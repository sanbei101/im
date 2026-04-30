package bench

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/internal/worker"
	"github.com/sanbei101/im/pkg/config"
)

func BenchmarkWorkerProcessInbound(b *testing.B) {
	cfg := config.NewTest()
	ctx := context.Background()

	redisStore := db.NewRedis(cfg)
	redisStore.InitStreamGroups(ctx)
	w := worker.New(cfg)

	batchSize := 10000

	for b.Loop() {
		b.StopTimer()

		msgs := make([]*db.Message, 0, batchSize)
		for j := range batchSize {
			msgID := uuid.Must(uuid.NewV7())
			clientID := uuid.Must(uuid.NewV7())
			senderID := uuid.Must(uuid.NewV7())
			receiverID := uuid.Must(uuid.NewV7())
			msg := db.Message{
				MsgID:       msgID,
				ClientMsgID: clientID,
				SenderID:    senderID,
				RoomID:      receiverID,
				MsgType:     db.MessageTypeText,
				ServerTime:  time.Now().UnixNano(),
				Payload:     jsontext.Value(fmt.Sprintf("{\"text\": \"bench message %d\"}", j)),
			}
			msgs = append(msgs, &msg)
		}
		redisStore.GatewayPushMessage(ctx, msgs)

		b.StartTimer()

		err := w.ProcessInbound(ctx, int64(batchSize))
		if err != nil {
			b.Fatalf("ProcessInbound failed: %v", err)
		}
	}
}
