package engine

import (
	"context"
	"errors"
	"testing"
	"uuid"
)

func TestEnginePins(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine(t)
	room := uuid.NewV7()
	sender := uuid.NewV7()
	first := mustWriteMessage(t, e, room, sender, "first")
	second := mustWriteMessage(t, e, room, sender, "second")

	t.Run("pin, list, unpin", func(t *testing.T) {
		if err := e.PinMessage(ctx, room, second.RoomSeq, sender); err != nil {
			t.Fatalf("pin second: %v", err)
		}
		if err := e.PinMessage(ctx, room, first.RoomSeq, sender); err != nil {
			t.Fatalf("pin first: %v", err)
		}
		pinned, err := e.PinnedMessages(ctx, room)
		if err != nil {
			t.Fatalf("list pins: %v", err)
		}
		// Pin keys are big-endian seqs, so the listing is seq-ascending no
		// matter the pin order.
		if len(pinned) != 2 || pinned[0].RoomSeq != first.RoomSeq || pinned[1].RoomSeq != second.RoomSeq {
			t.Fatalf("pinned = %+v, want seqs [%d %d]", pinned, first.RoomSeq, second.RoomSeq)
		}

		if err := e.UnpinMessage(ctx, room, first.RoomSeq); err != nil {
			t.Fatalf("unpin: %v", err)
		}
		pinned, err = e.PinnedMessages(ctx, room)
		if err != nil || len(pinned) != 1 || pinned[0].RoomSeq != second.RoomSeq {
			t.Fatalf("after unpin: pinned=%+v err=%v", pinned, err)
		}
	})

	t.Run("pin targets an existing message", func(t *testing.T) {
		if err := e.PinMessage(ctx, room, 1<<30, sender); !errors.Is(err, ErrNotFound) {
			t.Fatalf("pin absent seq: err=%v, want %v", err, ErrNotFound)
		}
		if err := e.PinMessage(ctx, uuid.NewV7(), first.RoomSeq, sender); !errors.Is(err, ErrNotFound) {
			t.Fatalf("pin in unknown room: err=%v, want %v", err, ErrNotFound)
		}
	})

	t.Run("pin survives recall", func(t *testing.T) {
		// Recall rewrites the record in place, so a pinned message stays
		// addressable by seq afterwards.
		recalled := mustWriteMessage(t, e, room, sender, "to recall")
		if _, err := e.RecallMessage(ctx, room, recalled.RoomSeq, sender, false); err != nil {
			t.Fatalf("recall: %v", err)
		}
		if err := e.PinMessage(ctx, room, recalled.RoomSeq, sender); err != nil {
			t.Fatalf("pin recalled message: %v", err)
		}
		pinned, err := e.PinnedMessages(ctx, room)
		if err != nil || len(pinned) == 0 || pinned[len(pinned)-1].RoomSeq != recalled.RoomSeq {
			t.Fatalf("recalled pin missing: pinned=%+v err=%v", pinned, err)
		}
	})

	t.Run("stale pin is skipped", func(t *testing.T) {
		// A pin whose message is gone for good must not break the listing.
		// Written straight to Pebble to simulate a message lost under the pin.
		staleSeq := uint64(1 << 28)
		if err := e.db.Set(appendPinKey(nil, room, staleSeq), make([]byte, 24), nil); err != nil {
			t.Fatal(err)
		}
		pinned, err := e.PinnedMessages(ctx, room)
		if err != nil {
			t.Fatalf("list with stale pin: %v", err)
		}
		for _, message := range pinned {
			if message.RoomSeq == staleSeq {
				t.Fatal("stale pin surfaced in listing")
			}
		}
	})

	t.Run("unpin without a pin is harmless", func(t *testing.T) {
		if err := e.UnpinMessage(ctx, room, second.RoomSeq+100); err != nil {
			t.Fatalf("unpin absent pin: %v", err)
		}
	})
}

func BenchmarkEnginePins(b *testing.B) {
	e := newTestEngine(b)
	ctx := context.Background()
	room, sender := benchIDs()
	messages := benchWrite(b, e, room, sender, 64)

	b.Run("pin-and-unpin", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			seq := messages[i%len(messages)].RoomSeq
			if err := e.PinMessage(ctx, room, seq, sender); err != nil {
				b.Fatal(err)
			}
			if err := e.UnpinMessage(ctx, room, seq); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})

	const pinnedCount = 16
	b.Run("pinned-list", func(b *testing.B) {
		for i := range pinnedCount {
			if err := e.PinMessage(ctx, room, messages[i].RoomSeq, sender); err != nil {
				b.Fatal(err)
			}
		}
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			pinned, err := e.PinnedMessages(ctx, room)
			if err != nil || len(pinned) != pinnedCount {
				b.Fatalf("pinned=%d err=%v", len(pinned), err)
			}
		}
	})
}
