// bench.go - single-file Go benchmark for im
//
// Usage:
//
//	go run ./bench TARGET_HOST=10.0.0.1
//
// Env:
//
//	TARGET_HOST  server IP/host (default: 127.0.0.1)
//	API_PORT     API port (default: 8801)
//	WS_PORT      WebSocket port (default: 8800)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
)

// ===================== 1. 配置常量 =====================

const (
	SingleRoomNum = 500
	GroupRoomNum1 = 100
	GroupRoomNum2 = 100
	SendInterval  = 500 * time.Millisecond
	RunDuration   = 30 * time.Second
	DialTimeout   = 10 * time.Second
	WriteTimeout  = 5 * time.Second
	ReadTimeout   = 60 * time.Second
)

var GroupRoom = []int{GroupRoomNum1, GroupRoomNum2}

// ===================== 2. 类型定义 =====================

type BenchMockUserInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type SingleRoomResp struct {
	RoomID string              `json:"room_id"`
	Users  []BenchMockUserInfo `json:"users"`
}

type GroupRoomResp struct {
	RoomID   string              `json:"room_id"`
	RoomSize int                 `json:"room_size"`
	Users    []BenchMockUserInfo `json:"users"`
}

type BatchMockResp struct {
	SingleRooms  []SingleRoomResp `json:"single_rooms"`
	GroupRooms   []GroupRoomResp  `json:"group_rooms"`
	TotalUserNum int              `json:"total_user_num"`
}

type VuConfig struct {
	User   BenchMockUserInfo
	RoomID string
	Type   string // "single" | "group"
}

type apiResponse[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

// ===================== 3. 指标 =====================

// Trend 收集所有延迟样本并在最后输出 count / min / avg / p50 / p95 / p99 / max。
type trend struct {
	mu      sync.Mutex
	samples []time.Duration
}

func (t *trend) add(d time.Duration) {
	t.mu.Lock()
	t.samples = append(t.samples, d)
	t.mu.Unlock()
}

func (t *trend) summary() (count int, minVal, avg, p50, p95, p99, maxVal time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	count = len(t.samples)
	if count == 0 {
		return count, minVal, avg, p50, p95, p99, maxVal
	}
	clone := make([]time.Duration, count)
	copy(clone, t.samples)
	slices.Sort(clone)

	var sum time.Duration
	for _, s := range clone {
		sum += s
	}
	avg = sum / time.Duration(count)
	pick := func(q float64) time.Duration {
		idx := int(float64(count-1) * q)
		return clone[idx]
	}
	minVal, maxVal = clone[0], clone[count-1]
	p50 = pick(0.50)
	p95 = pick(0.95)
	p99 = pick(0.99)
	return count, minVal, avg, p50, p95, p99, maxVal
}

// ===================== 4. Setup:拉取 mock 数据 =====================

func apiBase() string {
	host := getEnv("TARGET_HOST", "127.0.0.1")
	return fmt.Sprintf("http://%s:%s", host, getEnv("API_PORT", "8801"))
}

func wsURL() string {
	host := getEnv("TARGET_HOST", "127.0.0.1")
	return fmt.Sprintf("ws://%s:%s/ws", host, getEnv("WS_PORT", "8800"))
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func createBenchMock(ctx context.Context) (*BatchMockResp, error) {
	body, _ := json.Marshal(map[string]any{
		"single_room_num": SingleRoomNum,
		"group_room":      GroupRoom,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, apiBase()+"/api/v1/bench/mock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("createBenchMock: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("createBenchMock: status=%d body=%s", resp.StatusCode, string(b))
	}

	var wrapper apiResponse[BatchMockResp]
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("createBenchMock decode: %w", err)
	}
	return &wrapper.Data, nil
}

func flattenMockData(b *BatchMockResp) []VuConfig {
	out := make([]VuConfig, 0, b.TotalUserNum)
	for _, r := range b.SingleRooms {
		for _, u := range r.Users {
			out = append(out, VuConfig{User: u, RoomID: r.RoomID, Type: "single"})
		}
	}
	for _, r := range b.GroupRooms {
		for _, u := range r.Users {
			out = append(out, VuConfig{User: u, RoomID: r.RoomID, Type: "group"})
		}
	}
	return out
}

// ===================== 5. VU 主循环 =====================

// runVU 模拟单个 VU: 建立 WebSocket, 每 500ms 发一条 SendMessageReq,
// 解析回包并匹配 client_msg_id 记录延迟, 关闭时把未匹配的累积到计数器.
func runVU(ctx context.Context, vuIndex int, cfg *VuConfig, latency *trend, unmatched *atomic.Int64) {
	dialCtx, cancel := context.WithTimeout(ctx, DialTimeout)
	defer cancel()

	url := wsURL() + "?token=" + cfg.User.Token
	conn, resp, err := websocket.Dial(dialCtx, url, nil)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Printf("[VU%04d] dial: %v", vuIndex+1, err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")

	conn.SetReadLimit(1 << 20) // 1 MiB

	// pending: clientMsgId -> send time
	pending := sync.Map{}

	// 读循环:goroutine 与 ticker 并发
	readErr := make(chan error, 1)
	go func() {
		defer close(readErr)
		for {
			mt, data, err := conn.Read(ctx)
			if err != nil {
				readErr <- err
				return
			}
			if mt != websocket.MessageBinary {
				continue
			}
			// 服务端 ack 走 SendMessageAck (字段 1 是 client_msg_id);
			// 错误路径也用 SendMessageAck, code!=0 时 err_msg 有值.
			var ack imv1.SendMessageAck
			if err := ack.UnmarshalVT(data); err != nil {
				continue
			}
			id := ack.GetClientMsgId()
			if id == "" {
				continue
			}
			if v, ok := pending.LoadAndDelete(id); ok {
				latency.add(time.Since(v.(time.Time)))
			}
		}
	}()

	// 写循环:按 SendInterval 定时发
	ticker := time.NewTicker(SendInterval)
	defer ticker.Stop()

	vuLabel := fmt.Sprintf("[VU%04d] ", vuIndex+1)
	sent := 0
	for {
		select {
		case <-ctx.Done():
			n := 0
			pending.Range(func(_, _ any) bool { n++; return true })
			unmatched.Add(int64(n))
			conn.Close(websocket.StatusNormalClosure, "ctx done")
			<-readErr
			return
		case <-ticker.C:
			id := uuid.Must(uuid.NewV7()).String()
			text := vuLabel + "hello"
			now := time.Now()
			pending.Store(id, now)

			payloadBytes, err := (&imv1.TextMessagePayload{
				Text:      &text,
				AtUserIds: nil,
			}).MarshalVT()
			if err != nil {
				log.Printf("%smarshal text: %v", vuLabel, err)
				pending.Delete(id)
				continue
			}
			req := &imv1.SendMessageReq{
				ClientMsgId: &id,
				RoomId:      &cfg.RoomID,
				MsgType:     new(imv1.MessageType_MESSAGE_TYPE_TEXT),
				Payload:     payloadBytes,
			}
			data, err := req.MarshalVT()
			if err != nil {
				log.Printf("%smarshal req: %v", vuLabel, err)
				pending.Delete(id)
				continue
			}

			wctx, wcancel := context.WithTimeout(ctx, WriteTimeout)
			if err := conn.Write(wctx, websocket.MessageBinary, data); err != nil {
				wcancel()
				log.Printf("%swrite: %v", vuLabel, err)
				conn.Close(websocket.StatusGoingAway, "write err")
				<-readErr
				n := 0
				pending.Range(func(_, _ any) bool { n++; return true })
				unmatched.Add(int64(n))
				return
			}
			wcancel()
			sent++

			if sent%20 == 0 {
				n := 0
				pending.Range(func(_, _ any) bool { n++; return true })
				log.Printf("%ssent=%d in_flight=%d", vuLabel, sent, n)
			}
		case err := <-readErr:
			if err != nil {
				log.Printf("%sread: %v", vuLabel, err)
			}
			n := 0
			pending.Range(func(_, _ any) bool { n++; return true })
			unmatched.Add(int64(n))
			return
		}
	}
}

// ===================== 6. 入口 =====================

func main() {
	host := getEnv("TARGET_HOST", "127.0.0.1")
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("bench starting, target=%s duration=%s", host, RunDuration)

	setupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	batch, err := createBenchMock(setupCtx)
	cancel()
	if err != nil {
		log.Fatalf("setup failed: %v", err)
	}

	allVUs := flattenMockData(batch)
	if len(allVUs) < batch.TotalUserNum {
		log.Fatalf("expected %d users, got %d", batch.TotalUserNum, len(allVUs))
	}
	log.Printf("setup ok: %d single rooms, %d group rooms, %d users",
		len(batch.SingleRooms), len(batch.GroupRooms), len(allVUs))

	latency := &trend{}
	unmatched := &atomic.Int64{}

	runCtx, runCancel := context.WithTimeout(context.Background(), RunDuration)
	defer runCancel()

	var wg sync.WaitGroup
	for i := range allVUs {
		wg.Add(1)
		go func(i int, cfg *VuConfig) {
			defer wg.Done()
			runVU(runCtx, i, cfg, latency, unmatched)
		}(i, &allVUs[i])
	}

	// 等待:ctx 到期后 VU 也会退出
	<-runCtx.Done()
	wg.Wait()

	// 汇总
	count, minVal, avg, p50, p95, p99, maxVal := latency.summary()
	fmt.Println("================ bench result ================")
	fmt.Printf("ws_msg_unmatched : %d\n", unmatched.Load())
	fmt.Printf("ws_msg_latency   : count=%d min=%s avg=%s max=%s p50=%s p95=%s p99=%s\n",
		count, minVal, avg, maxVal, p50, p95, p99)
	fmt.Println("==============================================")
}
