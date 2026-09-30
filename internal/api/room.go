package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"uuid"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

type RoomAPI struct {
	store *store.Store
}

type CreateSingleRoomReq struct {
	UserID2 string `json:"user_id_2" validate:"required,uuid"`
}

type CreateGroupRoomReq struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids" validate:"required,min=2"`
}

type RoomResp struct {
	RoomID string `json:"room_id"`
}

type RoomInfo struct {
	RoomID    string `json:"room_id"`
	ChatType  string `json:"chat_type"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func (a *RoomAPI) CreateOrGetSingleChatRoom(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[CreateSingleRoomReq](w, r)
	if err != nil {
		return
	}
	userIDStr := jwt.GetUserIDFromContext(r)
	if userIDStr == "" {
		render.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	user1, err := uuid.Parse(userIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	user2, err := uuid.Parse(req.UserID2)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user_id_2")
		return
	}
	if user1 == user2 {
		render.Error(w, http.StatusBadRequest, "cannot create chat room with same user")
		return
	}

	hash := singleHash(user1, user2)
	if room, err := a.store.RoomBySingleHash(r.Context(), hash); err == nil {
		render.Success(w, "获取或创建单聊房间成功", RoomResp{RoomID: room.RoomID.String()})
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	roomID := uuid.NewV7()
	name, avatar := generateRoomInfo(roomID)
	err = a.store.CreateRoom(
		r.Context(),
		store.Room{RoomID: roomID, ChatType: "single", Name: name, AvatarURL: avatar, SingleChatHash: hash},
		[]store.Member{{UserID: user1, Role: "member"}, {UserID: user2, Role: "member"}},
	)
	if errors.Is(err, store.ErrAlreadyExists) {
		room, lookupErr := a.store.RoomBySingleHash(r.Context(), hash)
		if lookupErr != nil {
			render.Error(w, http.StatusInternalServerError, lookupErr.Error())
			return
		}
		render.Success(w, "获取或创建单聊房间成功", RoomResp{RoomID: room.RoomID.String()})
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取或创建单聊房间成功", RoomResp{RoomID: roomID.String()})
}

func (a *RoomAPI) CreateGroupRoom(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[CreateGroupRoomReq](w, r)
	if err != nil {
		return
	}
	if len(req.MemberIDs) < 2 {
		render.Error(w, http.StatusBadRequest, "group room requires at least 2 members")
		return
	}

	members := make([]store.Member, 0, len(req.MemberIDs))
	seen := make(map[uuid.UUID]struct{}, len(req.MemberIDs))
	for _, raw := range req.MemberIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			render.Error(w, http.StatusBadRequest, "invalid member id: "+raw)
			return
		}
		if _, ok := seen[id]; ok {
			render.Error(w, http.StatusBadRequest, "duplicate group member")
			return
		}
		seen[id] = struct{}{}
		role := "member"
		if len(members) == 0 {
			role = "owner"
		}
		members = append(members, store.Member{UserID: id, Role: role})
	}

	roomID := uuid.NewV7()
	name, avatar := generateRoomInfo(roomID)
	if req.Name != "" {
		name = req.Name
	}
	if err := a.store.CreateRoom(
		r.Context(),
		store.Room{RoomID: roomID, ChatType: "group", Name: name, AvatarURL: avatar},
		members,
	); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "创建群聊房间成功", RoomResp{RoomID: roomID.String()})
}

func (a *RoomAPI) ListRooms(w http.ResponseWriter, r *http.Request) {
	userIDStr := jwt.GetUserIDFromContext(r)
	id, err := uuid.Parse(userIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	rooms, err := a.store.RoomsByUser(r.Context(), id)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]RoomInfo, len(rooms))
	for i := range rooms {
		result[i] = RoomInfo{
			RoomID:    rooms[i].Room.RoomID.String(),
			ChatType:  rooms[i].Room.ChatType,
			Name:      rooms[i].Room.Name,
			AvatarURL: rooms[i].Room.AvatarURL,
		}
	}
	render.Success(w, "获取房间列表成功", map[string]any{"rooms": result})
}

func singleHash(a, b uuid.UUID) []byte {
	if a.Compare(b) > 0 {
		a, b = b, a
	}
	hash := sha256.New()
	hash.Write(a[:])
	hash.Write(b[:])
	return hash.Sum(nil)
}

var (
	adjectives = []string{"快乐的", "神秘的", "热情的", "冷静的", "勇敢的", "温柔的", "酷炫的", "安静的"}
	nouns      = []string{"会议室", "小屋", "角落", "广场", "花园", "沙龙", "茶馆", "驿站"}
)

func generateRoomInfo(roomID uuid.UUID) (string, string) {
	return adjectives[rand.IntN(len(adjectives))] + nouns[rand.IntN(len(nouns))], "room://" + fmt.Sprint(roomID)
}
