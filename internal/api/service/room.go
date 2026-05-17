package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/sanbei101/im/internal/db"
)

type RoomService struct {
	q *db.Queries
}

func NewRoomService(q *db.Queries) *RoomService {
	return &RoomService{q: q}
}

type CreateRoomReq struct {
	UserID2 string `json:"user_id_2" validate:"required,uuid"`
}

type CreateGroupRoomReq struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids" validate:"required,min=2"`
}

type BatchCreateRoomsReq struct {
	SingleRooms []BatchSingleRoomReq `json:"single_rooms"`
	GroupRooms  []BatchGroupRoomReq  `json:"group_rooms"`
}

type BatchSingleRoomReq struct {
	UserID1 string `json:"user_id_1" validate:"required,uuid"`
	UserID2 string `json:"user_id_2" validate:"required,uuid"`
}

type BatchGroupRoomReq struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids" validate:"required,min=2"`
}

type BatchCreateRoomsResp struct {
	SingleRooms []BatchRoomResult `json:"single_rooms"`
	GroupRooms  []BatchRoomResult `json:"group_rooms"`
}

type BatchRoomResult struct {
	RoomID string `json:"room_id"`
}

type RoomResp struct {
	RoomID string `json:"room_id"`
}

type ListRoomsResp struct {
	Rooms []RoomInfo `json:"rooms"`
}

type RoomInfo struct {
	RoomID    string `json:"room_id"`
	ChatType  string `json:"chat_type"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func (s *RoomService) ListRooms(ctx context.Context, userID string) (*ListRoomsResp, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	rooms, err := s.q.GetUserRooms(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	if len(rooms) == 0 {
		return nil, nil
	}

	result := make([]RoomInfo, len(rooms))
	for i, r := range rooms {
		result[i] = RoomInfo{
			RoomID:    r.RoomID.String(),
			ChatType:  string(r.ChatType),
			Name:      r.Name,
			AvatarURL: r.AvatarUrl,
		}
	}

	return &ListRoomsResp{Rooms: result}, nil
}

func (s *RoomService) CreateOrGetSingleChatRoom(ctx context.Context, userID1 string, req CreateRoomReq) (*RoomResp, error) {
	user1, err := uuid.Parse(userID1)
	if err != nil {
		return nil, err
	}
	user2, err := uuid.Parse(req.UserID2)
	if err != nil {
		return nil, err
	}

	if user1 == user2 {
		return nil, fmt.Errorf("cannot create chat room with same user")
	}

	hash := computeSingleChatHash(user1, user2)

	room, err := s.q.GetRoomByHash(ctx, hash)
	if err == nil && room != nil {
		return &RoomResp{RoomID: room.RoomID.String()}, nil
	}

	roomUUID := uuid.Must(uuid.NewV7())
	roomName, roomAvatar := s.generateRoomInfo(roomUUID)
	_, err = s.q.CreateRoom(ctx, db.CreateRoomParams{
		RoomID:         roomUUID,
		ChatType:       db.ChatTypeSingle,
		Name:           roomName,
		AvatarUrl:      roomAvatar,
		SingleChatHash: hash,
	})
	if err != nil {
		return nil, err
	}

	err = s.q.AddRoomMember(ctx, db.AddRoomMemberParams{
		RoomID: roomUUID,
		UserID: user1,
		Role:   db.MemberRoleMember,
	})
	if err != nil {
		return nil, err
	}

	err = s.q.AddRoomMember(ctx, db.AddRoomMemberParams{
		RoomID: roomUUID,
		UserID: user2,
		Role:   db.MemberRoleMember,
	})
	if err != nil {
		return nil, err
	}

	return &RoomResp{RoomID: roomUUID.String()}, nil
}

func (s *RoomService) CreateGroupRoom(ctx context.Context, req CreateGroupRoomReq) (*RoomResp, error) {
	if len(req.MemberIDs) < 2 {
		return nil, fmt.Errorf("group room requires at least 2 members")
	}

	memberUUIDs := make([]uuid.UUID, 0, len(req.MemberIDs))
	for _, id := range req.MemberIDs {
		u, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		memberUUIDs = append(memberUUIDs, u)
	}

	roomUUID := uuid.Must(uuid.NewV7())
	roomName, roomUrl := s.generateRoomInfo(roomUUID)
	if req.Name != "" {
		roomName = req.Name
	}

	_, err := s.q.CreateGroupRoom(ctx, db.CreateGroupRoomParams{
		RoomID:    roomUUID,
		Name:      roomName,
		AvatarUrl: roomUrl,
	})
	if err != nil {
		return nil, err
	}

	err = s.q.AddRoomMembers(ctx, db.AddRoomMembersParams{
		RoomID:  roomUUID,
		UserIds: memberUUIDs,
	})
	if err != nil {
		return nil, err
	}

	return &RoomResp{RoomID: roomUUID.String()}, nil
}

var (
	adjectives = []string{"快乐的", "神秘的", "热情的", "冷静的", "勇敢的", "温柔的", "酷炫的", "安静的"}
	nouns      = []string{"会议室", "小屋", "角落", "广场", "花园", "沙龙", "茶馆", "驿站"}
)

func (s *RoomService) generateRoomInfo(roomID uuid.UUID) (name string, avatarURL string) {
	seed := int64(binary.BigEndian.Uint64(roomID[:8]))
	rng := rand.New(rand.NewSource(seed))

	adj := adjectives[rng.Intn(len(adjectives))]
	noun := nouns[rng.Intn(len(nouns))]
	name = adj + noun

	avatarURL = fmt.Sprintf("https://api.dicebear.com/7.x/identicon/svg?seed=%s", roomID.String())
	return name, avatarURL
}

func (s *RoomService) BatchCreateRooms(ctx context.Context, req BatchCreateRoomsReq) (*BatchCreateRoomsResp, error) {
	singleResults := make([]BatchRoomResult, 0, len(req.SingleRooms))
	for _, sr := range req.SingleRooms {
		resp, err := s.CreateOrGetSingleChatRoom(ctx, sr.UserID1, CreateRoomReq{UserID2: sr.UserID2})
		if err != nil {
			return nil, fmt.Errorf("failed to create single room for %s-%s: %w", sr.UserID1, sr.UserID2, err)
		}
		singleResults = append(singleResults, BatchRoomResult{RoomID: resp.RoomID})
	}

	groupResults := make([]BatchRoomResult, 0, len(req.GroupRooms))
	for _, gr := range req.GroupRooms {
		resp, err := s.CreateGroupRoom(ctx, CreateGroupRoomReq{Name: gr.Name, MemberIDs: gr.MemberIDs})
		if err != nil {
			return nil, fmt.Errorf("failed to create group room %s: %w", gr.Name, err)
		}
		groupResults = append(groupResults, BatchRoomResult{RoomID: resp.RoomID})
	}

	return &BatchCreateRoomsResp{
		SingleRooms: singleResults,
		GroupRooms:  groupResults,
	}, nil
}

func computeSingleChatHash(user1, user2 uuid.UUID) []byte {
	if user1.String() > user2.String() {
		user1, user2 = user2, user1
	}
	combined := make([]byte, 32)
	copy(combined[:16], user1[:])
	copy(combined[16:], user2[:])
	return combined
}
