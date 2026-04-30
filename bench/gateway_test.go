package bench

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/internal/gateway"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/jwt"
)

func startMockWorker(rdb *db.Redis) {
	ctx := context.Background()
	for {
		msgs, err := rdb.WorkerPullMessage(ctx, 100)
		if err != nil || len(msgs) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		msgIDS := make([]string, 0, len(msgs))
		messages := make([]*db.Message, 0, len(msgs))
		for _, msg := range msgs {
			msgIDS = append(msgIDS, msg.ID)
			messages = append(messages, msg.Data)
		}
		rdb.WorkerPushMessage(ctx, messages)
		rdb.WorkerAckMessage(ctx, msgIDS...)
	}
}

func BenchmarkGatewayMessageSend(b *testing.B) {
	cfg := config.NewTest()
	rdb := db.NewRedis(cfg)
	rdb.InitStreamGroups(context.Background())

	gw := gateway.New(cfg)
	ctx := b.Context()

	go gw.HandleWorkerMessages(ctx)
	go startMockWorker(rdb)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", gw.HandleUserMessage)
	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"

	userCount := 1000
	var users []uuid.UUID
	conns := make([]*websocket.Conn, userCount)

	for range userCount {
		u := uuid.Must(uuid.NewV7())
		users = append(users, u)
	}

	var errCount atomic.Int64
	var receivedCount atomic.Int64

	var wgConnect sync.WaitGroup
	wgConnect.Add(userCount)
	sem := make(chan struct{}, 100)

	for i := range userCount {
		sem <- struct{}{}
		go func(idx int) {
			defer wgConnect.Done()
			defer func() { <-sem }()

			token, _ := jwt.GenerateToken(users[idx].String())
			dialCtx, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer dialCancel()

			c, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
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

	if errCount.Load() > 0 {
		b.Fatalf("Failed to establish %d connections", errCount.Load())
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			senderIdx := rand.Intn(userCount)
			receiverIdx := rand.Intn(userCount)
			conn := conns[senderIdx]

			msg := db.Message{
				ClientMsgID: uuid.New(),
				RoomID:      users[receiverIdx],
				Payload:     jsontext.Value(fmt.Sprintf("{\"text\": \"Bench message to %d\"}", receiverIdx)),
			}
			bin, _ := json.Marshal(msg)

			err := conn.Write(context.Background(), websocket.MessageText, bin)
			if err != nil {
				b.Error("Write failed:", err)
			}
		}
	})

	b.StopTimer()
	for _, c := range conns {
		if c != nil {
			c.Close(websocket.StatusNormalClosure, "")
		}
	}
}
