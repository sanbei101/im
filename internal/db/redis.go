package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/phuslu/log"
	"github.com/redis/go-redis/v9"
	"github.com/sanbei101/im/pkg/config"
)

type StreamMessage struct {
	ID   string
	Data *Message
}

var MachineFingerprint = GenerateMachineFingerprint(16)
var WorkerName = "worker-" + MachineFingerprint
var GatewayName = "gateway-" + MachineFingerprint

const (
	MessageSteamInbound = "message:inbound"
	MessageSteamDeliver = "message:deliver"
	MessageWorkerGroup  = "worker_group"
	MessageGatewayGroup = "gateway_group"
	MessageMaxLen       = 1000000
)

type Redis struct {
	client *redis.Client
}

func NewRedis(conf *config.Config) *Redis {
	client := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Addr,
		Password: conf.Redis.Password,
		DB:       conf.Redis.DB,
		PoolSize: 50,
	})

	return &Redis{client: client}
}

func (r *Redis) InitStreamGroups(ctx context.Context) error {
	groups := map[string]string{
		MessageSteamInbound: MessageWorkerGroup,
		MessageSteamDeliver: MessageGatewayGroup,
	}
	for stream, group := range groups {
		err := r.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
		if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create group %s failed: %w", group, err)
		}
	}
	return nil
}

func (r *Redis) WorkerPullMessage(ctx context.Context, batch int64) ([]*StreamMessage, error) {
	return r.pullMessageFromStream(ctx, MessageSteamInbound, MessageWorkerGroup, WorkerName, batch)
}

func (r *Redis) GatewayPullMessage(ctx context.Context, batch int64) ([]*StreamMessage, error) {
	return r.pullMessageFromStream(ctx, MessageSteamDeliver, MessageGatewayGroup, GatewayName, batch)
}

func (r *Redis) WorkerPushMessage(ctx context.Context, messages []*Message) error {
	return r.pushMessageToStream(ctx, MessageSteamDeliver, messages)
}

func (r *Redis) GatewayPushMessage(ctx context.Context, messages []*Message) error {
	return r.pushMessageToStream(ctx, MessageSteamInbound, messages)
}

func (r *Redis) WorkerAckMessage(ctx context.Context, ids ...string) error {
	return r.ackMessages(ctx, MessageSteamInbound, MessageWorkerGroup, ids...)
}

func (r *Redis) GatewayAckMessage(ctx context.Context, ids ...string) error {
	return r.ackMessages(ctx, MessageSteamDeliver, MessageGatewayGroup, ids...)
}

func (r *Redis) pullMessageFromStream(ctx context.Context, stream, group, consumer string, batch int64) ([]*StreamMessage, error) {
	result, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    batch,
		Block:    5 * time.Second,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("xread group failed: %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}

	var messages []*StreamMessage
	for _, msg := range result[0].Messages {
		data, ok := msg.Values["data"].(string)
		if !ok {
			log.Error().Str("id", msg.ID).Msg("Missing 'data' field in stream message")
			continue
		}
		var m Message
		if err := json.Unmarshal([]byte(data), &m); err != nil {
			log.Error().Str("id", msg.ID).Str("raw", data).Msg("Failed to unmarshal message")
			continue
		}
		messages = append(messages, &StreamMessage{ID: msg.ID, Data: &m})
	}
	return messages, nil
}

func (r *Redis) pushMessageToStream(ctx context.Context, stream string, messages []*Message) error {
	if len(messages) == 0 {
		return nil
	}
	pipe := r.client.Pipeline()
	for _, msg := range messages {
		if msg == nil {
			log.Error().Msg("Skipping nil message pointer")
			continue
		}
		bin, err := json.Marshal(msg)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal message for stream")
			continue
		}
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			MaxLen: MessageMaxLen,
			Approx: true,
			Values: map[string]any{"data": bin},
		})
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Redis) ackMessages(ctx context.Context, stream, group string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.client.XAck(ctx, stream, group, ids...).Err()
}

func GenerateMachineFingerprint(truncateLen int) string {
	if truncateLen < 8 || truncateLen > 64 {
		truncateLen = 16
	}

	var parts []string

	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			id := strings.TrimSpace(string(data))
			if len(id) >= 32 {
				parts = append(parts, id[:32])
			}
		}
	}

	if mac := getPrimaryMAC(); mac != "" {
		parts = append(parts, mac)
	}

	if len(parts) == 0 {
		host, _ := os.Hostname()
		if host != "" {
			parts = append(parts, host)
		} else {
			parts = append(parts, "unknown")
		}
	}
	raw := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(raw))
	fullHex := hex.EncodeToString(hash[:])

	if truncateLen > len(fullHex) {
		truncateLen = len(fullHex)
	}
	return fullHex[:truncateLen]
}

func getPrimaryMAC() string {
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if strings.Contains(iface.Name, "docker") ||
			strings.Contains(iface.Name, "veth") ||
			strings.Contains(iface.Name, "br-") {
			continue
		}
		mac := iface.HardwareAddr.String()
		if mac != "" && mac != "00:00:00:00:00:00" {
			return strings.ReplaceAll(mac, ":", "")
		}
	}
	return ""
}
