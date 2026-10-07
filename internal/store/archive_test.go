package store

import (
	"bytes"
	"context"
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

func newTestObjectStore(t *testing.T) *MinioObjectStore {
	t.Helper()
	cfg := config.NewTest()
	objects, err := NewMinioObjectStore(
		cfg.Storage.Endpoint, cfg.Storage.Bucket,
		cfg.Storage.AccessKeyID, cfg.Storage.SecretAccessKey, cfg.Storage.UseSSL,
	)
	if err != nil {
		t.Fatalf("create minio client for %s: %v", cfg.Storage.Endpoint, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := objects.EnsureBucket(ctx); err != nil {
		t.Skipf("object store %s unreachable, start it with `make test-storage`: %v", cfg.Storage.Endpoint, err)
	}
	return objects
}

func TestManifestCodec(t *testing.T) {
	want := ArchiveManifest{
		EndSeq: 42, MsgCount: 7, MinTime: -1000, MaxTime: 2000,
		S3Key: "archives/rooms/abc/000001_000042.parquet",
	}
	got, err := decodeManifest(encodeManifest(want))
	if err != nil || got != want {
		t.Fatalf("manifest round trip: got=%+v err=%v", got, err)
	}
	if _, err := decodeManifest([]byte("short")); err == nil {
		t.Fatal("expected error for corrupt manifest")
	}
}

func TestStoreArchive(t *testing.T) {
	objects := newTestObjectStore(t)
	ctx := context.Background()

	t.Run("cold reads across the archive boundary", func(t *testing.T) {
		s := newTestStore(t)
		if err := s.AttachRemote(objects, t.TempDir(), 8<<20); err != nil {
			t.Fatal(err)
		}
		sender := mustCreateUser(t, s, "archive-sender")
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		written := make([]Message, 10)
		for i := range written {
			written[i] = mustWriteMessage(t, s, room.RoomID, sender.UserID, fmt.Sprintf("m%d", i))
		}

		params := ArchiveParams{
			MaxAge:      -time.Hour, // everything is instantly eligible
			MaxMessages: 4, MaxBytes: 1 << 20, Prefix: "archives",
		}
		if err := s.ArchiveOnce(ctx, objects, params); err != nil {
			t.Fatal(err)
		}
		watermark, err := s.archiveWatermark(room.RoomID)
		if err != nil || watermark != 4 {
			t.Fatalf("watermark after first cycle: seq=%d err=%v", watermark, err)
		}

		page, err := s.Messages(ctx, room.RoomID, 0, 20)
		if err != nil || len(page.Messages) != 10 {
			t.Fatalf("messages spanning tiers: page=%+v err=%v", page, err)
		}
		for i, msg := range page.Messages {
			if want := uint64(10 - i); msg.RoomSeq != want {
				t.Fatalf("mixed tier page broken at %d: seq=%d want %d", i, msg.RoomSeq, want)
			}
		}
		cold, err := s.MessageByID(ctx, room.RoomID, written[0].MsgID)
		if err != nil || cold.RoomSeq != written[0].RoomSeq || cold.SenderID != sender.UserID {
			t.Fatalf("archived message by id: msg=%+v err=%v", cold, err)
		}

		// MaxMessages bounds every cycle to 4: the second cycle archives
		// seqs 5..8, the third drains the remaining tail, the fourth is a no-op.
		for cycle, want := range []uint64{8, 10, 10} {
			if err := s.ArchiveOnce(ctx, objects, params); err != nil {
				t.Fatal(err)
			}
			watermark, err := s.archiveWatermark(room.RoomID)
			if err != nil || watermark != want {
				t.Fatalf("watermark after cycle %d: seq=%d err=%v", cycle+2, watermark, err)
			}
		}
		if drained, err := s.Messages(ctx, room.RoomID, 0, 20); err != nil || len(drained.Messages) != 10 {
			t.Fatalf("fully archived room: page=%+v err=%v", drained, err)
		}

		// The body left Pebble: nobody can recall it, not even an owner.
		if _, err := s.RecallMessage(
			ctx,
			room.RoomID,
			written[3].MsgID,
			sender.UserID,
			true,
		); !errors.Is(
			err,
			ErrRecallTimeout,
		) {
			t.Fatalf("recall of archived message error = %v, want %v", err, ErrRecallTimeout)
		}
	})

	t.Run("archived message is unresolvable without remote", func(t *testing.T) {
		s := newTestStore(t)
		sender := mustCreateUser(t, s, "cold-sender")
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		msg := mustWriteMessage(t, s, room.RoomID, sender.UserID, "will go cold")
		params := ArchiveParams{MaxAge: -time.Hour, MaxMessages: 10, MaxBytes: 1 << 20, Prefix: "archives"}
		if err := s.ArchiveOnce(ctx, objects, params); err != nil {
			t.Fatal(err)
		}
		// No AttachRemote: the object store tier is not wired up yet.
		if _, err := s.MessageByID(ctx, room.RoomID, msg.MsgID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("archived message without remote error = %v, want %v", err, ErrNotFound)
		}
		if _, err := s.RecallMessage(ctx, room.RoomID, msg.MsgID, sender.UserID, false); !errors.Is(err, ErrNotFound) {
			t.Fatalf("recall without remote error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("disk cache evicts and survives stale tmp files", func(t *testing.T) {
		s := newTestStore(t)
		cacheDir := t.TempDir()
		stale := filepath.Join(cacheDir, "deadbeef.tmp")
		if err := os.WriteFile(stale, []byte("crashed upload"), 0o644); err != nil {
			t.Fatal(err)
		}
		// A one-byte budget forces eviction on every cached file: reads must
		// still be correct by re-fetching from MinIO.
		if err := s.AttachRemote(objects, cacheDir, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale tmp file error = %v, want removed", err)
		}

		sender := mustCreateUser(t, s, "evict-sender")
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		written := make([]Message, 3)
		for i := range written {
			written[i] = mustWriteMessage(t, s, room.RoomID, sender.UserID, "evict me")
		}
		params := ArchiveParams{MaxAge: -time.Hour, MaxMessages: 10, MaxBytes: 1 << 20, Prefix: "archives"}
		if err := s.ArchiveOnce(ctx, objects, params); err != nil {
			t.Fatal(err)
		}
		page, err := s.Messages(ctx, room.RoomID, 0, 20)
		if err != nil || len(page.Messages) != 3 {
			t.Fatalf("messages after eviction: page=%+v err=%v", page, err)
		}
		for _, msg := range written {
			got, err := s.MessageByID(ctx, room.RoomID, msg.MsgID)
			if err != nil || got.MsgID != msg.MsgID {
				t.Fatalf("cold read after eviction: msg=%+v err=%v", got, err)
			}
		}
	})

	t.Run("owns filter skips foreign rooms", func(t *testing.T) {
		s := newTestStore(t)
		if err := s.AttachRemote(objects, t.TempDir(), 8<<20); err != nil {
			t.Fatal(err)
		}
		sender := mustCreateUser(t, s, "foreign-sender")
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		mustWriteMessage(t, s, room.RoomID, sender.UserID, "not mine")

		params := ArchiveParams{
			Owns:   func(uuid.UUID) bool { return false },
			MaxAge: -time.Hour, MaxMessages: 10, MaxBytes: 1 << 20, Prefix: "archives",
		}
		if err := s.ArchiveOnce(ctx, objects, params); err != nil {
			t.Fatal(err)
		}
		watermark, err := s.archiveWatermark(room.RoomID)
		if err != nil || watermark != 0 {
			t.Fatalf("foreign room watermark: seq=%d err=%v", watermark, err)
		}
	})

	t.Run("invalid params rejected", func(t *testing.T) {
		s := newTestStore(t)
		params := ArchiveParams{MaxAge: time.Hour, MaxMessages: 0, MaxBytes: 1 << 20}
		if err := s.ArchiveOnce(ctx, objects, params); err == nil {
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
