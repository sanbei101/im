package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

type RoomAPI struct {
	store         *store.Store
	streamHandler *StreamHandler
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
	Notice    string `json:"notice"`
}

type RoomDetailResp struct {
	RoomID    string `json:"room_id"`
	ChatType  string `json:"chat_type"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Notice    string `json:"notice"`
	MyRole    string `json:"my_role"`
}

type UpdateRoomReq struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Notice    string `json:"notice"`
}

type AddMembersReq struct {
	MemberIDs []string `json:"member_ids" validate:"required"`
}

type TransferOwnerReq struct {
	NewOwnerID string `json:"new_owner_id" validate:"required,uuid"`
}

type MemberInfoResp struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
	IsPinned  bool   `json:"is_pinned"`
	IsMuted   bool   `json:"is_muted"`
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
		store.Room{RoomID: roomID, ChatType: store.ChatTypeSingle, Name: name, AvatarURL: avatar, SingleChatHash: hash},
		[]store.Member{{UserID: user1, Role: store.RoleMember}, {UserID: user2, Role: store.RoleMember}},
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
	creatorIDStr := jwt.GetUserIDFromContext(r)
	if creatorIDStr == "" {
		render.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	creatorID, err := uuid.Parse(creatorIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	req, err := render.ReadBody[CreateGroupRoomReq](w, r)
	if err != nil {
		return
	}

	members := make([]store.Member, 0, len(req.MemberIDs)+1)
	seen := make(map[uuid.UUID]struct{}, len(req.MemberIDs)+1)

	members = append(members, store.Member{UserID: creatorID, Role: store.RoleOwner})
	seen[creatorID] = struct{}{}

	for _, raw := range req.MemberIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			render.Error(w, http.StatusBadRequest, "invalid member id: "+raw)
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		members = append(members, store.Member{UserID: id, Role: store.RoleMember})
	}

	if len(members) < 2 {
		render.Error(w, http.StatusBadRequest, "group room requires at least 2 members")
		return
	}

	roomID := uuid.NewV7()
	name, avatar := generateRoomInfo(roomID)
	if req.Name != "" {
		name = req.Name
	}
	if err := a.store.CreateRoom(
		r.Context(),
		store.Room{RoomID: roomID, ChatType: store.ChatTypeGroup, Name: name, AvatarURL: avatar},
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

func (a *RoomAPI) GetRoom(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	room, err := a.store.Room(r.Context(), roomID)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusNotFound, "room not found")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	member, err := a.store.Member(r.Context(), roomID, myID)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取房间信息成功", RoomDetailResp{
		RoomID:    room.RoomID.String(),
		ChatType:  room.ChatType,
		Name:      room.Name,
		AvatarURL: room.AvatarURL,
		Notice:    room.Notice,
		MyRole:    member.Role,
	})
}

func (a *RoomAPI) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	member, err := a.store.Member(r.Context(), roomID, myID)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if member.Role != store.RoleOwner && member.Role != store.RoleAdmin {
		render.Error(w, http.StatusForbidden, "only owner or admin can update room")
		return
	}

	req, err := render.ReadBody[UpdateRoomReq](w, r)
	if err != nil {
		return
	}

	updated, err := a.store.UpdateRoom(r.Context(), roomID, req.Name, req.AvatarURL, req.Notice)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "更新房间信息成功", RoomDetailResp{
		RoomID:    updated.RoomID.String(),
		ChatType:  updated.ChatType,
		Name:      updated.Name,
		AvatarURL: updated.AvatarURL,
		Notice:    updated.Notice,
		MyRole:    member.Role,
	})
}

func (a *RoomAPI) ListMembers(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	if _, err := a.store.Member(r.Context(), roomID, myID); err != nil {
		render.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}

	members, err := a.store.Members(r.Context(), roomID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]MemberInfoResp, 0, len(members))
	for _, m := range members {
		u, err := a.store.UserByID(r.Context(), m.UserID)
		if err != nil {
			continue
		}
		result = append(result, MemberInfoResp{
			UserID:    u.UserID.String(),
			Username:  u.Username,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
			Role:      m.Role,
			IsPinned:  m.IsPinned,
			IsMuted:   m.IsMuted,
		})
	}

	render.Success(w, "获取群成员列表成功", result)
}

func (a *RoomAPI) AddMembers(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	if _, err := a.store.Member(r.Context(), roomID, myID); err != nil {
		render.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}

	req, err := render.ReadBody[AddMembersReq](w, r)
	if err != nil {
		return
	}

	toAdd := make([]store.Member, 0, len(req.MemberIDs))
	for _, raw := range req.MemberIDs {
		uid, err := uuid.Parse(raw)
		if err != nil {
			render.Error(w, http.StatusBadRequest, "invalid member id: "+raw)
			return
		}
		if _, err := a.store.Member(r.Context(), roomID, uid); err == nil {
			continue
		}
		toAdd = append(toAdd, store.Member{UserID: uid, Role: store.RoleMember})
	}

	if len(toAdd) > 0 {
		if err := a.store.AddMembers(r.Context(), roomID, toAdd); err != nil {
			render.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if a.streamHandler != nil {
			a.streamHandler.InvalidateRoomMembers(roomID)
		}
	}

	render.SuccessNoData(w, http.StatusOK, "添加群成员成功")
}

func (a *RoomAPI) RemoveMember(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid target user id")
		return
	}

	myMember, err := a.store.Member(r.Context(), roomID, myID)
	if err != nil {
		render.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}
	targetMember, err := a.store.Member(r.Context(), roomID, targetID)
	if err != nil {
		render.Error(w, http.StatusNotFound, "target not in room")
		return
	}

	if targetMember.Role == store.RoleOwner {
		render.Error(w, http.StatusForbidden, "cannot remove group owner")
		return
	}

	if myID != targetID && myMember.Role != store.RoleOwner &&
		(myMember.Role != store.RoleAdmin || targetMember.Role != store.RoleMember) {
		render.Error(w, http.StatusForbidden, "no permission to remove this member")
		return
	}

	if err := a.store.RemoveMember(r.Context(), roomID, targetID); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a.streamHandler != nil {
		a.streamHandler.InvalidateRoomMembers(roomID)
	}

	render.SuccessNoData(w, http.StatusOK, "移除群成员成功")
}

type UpdateMemberRoleReq struct {
	Role string `json:"role" validate:"required"`
}

func (a *RoomAPI) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid target user id")
		return
	}

	myMember, err := a.store.Member(r.Context(), roomID, myID)
	if err != nil {
		render.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}
	if myMember.Role != store.RoleOwner {
		render.Error(w, http.StatusForbidden, "only group owner can manage member roles")
		return
	}

	targetMember, err := a.store.Member(r.Context(), roomID, targetID)
	if err != nil {
		render.Error(w, http.StatusNotFound, "target not in room")
		return
	}

	if targetMember.Role == store.RoleOwner {
		render.Error(w, http.StatusBadRequest, "cannot modify owner role")
		return
	}

	req, err := render.ReadBody[UpdateMemberRoleReq](w, r)
	if err != nil {
		return
	}

	if req.Role != store.RoleAdmin && req.Role != store.RoleMember {
		render.Error(w, http.StatusBadRequest, "invalid role: must be 'admin' or 'member'")
		return
	}

	if err := a.store.UpdateMemberRole(r.Context(), roomID, targetID, req.Role); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if a.streamHandler != nil {
		a.streamHandler.InvalidateRoomMembers(roomID)
	}

	render.SuccessNoData(w, http.StatusOK, "更新成员角色成功")
}

func (a *RoomAPI) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	member, err := a.store.Member(r.Context(), roomID, myID)
	if err != nil {
		render.Error(w, http.StatusNotFound, "not a member of this room")
		return
	}

	if member.Role == store.RoleOwner {
		render.Error(w, http.StatusBadRequest, "群主无法直接退群，请先转让群主或解散群聊")
		return
	}

	if err := a.store.RemoveMember(r.Context(), roomID, myID); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a.streamHandler != nil {
		a.streamHandler.InvalidateRoomMembers(roomID)
	}

	render.SuccessNoData(w, http.StatusOK, "退出群聊成功")
}

func (a *RoomAPI) TransferOwner(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	myMember, err := a.store.Member(r.Context(), roomID, myID)
	if err != nil || myMember.Role != store.RoleOwner {
		render.Error(w, http.StatusForbidden, "only owner can transfer ownership")
		return
	}

	req, err := render.ReadBody[TransferOwnerReq](w, r)
	if err != nil {
		return
	}
	newOwnerID, err := uuid.Parse(req.NewOwnerID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid new_owner_id")
		return
	}
	if newOwnerID == myID {
		render.Error(w, http.StatusBadRequest, "already the owner")
		return
	}

	if _, err := a.store.Member(r.Context(), roomID, newOwnerID); err != nil {
		render.Error(w, http.StatusNotFound, "new owner is not a member of this room")
		return
	}

	if err := a.store.UpdateMemberRole(r.Context(), roomID, newOwnerID, store.RoleOwner); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.store.UpdateMemberRole(r.Context(), roomID, myID, store.RoleMember); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "转让群主成功")
}

func (a *RoomAPI) DissolveRoom(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	member, err := a.store.Member(r.Context(), roomID, myID)
	if err != nil || member.Role != store.RoleOwner {
		render.Error(w, http.StatusForbidden, "only owner can dissolve room")
		return
	}

	if err := a.store.DissolveRoom(r.Context(), roomID); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a.streamHandler != nil {
		a.streamHandler.InvalidateRoomMembers(roomID)
	}

	render.SuccessNoData(w, http.StatusOK, "解散群聊成功")
}

type PinReq struct {
	MsgID string `json:"msg_id" validate:"required,uuid"`
}

func (a *RoomAPI) PinMessage(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}
	req, err := render.ReadBody[PinReq](w, r)
	if err != nil {
		return
	}
	msgID, err := uuid.Parse(req.MsgID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid msg_id")
		return
	}

	member, err := a.store.Member(r.Context(), roomID, myID)
	if err != nil || (member.Role != store.RoleOwner && member.Role != store.RoleAdmin) {
		render.Error(w, http.StatusForbidden, "只有群主或管理员可以置顶消息")
		return
	}

	if err := a.store.PinMessage(r.Context(), roomID, msgID, myID); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.SuccessNoData(w, http.StatusOK, "置顶消息成功")
}

func (a *RoomAPI) UnpinMessage(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msg_id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid msg_id")
		return
	}

	member, err := a.store.Member(r.Context(), roomID, myID)
	if err != nil || (member.Role != store.RoleOwner && member.Role != store.RoleAdmin) {
		render.Error(w, http.StatusForbidden, "只有群主或管理员可以取消置顶消息")
		return
	}

	if err := a.store.UnpinMessage(r.Context(), roomID, msgID); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.SuccessNoData(w, http.StatusOK, "取消置顶成功")
}

func (a *RoomAPI) GetPinnedMessages(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	if _, err := a.store.Member(r.Context(), roomID, myID); err != nil {
		render.Error(w, http.StatusForbidden, "not a room member")
		return
	}

	pins, err := a.store.PinnedMessages(r.Context(), roomID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(w, "获取置顶消息成功", pins)
}
