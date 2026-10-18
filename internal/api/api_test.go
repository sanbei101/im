package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"uuid"

	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
	"github.com/sanbei101/im/proto/pb"
)

func TestMain(m *testing.M) {
	if err := jwt.Init("test-secret"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

type testEnv struct {
	store   *store.Store
	stream  *StreamHandler
	handler http.Handler
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	data, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := data.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	stream := NewStreamHandler(data, 1024, 0, 1)
	return &testEnv{
		store:   data,
		stream:  stream,
		handler: NewRouter(data, stream, nil, "https://cdn.example.test"),
	}
}

func (e *testEnv) addUser(t *testing.T, name string) (store.User, string) {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte("secret1"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := e.store.CreateUser(context.Background(), name, string(hashed))
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwt.GenerateToken(user.UserID.String())
	if err != nil {
		t.Fatal(err)
	}
	return user, token
}

func (e *testEnv) addRoom(t *testing.T, owner store.User, members ...store.User) store.Room {
	t.Helper()
	return e.createRoom(t, uuid.NewV7(), owner, members...)
}

func (e *testEnv) createRoom(
	t *testing.T,
	roomID uuid.UUID,
	owner store.User,
	members ...store.User,
) store.Room {
	t.Helper()
	list := make([]store.Member, 0, len(members)+1)
	list = append(list, store.Member{UserID: owner.UserID, Role: store.RoleOwner})
	for _, member := range members {
		list = append(list, store.Member{UserID: member.UserID, Role: store.RoleMember})
	}
	room := store.Room{RoomID: roomID, ChatType: store.ChatTypeGroup, Name: "test-room"}
	if err := e.store.CreateRoom(context.Background(), room, list); err != nil {
		t.Fatal(err)
	}
	return room
}

func roomIDForNode(t *testing.T, nodeIndex int) uuid.UUID {
	t.Helper()
	for range 10_000 {
		roomID := uuid.NewV7()
		slot, err := pkg.RoomSlot(roomID, 1024)
		if err != nil {
			t.Fatal(err)
		}
		index, err := pkg.NodeIndex(slot, 1024, 2)
		if err != nil {
			t.Fatal(err)
		}
		if index == nodeIndex {
			return roomID
		}
	}
	t.Fatal("could not find a room for the requested node")
	return uuid.Nil()
}

func doRequest(
	t *testing.T,
	handler http.Handler,
	method, path, token string,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func jsonBody[T any](t *testing.T, value T) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
	return out
}

func decodeResponse[T any](t *testing.T, recorder *httptest.ResponseRecorder) render.Response[T] {
	t.Helper()
	return decode[render.Response[T]](t, recorder)
}

func requireStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, want, recorder.Body.String())
	}
}

func newConnection() *apiConnection {
	return &apiConnection{
		outbox: make(chan *pb.APIFrame, 8),
		quit:   make(chan struct{}),
		users:  make(map[string]struct{}),
	}
}

func nextFrame(t *testing.T, connection *apiConnection) *pb.APIFrame {
	t.Helper()
	select {
	case frame := <-connection.outbox:
		return frame
	case <-time.After(time.Second):
		t.Fatal("no frame was enqueued")
		return nil
	}
}

func TestStreamHandler(t *testing.T) {
	t.Run("sessions", func(t *testing.T) {
		env := newTestEnv(t)
		userID := uuid.NewV7().String()
		connection := newConnection()

		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: userID,
			Online: true,
		}}})
		if !env.stream.IsOnline(userID) {
			t.Fatal("online session was not registered")
		}
		if got := env.stream.onlineConnection(userID); got != connection {
			t.Fatal("online lookup returned the wrong connection")
		}

		stale := newConnection()
		env.stream.updateSessions(stale, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: userID,
			Online: false,
		}}})
		if !env.stream.IsOnline(userID) {
			t.Fatal("offline from a non-owner connection cleared the session")
		}

		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: userID,
			Online: false,
		}}})
		if env.stream.IsOnline(userID) {
			t.Fatal("offline from the owning connection did not clear the session")
		}

		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: userID,
			Online: true,
		}}})
		env.stream.removeConnection(connection)
		if env.stream.IsOnline(userID) {
			t.Fatal("removeConnection did not clear the session")
		}
	})

	t.Run("notifications", func(t *testing.T) {
		env := newTestEnv(t)
		alice, _ := env.addUser(t, "alice")
		bob, _ := env.addUser(t, "bob")
		room := env.addRoom(t, alice, bob)

		connection := newConnection()
		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: bob.UserID.String(),
			Online: true,
		}}})

		env.stream.NotifyUsers([]uuid.UUID{bob.UserID, alice.UserID}, FriendAccepted{
			UserID:   alice.UserID,
			Username: "alice",
		})
		frame := nextFrame(t, connection)
		batch := frame.GetNotificationBatch()
		if batch == nil || len(batch.GetNotifications()) != 1 {
			t.Fatalf("notification batch = %+v", batch)
		}
		notification := batch.GetNotifications()[0]
		if notification.GetUserId() != bob.UserID.String() {
			t.Fatalf("notification user = %q", notification.GetUserId())
		}
		if notification.GetEvent() != "friend_accepted" {
			t.Fatalf("notification event = %q", notification.GetEvent())
		}
		var accepted FriendAccepted
		if err := json.Unmarshal(notification.GetData(), &accepted); err != nil {
			t.Fatalf("decode notification payload: %v", err)
		}
		if accepted.UserID != alice.UserID {
			t.Fatalf("notification payload user = %s", accepted.UserID)
		}

		env.stream.NotifyRoom(context.Background(), room.RoomID, uuid.Nil(), RoomMembersChanged{
			RoomID: room.RoomID,
			Action: "joined",
		})
		frame = nextFrame(t, connection)
		batch = frame.GetNotificationBatch()
		if batch == nil || len(batch.GetNotifications()) != 1 {
			t.Fatalf("room notification batch = %+v", batch)
		}
		if batch.GetNotifications()[0].GetEvent() != "room_members_changed" {
			t.Fatalf("room notification event = %q", batch.GetNotifications()[0].GetEvent())
		}

		select {
		case extra := <-connection.outbox:
			t.Fatalf("unexpected extra frame: %+v", extra)
		default:
		}
	})

	t.Run("room member cache", func(t *testing.T) {
		env := newTestEnv(t)
		alice, _ := env.addUser(t, "alice")
		bob, _ := env.addUser(t, "bob")
		carol, _ := env.addUser(t, "carol")
		room := env.addRoom(t, alice, bob)

		members, err := env.stream.getRoomMembers(context.Background(), room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		if len(members) != 2 {
			t.Fatalf("members = %d, want 2", len(members))
		}

		if err := env.store.AddMembers(context.Background(), room.RoomID, []store.Member{{
			UserID: carol.UserID,
			Role:   store.RoleMember,
		}}); err != nil {
			t.Fatal(err)
		}

		members, err = env.stream.getRoomMembers(context.Background(), room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		if len(members) != 2 {
			t.Fatalf("cached members = %d, want 2", len(members))
		}

		env.stream.InvalidateRoomMembers(room.RoomID)
		members, err = env.stream.getRoomMembers(context.Background(), room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		if len(members) != 3 {
			t.Fatalf("refreshed members = %d, want 3", len(members))
		}
	})

	t.Run("write batch", func(t *testing.T) {
		env := newTestEnv(t)
		sender, _ := env.addUser(t, "sender")
		room := env.createRoom(t, roomIDForNode(t, 0), sender)
		owner := NewStreamHandler(env.store, 1024, 0, 2)
		foreign := NewStreamHandler(env.store, 1024, 1, 2)

		clientMsgID := uuid.NewV7()
		input := &pb.SendMessage{
			RequestId:   "req-1",
			ClientMsgId: clientMsgID.String(),
			SenderId:    sender.UserID.String(),
			RoomId:      room.RoomID.String(),
			MsgType:     int32(store.MsgTypeText),
			Payload:     []byte(`{"text":"hello"}`),
			Ext:         []byte(`{}`),
		}

		result, written, err := owner.writeBatch(context.Background(), &pb.SendBatch{
			BatchId:  "batch-1",
			Messages: []*pb.SendMessage{input},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.GetResults()) != 1 || result.GetResults()[0].GetCode() != 0 {
			t.Fatalf("write result = %+v", result.GetResults())
		}
		if len(written) != 1 || written[0].RoomSeq != 1 {
			t.Fatalf("written messages = %+v", written)
		}
		first := result.GetResults()[0]

		duplicate, _, err := owner.writeBatch(context.Background(), &pb.SendBatch{
			BatchId:  "batch-2",
			Messages: []*pb.SendMessage{input},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := duplicate.GetResults()[0]; got.GetMsgId() != first.GetMsgId() ||
			got.GetRoomSeq() != first.GetRoomSeq() {
			t.Fatalf("duplicate result = %+v, want msg_id=%s seq=%d", got, first.GetMsgId(), first.GetRoomSeq())
		}

		conflictInput := *input
		conflictInput.Payload = []byte(`{"text":"different"}`)
		conflict, _, err := owner.writeBatch(context.Background(), &pb.SendBatch{
			BatchId:  "batch-3",
			Messages: []*pb.SendMessage{&conflictInput},
		})
		if err != nil {
			t.Fatal(err)
		}
		if conflict.GetResults()[0].GetCode() == 0 {
			t.Fatal("conflicting request was accepted")
		}

		foreignInput := *input
		foreignInput.ClientMsgId = uuid.NewV7().String()
		foreignResult, _, err := foreign.writeBatch(context.Background(), &pb.SendBatch{
			BatchId:  "batch-4",
			Messages: []*pb.SendMessage{&foreignInput},
		})
		if err != nil {
			t.Fatal(err)
		}
		if foreignResult.GetResults()[0].GetCode() == 0 {
			t.Fatal("a non-owner node accepted the message")
		}

		if err := foreign.checkRoomOwned(room.RoomID); err == nil {
			t.Fatal("non-owner node reported ownership")
		}
	})

	t.Run("push", func(t *testing.T) {
		env := newTestEnv(t)
		alice, _ := env.addUser(t, "alice")
		bob, _ := env.addUser(t, "bob")
		room := env.addRoom(t, alice, bob)

		message, err := env.store.WriteMessage(context.Background(), store.Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    alice.UserID,
			RoomID:      room.RoomID,
			MsgType:     store.MsgTypeText,
			Payload:     []byte(`{"text":"hello"}`),
			Ext:         []byte(`{}`),
		})
		if err != nil {
			t.Fatal(err)
		}

		connection := newConnection()
		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: bob.UserID.String(),
			Online: true,
		}}})

		if err := env.stream.Push(context.Background(), &message); err != nil {
			t.Fatal(err)
		}
		frame := nextFrame(t, connection)
		batch := frame.GetRoomPushBatch()
		if batch == nil || len(batch.GetRoomPushes()) != 1 {
			t.Fatalf("room push batch = %+v", batch)
		}
		push := batch.GetRoomPushes()[0]
		if push.GetRoomId() != room.RoomID.String() || push.GetRoomSeq() != message.RoomSeq {
			t.Fatalf("room push = %+v", push)
		}
		if len(push.GetUserIds()) != 1 || push.GetUserIds()[0] != bob.UserID.String() {
			t.Fatalf("room push users = %+v", push.GetUserIds())
		}
	})

	t.Run("outbox", func(t *testing.T) {
		connection := newAPIConnection(nil)
		frame := &pb.APIFrame{Body: &pb.APIFrame_NotificationBatch{
			NotificationBatch: &pb.NotificationBatch{},
		}}
		for range connectionOutboxSize {
			if err := connection.enqueue(frame); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
		}
		if err := connection.enqueue(frame); !errors.Is(err, errOutboxFull) {
			t.Fatalf("enqueue on full outbox = %v", err)
		}
	})
}
