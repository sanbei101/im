package store

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"testing"
	"uuid"
)

func newBenchStore(b *testing.B) *Store {
	b.Helper()
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := data.Close(); err != nil {
			b.Errorf("close store: %v", err)
		}
	})
	return data
}

func benchRoom(b *testing.B, s *Store) (uuid.UUID, uuid.UUID) {
	b.Helper()
	sender := uuid.NewV7()
	room := uuid.NewV7()
	if err := s.CreateRoom(context.Background(),
		Room{RoomID: room, ChatType: ChatTypeGroup},
		[]Member{{UserID: sender, Role: RoleOwner}}); err != nil {
		b.Fatal(err)
	}
	return room, sender
}

func BenchmarkStoreMessageWrite(b *testing.B) {
	for _, batch := range []int{1, 32, 100} {
		b.Run(fmt.Sprintf("batch-%d", batch), func(b *testing.B) {
			s := newBenchStore(b)
			room, sender := benchRoom(b, s)
			payload := []byte(`{"text":"benchmark payload"}`)

			b.ReportAllocs()
			for b.Loop() {
				messages := make([]Message, batch)
				for i := range messages {
					messages[i] = Message{
						ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
						MsgType: MsgTypeText, Payload: payload,
					}
				}
				results := s.WriteMessages(context.Background(), messages)
				for i := range results {
					if results[i].Err != nil {
						b.Fatal(results[i].Err)
					}
				}
			}
		})
	}
}

func BenchmarkStoreMessageWriteParallel(b *testing.B) {
	for name, rooms := range map[string]int{"single-room": 1, "multi-room": 64} {
		b.Run(name, func(b *testing.B) {
			s := newBenchStore(b)
			sender := uuid.NewV7()
			roomIDs := make([]uuid.UUID, rooms)
			for i := range roomIDs {
				roomIDs[i] = uuid.NewV7()
				if err := s.CreateRoom(context.Background(),
					Room{RoomID: roomIDs[i], ChatType: ChatTypeGroup},
					[]Member{{UserID: sender, Role: RoleOwner}}); err != nil {
					b.Fatal(err)
				}
			}
			payload := []byte(`{"text":"benchmark payload"}`)

			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for i := 0; pb.Next(); i++ {
					if _, err := s.WriteMessage(context.Background(), Message{
						ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: roomIDs[i%rooms],
						MsgType: MsgTypeText, Payload: payload,
					}); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkStoreRead(b *testing.B) {
	b.Run("newest-page", func(b *testing.B) {
		s := newBenchStore(b)
		room, sender := benchRoom(b, s)
		benchWrite(b, s, room, sender, 200)

		b.ReportAllocs()
		for b.Loop() {
			page, err := s.Messages(context.Background(), room, 0, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("read messages: count=%d err=%v", len(page.Messages), err)
			}
		}
	})
	b.Run("by-id", func(b *testing.B) {
		s := newBenchStore(b)
		room, sender := benchRoom(b, s)
		messages := benchWrite(b, s, room, sender, 50)
		target := messages[len(messages)-1].MsgID

		b.ReportAllocs()
		for b.Loop() {
			msg, err := s.MessageByID(context.Background(), room, target)
			if err != nil || msg.MsgID != target {
				b.Fatalf("message by id: err=%v", err)
			}
		}
	})
	b.Run("conversations", func(b *testing.B) {
		s := newBenchStore(b)
		user := uuid.NewV7()
		for i := range 50 {
			room := uuid.NewV7()
			if err := s.CreateRoom(context.Background(),
				Room{RoomID: room, ChatType: ChatTypeGroup, Name: fmt.Sprintf("Room %d", i)},
				[]Member{{UserID: user, Role: RoleMember}}); err != nil {
				b.Fatal(err)
			}
			if _, err := s.WriteMessage(context.Background(), Message{
				ClientMsgID: uuid.NewV7(), SenderID: user, RoomID: room,
				MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"hello"}`),
			}); err != nil {
				b.Fatal(err)
			}
			if err := s.MarkRoomRead(context.Background(), user, room, 1); err != nil {
				b.Fatal(err)
			}
		}

		b.ReportAllocs()
		for b.Loop() {
			convs, err := s.Conversations(context.Background(), user)
			if err != nil || len(convs) != 50 {
				b.Fatalf("conversations: count=%d err=%v", len(convs), err)
			}
		}
	})
	b.Run("members-500", func(b *testing.B) {
		s := newBenchStore(b)
		room := uuid.NewV7()
		members := make([]Member, 500)
		for i := range members {
			members[i] = Member{UserID: uuid.NewV7(), Role: RoleMember}
		}
		if err := s.CreateRoom(context.Background(), Room{RoomID: room, ChatType: ChatTypeGroup}, members); err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			list, err := s.Members(context.Background(), room)
			if err != nil || len(list) != 500 {
				b.Fatalf("members: count=%d err=%v", len(list), err)
			}
		}
	})
}

func benchWrite(b *testing.B, s *Store, room, sender uuid.UUID, count int) []Message {
	b.Helper()
	messages := make([]Message, count)
	for i := range count {
		message, err := s.WriteMessage(context.Background(), Message{
			ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
			MsgType: MsgTypeText, Payload: []byte("payload"),
		})
		if err != nil {
			b.Fatal(err)
		}
		messages[i] = message
	}
	return messages
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

	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(2000)
		for b.Loop() {
			if _, err := encodeParquet(rows); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("decode", func(b *testing.B) {
		data, err := encodeParquet(rows)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for b.Loop() {
			decoded, err := decodeParquet(data)
			if err != nil || len(decoded) != len(rows) {
				b.Fatalf("decode parquet: rows=%d err=%v", len(decoded), err)
			}
		}
	})
}

func BenchmarkMessageCache(b *testing.B) {
	b.Run("append", func(b *testing.B) {
		cache := newMessageCache()
		room := uuid.NewV7()
		message := Message{RoomID: room, RoomSeq: 1, Payload: jsontext.Value(`{"text":"cache benchmark"}`)}

		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			message.RoomSeq = uint64(i + 1)
			cache.append(&message)
		}
	})
	b.Run("page", func(b *testing.B) {
		cache := newMessageCache()
		room := uuid.NewV7()
		for i := range 128 {
			cache.append(
				&Message{RoomID: room, RoomSeq: uint64(i + 1), Payload: jsontext.Value(`{"text":"cache benchmark"}`)},
			)
		}

		b.ReportAllocs()
		for b.Loop() {
			messages, _, ok := cache.page(room, 0, 20)
			if !ok || len(messages) != 20 {
				b.Fatalf("page: ok=%v len=%d", ok, len(messages))
			}
		}
	})
}
