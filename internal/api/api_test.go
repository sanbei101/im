package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/engine"
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
	db      *store.DB
	data    *engine.Engine
	stream  *StreamHandler
	members *Membership
	handler http.Handler
}

func testDBURL() string {
	if value := os.Getenv("TEST_POSTGRES_URL"); value != "" {
		return value
	}
	return "postgres://127.0.0.1:5432/im_test?sslmode=disable"
}

func newTestEnv(tb testing.TB) *testEnv {
	tb.Helper()
	ctx := context.Background()
	db, err := store.OpenDB(ctx, testDBURL())
	if err != nil {
		tb.Fatalf("open postgres: %v", err)
	}
	tb.Cleanup(db.Close)

	admin, err := pgxpool.New(ctx, testDBURL())
	if err != nil {
		tb.Fatalf("open postgres admin: %v", err)
	}
	if _, err := admin.Exec(ctx, `TRUNCATE users, rooms CASCADE`); err != nil {
		admin.Close()
		tb.Fatalf("truncate test database: %v", err)
	}
	admin.Close()

	data, err := engine.Open(tb.TempDir())
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := data.Close(); err != nil {
			tb.Errorf("close engine: %v", err)
		}
	})
	members := NewMembership(db)
	stream := NewStreamHandler(data, members, 1024, 0, 1)
	return &testEnv{
		db:      db,
		data:    data,
		stream:  stream,
		members: members,
		handler: NewRouter(db, data, stream, members, nil, "https://cdn.example.test", "test-node"),
	}
}

func (e *testEnv) addUser(tb testing.TB, name string) (store.User, string) {
	tb.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte("secret1"), bcrypt.MinCost)
	if err != nil {
		tb.Fatal(err)
	}
	user, err := e.db.CreateUser(context.Background(), name, string(hashed))
	if err != nil {
		tb.Fatal(err)
	}
	token, err := jwt.GenerateToken(user.UserID.String())
	if err != nil {
		tb.Fatal(err)
	}
	return user, token
}

func (e *testEnv) addRoom(tb testing.TB, owner store.User, members ...store.User) store.Room {
	tb.Helper()
	return e.createRoom(tb, uuid.NewV7(), owner, members...)
}

func (e *testEnv) createRoom(
	tb testing.TB,
	roomID uuid.UUID,
	owner store.User,
	members ...store.User,
) store.Room {
	tb.Helper()
	list := make([]store.Member, 0, len(members)+1)
	list = append(list, store.Member{UserID: owner.UserID, Role: store.RoleOwner})
	for _, member := range members {
		list = append(list, store.Member{UserID: member.UserID, Role: store.RoleMember})
	}
	room := store.Room{RoomID: roomID, ChatType: store.ChatTypeGroup, Name: "test-room"}
	if err := e.db.CreateRoom(context.Background(), room, list); err != nil {
		tb.Fatal(err)
	}
	return room
}

func roomIDForNode(tb testing.TB, nodeIndex int) uuid.UUID {
	tb.Helper()
	for range 10_000 {
		roomID := uuid.NewV7()
		slot, err := pkg.RoomSlot(roomID, 1024)
		if err != nil {
			tb.Fatal(err)
		}
		index, err := pkg.NodeIndex(slot, 1024, 2)
		if err != nil {
			tb.Fatal(err)
		}
		if index == nodeIndex {
			return roomID
		}
	}
	tb.Fatal("could not find a room for the requested node")
	return uuid.Nil()
}

func doRequest(
	tb testing.TB,
	handler http.Handler,
	method, path, token string,
	body []byte,
) *httptest.ResponseRecorder {
	tb.Helper()
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

func jsonBody[T any](tb testing.TB, value T) []byte {
	tb.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		tb.Fatal(err)
	}
	return body
}

func decode[T any](tb testing.TB, recorder *httptest.ResponseRecorder) T {
	tb.Helper()
	var out T
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		tb.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
	return out
}

func decodeResponse[T any](tb testing.TB, recorder *httptest.ResponseRecorder) render.Response[T] {
	tb.Helper()
	return decode[render.Response[T]](tb, recorder)
}

func requireStatus(tb testing.TB, recorder *httptest.ResponseRecorder, want int) {
	tb.Helper()
	if recorder.Code != want {
		tb.Fatalf("status = %d, want %d; body=%s", recorder.Code, want, recorder.Body.String())
	}
}

func newConnection() *apiConnection {
	return &apiConnection{
		outbox: make(chan *pb.APIFrame, 8),
		quit:   make(chan struct{}),
		users:  make(map[uuid.UUID]struct{}),
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
		userID := uuid.NewV7()
		connection := newConnection()

		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: userID.String(),
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
			UserId: userID.String(),
			Online: false,
		}}})
		if !env.stream.IsOnline(userID) {
			t.Fatal("offline from a non-owner connection cleared the session")
		}

		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: userID.String(),
			Online: false,
		}}})
		if env.stream.IsOnline(userID) {
			t.Fatal("offline from the owning connection did not clear the session")
		}

		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: userID.String(),
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

		members, err := env.members.Members(context.Background(), room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		if len(members) != 2 {
			t.Fatalf("members = %d, want 2", len(members))
		}

		if _, err := env.db.AddMembers(context.Background(), room.RoomID, []store.Member{{
			UserID: carol.UserID,
			Role:   store.RoleMember,
		}}); err != nil {
			t.Fatal(err)
		}

		members, err = env.members.Members(context.Background(), room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		if len(members) != 2 {
			t.Fatalf("cached members = %d, want 2", len(members))
		}

		env.members.Invalidate(room.RoomID)
		members, err = env.members.Members(context.Background(), room.RoomID)
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
		owner := NewStreamHandler(env.data, env.members, 1024, 0, 2)
		foreign := NewStreamHandler(env.data, env.members, 1024, 1, 2)

		clientMsgID := uuid.NewV7()
		input := &pb.SendMessage{
			RequestId:   "req-1",
			ClientMsgId: clientMsgID.String(),
			SenderId:    sender.UserID.String(),
			RoomId:      room.RoomID.String(),
			MsgType:     int32(engine.MsgTypeText),
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
		if got := duplicate.GetResults()[0]; got.GetRoomSeq() != first.GetRoomSeq() {
			t.Fatalf("duplicate result = %+v, want seq=%d", got, first.GetRoomSeq())
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

		message, err := env.data.WriteMessage(context.Background(), engine.Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    alice.UserID,
			RoomID:      room.RoomID,
			MsgType:     engine.MsgTypeText,
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

func TestUserHTTP(t *testing.T) {
	t.Run("auth and presence", func(t *testing.T) {
		env := newTestEnv(t)
		bob, _ := env.addUser(t, "bob")

		recorder := doRequest(t, env.handler, http.MethodPost, "/api/v1/users/register", "",
			jsonBody(t, UserAuthReq{Username: "alice", Password: "secret1"}))
		requireStatus(t, recorder, http.StatusOK)
		registered := decodeResponse[UserResp](t, recorder)
		if registered.Code != http.StatusOK || registered.Data.Token == "" {
			t.Fatalf("register response = %+v", registered)
		}

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/users/login", "",
			jsonBody(t, UserAuthReq{Username: "alice", Password: "secret1"}))
		requireStatus(t, recorder, http.StatusOK)
		loggedIn := decodeResponse[UserResp](t, recorder)
		if loggedIn.Data.Token == "" {
			t.Fatal("login did not return a token")
		}

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/users/profile",
			registered.Data.Token, nil)
		requireStatus(t, recorder, http.StatusOK)
		profile := decodeResponse[UserProfileResp](t, recorder)
		if profile.Data.Username != "alice" {
			t.Fatalf("profile username = %q", profile.Data.Username)
		}

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/users/profile", "", nil)
		requireStatus(t, recorder, http.StatusUnauthorized)
		apiError := decode[render.ResponseWithoutData](t, recorder)
		if apiError.Code != http.StatusUnauthorized {
			t.Fatalf("error response code = %d", apiError.Code)
		}

		connection := newConnection()
		env.stream.updateSessions(connection, &pb.SessionBatch{Sessions: []*pb.Session{{
			UserId: bob.UserID.String(),
			Online: true,
		}}})

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/users/presence",
			registered.Data.Token, jsonBody(t, PresenceReq{UserIDs: []string{
				bob.UserID.String(),
				registered.Data.UserID,
			}}))
		requireStatus(t, recorder, http.StatusOK)
		presence := decodeResponse[PresenceResp](t, recorder)
		if !presence.Data.Presence[bob.UserID.String()] {
			t.Fatal("online user reported offline")
		}
		if presence.Data.Presence[registered.Data.UserID] {
			t.Fatal("offline user reported online")
		}
	})

	t.Run("account", func(t *testing.T) {
		env := newTestEnv(t)
		alice, token := env.addUser(t, "alice")

		recorder := doRequest(t, env.handler, http.MethodPut, "/api/v1/users/profile", token,
			jsonBody(t, UpdateProfileReq{Nickname: "Alice", AvatarURL: "https://example.test/a.png"}))
		requireStatus(t, recorder, http.StatusOK)
		profile := decodeResponse[UserProfileResp](t, recorder)
		if profile.Data.Nickname != "Alice" || profile.Data.AvatarURL != "https://example.test/a.png" {
			t.Fatalf("profile = %+v", profile.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/users/search?keyword=ali", token, nil)
		requireStatus(t, recorder, http.StatusOK)
		results := decodeResponse[[]UserProfileResp](t, recorder)
		if len(results.Data) != 1 || results.Data[0].UserID != alice.UserID.String() {
			t.Fatalf("search results = %+v", results.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/devices/token", token,
			jsonBody(t, DeviceTokenReq{Token: "device-1", Platform: "ios"}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/users/logout", token, nil)
		requireStatus(t, recorder, http.StatusOK)
		if _, err := env.db.DeviceToken(context.Background(), alice.UserID); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("device token after logout: %v", err)
		}

		recorder = doRequest(t, env.handler, http.MethodPut, "/api/v1/users/password", token,
			jsonBody(t, UpdatePasswordReq{OldPassword: "secret1", NewPassword: "secret2"}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/users/login", "",
			jsonBody(t, UserAuthReq{Username: "alice", Password: "secret2"}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/users/login", "",
			jsonBody(t, UserAuthReq{Username: "alice", Password: "secret1"}))
		requireStatus(t, recorder, http.StatusUnauthorized)
	})
}

func TestRoomHTTP(t *testing.T) {
	t.Run("group and messages", func(t *testing.T) {
		env := newTestEnv(t)
		alice, aliceToken := env.addUser(t, "alice")
		bob, bobToken := env.addUser(t, "bob")
		carol, _ := env.addUser(t, "carol")

		wantRoomID := uuid.NewV7()
		recorder := doRequest(t, env.handler, http.MethodPost, "/api/v1/rooms/group", aliceToken,
			jsonBody(t, CreateGroupRoomReq{
				RoomID:    wantRoomID.String(),
				Name:      "team",
				MemberIDs: []string{bob.UserID.String(), carol.UserID.String()},
			}))
		requireStatus(t, recorder, http.StatusOK)
		roomResp := decodeResponse[RoomResp](t, recorder)
		if roomResp.Data.RoomID != wantRoomID.String() {
			t.Fatalf("room id = %q, want %q", roomResp.Data.RoomID, wantRoomID)
		}
		roomID := wantRoomID

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/rooms/list", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		rooms := decodeResponse[RoomListResp](t, recorder)
		found := false
		for _, room := range rooms.Data.Rooms {
			if room.RoomID == roomID.String() {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("room %s missing from %+v", roomID, rooms.Data.Rooms)
		}

		recorder = doRequest(t, env.handler, http.MethodGet,
			"/api/v1/rooms/"+roomID.String()+"/members", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		members := decodeResponse[[]MemberInfoResp](t, recorder)
		if len(members.Data) != 3 {
			t.Fatalf("members = %d, want 3", len(members.Data))
		}

		message, err := env.data.WriteMessage(context.Background(), engine.Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    alice.UserID,
			RoomID:      roomID,
			MsgType:     engine.MsgTypeText,
			Payload:     []byte(`{"text":"hello"}`),
			Ext:         []byte(`{}`),
		})
		if err != nil {
			t.Fatal(err)
		}

		recorder = doRequest(t, env.handler, http.MethodGet,
			"/api/v1/messages/history?room_id="+roomID.String()+"&page_size=10", bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		page := decodeResponse[engine.MessagePage](t, recorder)
		if len(page.Data.Messages) != 1 || page.Data.Messages[0].RoomSeq != message.RoomSeq {
			t.Fatalf("history = %+v", page.Data.Messages)
		}

		reactionPath := "/api/v1/messages/" + strconv.FormatUint(message.RoomSeq, 10) + "/reactions"
		recorder = doRequest(t, env.handler, http.MethodPost, reactionPath+"?room_id="+roomID.String(), bobToken,
			jsonBody(t, ReactionReq{Emoji: "👍"}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodGet,
			reactionPath+"?room_id="+roomID.String(), bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		groups := decodeResponse[[]engine.ReactionGroup](t, recorder)
		if len(groups.Data) != 1 || groups.Data[0].Emoji != "👍" || groups.Data[0].Count != 1 {
			t.Fatalf("reaction groups = %+v", groups.Data)
		}
		if len(groups.Data[0].UserIDs) != 1 || groups.Data[0].UserIDs[0] != bob.UserID {
			t.Fatalf("reaction users = %+v", groups.Data[0].UserIDs)
		}

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/conversations", bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		conversations := decodeResponse[ConversationListResp](t, recorder)
		if len(conversations.Data.Conversations) != 1 ||
			conversations.Data.Conversations[0].Room.RoomID != roomID {
			t.Fatalf("conversations = %+v", conversations.Data.Conversations)
		}

		recorder = doRequest(t, env.handler, http.MethodPost,
			"/api/v1/rooms/"+roomID.String()+"/read", bobToken,
			jsonBody(t, MarkReadReq{ReadSeq: message.RoomSeq}))
		requireStatus(t, recorder, http.StatusOK)
		readSeq, err := env.data.ReadSeq(context.Background(), roomID, bob.UserID)
		if err != nil || readSeq != message.RoomSeq {
			t.Fatalf("read seq = %d, want %d; err=%v", readSeq, message.RoomSeq, err)
		}

		recorder = doRequest(
			t,
			env.handler,
			http.MethodGet,
			"/api/v1/messages/"+strconv.FormatUint(
				message.RoomSeq,
				10,
			)+"/read_users?room_id="+roomID.String(),
			bobToken,
			nil,
		)
		requireStatus(t, recorder, http.StatusOK)
		readUsers := decodeResponse[ReadUsersResp](t, recorder)
		if readUsers.Data.Count != 1 || len(readUsers.Data.ReadUserIDs) != 1 ||
			readUsers.Data.ReadUserIDs[0] != bob.UserID {
			t.Fatalf("read users = %+v", readUsers.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/rooms/"+roomID.String(), bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		detail := decodeResponse[RoomDetailResp](t, recorder)
		if detail.Data.MyRole != store.RoleMember {
			t.Fatalf("role = %q", detail.Data.MyRole)
		}
	})

	t.Run("conversation settings", func(t *testing.T) {
		env := newTestEnv(t)
		alice, token := env.addUser(t, "alice")
		bob, _ := env.addUser(t, "bob")
		room := env.addRoom(t, alice, bob)

		message, err := env.data.WriteMessage(context.Background(), engine.Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    alice.UserID,
			RoomID:      room.RoomID,
			MsgType:     engine.MsgTypeText,
			Payload:     []byte(`{"text":"hello"}`),
			Ext:         []byte(`{}`),
		})
		if err != nil {
			t.Fatal(err)
		}

		recorder := doRequest(t, env.handler, http.MethodPut,
			"/api/v1/conversations/"+room.RoomID.String()+"/pin", token,
			jsonBody(t, PinConversationReq{IsPinned: true}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodPut,
			"/api/v1/conversations/"+room.RoomID.String()+"/mute", token,
			jsonBody(t, MuteConversationReq{IsMuted: true}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/conversations", token, nil)
		requireStatus(t, recorder, http.StatusOK)
		conversations := decodeResponse[ConversationListResp](t, recorder)
		if len(conversations.Data.Conversations) != 1 {
			t.Fatalf("conversations = %+v", conversations.Data.Conversations)
		}
		member := conversations.Data.Conversations[0].Member
		if !member.IsPinned || !member.IsMuted {
			t.Fatalf("member settings = %+v", member)
		}

		recorder = doRequest(t, env.handler, http.MethodPut,
			"/api/v1/rooms/"+room.RoomID.String()+"/clear_unread", token, nil)
		requireStatus(t, recorder, http.StatusOK)

		readSeq, err := env.data.ReadSeq(context.Background(), room.RoomID, alice.UserID)
		if err != nil || readSeq != message.RoomSeq {
			t.Fatalf("read seq = %d, want %d; err=%v", readSeq, message.RoomSeq, err)
		}
	})

	t.Run("moderation", func(t *testing.T) {
		env := newTestEnv(t)
		alice, aliceToken := env.addUser(t, "alice")
		bob, bobToken := env.addUser(t, "bob")
		room := env.addRoom(t, alice, bob)

		message, err := env.data.WriteMessage(context.Background(), engine.Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    alice.UserID,
			RoomID:      room.RoomID,
			MsgType:     engine.MsgTypeText,
			Payload:     []byte(`{"text":"hello"}`),
			Ext:         []byte(`{}`),
		})
		if err != nil {
			t.Fatal(err)
		}
		messagePath := "/api/v1/messages/" + strconv.FormatUint(message.RoomSeq, 10)

		recorder := doRequest(t, env.handler, http.MethodGet,
			messagePath+"/read_users?room_id="+room.RoomID.String(), bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		readUsers := decodeResponse[ReadUsersResp](t, recorder)
		if readUsers.Data.Count != 0 {
			t.Fatalf("read users = %+v", readUsers.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodGet,
			"/api/v1/messages/search?room_id="+room.RoomID.String()+"&keyword=hello", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		found := decodeResponse[[]engine.Message](t, recorder)
		if len(found.Data) != 1 || found.Data[0].RoomSeq != message.RoomSeq {
			t.Fatalf("search results = %+v", found.Data)
		}

		reactionPath := messagePath + "/reactions"
		recorder = doRequest(t, env.handler, http.MethodPost, reactionPath+"?room_id="+room.RoomID.String(), bobToken,
			jsonBody(t, ReactionReq{Emoji: "👍"}))
		requireStatus(t, recorder, http.StatusOK)
		recorder = doRequest(t, env.handler, http.MethodDelete,
			reactionPath+"?room_id="+room.RoomID.String()+"&emoji=%F0%9F%91%8D", bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		recorder = doRequest(t, env.handler, http.MethodGet,
			reactionPath+"?room_id="+room.RoomID.String(), bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		groups := decodeResponse[[]engine.ReactionGroup](t, recorder)
		if len(groups.Data) != 0 {
			t.Fatalf("reaction groups = %+v", groups.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodPost,
			"/api/v1/rooms/"+room.RoomID.String()+"/pins", aliceToken,
			jsonBody(t, PinReq{RoomSeq: message.RoomSeq}))
		requireStatus(t, recorder, http.StatusOK)
		recorder = doRequest(t, env.handler, http.MethodGet,
			"/api/v1/rooms/"+room.RoomID.String()+"/pins", bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		pins := decodeResponse[[]engine.Message](t, recorder)
		if len(pins.Data) != 1 || pins.Data[0].RoomSeq != message.RoomSeq {
			t.Fatalf("pins = %+v", pins.Data)
		}
		recorder = doRequest(t, env.handler, http.MethodDelete,
			"/api/v1/rooms/"+room.RoomID.String()+"/pins/"+strconv.FormatUint(message.RoomSeq, 10), aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodPost,
			"/api/v1/messages/recall?room_id="+room.RoomID.String(), aliceToken,
			jsonBody(t, RecallReq{RoomSeq: message.RoomSeq}))
		requireStatus(t, recorder, http.StatusOK)
		recalled := decodeResponse[engine.Message](t, recorder)
		if recalled.Data.MsgType != engine.MsgTypeRecall {
			t.Fatalf("recalled msg type = %s", recalled.Data.MsgType)
		}
	})

	t.Run("membership", func(t *testing.T) {
		env := newTestEnv(t)
		alice, aliceToken := env.addUser(t, "alice")
		bob, bobToken := env.addUser(t, "bob")
		carol, _ := env.addUser(t, "carol")
		room := env.addRoom(t, alice, bob)
		roomPath := "/api/v1/rooms/" + room.RoomID.String()

		recorder := doRequest(t, env.handler, http.MethodPost, roomPath+"/members", aliceToken,
			jsonBody(t, AddMembersReq{MemberIDs: []string{carol.UserID.String()}}))
		requireStatus(t, recorder, http.StatusOK)
		if _, err := env.db.Member(context.Background(), room.RoomID, carol.UserID); err != nil {
			t.Fatalf("load added member: %v", err)
		}

		recorder = doRequest(t, env.handler, http.MethodPut,
			roomPath+"/members/"+carol.UserID.String()+"/role", aliceToken,
			jsonBody(t, UpdateMemberRoleReq{Role: store.RoleAdmin}))
		requireStatus(t, recorder, http.StatusOK)
		member, err := env.db.Member(context.Background(), room.RoomID, carol.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if member.Role != store.RoleAdmin {
			t.Fatalf("role = %q", member.Role)
		}

		recorder = doRequest(t, env.handler, http.MethodDelete,
			roomPath+"/members/"+carol.UserID.String(), aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		if _, err := env.db.Member(
			context.Background(),
			room.RoomID,
			carol.UserID,
		); !errors.Is(
			err,
			store.ErrNotFound,
		) {
			t.Fatalf("removed member lookup: %v", err)
		}

		recorder = doRequest(t, env.handler, http.MethodPost, roomPath+"/transfer", aliceToken,
			jsonBody(t, TransferOwnerReq{NewOwnerID: bob.UserID.String()}))
		requireStatus(t, recorder, http.StatusOK)
		owner, err := env.db.Member(context.Background(), room.RoomID, bob.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if owner.Role != store.RoleOwner {
			t.Fatalf("new owner role = %q", owner.Role)
		}

		recorder = doRequest(t, env.handler, http.MethodPost, roomPath+"/leave", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		if _, err := env.db.Member(
			context.Background(),
			room.RoomID,
			alice.UserID,
		); !errors.Is(
			err,
			store.ErrNotFound,
		) {
			t.Fatalf("left member lookup: %v", err)
		}

		recorder = doRequest(t, env.handler, http.MethodDelete, roomPath, bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		if _, err := env.db.Room(context.Background(), room.RoomID); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("dissolved room lookup: %v", err)
		}
	})

	t.Run("single chat and update", func(t *testing.T) {
		env := newTestEnv(t)
		alice, aliceToken := env.addUser(t, "alice")
		bob, bobToken := env.addUser(t, "bob")

		recorder := doRequest(t, env.handler, http.MethodPost, "/api/v1/rooms/single", aliceToken,
			jsonBody(t, CreateSingleRoomReq{UserID2: bob.UserID.String()}))
		requireStatus(t, recorder, http.StatusOK)
		created := decodeResponse[RoomResp](t, recorder)

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/rooms/single", bobToken,
			jsonBody(t, CreateSingleRoomReq{UserID2: alice.UserID.String()}))
		requireStatus(t, recorder, http.StatusOK)
		again := decodeResponse[RoomResp](t, recorder)
		if again.Data.RoomID != created.Data.RoomID {
			t.Fatalf("single chat room changed: %s -> %s", created.Data.RoomID, again.Data.RoomID)
		}

		roomPath := "/api/v1/rooms/" + created.Data.RoomID
		recorder = doRequest(t, env.handler, http.MethodGet, roomPath, bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		single := decodeResponse[RoomDetailResp](t, recorder)
		if single.Data.ChatType != store.ChatTypeSingle {
			t.Fatalf("single chat type = %q", single.Data.ChatType)
		}

		group := env.addRoom(t, alice, bob)
		groupPath := "/api/v1/rooms/" + group.RoomID.String()
		recorder = doRequest(t, env.handler, http.MethodPut, groupPath, aliceToken,
			jsonBody(t, UpdateRoomReq{
				Name:      "renamed",
				AvatarURL: "https://example.test/room.png",
				Notice:    "welcome",
			}))
		requireStatus(t, recorder, http.StatusOK)
		updated := decodeResponse[RoomDetailResp](t, recorder)
		if updated.Data.Name != "renamed" || updated.Data.Notice != "welcome" {
			t.Fatalf("updated room = %+v", updated.Data)
		}
		if updated.Data.MyRole != store.RoleOwner {
			t.Fatalf("group owner role = %q", updated.Data.MyRole)
		}

		recorder = doRequest(t, env.handler, http.MethodGet, groupPath, bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		detail := decodeResponse[RoomDetailResp](t, recorder)
		if detail.Data.Name != "renamed" || detail.Data.Notice != "welcome" {
			t.Fatalf("room detail = %+v", detail.Data)
		}
	})
}

func TestFriendHTTP(t *testing.T) {
	t.Run("flow", func(t *testing.T) {
		env := newTestEnv(t)
		alice, aliceToken := env.addUser(t, "alice")
		bob, bobToken := env.addUser(t, "bob")

		recorder := doRequest(t, env.handler, http.MethodPost, "/api/v1/friends/apply", aliceToken,
			jsonBody(t, ApplyFriendReq{TargetID: bob.UserID.String(), Greeting: "hi"}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/friends/applications", bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		applications := decodeResponse[[]store.FriendApplication](t, recorder)
		if len(applications.Data) != 1 || applications.Data[0].FromUserID != alice.UserID {
			t.Fatalf("applications = %+v", applications.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/friends/audit", bobToken,
			jsonBody(t, AuditFriendReq{FromUserID: alice.UserID.String(), Action: "accept"}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/friends/", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		friends := decodeResponse[[]FriendItemResp](t, recorder)
		if len(friends.Data) != 1 || friends.Data[0].UserID != bob.UserID.String() {
			t.Fatalf("friends = %+v", friends.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/friends/blacklist", aliceToken,
			jsonBody(t, BlacklistReq{TargetID: bob.UserID.String()}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/friends/blacklist", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		blocked := decodeResponse[[]UserProfileResp](t, recorder)
		if len(blocked.Data) != 1 || blocked.Data[0].UserID != bob.UserID.String() {
			t.Fatalf("blacklist = %+v", blocked.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodDelete,
			"/api/v1/friends/blacklist/"+bob.UserID.String(), aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
	})

	t.Run("remark and delete", func(t *testing.T) {
		env := newTestEnv(t)
		alice, aliceToken := env.addUser(t, "alice")
		bob, _ := env.addUser(t, "bob")

		if err := env.db.ApplyFriend(context.Background(), alice.UserID, bob.UserID, "hi"); err != nil {
			t.Fatal(err)
		}
		if err := env.db.AuditFriend(context.Background(), bob.UserID, alice.UserID, true); err != nil {
			t.Fatal(err)
		}

		recorder := doRequest(t, env.handler, http.MethodPut,
			"/api/v1/friends/"+bob.UserID.String()+"/remark", aliceToken,
			jsonBody(t, UpdateRemarkReq{Remark: "Bobby"}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/friends/", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		friends := decodeResponse[[]FriendItemResp](t, recorder)
		if len(friends.Data) != 1 || friends.Data[0].Remark != "Bobby" {
			t.Fatalf("friends = %+v", friends.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodDelete,
			"/api/v1/friends/"+bob.UserID.String(), aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/friends/", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		friends = decodeResponse[[]FriendItemResp](t, recorder)
		if len(friends.Data) != 0 {
			t.Fatalf("friends after delete = %+v", friends.Data)
		}
	})
}

func TestFileHTTP(t *testing.T) {
	t.Run("presign upload", func(t *testing.T) {
		env := newTestEnv(t)
		_, token := env.addUser(t, "alice")

		objects, err := NewMinioObjectStore(ObjectStoreConfig{
			Endpoint:  "127.0.0.1:9000",
			Bucket:    "im-media",
			AccessKey: "access",
			SecretKey: "secret",
			Region:    "us-east-1",
			PathStyle: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		handler := NewRouter(
			env.db,
			env.data,
			env.stream,
			env.members,
			objects,
			"https://cdn.example.test",
			"test-node",
		)

		recorder := doRequest(t, handler, http.MethodPost, "/api/v1/files/presign", token,
			jsonBody(t, PresignUploadReq{FileName: "photo.PNG", ContentType: "image/png"}))
		requireStatus(t, recorder, http.StatusOK)
		response := decodeResponse[PresignUploadResp](t, recorder)
		if !strings.HasPrefix(response.Data.FileKey, "uploads/") ||
			!strings.HasSuffix(response.Data.FileKey, ".png") {
			t.Fatalf("file key = %q", response.Data.FileKey)
		}
		if !strings.Contains(response.Data.UploadURL, "im-media") {
			t.Fatalf("upload url = %q", response.Data.UploadURL)
		}
		wantDownload := "https://cdn.example.test/" + response.Data.FileKey
		if response.Data.DownloadURL != wantDownload {
			t.Fatalf("download url = %q", response.Data.DownloadURL)
		}
	})
}

// benchMessage seeds one message and returns it.
func (e *testEnv) benchMessage(tb testing.TB, roomID, senderID uuid.UUID) engine.Message {
	tb.Helper()
	message, err := e.data.WriteMessage(context.Background(), engine.Message{
		ClientMsgID: uuid.NewV7(),
		SenderID:    senderID,
		RoomID:      roomID,
		MsgType:     engine.MsgTypeText,
		Payload:     []byte(`{"text":"bench"}`),
		Ext:         []byte(`{}`),
	})
	if err != nil {
		tb.Fatal(err)
	}
	return message
}

// BenchmarkAPIHistory measures the highest-QPS message-plane endpoint end to
// end: JWT auth, membership authorization, Pebble read, JSON render.
func BenchmarkAPIHistory(b *testing.B) {
	env := newTestEnv(b)
	alice, _ := env.addUser(b, "bench-alice")
	bob, bobToken := env.addUser(b, "bench-bob")
	room := env.addRoom(b, alice, bob)
	for range 50 {
		env.benchMessage(b, room.RoomID, alice.UserID)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		recorder := doRequest(b, env.handler, http.MethodGet,
			"/api/v1/messages/history?room_id="+room.RoomID.String()+"&page_size=20", bobToken, nil)
		if recorder.Code != http.StatusOK {
			b.Fatalf("history: %d %s", recorder.Code, recorder.Body.String())
		}
	}
}

// BenchmarkAPISendBatch measures the WebSocket send hot path: per-message
// admission checks plus one batched Pebble commit per gateway batch.
func BenchmarkAPISendBatch(b *testing.B) {
	env := newTestEnv(b)
	sender, _ := env.addUser(b, "bench-sender")
	room := env.createRoom(b, roomIDForNode(b, 0), sender)
	owner := NewStreamHandler(env.data, env.members, 1024, 0, 1)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		messages := make([]*pb.SendMessage, 32)
		for i := range messages {
			messages[i] = &pb.SendMessage{
				ClientMsgId: uuid.NewV7().String(),
				SenderId:    sender.UserID.String(),
				RoomId:      room.RoomID.String(),
				MsgType:     int32(engine.MsgTypeText),
				Payload:     []byte(`{"text":"bench"}`),
			}
		}
		result, written, err := owner.writeBatch(ctx, &pb.SendBatch{Messages: messages})
		if err != nil {
			b.Fatal(err)
		}
		if len(written) != 32 {
			b.Fatalf("written = %d, want 32; results = %+v", len(written), result.GetResults())
		}
	}
}

// BenchmarkAPIAddReaction measures a room-wide-fanout write endpoint: auth,
// membership, reaction commit, and the reaction broadcast.
func BenchmarkAPIAddReaction(b *testing.B) {
	env := newTestEnv(b)
	alice, _ := env.addUser(b, "bench-alice")
	bob, bobToken := env.addUser(b, "bench-bob")
	room := env.addRoom(b, alice, bob)
	message := env.benchMessage(b, room.RoomID, alice.UserID)
	body := jsonBody(b, ReactionReq{Emoji: "👍"})

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		recorder := doRequest(
			b,
			env.handler,
			http.MethodPost,
			"/api/v1/messages/"+strconv.FormatUint(
				message.RoomSeq,
				10,
			)+"/reactions?room_id="+room.RoomID.String(),
			bobToken,
			body,
		)
		if recorder.Code != http.StatusOK {
			b.Fatalf("add reaction: %d %s", recorder.Code, recorder.Body.String())
		}
	}
}
