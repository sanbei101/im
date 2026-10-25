package engine

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"
)

// instantArchive makes every message eligible and bounds one export to n messages.
func instantArchive(n int) ArchiveParams {
	return ArchiveParams{MaxAge: -time.Hour, MaxMessages: n, MaxBytes: 1 << 20}
}

func TestManifestCodec(t *testing.T) {
	want := ArchiveManifest{
		EndSeq: 42, MsgCount: 7, MinTime: -1000, MaxTime: 2000,
		File: "rooms/00000000-0000-0000-0000-000000000000/000001_000042.parquet",
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

func TestEngineArchive(t *testing.T) {
	ctx := context.Background()

	t.Run("cold reads across the archive boundary", func(t *testing.T) {
		e := newTestEngine(t, ArchiveDir(t.TempDir()))
		sender := uuid.NewV7()
		room := uuid.NewV7()
		written := make([]Message, 10)
		for i := range written {
			written[i] = mustWriteMessage(t, e, room, sender, fmt.Sprintf("m%d", i))
		}

		params := instantArchive(4)
		if err := e.ArchiveOnce(ctx, params); err != nil {
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
		cold, err := messageAt(ctx, e, room, written[0].RoomSeq)
		if err != nil || cold.RoomSeq != written[0].RoomSeq || cold.SenderID != sender {
			t.Fatalf("archived message by seq: msg=%+v err=%v", cold, err)
		}

		// MaxMessages bounds every cycle to 4: the second cycle archives
		// seqs 5..8, the third drains the remaining tail, the fourth is a no-op.
		for cycle, want := range []uint64{8, 10, 10} {
			if err := e.ArchiveOnce(ctx, params); err != nil {
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
		if _, err := e.RecallMessage(ctx, room, written[3].RoomSeq, sender, true); !errors.Is(err, ErrRecallTimeout) {
			t.Fatalf("recall of archived message error = %v, want %v", err, ErrRecallTimeout)
		}
	})

	t.Run("cold history survives a restart", func(t *testing.T) {
		dir := t.TempDir()
		e := openTestEngine(t, dir+"/engine", ArchiveDir(dir+"/archive"))
		sender := uuid.NewV7()
		room := uuid.NewV7()
		written := make([]Message, 5)
		for i := range written {
			written[i] = mustWriteMessage(t, e, room, sender, "cold restart")
		}
		if err := e.ArchiveOnce(ctx, instantArchive(10)); err != nil {
			t.Fatal(err)
		}
		if err := e.Close(); err != nil {
			t.Fatal(err)
		}

		// A fresh process reopens the same store: the watermark and the
		// parquet files are all it needs to serve cold history again.
		reopened := openTestEngine(t, dir+"/engine", ArchiveDir(dir+"/archive"))
		page, err := reopened.Messages(ctx, room, 0, 20)
		if err != nil || len(page.Messages) != 5 {
			t.Fatalf("cold history after restart: page=%+v err=%v", page, err)
		}
		for i, msg := range page.Messages {
			if want := uint64(5 - i); msg.RoomSeq != want {
				t.Fatalf("cold order broken at %d: seq=%d want %d", i, msg.RoomSeq, want)
			}
			if string(msg.Payload) == "" {
				t.Fatalf("cold payload lost at seq %d", msg.RoomSeq)
			}
		}
		cold, err := messageAt(ctx, reopened, room, written[0].RoomSeq)
		if err != nil || cold.RoomSeq != written[0].RoomSeq {
			t.Fatalf("cold by-id after restart: msg=%+v err=%v", cold, err)
		}
		// New writes continue the sequence where the old engine left off.
		fresh := mustWriteMessage(t, reopened, room, sender, "after restart")
		if fresh.RoomSeq != 6 {
			t.Fatalf("seq after restart = %d, want 6", fresh.RoomSeq)
		}
	})

	t.Run("archived message is unresolvable without an archive dir", func(t *testing.T) {
		dir := t.TempDir()
		e := openTestEngine(t, dir+"/engine", ArchiveDir(dir+"/archive"))
		sender := uuid.NewV7()
		room := uuid.NewV7()
		msg := mustWriteMessage(t, e, room, sender, "will go cold")
		if err := e.ArchiveOnce(ctx, instantArchive(10)); err != nil {
			t.Fatal(err)
		}
		if err := e.Close(); err != nil {
			t.Fatal(err)
		}

		// Reopened without ArchiveDir: the tier is not wired up, so archived
		// bodies stay gone and warm history still works. The read path reports
		// the misconfigured tier; recall reports the missing message.
		bare := openTestEngine(t, dir+"/engine")
		if _, err := messageAt(ctx, bare, room, msg.RoomSeq); err == nil {
			t.Fatal("archived message resolved without an archive dir")
		}
		if _, err := bare.RecallMessage(ctx, room, msg.RoomSeq, sender, false); !errors.Is(err, ErrNotFound) {
			t.Fatalf("recall without dir error = %v, want %v", err, ErrNotFound)
		}
		if err := bare.ArchiveOnce(ctx, instantArchive(10)); err == nil {
			t.Fatal("expected ArchiveOnce to fail without an archive dir")
		}
	})

	t.Run("owns filter skips foreign rooms", func(t *testing.T) {
		e := newTestEngine(t, ArchiveDir(t.TempDir()))
		sender := uuid.NewV7()
		room := uuid.NewV7()
		mustWriteMessage(t, e, room, sender, "not mine")

		params := instantArchive(10)
		params.Owns = func(uuid.UUID) bool { return false }
		if err := e.ArchiveOnce(ctx, params); err != nil {
			t.Fatal(err)
		}
		watermark, err := e.archiveWatermark(room)
		if err != nil || watermark != 0 {
			t.Fatalf("foreign room watermark: seq=%d err=%v", watermark, err)
		}
	})

	t.Run("invalid params rejected", func(t *testing.T) {
		e := newTestEngine(t, ArchiveDir(t.TempDir()))
		params := ArchiveParams{MaxAge: time.Hour, MaxMessages: 0, MaxBytes: 1 << 20}
		if err := e.ArchiveOnce(ctx, params); err == nil {
			t.Fatal("expected error for MaxMessages=0")
		}
	})
}

func BenchmarkArchiveParquetCodec(b *testing.B) {
	rows := make([]ParquetMessage, 2000)
	for i := range rows {
		rows[i] = newParquetMessage(&Message{
			SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
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

// BenchmarkEngineArchivedRead serves a page from the exported parquet files:
// the decoded cache makes the second read of the same file free.
func BenchmarkEngineArchivedRead(b *testing.B) {
	ctx := context.Background()
	e := newTestEngine(b, ArchiveDir(b.TempDir()))
	room, sender := benchIDs()
	messages := benchWrite(b, e, room, sender, 2000)
	if err := e.ArchiveOnce(ctx, instantArchive(2000)); err != nil {
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
	b.Run("by-seq", func(b *testing.B) {
		target := messages[500].RoomSeq
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			if _, err := messageAt(ctx, e, room, target); err != nil {
				b.Fatal(err)
			}
		}
	})
}
