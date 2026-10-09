package store

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"
)

// newTestStore opens a throwaway store and registers its cleanup.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := data.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	return data
}

func mustCreateUser(t *testing.T, s *Store, name string) User {
	t.Helper()
	user, err := s.CreateUser(context.Background(), name, "password")
	if err != nil {
		t.Fatalf("create user %s: %v", name, err)
	}
	return user
}

func mustCreateRoom(t *testing.T, s *Store, chatType string, members ...Member) Room {
	t.Helper()
	room := Room{RoomID: uuid.NewV7(), ChatType: chatType}
	if chatType == ChatTypeGroup {
		room.Name = "test room"
	}
	if err := s.CreateRoom(context.Background(), room, members); err != nil {
		t.Fatalf("create room: %v", err)
	}
	return room
}

func mustWriteMessage(t *testing.T, s *Store, roomID, senderID uuid.UUID, payload string) Message {
	t.Helper()
	message, err := s.WriteMessage(context.Background(), Message{
		ClientMsgID: uuid.NewV7(),
		SenderID:    senderID,
		RoomID:      roomID,
		MsgType:     MsgTypeText,
		Payload:     jsontext.Value(`{"text":"` + payload + `"}`),
	})
	if err != nil {
		t.Fatalf("write message: %v", err)
	}
	return message
}

// messagesEqual compares Messages field by field; the struct itself is not
// comparable because Payload and Ext are slices.
func messagesEqual(a, b Message) bool {
	return a.MsgID == b.MsgID && a.ClientMsgID == b.ClientMsgID && a.SenderID == b.SenderID &&
		a.RoomID == b.RoomID && a.RoomSeq == b.RoomSeq && a.ServerTime == b.ServerTime &&
		a.ReplyToMsgID == b.ReplyToMsgID && a.MsgType == b.MsgType &&
		string(a.Payload) == string(b.Payload) && string(a.Ext) == string(b.Ext)
}

func TestMessageCodec(t *testing.T) {
	t.Run("round trip full record", func(t *testing.T) {
		want := Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: 10, ServerTime: 456, ReplyToMsgID: uuid.NewV7(),
			MsgType: MsgTypeFile, Payload: jsontext.Value(`{"u":"x"}`), Ext: jsontext.Value(`{"e":1}`),
		}
		got, err := decodeMessage(encodeMessage(want))
		if err != nil {
			t.Fatal(err)
		}
		if !messagesEqual(got, want) {
			t.Fatalf("round trip mismatch:\nwant %+v\ngot  %+v", want, got)
		}
	})
	t.Run("round trip minimal record", func(t *testing.T) {
		want := Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: 1, ServerTime: -1, MsgType: MsgTypeSystem, Payload: jsontext.Value(`{}`),
		}
		got, err := decodeMessage(encodeMessage(want))
		if err != nil {
			t.Fatal(err)
		}
		if !messagesEqual(got, want) {
			t.Fatalf("round trip mismatch:\nwant %+v\ngot  %+v", want, got)
		}
	})
	t.Run("truncated records rejected", func(t *testing.T) {
		full := encodeMessage(Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: 1, ServerTime: 1, MsgType: MsgTypeText, Payload: jsontext.Value(`{}`),
		})
		for _, broken := range [][]byte{{}, full[:len(full)/2], append(full, 0)} {
			if _, err := decodeMessage(broken); err == nil {
				t.Fatalf("expected error for %d-byte record", len(broken))
			}
		}
	})
	t.Run("dedup round trip", func(t *testing.T) {
		want := Dedup{MsgID: uuid.NewV7(), RoomSeq: 7, ServerTime: 42, PayloadSum: [32]byte{1, 2, 3}}
		got, err := decodeDedup(encodeDedup(want))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("dedup round trip mismatch: want %+v got %+v", want, got)
		}
		if _, err := decodeDedup(want.PayloadSum[:16]); err == nil {
			t.Fatal("expected error for short dedup record")
		}
	})
}

func TestStoreUsers(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	t.Run("create and duplicate", func(t *testing.T) {
		user := mustCreateUser(t, s, "alice")
		if user.Username != "alice" || user.Nickname != "alice" || user.Password != "password" {
			t.Fatalf("unexpected user: %+v", user)
		}
		if _, err := s.CreateUser(ctx, "alice", "password"); !errors.Is(err, ErrAlreadyExists) {
			t.Fatalf("duplicate error = %v, want %v", err, ErrAlreadyExists)
		}
		if _, err := s.CreateUser(ctx, "   ", "password"); err == nil {
			t.Fatal("expected error for blank username")
		}
	})

	t.Run("lookup trims and matches case", func(t *testing.T) {
		mustCreateUser(t, s, "Bob")
		for _, name := range []string{"Bob", " bob ", "BOB"} {
			user, err := s.UserByUsername(ctx, name)
			if err != nil || user.Username != "Bob" {
				t.Fatalf("lookup %q: user=%+v err=%v", name, user, err)
			}
		}
		if _, err := s.UserByUsername(ctx, "nobody"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing user error = %v, want %v", err, ErrNotFound)
		}
		byID, err := s.UserByID(ctx, mustCreateUser(t, s, "carol").UserID)
		if err != nil || byID.Username != "carol" {
			t.Fatalf("lookup by id: user=%+v err=%v", byID, err)
		}
	})

	t.Run("update profile keeps empty fields", func(t *testing.T) {
		user := mustCreateUser(t, s, "dave")
		updated, err := s.UpdateUserProfile(ctx, user.UserID, "  Dave Jr.  ", "https://avatar/dave")
		if err != nil || updated.Nickname != "Dave Jr." || updated.AvatarURL != "https://avatar/dave" {
			t.Fatalf("update profile: user=%+v err=%v", updated, err)
		}
		kept, err := s.UpdateUserProfile(ctx, user.UserID, "", "")
		if err != nil || kept.Nickname != "Dave Jr." || kept.AvatarURL != "https://avatar/dave" {
			t.Fatalf("empty update must not clobber fields: %+v err=%v", kept, err)
		}
		if _, err := s.UpdateUserProfile(ctx, uuid.NewV7(), "x", ""); !errors.Is(err, ErrNotFound) {
			t.Fatalf("unknown user error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("password update", func(t *testing.T) {
		user := mustCreateUser(t, s, "erin")
		if err := s.UpdateUserPassword(ctx, user.UserID, "new-secret"); err != nil {
			t.Fatal(err)
		}
		stored, err := s.UserByID(ctx, user.UserID)
		if err != nil || stored.Password != "new-secret" {
			t.Fatalf("password update: user=%+v err=%v", stored, err)
		}
	})

	t.Run("search by username and nickname", func(t *testing.T) {
		mustCreateUser(t, s, "frank_zhang")
		mustCreateUser(t, s, "grace")
		updated := mustCreateUser(t, s, "heidi")
		if _, err := s.UpdateUserProfile(ctx, updated.UserID, "张小Heidi", ""); err != nil {
			t.Fatal(err)
		}

		for keyword, want := range map[string]int{"zhang": 1, "GRACE": 1, "heidi": 1} {
			found, err := s.SearchUsers(ctx, keyword, 10)
			if err != nil || len(found) != want {
				t.Fatalf("search %q: found=%+v err=%v", keyword, found, err)
			}
		}
		limited, err := s.SearchUsers(ctx, "i", 1)
		if err != nil || len(limited) != 1 {
			t.Fatalf("search with limit: found=%+v err=%v", limited, err)
		}
	})
}

func TestStoreRooms(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	t.Run("create group room", func(t *testing.T) {
		owner := mustCreateUser(t, s, "room-owner")
		room := mustCreateRoom(t, s, ChatTypeGroup,
			Member{UserID: owner.UserID, Role: RoleOwner},
			Member{UserID: uuid.NewV7(), Role: RoleMember},
		)
		got, err := s.Room(ctx, room.RoomID)
		if err != nil || got.RoomID != room.RoomID || got.LastSeq != 0 {
			t.Fatalf("load room: room=%+v err=%v", got, err)
		}
		members, err := s.Members(ctx, room.RoomID)
		if err != nil || len(members) != 2 {
			t.Fatalf("members: list=%+v err=%v", members, err)
		}
		if err := s.CreateRoom(
			ctx,
			Room{RoomID: room.RoomID, ChatType: ChatTypeGroup},
			nil,
		); !errors.Is(
			err,
			ErrAlreadyExists,
		) {
			t.Fatalf("duplicate room error = %v, want %v", err, ErrAlreadyExists)
		}
		if _, err := s.Room(ctx, uuid.NewV7()); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing room error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("single chat hash uniqueness", func(t *testing.T) {
		hash := []byte("pair-hash-1")
		room := Room{RoomID: uuid.NewV7(), ChatType: ChatTypeSingle, SingleChatHash: hash}
		if err := s.CreateRoom(ctx, room, []Member{{UserID: uuid.NewV7(), Role: RoleMember}}); err != nil {
			t.Fatal(err)
		}
		found, err := s.RoomBySingleHash(ctx, hash)
		if err != nil || found.RoomID != room.RoomID {
			t.Fatalf("room by hash: room=%+v err=%v", found, err)
		}
		dup := Room{RoomID: uuid.NewV7(), ChatType: ChatTypeSingle, SingleChatHash: hash}
		if err := s.CreateRoom(ctx, dup, nil); !errors.Is(err, ErrAlreadyExists) {
			t.Fatalf("duplicate hash error = %v, want %v", err, ErrAlreadyExists)
		}
	})

	t.Run("member management", func(t *testing.T) {
		owner := mustCreateUser(t, s, "mgmt-owner")
		member := mustCreateUser(t, s, "mgmt-member")
		outsider := mustCreateUser(t, s, "mgmt-outsider")
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: owner.UserID, Role: RoleOwner})

		if err := s.AddMembers(ctx, room.RoomID, []Member{{UserID: member.UserID, Role: RoleMember}}); err != nil {
			t.Fatal(err)
		}
		got, err := s.Member(ctx, room.RoomID, member.UserID)
		if err != nil || got.Role != RoleMember {
			t.Fatalf("member lookup: member=%+v err=%v", got, err)
		}
		if _, err := s.Member(ctx, room.RoomID, outsider.UserID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("outsider error = %v, want %v", err, ErrNotFound)
		}

		if err := s.UpdateMemberRole(ctx, room.RoomID, member.UserID, RoleAdmin); err != nil {
			t.Fatal(err)
		}
		pinned, muted := true, true
		if err := s.UpdateMemberSettings(ctx, room.RoomID, member.UserID, &pinned, &muted); err != nil {
			t.Fatal(err)
		}
		updated, err := s.Member(ctx, room.RoomID, member.UserID)
		if err != nil || updated.Role != RoleAdmin || !updated.IsPinned || !updated.IsMuted {
			t.Fatalf("member update: member=%+v err=%v", updated, err)
		}

		if err := s.RemoveMember(ctx, room.RoomID, member.UserID); err != nil {
			t.Fatal(err)
		}
		rooms, err := s.RoomsByUser(ctx, member.UserID)
		if err != nil || len(rooms) != 0 {
			t.Fatalf("rooms after removal: list=%+v err=%v", rooms, err)
		}
	})

	t.Run("update room metadata", func(t *testing.T) {
		owner := mustCreateUser(t, s, "meta-owner")
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: owner.UserID, Role: RoleOwner})
		updated, err := s.UpdateRoom(ctx, room.RoomID, "新名字", "https://avatar/room", "公告")
		if err != nil || updated.Name != "新名字" || updated.Notice != "公告" {
			t.Fatalf("update room: room=%+v err=%v", updated, err)
		}
		kept, err := s.UpdateRoom(ctx, room.RoomID, "", "", "")
		if err != nil || kept.Name != "新名字" || kept.Notice != "公告" {
			t.Fatalf("empty update must not clobber fields: %+v err=%v", kept, err)
		}
	})

	t.Run("dissolve removes room and members", func(t *testing.T) {
		owner := mustCreateUser(t, s, "dissolve-owner")
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: owner.UserID, Role: RoleOwner})
		if err := s.DissolveRoom(ctx, room.RoomID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Room(ctx, room.RoomID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("dissolved room error = %v, want %v", err, ErrNotFound)
		}
		rooms, err := s.RoomsByUser(ctx, owner.UserID)
		if err != nil || len(rooms) != 0 {
			t.Fatalf("rooms after dissolve: list=%+v err=%v", rooms, err)
		}
	})
}

func TestStoreMessages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	sender := mustCreateUser(t, s, "msg-sender")
	room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})

	t.Run("write assigns identity", func(t *testing.T) {
		msg := mustWriteMessage(t, s, room.RoomID, sender.UserID, "first")
		if msg.RoomSeq != 1 || msg.MsgID == uuid.Nil() || msg.ServerTime == 0 {
			t.Fatalf("server did not stamp message: %+v", msg)
		}
		got, err := s.Room(ctx, room.RoomID)
		if err != nil || got.LastSeq != 1 {
			t.Fatalf("room last seq: room=%+v err=%v", got, err)
		}
		if _, err := s.WriteMessage(ctx, Message{
			ClientMsgID: uuid.NewV7(), SenderID: sender.UserID, RoomID: uuid.NewV7(), MsgType: MsgTypeText,
		}); err == nil {
			t.Fatal("expected error writing to missing room")
		}
	})

	t.Run("idempotent retry and conflict", func(t *testing.T) {
		clientID := uuid.NewV7()
		first, err := s.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender.UserID, RoomID: room.RoomID,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"same"}`),
		})
		if err != nil {
			t.Fatal(err)
		}
		retry, err := s.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender.UserID, RoomID: room.RoomID,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"same"}`),
		})
		if err != nil || retry.MsgID != first.MsgID || retry.RoomSeq != first.RoomSeq {
			t.Fatalf("retry changed result: first=%+v retry=%+v err=%v", first, retry, err)
		}
		if _, err := s.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender.UserID, RoomID: room.RoomID,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"changed"}`),
		}); !errors.Is(err, ErrConflict) {
			t.Fatalf("changed payload error = %v, want %v", err, ErrConflict)
		}
	})

	t.Run("batch dedup within one commit", func(t *testing.T) {
		clientID := uuid.NewV7()
		results := s.WriteMessages(ctx, []Message{
			{
				ClientMsgID: clientID,
				SenderID:    sender.UserID,
				RoomID:      room.RoomID,
				MsgType:     MsgTypeText,
				Payload:     []byte("dup"),
			},
			{
				ClientMsgID: clientID,
				SenderID:    sender.UserID,
				RoomID:      room.RoomID,
				MsgType:     MsgTypeText,
				Payload:     []byte("dup"),
			},
			{
				ClientMsgID: uuid.NewV7(),
				SenderID:    sender.UserID,
				RoomID:      room.RoomID,
				MsgType:     MsgTypeText,
				Payload:     []byte("other"),
			},
		})
		if len(results) != 3 {
			t.Fatalf("batch returned %d results", len(results))
		}
		for i := range results {
			if results[i].Err != nil {
				t.Fatalf("batch result %d: %v", i, results[i].Err)
			}
		}
		if results[0].Message.MsgID != results[1].Message.MsgID {
			t.Fatalf("duplicate batch entries diverged: %+v vs %+v", results[0], results[1])
		}
		if results[2].Message.RoomSeq != results[0].Message.RoomSeq+1 {
			t.Fatalf("distinct batch message shares seq with duplicate: %+v", results[2])
		}
	})

	t.Run("pagination newest first with cursor", func(t *testing.T) {
		paged := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		for i := range 7 {
			mustWriteMessage(t, s, paged.RoomID, sender.UserID, "m"+string(rune('a'+i)))
		}

		newest, err := s.Messages(ctx, paged.RoomID, 0, 3)
		if err != nil || len(newest.Messages) != 3 || !newest.HasMore {
			t.Fatalf("newest page: page=%+v err=%v", newest, err)
		}
		if newest.Messages[0].RoomSeq != 7 || newest.Messages[2].RoomSeq != 5 {
			t.Fatalf("newest page order wrong: %+v", newest.Messages)
		}

		next, err := s.Messages(ctx, paged.RoomID, newest.Messages[2].RoomSeq, 3)
		if err != nil || len(next.Messages) != 3 || next.Messages[0].RoomSeq != 4 {
			t.Fatalf("cursor page: page=%+v err=%v", next, err)
		}
		tail, err := s.Messages(ctx, paged.RoomID, next.Messages[2].RoomSeq, 3)
		if err != nil || len(tail.Messages) != 1 || tail.HasMore {
			t.Fatalf("tail page: page=%+v err=%v", tail, err)
		}

		whole, err := s.Messages(ctx, paged.RoomID, 0, 20)
		if err != nil || len(whole.Messages) != 7 || whole.HasMore {
			t.Fatalf("full page: page=%+v err=%v", whole, err)
		}
		oversized, err := s.Messages(ctx, paged.RoomID, 0, 500)
		if err != nil || len(oversized.Messages) != 7 {
			t.Fatalf("oversized limit must clamp: page=%+v err=%v", oversized, err)
		}
		if empty, err := s.Messages(ctx, uuid.NewV7(), 0, 20); err != nil || len(empty.Messages) != 0 {
			t.Fatalf("missing room page: page=%+v err=%v", empty, err)
		}

		afterPage, err := s.MessagesAfter(ctx, paged.RoomID, 4, 10)
		if err != nil || len(afterPage.Messages) != 3 {
			t.Fatalf("after page: page=%+v err=%v", afterPage, err)
		}
		if afterPage.Messages[0].RoomSeq != 5 || afterPage.Messages[1].RoomSeq != 6 ||
			afterPage.Messages[2].RoomSeq != 7 {
			t.Fatalf("after page ordering wrong: %+v", afterPage.Messages)
		}
	})

	t.Run("concurrent writes keep sequence dense", func(t *testing.T) {
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		const writers, perWriter = 4, 25
		var wg sync.WaitGroup
		wg.Add(writers)
		for range writers {
			go func() {
				defer wg.Done()
				for range perWriter {
					if _, err := s.WriteMessage(ctx, Message{
						ClientMsgID: uuid.NewV7(), SenderID: sender.UserID, RoomID: room.RoomID,
						MsgType: MsgTypeText, Payload: []byte("concurrent"),
					}); err != nil {
						t.Errorf("concurrent write: %v", err)
						return
					}
				}
			}()
		}
		wg.Wait()

		page, err := s.Messages(ctx, room.RoomID, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Messages) != writers*perWriter {
			t.Fatalf("page holds %d messages, want %d", len(page.Messages), writers*perWriter)
		}
		for i, msg := range page.Messages {
			if want := uint64(writers * perWriter); msg.RoomSeq != want-uint64(i) {
				t.Fatalf("sequence not dense at %d: seq=%d", i, msg.RoomSeq)
			}
		}
	})

	t.Run("search skips recalled and honours cursor", func(t *testing.T) {
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		hit := mustWriteMessage(t, s, room.RoomID, sender.UserID, "Needle Here")
		mustWriteMessage(t, s, room.RoomID, sender.UserID, "nothing special")
		recalled := mustWriteMessage(t, s, room.RoomID, sender.UserID, "NEEDLE to remove")
		if _, err := s.RecallMessage(ctx, room.RoomID, recalled.MsgID, sender.UserID, false); err != nil {
			t.Fatal(err)
		}

		found, err := s.SearchRoomMessages(ctx, room.RoomID, "  NEEDLE  ", 0, 10)
		if err != nil || len(found) != 1 || found[0].MsgID != hit.MsgID {
			t.Fatalf("search: found=%+v err=%v", found, err)
		}
		if _, err := s.SearchRoomMessages(ctx, room.RoomID, "", 0, 10); err != nil || len(found) != 1 {
			t.Fatalf("empty keyword must return nothing, got err=%v", err)
		}
	})

	t.Run("message by id", func(t *testing.T) {
		msg := mustWriteMessage(t, s, room.RoomID, sender.UserID, "byid")
		got, err := s.MessageByID(ctx, room.RoomID, msg.MsgID)
		if err != nil || got.RoomSeq != msg.RoomSeq {
			t.Fatalf("message by id: msg=%+v err=%v", got, err)
		}
		if _, err := s.MessageByID(ctx, room.RoomID, uuid.NewV7()); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing message error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("recall rules", func(t *testing.T) {
		room := mustCreateRoom(t, s, ChatTypeGroup,
			Member{UserID: sender.UserID, Role: RoleMember},
			Member{UserID: mustCreateUser(t, s, "recall-owner").UserID, Role: RoleOwner},
		)
		msg := mustWriteMessage(t, s, room.RoomID, sender.UserID, "recall me")
		outsider := uuid.NewV7()

		if _, err := s.RecallMessage(ctx, room.RoomID, msg.MsgID, outsider, false); !errors.Is(err, ErrForbidden) {
			t.Fatalf("outsider recall error = %v, want %v", err, ErrForbidden)
		}
		stale := mustWriteMessage(t, s, room.RoomID, sender.UserID, "too old to recall")
		if err := s.backdateMessage(stale, time.Hour); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RecallMessage(
			ctx,
			room.RoomID,
			stale.MsgID,
			sender.UserID,
			false,
		); !errors.Is(
			err,
			ErrRecallTimeout,
		) {
			t.Fatalf("stale recall error = %v, want %v", err, ErrRecallTimeout)
		}
		recalled, err := s.RecallMessage(ctx, room.RoomID, stale.MsgID, outsider, true)
		if err != nil || recalled.MsgType != MsgTypeRecall {
			t.Fatalf("owner recall: msg=%+v err=%v", recalled, err)
		}
		again, err := s.RecallMessage(ctx, room.RoomID, msg.MsgID, sender.UserID, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.RecallMessage(
			ctx,
			room.RoomID,
			msg.MsgID,
			sender.UserID,
			false,
		); err != nil ||
			again.MsgType != MsgTypeRecall {
			t.Fatalf("repeat recall: msg=%+v err=%v", again, err)
		}

		page, err := s.Messages(ctx, room.RoomID, 0, 10)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range page.Messages {
			if m.MsgID == msg.MsgID && m.MsgType != MsgTypeRecall {
				t.Fatalf("recall not visible in tail cache: %+v", m)
			}
		}
	})

	t.Run("cancelled context fails fast", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := s.WriteMessage(cancelled, Message{RoomID: room.RoomID}); !errors.Is(err, context.Canceled) {
			t.Fatalf("write error = %v, want %v", err, context.Canceled)
		}
		if _, err := s.Messages(cancelled, room.RoomID, 0, 10); !errors.Is(err, context.Canceled) {
			t.Fatalf("read error = %v, want %v", err, context.Canceled)
		}
	})
}

// backdateMessage rewrites the stored message with an ancient ServerTime so
// recall-window logic can be exercised without sleeping.
func (s *Store) backdateMessage(message Message, age time.Duration) error {
	message.ServerTime = time.Now().Add(-age).UnixMicro()
	batch := s.db.NewBatch()
	defer batch.Close()
	var keyBuf [25]byte
	if err := s.setBytes(
		batch,
		appendMessageKey(keyBuf[:0], message.RoomID, message.RoomSeq),
		encodeMessage(message),
	); err != nil {
		return err
	}
	return commit(batch)
}

func TestStoreFriends(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, s, "alice")
	bob := mustCreateUser(t, s, "bob")

	t.Run("apply and duplicate", func(t *testing.T) {
		if err := s.ApplyFriend(ctx, alice.UserID, bob.UserID, "加个好友"); err != nil {
			t.Fatal(err)
		}
		if err := s.ApplyFriend(ctx, alice.UserID, alice.UserID, "self"); err == nil {
			t.Fatal("expected error for self application")
		}
		apps, err := s.Applications(ctx, bob.UserID)
		if err != nil || len(apps) != 1 || apps[0].FromUserID != alice.UserID ||
			apps[0].Status != FriendStatusPending || apps[0].Greeting != "加个好友" {
			t.Fatalf("applications: list=%+v err=%v", apps, err)
		}
	})

	t.Run("audit accept creates both sides", func(t *testing.T) {
		if err := s.AuditFriend(ctx, bob.UserID, alice.UserID, true); err != nil {
			t.Fatal(err)
		}
		for _, user := range []User{alice, bob} {
			friends, err := s.Friends(ctx, user.UserID)
			if err != nil || len(friends) != 1 {
				t.Fatalf("friends of %s: list=%+v err=%v", user.Username, friends, err)
			}
		}
	})

	t.Run("remark and delete", func(t *testing.T) {
		if err := s.UpdateFriendRemark(ctx, alice.UserID, bob.UserID, "老鲍"); err != nil {
			t.Fatal(err)
		}
		friends, err := s.Friends(ctx, alice.UserID)
		if err != nil || friends[0].Remark != "老鲍" {
			t.Fatalf("remark: list=%+v err=%v", friends, err)
		}
		if err := s.DeleteFriend(ctx, alice.UserID, bob.UserID); err != nil {
			t.Fatal(err)
		}
		for _, user := range []User{alice, bob} {
			friends, err := s.Friends(ctx, user.UserID)
			if err != nil || len(friends) != 0 {
				t.Fatalf("friends after delete: list=%+v err=%v", friends, err)
			}
		}
	})

	t.Run("blacklist", func(t *testing.T) {
		carol := mustCreateUser(t, s, "carol")
		if err := s.AddBlacklist(ctx, alice.UserID, carol.UserID); err != nil {
			t.Fatal(err)
		}
		if err := s.AddBlacklist(ctx, alice.UserID, alice.UserID); err == nil {
			t.Fatal("expected error for self blacklist")
		}
		blocked, err := s.IsBlacklisted(ctx, alice.UserID, carol.UserID)
		if err != nil || !blocked {
			t.Fatalf("blacklisted=%v err=%v", blocked, err)
		}
		list, err := s.Blacklist(ctx, alice.UserID)
		if err != nil || len(list) != 1 || list[0] != carol.UserID {
			t.Fatalf("blacklist: list=%+v err=%v", list, err)
		}
		if err := s.RemoveBlacklist(ctx, alice.UserID, carol.UserID); err != nil {
			t.Fatal(err)
		}
		blocked, err = s.IsBlacklisted(ctx, alice.UserID, carol.UserID)
		if err != nil || blocked {
			t.Fatalf("after removal blacklisted=%v err=%v", blocked, err)
		}
	})
}

func TestStoreInteractions(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	owner := mustCreateUser(t, s, "inter-owner")
	readers := []User{owner, mustCreateUser(t, s, "inter-r1"), mustCreateUser(t, s, "inter-r2")}
	members := make([]Member, len(readers))
	for i, user := range readers {
		members[i] = Member{UserID: user.UserID, Role: RoleMember}
	}
	room := mustCreateRoom(t, s, ChatTypeGroup, members...)
	msg := mustWriteMessage(t, s, room.RoomID, owner.UserID, "react to me")

	t.Run("reactions aggregate per emoji", func(t *testing.T) {
		for _, user := range readers {
			if err := s.AddReaction(ctx, room.RoomID, msg.MsgID, user.UserID, "👍"); err != nil {
				t.Fatal(err)
			}
		}
		if err := s.AddReaction(ctx, room.RoomID, msg.MsgID, owner.UserID, "🔥"); err != nil {
			t.Fatal(err)
		}
		if err := s.AddReaction(ctx, room.RoomID, msg.MsgID, owner.UserID, ""); err == nil {
			t.Fatal("expected error for empty emoji")
		}

		groups, err := s.Reactions(ctx, room.RoomID, msg.MsgID)
		if err != nil || len(groups) != 2 {
			t.Fatalf("reactions: groups=%+v err=%v", groups, err)
		}
		for _, group := range groups {
			if group.Emoji == "👍" && group.Count != 3 {
				t.Fatalf("thumbs up count = %d, want 3", group.Count)
			}
		}
		if err := s.RemoveReaction(ctx, room.RoomID, msg.MsgID, readers[1].UserID, "👍"); err != nil {
			t.Fatal(err)
		}
		groups, err = s.Reactions(ctx, room.RoomID, msg.MsgID)
		if err != nil {
			t.Fatal(err)
		}
		for _, group := range groups {
			if group.Emoji == "👍" && group.Count != 2 {
				t.Fatalf("thumbs up count after removal = %d, want 2", group.Count)
			}
		}

		page, err := s.Messages(ctx, room.RoomID, 0, 10)
		if err != nil {
			t.Fatal(err)
		}
		var foundMsg *Message
		for i := range page.Messages {
			if page.Messages[i].MsgID == msg.MsgID {
				foundMsg = &page.Messages[i]
				break
			}
		}
		if foundMsg == nil || len(foundMsg.Reactions) != 2 {
			t.Fatalf("history message reactions: %+v", foundMsg)
		}

		gotMsg, err := s.MessageByID(ctx, room.RoomID, msg.MsgID)
		if err != nil || len(gotMsg.Reactions) != 2 {
			t.Fatalf("message by id reactions: gotMsg=%+v err=%v", gotMsg, err)
		}
	})

	t.Run("read users follow read sequence", func(t *testing.T) {
		if err := s.MarkRoomRead(ctx, readers[0].UserID, room.RoomID, msg.RoomSeq); err != nil {
			t.Fatal(err)
		}
		if err := s.MarkRoomRead(ctx, readers[1].UserID, room.RoomID, msg.RoomSeq-1); err != nil {
			t.Fatal(err)
		}
		readUsers, err := s.ReadUsers(ctx, room.RoomID, msg.MsgID)
		if err != nil || len(readUsers) != 1 || readUsers[0] != readers[0].UserID {
			t.Fatalf("read users: list=%+v err=%v", readUsers, err)
		}
		seq, err := s.ReadSeq(ctx, readers[1].UserID, room.RoomID)
		if err != nil || seq != msg.RoomSeq-1 {
			t.Fatalf("read seq: seq=%d err=%v", seq, err)
		}
	})

	t.Run("pinned messages", func(t *testing.T) {
		if err := s.PinMessage(ctx, room.RoomID, uuid.NewV7(), owner.UserID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("pin missing message error = %v, want %v", err, ErrNotFound)
		}
		if err := s.PinMessage(ctx, room.RoomID, msg.MsgID, owner.UserID); err != nil {
			t.Fatal(err)
		}
		pins, err := s.PinnedMessages(ctx, room.RoomID)
		if err != nil || len(pins) != 1 || pins[0].MsgID != msg.MsgID {
			t.Fatalf("pins: list=%+v err=%v", pins, err)
		}
		if err := s.UnpinMessage(ctx, room.RoomID, msg.MsgID); err != nil {
			t.Fatal(err)
		}
		pins, err = s.PinnedMessages(ctx, room.RoomID)
		if err != nil || len(pins) != 0 {
			t.Fatalf("pins after unpin: list=%+v err=%v", pins, err)
		}
	})

	t.Run("device token lifecycle", func(t *testing.T) {
		user := readers[2]
		if _, err := s.DeviceToken(ctx, user.UserID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing token error = %v, want %v", err, ErrNotFound)
		}
		info := DeviceInfo{Token: "apns-token", Platform: "ios"}
		if err := s.SaveDeviceToken(ctx, user.UserID, info); err != nil {
			t.Fatal(err)
		}
		got, err := s.DeviceToken(ctx, user.UserID)
		if err != nil || got.Token != info.Token || got.Platform != info.Platform {
			t.Fatalf("device token: info=%+v err=%v", got, err)
		}
		if err := s.DeleteDeviceToken(ctx, user.UserID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DeviceToken(ctx, user.UserID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("deleted token error = %v, want %v", err, ErrNotFound)
		}
	})
}

func TestStoreConversations(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, s, "conv-alice")
	bob := mustCreateUser(t, s, "conv-bob")
	room := mustCreateRoom(t, s, ChatTypeGroup,
		Member{UserID: alice.UserID, Role: RoleOwner},
		Member{UserID: bob.UserID, Role: RoleMember},
	)

	t.Run("unread count and last message", func(t *testing.T) {
		last := mustWriteMessage(t, s, room.RoomID, alice.UserID, "hello bob")
		convs, err := s.Conversations(ctx, bob.UserID)
		if err != nil || len(convs) != 1 {
			t.Fatalf("conversations: list=%+v err=%v", convs, err)
		}
		info := convs[0]
		if info.UnreadCount != 1 || info.LastMessage == nil || info.LastMessage.MsgID != last.MsgID {
			t.Fatalf("conversation: info=%+v", info)
		}
	})

	t.Run("mark read and pin reorder", func(t *testing.T) {
		if err := s.MarkRoomRead(ctx, bob.UserID, room.RoomID, 1); err != nil {
			t.Fatal(err)
		}
		convs, err := s.Conversations(ctx, bob.UserID)
		if err != nil || convs[0].UnreadCount != 0 {
			t.Fatalf("conversations after read: list=%+v err=%v", convs, err)
		}

		other := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: bob.UserID, Role: RoleMember})
		mustWriteMessage(t, s, other.RoomID, bob.UserID, "newer room")

		pinned := true
		if err := s.UpdateMemberSettings(ctx, room.RoomID, bob.UserID, &pinned, nil); err != nil {
			t.Fatal(err)
		}
		convs, err = s.Conversations(ctx, bob.UserID)
		if err != nil || len(convs) != 2 || convs[0].Room.RoomID != room.RoomID {
			t.Fatalf("pinned room must sort first: list=%+v err=%v", convs, err)
		}
	})
}

func TestStoreLifecycle(t *testing.T) {
	ctx := context.Background()

	t.Run("ping and checkpoint reopen", func(t *testing.T) {
		dir := t.TempDir()
		data, err := Open(dir + "/store")
		if err != nil {
			t.Fatal(err)
		}
		sender := mustCreateUser(t, data, "ckpt-user")
		room := mustCreateRoom(t, data, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		mustWriteMessage(t, data, room.RoomID, sender.UserID, "durable")
		if err := data.Ping(ctx); err != nil {
			t.Fatalf("ping: %v", err)
		}
		if err := data.Checkpoint(ctx, dir+"/checkpoint"); err != nil {
			t.Fatal(err)
		}
		if err := data.Close(); err != nil {
			t.Fatal(err)
		}
		if err := data.Close(); err != nil {
			t.Fatalf("double close must be idempotent: %v", err)
		}

		checkpoint, err := Open(dir + "/checkpoint")
		if err != nil {
			t.Fatal(err)
		}
		defer checkpoint.Close()
		room2, err := checkpoint.Room(ctx, room.RoomID)
		if err != nil || room2.LastSeq != 1 {
			t.Fatalf("checkpoint room: room=%+v err=%v", room2, err)
		}
		page, err := checkpoint.Messages(ctx, room.RoomID, 0, 10)
		if err != nil || len(page.Messages) != 1 {
			t.Fatalf("checkpoint messages: page=%+v err=%v", page, err)
		}
	})

	t.Run("writes after close fail", func(t *testing.T) {
		data, err := Open(t.TempDir() + "/store")
		if err != nil {
			t.Fatal(err)
		}
		if err := data.Close(); err != nil {
			t.Fatal(err)
		}
		if err := data.Ping(ctx); !errors.Is(err, ErrClosed) {
			t.Fatalf("ping error = %v, want %v", err, ErrClosed)
		}
		results := data.WriteMessages(ctx, []Message{{RoomID: uuid.NewV7()}})
		if len(results) != 1 || !errors.Is(results[0].Err, ErrClosed) {
			t.Fatalf("closed write result: %+v", results)
		}
		if empty := data.WriteMessages(ctx, nil); empty != nil {
			t.Fatalf("empty batch must return nil, got %+v", empty)
		}
	})
}
