package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
	"github.com/sanbei101/im/proto/pb"
)

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
		if _, err := env.store.DeviceToken(context.Background(), alice.UserID); !errors.Is(err, store.ErrNotFound) {
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

		message, err := env.store.WriteMessage(context.Background(), store.Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    alice.UserID,
			RoomID:      roomID,
			MsgType:     store.MsgTypeText,
			Payload:     []byte(`{"text":"hello"}`),
			Ext:         []byte(`{}`),
		})
		if err != nil {
			t.Fatal(err)
		}

		recorder = doRequest(t, env.handler, http.MethodGet,
			"/api/v1/messages/history?room_id="+roomID.String()+"&page_size=10", bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		page := decodeResponse[store.MessagePage](t, recorder)
		if len(page.Data.Messages) != 1 || page.Data.Messages[0].MsgID != message.MsgID {
			t.Fatalf("history = %+v", page.Data.Messages)
		}

		reactionPath := "/api/v1/messages/" + message.MsgID.String() + "/reactions"
		recorder = doRequest(t, env.handler, http.MethodPost, reactionPath, bobToken,
			jsonBody(t, ReactionReq{RoomID: roomID.String(), Emoji: "👍"}))
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodGet,
			reactionPath+"?room_id="+roomID.String(), bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		groups := decodeResponse[[]store.ReactionGroup](t, recorder)
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

		if _, err := env.store.WriteMessage(context.Background(), store.Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    alice.UserID,
			RoomID:      room.RoomID,
			MsgType:     store.MsgTypeText,
			Payload:     []byte(`{"text":"hello"}`),
			Ext:         []byte(`{}`),
		}); err != nil {
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

		recorder = doRequest(t, env.handler, http.MethodGet, "/api/v1/conversations", token, nil)
		requireStatus(t, recorder, http.StatusOK)
		conversations = decodeResponse[ConversationListResp](t, recorder)
		if conversations.Data.Conversations[0].UnreadCount != 0 {
			t.Fatalf("unread count = %d", conversations.Data.Conversations[0].UnreadCount)
		}
	})

	t.Run("moderation", func(t *testing.T) {
		env := newTestEnv(t)
		alice, aliceToken := env.addUser(t, "alice")
		bob, bobToken := env.addUser(t, "bob")
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
		messagePath := "/api/v1/messages/" + message.MsgID.String()

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
		found := decodeResponse[[]store.Message](t, recorder)
		if len(found.Data) != 1 || found.Data[0].MsgID != message.MsgID {
			t.Fatalf("search results = %+v", found.Data)
		}

		reactionPath := messagePath + "/reactions"
		recorder = doRequest(t, env.handler, http.MethodPost, reactionPath, bobToken,
			jsonBody(t, ReactionReq{RoomID: room.RoomID.String(), Emoji: "👍"}))
		requireStatus(t, recorder, http.StatusOK)
		recorder = doRequest(t, env.handler, http.MethodDelete,
			reactionPath+"?room_id="+room.RoomID.String()+"&emoji=%F0%9F%91%8D", bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		recorder = doRequest(t, env.handler, http.MethodGet,
			reactionPath+"?room_id="+room.RoomID.String(), bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		groups := decodeResponse[[]store.ReactionGroup](t, recorder)
		if len(groups.Data) != 0 {
			t.Fatalf("reaction groups = %+v", groups.Data)
		}

		recorder = doRequest(t, env.handler, http.MethodPost,
			"/api/v1/rooms/"+room.RoomID.String()+"/pins", aliceToken,
			jsonBody(t, PinReq{MsgID: message.MsgID.String()}))
		requireStatus(t, recorder, http.StatusOK)
		recorder = doRequest(t, env.handler, http.MethodGet,
			"/api/v1/rooms/"+room.RoomID.String()+"/pins", bobToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		pins := decodeResponse[[]store.Message](t, recorder)
		if len(pins.Data) != 1 || pins.Data[0].MsgID != message.MsgID {
			t.Fatalf("pins = %+v", pins.Data)
		}
		recorder = doRequest(t, env.handler, http.MethodDelete,
			"/api/v1/rooms/"+room.RoomID.String()+"/pins/"+message.MsgID.String(), aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)

		recorder = doRequest(t, env.handler, http.MethodPost, "/api/v1/messages/recall", aliceToken,
			jsonBody(t, RecallReq{RoomID: room.RoomID.String(), MsgID: message.MsgID.String()}))
		requireStatus(t, recorder, http.StatusOK)
		recalled := decodeResponse[store.Message](t, recorder)
		if recalled.Data.MsgType != store.MsgTypeRecall {
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
		if _, err := env.store.Member(context.Background(), room.RoomID, carol.UserID); err != nil {
			t.Fatalf("load added member: %v", err)
		}

		recorder = doRequest(t, env.handler, http.MethodPut,
			roomPath+"/members/"+carol.UserID.String()+"/role", aliceToken,
			jsonBody(t, UpdateMemberRoleReq{Role: store.RoleAdmin}))
		requireStatus(t, recorder, http.StatusOK)
		member, err := env.store.Member(context.Background(), room.RoomID, carol.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if member.Role != store.RoleAdmin {
			t.Fatalf("role = %q", member.Role)
		}

		recorder = doRequest(t, env.handler, http.MethodDelete,
			roomPath+"/members/"+carol.UserID.String(), aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		if _, err := env.store.Member(
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
		owner, err := env.store.Member(context.Background(), room.RoomID, bob.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if owner.Role != store.RoleOwner {
			t.Fatalf("new owner role = %q", owner.Role)
		}

		recorder = doRequest(t, env.handler, http.MethodPost, roomPath+"/leave", aliceToken, nil)
		requireStatus(t, recorder, http.StatusOK)
		if _, err := env.store.Member(
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
		if _, err := env.store.Room(context.Background(), room.RoomID); !errors.Is(err, store.ErrNotFound) {
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

		if err := env.store.ApplyFriend(context.Background(), alice.UserID, bob.UserID, "hi"); err != nil {
			t.Fatal(err)
		}
		if err := env.store.AuditFriend(context.Background(), bob.UserID, alice.UserID, true); err != nil {
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
	env := newTestEnv(t)
	_, token := env.addUser(t, "alice")

	objects, err := store.NewMinioObjectStore(store.ObjectStoreConfig{
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
	handler := NewRouter(env.store, env.stream, objects, "https://cdn.example.test")

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
		t.Fatalf("download url = %q, want %q", response.Data.DownloadURL, wantDownload)
	}
}
