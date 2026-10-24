package engine

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/sanbei101/im/pkg/config"
)

func newTestObjectStore(tb testing.TB) *MinioObjectStore {
	tb.Helper()
	cfg := config.NewTest()
	objects, err := NewMinioObjectStore(ObjectStoreConfig{
		Endpoint:  cfg.Storage.Endpoint,
		Bucket:    cfg.Storage.Bucket,
		AccessKey: cfg.Storage.AccessKey,
		SecretKey: cfg.Storage.SecretKey,
		Region:    cfg.Storage.Region,
		UseSSL:    cfg.Storage.UseSSL,
		PathStyle: cfg.Storage.PathStyle,
	})
	if err != nil {
		tb.Fatalf("create minio client for %s: %v", cfg.Storage.Endpoint, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := objects.EnsureBucket(ctx); err != nil {
		tb.Skipf("object store %s unreachable, start it with `make test-storage`: %v", cfg.Storage.Endpoint, err)
	}
	return objects
}

func TestManifestCodec(t *testing.T) {
	want := ArchiveManifest{
		EndSeq: 42, MsgCount: 7, MinTime: -1000, MaxTime: 2000,
		S3Key: "archives/rooms/abc/000001_000042.parquet",
	}

	t.Run("round trip", func(t *testing.T) {
		got, err := decodeManifest(encodeManifest(want))
		if err != nil || got != want {
			t.Fatalf("manifest round trip: got=%+v err=%v", got, err)
		}
	})

	t.Run("corrupt manifest rejected", func(t *testing.T) {
		if _, err := decodeManifest([]byte("short")); err == nil {
			t.Fatal("expected error for corrupt manifest")
		}
	})
}

func TestStoreArchive(t *testing.T) {
	objects := newTestObjectStore(t)
	ctx := context.Background()

	t.Run("cold reads across the archive boundary", func(t *testing.T) {
		e := newTestEngine(t)
		if err := e.AttachRemote(objects, t.TempDir(), 8<<20); err != nil {
			t.Fatal(err)
		}
		sender := uuid.NewV7()
		room := uuid.NewV7()
		written := make([]Message, 10)
		for i := range written {
			written[i] = mustWriteMessage(t, e, room, sender, fmt.Sprintf("m%d", i))
		}

		params := ArchiveParams{
			MaxAge:      -time.Hour, // everything is instantly eligible
			MaxMessages: 4, MaxBytes: 1 << 20, Prefix: "archives",
		}
		if err := e.ArchiveOnce(ctx, objects, params); err != nil {
			t.Fatal(err)
		}
		watermark, err := e.archiveWatermark(room)
		if err != nil || watermark != 4 {
			t.Fatalf("watermark after first cycle: seq=%d err=%v", watermark, err)
		}

		page, err := e.Messages(ctx, room, 0, 20)
		if err != nil || len(page.Messages) != 10 {
			t.Fatalf("messages spanning tiers: page=%+v err=%v", page, err)
		}
		for i, msg := range page.Messages {
			if want := uint64(10 - i); msg.RoomSeq != want {
				t.Fatalf("mixed tier page broken at %d: seq=%d want %d", i, msg.RoomSeq, want)
			}
		}
		cold, err := e.MessageByID(ctx, room, written[0].MsgID)
		if err != nil || cold.RoomSeq != written[0].RoomSeq || cold.SenderID != sender {
			t.Fatalf("archived message by id: msg=%+v err=%v", cold, err)
		}

		// MaxMessages bounds every cycle to 4: the second cycle archives
		// seqs 5..8, the third drains the remaining tail, the fourth is a no-op.
		for cycle, want := range []uint64{8, 10, 10} {
			if err := e.ArchiveOnce(ctx, objects, params); err != nil {
				t.Fatal(err)
			}
			watermark, err := e.archiveWatermark(room)
			if err != nil || watermark != want {
				t.Fatalf("watermark after cycle %d: seq=%d err=%v", cycle+2, watermark, err)
			}
		}
		if drained, err := e.Messages(ctx, room, 0, 20); err != nil || len(drained.Messages) != 10 {
			t.Fatalf("fully archived room: page=%+v err=%v", drained, err)
		}

		// The body left Pebble: nobody can recall it, not even an owner.
		if _, err := e.RecallMessage(
			ctx,
			room,
			written[3].MsgID,
			sender,
			true,
		); !errors.Is(
			err,
			ErrRecallTimeout,
		) {
			t.Fatalf("recall of archived message error = %v, want %v", err, ErrRecallTimeout)
		}
	})

	t.Run("archived message is unresolvable without remote", func(t *testing.T) {
		e := newTestEngine(t)
		sender := uuid.NewV7()
		room := uuid.NewV7()
		msg := mustWriteMessage(t, e, room, sender, "will go cold")
		params := ArchiveParams{MaxAge: -time.Hour, MaxMessages: 10, MaxBytes: 1 << 20, Prefix: "archives"}
		if err := e.ArchiveOnce(ctx, objects, params); err != nil {
			t.Fatal(err)
		}
		// No AttachRemote: the object store tier is not wired up yet.
		if _, err := e.MessageByID(ctx, room, msg.MsgID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("archived message without remote error = %v, want %v", err, ErrNotFound)
		}
		if _, err := e.RecallMessage(ctx, room, msg.MsgID, sender, false); !errors.Is(err, ErrNotFound) {
			t.Fatalf("recall without remote error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("disk cache evicts and survives stale tmp files", func(t *testing.T) {
		e := newTestEngine(t)
		cacheDir := t.TempDir()
		stale := filepath.Join(cacheDir, "deadbeef.tmp")
		if err := os.WriteFile(stale, []byte("crashed upload"), 0o644); err != nil {
			t.Fatal(err)
		}
		// A one-byte budget forces eviction on every cached file: reads must
		// still be correct by re-fetching from MinIO.
		if err := e.AttachRemote(objects, cacheDir, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale tmp file error = %v, want removed", err)
		}

		sender := uuid.NewV7()
		room := uuid.NewV7()
		written := make([]Message, 3)
		for i := range written {
			written[i] = mustWriteMessage(t, e, room, sender, "evict me")
		}
		params := ArchiveParams{MaxAge: -time.Hour, MaxMessages: 10, MaxBytes: 1 << 20, Prefix: "archives"}
		if err := e.ArchiveOnce(ctx, objects, params); err != nil {
			t.Fatal(err)
		}
		page, err := e.Messages(ctx, room, 0, 20)
		if err != nil || len(page.Messages) != 3 {
			t.Fatalf("messages after eviction: page=%+v err=%v", page, err)
		}
		for _, msg := range written {
			got, err := e.MessageByID(ctx, room, msg.MsgID)
			if err != nil || got.MsgID != msg.MsgID {
				t.Fatalf("cold read after eviction: msg=%+v err=%v", got, err)
			}
		}
	})

	t.Run("owns filter skips foreign rooms", func(t *testing.T) {
		e := newTestEngine(t)
		if err := e.AttachRemote(objects, t.TempDir(), 8<<20); err != nil {
			t.Fatal(err)
		}
		sender := uuid.NewV7()
		room := uuid.NewV7()
		mustWriteMessage(t, e, room, sender, "not mine")

		params := ArchiveParams{
			Owns:   func(uuid.UUID) bool { return false },
			MaxAge: -time.Hour, MaxMessages: 10, MaxBytes: 1 << 20, Prefix: "archives",
		}
		if err := e.ArchiveOnce(ctx, objects, params); err != nil {
			t.Fatal(err)
		}
		watermark, err := e.archiveWatermark(room)
		if err != nil || watermark != 0 {
			t.Fatalf("foreign room watermark: seq=%d err=%v", watermark, err)
		}
	})

	t.Run("invalid params rejected", func(t *testing.T) {
		e := newTestEngine(t)
		params := ArchiveParams{MaxAge: time.Hour, MaxMessages: 0, MaxBytes: 1 << 20}
		if err := e.ArchiveOnce(ctx, objects, params); err == nil {
			t.Fatal("expected error for MaxMessages=0")
		}
	})

	t.Run("object store put get and presign", func(t *testing.T) {
		key := "archives/rooms/probe/probe.bin"
		payload := []byte("archive probe payload")
		if err := objects.Put(ctx, key, bytes.NewReader(payload), int64(len(payload))); err != nil {
			t.Fatal(err)
		}
		got, err := objects.Get(ctx, key)
		if err != nil || !bytes.Equal(got, payload) {
			t.Fatalf("get object: data=%q err=%v", got, err)
		}

		// The media upload path: clients PUT straight to the presigned URL.
		url, err := objects.PresignPut(ctx, "media/probe.bin", time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		request, err := http.NewRequest(http.MethodPut, url.String(), bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("presigned upload status = %d", response.StatusCode)
		}
		uploaded, err := objects.Get(ctx, "media/probe.bin")
		if err != nil || !bytes.Equal(uploaded, payload) {
			t.Fatalf("presigned upload read back: data=%q err=%v", uploaded, err)
		}
	})
}

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
		b.ResetTimer()
		b.ReportAllocs()
		b.SetBytes(int64(len(sample)))
		for b.Loop() {
			if _, err := encodeParquet(rows); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("decode", func(b *testing.B) {
		b.ResetTimer()
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

func BenchmarkStoreArchivedRead(b *testing.B) {
	objects := newTestObjectStore(b)
	ctx := context.Background()
	e := newTestEngine(b)
	if err := e.AttachRemote(objects, b.TempDir(), 64<<20); err != nil {
		b.Fatal(err)
	}
	room, sender := benchIDs()
	messages := benchWrite(b, e, room, sender, 2000)
	params := ArchiveParams{MaxAge: -time.Hour, MaxMessages: 2000, MaxBytes: 32 << 20, Prefix: "archives"}
	if err := e.ArchiveOnce(ctx, objects, params); err != nil {
		b.Fatal(err)
	}

	b.Run("page", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			page, err := e.Messages(ctx, room, 1000, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("archived page: count=%d err=%v", len(page.Messages), err)
			}
		}
	})
	b.Run("by-id", func(b *testing.B) {
		target := messages[500].MsgID
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			if _, err := e.MessageByID(ctx, room, target); err != nil {
				b.Fatal(err)
			}
		}
	})
}
