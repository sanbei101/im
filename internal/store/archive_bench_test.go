package store

import (
	"context"
	"encoding/json/jsontext"
	"testing"
	"time"
	"uuid"

	"github.com/sanbei101/im/pkg/config"
)

func BenchmarkArchiveParquetCodec(b *testing.B) {
	rows := make([]ParquetMessage, 2000)
	for i := range rows {
		rows[i] = newParquetMessage(&Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: uint64(i + 1), ServerTime: int64(i), MsgType: MsgTypeText,
			Payload: jsontext.Value(`{"text":"parquet archive benchmark payload"}`),
		})
	}

	sample, err := encodeParquet(rows)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(sample)))
		for b.Loop() {
			if _, err := encodeParquet(rows); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(sample)))
		for b.Loop() {
			decoded, err := decodeParquet(sample)
			if err != nil || len(decoded) != len(rows) {
				b.Fatalf("decode parquet: rows=%d err=%v", len(decoded), err)
			}
		}
	})
}
func newBenchObjectStore(b *testing.B) *MinioObjectStore {
	b.Helper()
	cfg := config.NewTest()
	objects, err := NewMinioObjectStore(
		cfg.Storage.Endpoint, cfg.Storage.Bucket,
		cfg.Storage.AccessKeyID, cfg.Storage.SecretAccessKey, cfg.Storage.UseSSL,
	)
	if err != nil {
		b.Fatalf("create minio client for %s: %v", cfg.Storage.Endpoint, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := objects.EnsureBucket(ctx); err != nil {
		b.Skipf("object store %s unreachable, start it with `make test`: %v", cfg.Storage.Endpoint, err)
	}
	return objects
}

func BenchmarkStoreArchivedRead(b *testing.B) {
	objects := newBenchObjectStore(b)
	ctx := context.Background()
	s := newBenchStore(b)
	if err := s.AttachRemote(objects, b.TempDir(), 64<<20); err != nil {
		b.Fatal(err)
	}
	room, sender := benchRoom(b, s)
	messages := benchWrite(b, s, room, sender, 2000)
	params := ArchiveParams{MaxAge: -time.Hour, MaxMessages: 2000, MaxBytes: 32 << 20, Prefix: "archives"}
	if err := s.ArchiveOnce(ctx, objects, params); err != nil {
		b.Fatal(err)
	}

	b.Run("page", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			page, err := s.Messages(ctx, room, 1000, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("archived page: count=%d err=%v", len(page.Messages), err)
			}
		}
	})
	b.Run("by-id", func(b *testing.B) {
		target := messages[500].MsgID
		b.ReportAllocs()
		for b.Loop() {
			if _, err := s.MessageByID(ctx, room, target); err != nil {
				b.Fatal(err)
			}
		}
	})
}
