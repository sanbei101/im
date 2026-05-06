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
	UserID1 string `json:"user_id_1"`
	UserID2 string `json:"user_id_2"`
}

type RoomResp struct {
	RoomID string `json:"room_id"`
}

func (s *RoomService) CreateOrGetSingleChatRoom(ctx context.Context, req CreateRoomReq) (*RoomResp, error) {
	user1, err := uuid.Parse(req.UserID1)
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

func computeSingleChatHash(user1, user2 uuid.UUID) []byte {
	if user1.String() > user2.String() {
		user1, user2 = user2, user1
	}
	combined := make([]byte, 32)
	copy(combined[:16], user1[:])
	copy(combined[16:], user2[:])
	return combined
}
