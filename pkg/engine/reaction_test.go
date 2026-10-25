package engine

import (
	"context"
	"slices"
	"testing"
	"uuid"
)

func TestEngineReactions(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine(t)
	room := uuid.NewV7()
	sender := uuid.NewV7()
	message := mustWriteMessage(t, e, room, sender, "react to me")

	t.Run("add, list, remove", func(t *testing.T) {
		reactor := uuid.NewV7()
		if err := e.AddReaction(ctx, room, message.RoomSeq, reactor, "👍"); err != nil {
			t.Fatalf("add reaction: %v", err)
		}
		groups, err := e.Reactions(ctx, room, message.RoomSeq)
		if err != nil {
			t.Fatalf("list reactions: %v", err)
		}
		if len(groups) != 1 || groups[0].Emoji != "👍" || groups[0].Count != 1 {
			t.Fatalf("groups = %+v", groups)
		}

		if err := e.RemoveReaction(ctx, room, message.RoomSeq, reactor, "👍"); err != nil {
			t.Fatalf("remove reaction: %v", err)
		}
		groups, err = e.Reactions(ctx, room, message.RoomSeq)
		if err != nil || len(groups) != 0 {
			t.Fatalf("after remove: groups=%+v err=%v", groups, err)
		}
	})

	t.Run("one user, one emoji: re-adding is idempotent", func(t *testing.T) {
		// The key IS the record, so a repeated insert neither errors nor
		// double-counts.
		reactor := uuid.NewV7()
		for range 2 {
			if err := e.AddReaction(ctx, room, message.RoomSeq, reactor, "🔥"); err != nil {
				t.Fatalf("add reaction: %v", err)
			}
		}
		groups, err := e.Reactions(ctx, room, message.RoomSeq)
		if err != nil || len(groups) != 1 || groups[0].Count != 1 {
			t.Fatalf("after re-add: groups=%+v err=%v", groups, err)
		}
	})

	t.Run("groups sort largest first, then emoji", func(t *testing.T) {
		low := uuid.NewV7()
		highA, highB := uuid.NewV7(), uuid.NewV7()
		seq := mustWriteMessage(t, e, room, sender, "sorted").RoomSeq
		for _, reaction := range []struct {
			user  uuid.UUID
			emoji string
		}{
			{low, "🔥"}, {highA, "👍"}, {highB, "👍"},
		} {
			if err := e.AddReaction(ctx, room, seq, reaction.user, reaction.emoji); err != nil {
				t.Fatalf("add reaction: %v", err)
			}
		}
		groups, err := e.Reactions(ctx, room, seq)
		if err != nil {
			t.Fatalf("list reactions: %v", err)
		}
		if len(groups) != 2 {
			t.Fatalf("groups = %+v", groups)
		}
		if groups[0].Emoji != "👍" || groups[0].Count != 2 {
			t.Fatalf("largest group not first: %+v", groups[0])
		}
		if !slices.Contains(groups[0].UserIDs, highA) || !slices.Contains(groups[0].UserIDs, highB) {
			t.Fatalf("group members = %v", groups[0].UserIDs)
		}
	})

	t.Run("page reads carry reactions", func(t *testing.T) {
		fresh := mustWriteMessage(t, e, room, sender, "page carries reactions")
		if err := e.AddReaction(ctx, room, fresh.RoomSeq, uuid.NewV7(), "👍"); err != nil {
			t.Fatalf("add reaction: %v", err)
		}
		page, err := e.Messages(ctx, room, 0, 10)
		if err != nil {
			t.Fatalf("read page: %v", err)
		}
		for _, listed := range page.Messages {
			if listed.RoomSeq != fresh.RoomSeq {
				continue
			}
			if len(listed.Reactions) != 1 || listed.Reactions[0].Emoji != "👍" {
				t.Fatalf("page reactions = %+v", listed.Reactions)
			}
			return
		}
		t.Fatalf("message %d missing from page", fresh.RoomSeq)
	})

	t.Run("reactions are scoped per message", func(t *testing.T) {
		other := mustWriteMessage(t, e, room, sender, "no reactions here")
		groups, err := e.Reactions(ctx, room, other.RoomSeq)
		if err != nil || len(groups) != 0 {
			t.Fatalf("untouched message: groups=%+v err=%v", groups, err)
		}
	})

	t.Run("empty emoji is rejected, absent remove is harmless", func(t *testing.T) {
		if err := e.AddReaction(ctx, room, message.RoomSeq, uuid.NewV7(), ""); err == nil {
			t.Fatal("empty emoji must be rejected")
		}
		if err := e.RemoveReaction(ctx, room, message.RoomSeq, uuid.NewV7(), "👍"); err != nil {
			t.Fatalf("remove absent reaction: %v", err)
		}
	})
}

func BenchmarkEngineReactions(b *testing.B) {
	e := newTestEngine(b)
	ctx := context.Background()
	room, sender := benchIDs()
	messages := benchWrite(b, e, room, sender, 32)

	// Four users seed two emojis on every message, so group reads assemble
	// from a realistic key spread instead of an empty prefix probe.
	users := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	for i := range messages {
		for _, user := range users[:3] {
			if err := e.AddReaction(ctx, room, messages[i].RoomSeq, user, "👍"); err != nil {
				b.Fatal(err)
			}
		}
		if err := e.AddReaction(ctx, room, messages[i].RoomSeq, users[3], "🔥"); err != nil {
			b.Fatal(err)
		}
	}

	b.Run("add-remove", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			seq := messages[i%len(messages)].RoomSeq
			if err := e.AddReaction(ctx, room, seq, users[3], "⭐"); err != nil {
				b.Fatal(err)
			}
			if err := e.RemoveReaction(ctx, room, seq, users[3], "⭐"); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})

	b.Run("groups-read", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			groups, err := e.Reactions(ctx, room, messages[i%len(messages)].RoomSeq)
			if err != nil || len(groups) != 2 {
				b.Fatalf("groups=%d err=%v", len(groups), err)
			}
			i++
		}
	})
}
