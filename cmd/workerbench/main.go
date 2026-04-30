package main

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"os"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/internal/worker"
	"github.com/sanbei101/im/pkg/config"
)

var (
	processedCount atomic.Int64
	errorCount     atomic.Int64
)

const (
	MessageCount  = 500000
	BatchSize     = 10000
	pushBatchSize = 10000
)

func main() {
	cfg := config.NewTest()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	redis := db.NewRedis(cfg)
	redis.InitStreamGroups(ctx)
	woker := worker.New(cfg)
	cpuFile, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal().Err(err).Msg("create cpu profile failed")
	}
	defer cpuFile.Close()
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		log.Fatal().Err(err).Msg("start cpu profile failed")
	}
	defer pprof.StopCPUProfile()

	fmt.Println("Pre-populating messages:inbound stream...")
	prepopulateStart := time.Now()
	msgs := make([]*db.Message, 0, pushBatchSize)
	for i := range MessageCount {
		msgID, _ := uuid.NewV7()
		clientID, _ := uuid.NewV7()
		senderID, _ := uuid.NewV7()
		receiverID, _ := uuid.NewV7()
		msg := db.Message{
			MsgID:       msgID,
			ClientMsgID: clientID,
			SenderID:    senderID,
			RoomID:      receiverID,
			MsgType:     db.MessageTypeText,
			ServerTime:  time.Now().UnixNano(),
			Payload:     jsontext.Value(fmt.Sprintf(`{"text": "bench message %d"}`, i)),
		}
		msgs = append(msgs, &msg)

		if len(msgs) >= pushBatchSize {
			redis.GatewayPushMessage(ctx, msgs)
			msgs = msgs[:0]
		}
	}
	if len(msgs) > 0 {
		redis.GatewayPushMessage(ctx, msgs)
	}
	fmt.Printf("Pre-populated %d messages in %v\n", MessageCount, time.Since(prepopulateStart))
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := woker.ProcessInbound(ctx, BatchSize)
				if err != nil {
					log.Error().Err(err).Msg("worker process inbound failed")
					errorCount.Add(BatchSize)
				}
				processedCount.Add(BatchSize)
			}
		}
	}()
	fmt.Println("Waiting for processing to complete...")
	startTime := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastProcessed int64
	for {
		<-ticker.C
		currentProcessed := processedCount.Load()
		currentErrors := errorCount.Load()
		log.Info().
			Int64("processed", currentProcessed).
			Int64("errors", currentErrors).
			Int64("处理速率 msg/s", currentProcessed-lastProcessed).
			Msg("当前速率")
		lastProcessed = currentProcessed
		if currentProcessed+currentErrors >= int64(MessageCount) {
			cancel()
			break
		}
	}

	elapsed := time.Since(startTime)

	pprof.StopCPUProfile()

	memFile, err := os.Create("mem.prof")
	if err != nil {
		log.Error().Err(err).Msg("create mem profile failed")
	} else {
		if err := pprof.WriteHeapProfile(memFile); err != nil {
			log.Error().Err(err).Msg("write heap profile failed")
		}
		memFile.Close()
	}

	fmt.Printf("\n--- Bench Results ---\n")
	fmt.Printf("Total messages: %d\n", MessageCount)
	fmt.Printf("Processed: %d\n", processedCount.Load())
	fmt.Printf("Errors: %d\n", errorCount.Load())
	fmt.Printf("Elapsed: %v\n", elapsed)
	fmt.Printf("Worker Bench: %d messages, batch size %d\n", MessageCount, BatchSize)
}
