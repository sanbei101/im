package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"uuid"
)

func TestStoreMessageIdempotencyPaginationAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	storePath := t.TempDir() + "/store"
	data, err := Open(storePath)
	if err != nil {
		t.Fatal(err)
	}

	userID := uuid.NewV7()
	roomID := uuid.NewV7()
	if _, err := data.CreateUser(ctx, "alice", "password"); err != nil {
		t.Fatal(err)
	}
	if err := data.CreateRoom(ctx, Room{RoomID: roomID, ChatType: "single"}, []Member{{UserID: userID, Role: "member"}}); err != nil {
		t.Fatal(err)
	}

	clientID := uuid.NewV7()
	first, err := data.WriteMessage(ctx, Message{
		ClientMsgID: clientID,
		SenderID:    userID,
		RoomID:      roomID,
		MsgType:     "1",
		Payload:     []byte(`{"text":"first"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	retry, err := data.WriteMessage(ctx, Message{
		ClientMsgID: clientID,
		SenderID:    userID,
		RoomID:      roomID,
		MsgType:     "1",
		Payload:     []byte(`{"text":"first"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if retry.MsgID != first.MsgID || retry.RoomSeq != first.RoomSeq {
		t.Fatalf("idempotent retry changed result: first=%+v retry=%+v", first, retry)
	}
	_, err = data.WriteMessage(ctx, Message{
		ClientMsgID: clientID,
		SenderID:    userID,
		RoomID:      roomID,
		MsgType:     "1",
		Payload:     []byte(`{"text":"changed"}`),
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("changed idempotent payload error = %v, want %v", err, ErrConflict)
	}

	for _, text := range []string{"second", "third"} {
		if _, err := data.WriteMessage(ctx, Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    userID,
			RoomID:      roomID,
			MsgType:     "1",
			Payload:     []byte(`{"text":"` + text + `"}`),
		}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := data.Messages(ctx, roomID, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Messages) != 2 || !page.HasMore || string(page.Messages[0].Payload) != `{"text":"third"}` {
		t.Fatalf("unexpected latest page: %+v", page)
	}
	beforeFirst, err := data.Messages(ctx, roomID, first.RoomSeq+1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeFirst.Messages) != 1 || beforeFirst.Messages[0].RoomSeq != first.RoomSeq {
		t.Fatalf("unexpected before page: %+v", beforeFirst)
	}

	checkpointPath := t.TempDir() + "/checkpoint"
	if err := data.Checkpoint(ctx, checkpointPath); err != nil {
		t.Fatal(err)
	}
	if err := data.Close(); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := Open(checkpointPath)
	if err != nil {
		t.Fatal(err)
	}
	defer checkpoint.Close()
	room, err := checkpoint.Room(ctx, roomID)
	if err != nil {
		t.Fatal(err)
	}
	if room.LastSeq != 3 {
		t.Fatalf("checkpoint room last sequence = %d, want 3", room.LastSeq)
	}
}

func TestStoreContextCancellation(t *testing.T) {
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := data.CreateUser(ctx, "cancelled", "password"); !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateUser error = %v, want context.Canceled", err)
	}
	if _, err := data.Messages(ctx, uuid.NewV7(), 0, 20); !errors.Is(err, context.Canceled) {
		t.Fatalf("Messages error = %v, want context.Canceled", err)
	}
}

func TestStoreConcurrentMessageWrites(t *testing.T) {
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(context.Background(), Room{RoomID: roomID, ChatType: "group"}, []Member{{UserID: senderID, Role: "owner"}}); err != nil {
		t.Fatal(err)
	}

	const writers = 8
	const messagesPerWriter = 25
	var wg sync.WaitGroup
	wg.Add(writers)
	for range writers {
		go func() {
			defer wg.Done()
			for range messagesPerWriter {
				if _, err := data.WriteMessage(context.Background(), Message{
					ClientMsgID: uuid.NewV7(),
					SenderID:    senderID,
					RoomID:      roomID,
					MsgType:     "1",
					Payload:     []byte("concurrent"),
				}); err != nil {
					t.Errorf("concurrent WriteMessage: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	room, err := data.Room(context.Background(), roomID)
	if err != nil {
		t.Fatal(err)
	}
	want := uint64(writers * messagesPerWriter)
	if room.LastSeq != want {
		t.Fatalf("room last sequence = %d, want %d", room.LastSeq, want)
	}
	page, err := data.Messages(context.Background(), roomID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Messages) != 100 || page.Messages[0].RoomSeq != want {
		t.Fatalf("unexpected concurrent message page: len=%d first_seq=%d", len(page.Messages), page.Messages[0].RoomSeq)
	}
}

func BenchmarkWriteMessage(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		context.Background(),
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		b.Fatal(err)
	}

	payload := []byte(`{"text":"benchmark payload"}`)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := data.WriteMessage(context.Background(), Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    senderID,
			RoomID:      roomID,
			MsgType:     "1",
			Payload:     payload,
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadMessages(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		context.Background(),
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		b.Fatal(err)
	}

	for range 100 {
		if _, err := data.WriteMessage(context.Background(), Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    senderID,
			RoomID:      roomID,
			MsgType:     "1",
			Payload:     []byte("payload"),
		}); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		page, err := data.Messages(context.Background(), roomID, 0, 20)
		if err != nil || len(page.Messages) != 20 {
			b.Fatalf("read messages: page=%+v err=%v", page, err)
		}
	}
}
