package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/internal/gateway"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/jwt"
)

const (
	UserCount    = 10000
	MessageCount = 10
)

var (
	sentCount     atomic.Int64
	receivedCount atomic.Int64
	errCount      atomic.Int64
)

func startMockWorker(rdb *db.Redis) {
	ctx := context.Background()
	for {
		msgs, err := rdb.WorkerPullMessage(ctx, 100)
		if err != nil || len(msgs) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		var msgIDs []string
		var messages []*db.Message
		for _, msg := range msgs {
			msgIDs = append(msgIDs, msg.ID)
			messages = append(messages, msg.Data)
		}
		rdb.WorkerPushMessage(ctx, messages)
		rdb.WorkerAckMessage(ctx, msgIDs...)
	}
}

func main() {
	cfg := config.NewTest()
	rdb := db.NewRedis(cfg)
	rdb.InitStreamGroups(context.Background())

	gw := gateway.New(cfg)
	go gw.HandleWorkerMessages(context.Background())
	go startMockWorker(rdb)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", gw.HandleUserMessage)
	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"

	cpuFile, err := os.Create("cpu.prof")
	if err != nil {
		log.Error().Err(err).Msg("创建 CPU profile 文件失败")
		return
	}

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		log.Error().Err(err).Msg("启动 CPU profile 失败")
		cpuFile.Close()
		return
	}
	defer func() {
		pprof.StopCPUProfile()
		cpuFile.Close()
	}()

	fmt.Printf("正在生成 %d 个用户并建立连接...\n", UserCount)
	var users []uuid.UUID
	conns := make([]*websocket.Conn, UserCount)

	for range UserCount {
		u, _ := uuid.NewV7()
		users = append(users, u)
	}

	var wgConnect sync.WaitGroup
	wgConnect.Add(UserCount)

	sem := make(chan struct{}, 100)
	for i := range UserCount {
		sem <- struct{}{}
		go func(idx int) {
			defer wgConnect.Done()
			defer func() { <-sem }()

			token, _ := jwt.GenerateToken(users[idx].String())
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			c, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
				HTTPHeader: http.Header{"Authorization": []string{"Bearer " + token}},
			})
			if err != nil {
				errCount.Add(1)
				return
			}
			conns[idx] = c
			go func(conn *websocket.Conn) {
				for {
					_, _, err := conn.Read(context.Background())
					if err != nil {
						return
					}
					receivedCount.Add(1)
				}
			}(c)
		}(i)
	}
	wgConnect.Wait()
	fmt.Printf("成功建立 %d 个连接,失败 %d 个\n", UserCount-int(errCount.Load()), errCount.Load())

	fmt.Println("开始压测:互相发送消息...")
	startTime := time.Now()
	var lastSentCount int64
	var lastReceivedCount int64
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				currentSent := sentCount.Load()
				currentReceived := receivedCount.Load()
				log.Info().
					Int64("发送速率 msg/s", currentSent-lastSentCount).
					Int64("接收速率 msg/s", currentReceived-lastReceivedCount).
					Msg("当前速率")
				lastSentCount = currentSent
				lastReceivedCount = currentReceived
			case <-ctx.Done():
				return
			}
		}
	}()
	var wgSend sync.WaitGroup
	wgSend.Add(UserCount)

	for i := range UserCount {
		go func(senderIdx int) {
			defer wgSend.Done()
			conn := conns[senderIdx]
			if conn == nil {
				return
			}

			for j := range MessageCount {
				receiverIdx := rand.Intn(UserCount)

				msg := db.Message{
					ClientMsgID: uuid.New(),
					RoomID:      users[receiverIdx],
					ChatType:    db.ChatTypeSingle,
					Payload:     jsontext.Value(fmt.Sprintf(`{"text": "Hello from user %d to user %d, msg %d"}`, senderIdx, receiverIdx, j)),
				}
				bin, _ := json.Marshal(msg)

				err := conn.Write(context.Background(), websocket.MessageText, bin)
				if err != nil {
					errCount.Add(1)
				} else {
					sentCount.Add(1)
				}
				time.Sleep(1000 * time.Millisecond)
			}
		}(i)
	}

	wgSend.Wait()
	cancel()

	time.Sleep(2 * time.Second)

	elapsed := time.Since(startTime)

	pprof.StopCPUProfile()

	memFile, err := os.Create("mem.prof")
	if err != nil {
		log.Error().Err(err).Msg("创建内存 profile 文件失败")
	} else {
		if err := pprof.WriteHeapProfile(memFile); err != nil {
			log.Error().Err(err).Msg("写入内存 profile 失败")
		}
		memFile.Close()
		log.Info().Msg("已生成 mem.prof 文件")
	}

	fmt.Printf("--- 压测结果 ---\n")
	fmt.Printf("耗时: %v\n", elapsed)
	fmt.Printf("发送消息: %d\n", sentCount.Load())
	fmt.Printf("接收消息: %d\n", receivedCount.Load())
	fmt.Printf("错误数量: %d\n", errCount.Load())
}
