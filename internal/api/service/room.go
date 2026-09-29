package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand/v2"
	"uuid"

	"github.com/sanbei101/im/internal/store"
)

type RoomService struct{ store *store.Store }

type CreateRoomReq struct {
	UserID2 string `json:"user_id_2" validate:"required,uuid"`
}
type CreateGroupRoomReq struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids" validate:"required,min=2"`
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

func NewRoomService(s *store.Store) *RoomService { return &RoomService{store: s} }

func (s *RoomService) ListRooms(ctx context.Context, userID string) (*ListRoomsResp, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rooms, err := s.store.RoomsByUser(ctx, id)
	if err != nil {
		return nil, err
	}
	result := make([]RoomInfo, 0, len(rooms))
	for _, item := range rooms {
		result = append(
			result,
			RoomInfo{
				RoomID:    item.Room.RoomID.String(),
				ChatType:  item.Room.ChatType,
				Name:      item.Room.Name,
				AvatarURL: item.Room.AvatarURL,
			},
		)
	}
	return &ListRoomsResp{Rooms: result}, nil
}

func (s *RoomService) CreateOrGetSingleChatRoom(
	ctx context.Context,
	userID string,
	req CreateRoomReq,
) (*RoomResp, error) {
	user1, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	user2, err := uuid.Parse(req.UserID2)
	if err != nil {
		return nil, err
	}
	if user1 == user2 {
		return nil, errors.New("cannot create chat room with same user")
	}
	hash := singleHash(user1, user2)
	if room, err := s.store.RoomBySingleHash(ctx, hash); err == nil {
		return &RoomResp{RoomID: room.RoomID.String()}, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	roomID := uuid.NewV7()
	name, avatar := generateRoomInfo(roomID)
	err = s.store.CreateRoom(
		ctx,
		store.Room{RoomID: roomID, ChatType: "single", Name: name, AvatarURL: avatar, SingleChatHash: hash},
		[]store.Member{{UserID: user1, Role: "member"}, {UserID: user2, Role: "member"}},
	)
	if errors.Is(err, store.ErrAlreadyExists) {
		room, lookupErr := s.store.RoomBySingleHash(ctx, hash)
		if lookupErr != nil {
			return nil, lookupErr
		}
		return &RoomResp{RoomID: room.RoomID.String()}, nil
	}
	if err != nil {
		return nil, err
	}
	return &RoomResp{RoomID: roomID.String()}, nil
}

func (s *RoomService) CreateGroupRoom(ctx context.Context, req CreateGroupRoomReq) (*RoomResp, error) {
	if len(req.MemberIDs) < 2 {
		return nil, errors.New("group room requires at least 2 members")
	}
	members := make([]store.Member, 0, len(req.MemberIDs))
	seen := make(map[uuid.UUID]struct{}, len(req.MemberIDs))
	for _, raw := range req.MemberIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; ok {
			return nil, errors.New("duplicate group member")
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
	if err := s.store.CreateRoom(
		ctx,
		store.Room{RoomID: roomID, ChatType: "group", Name: name, AvatarURL: avatar},
		members,
	); err != nil {
		return nil, err
	}
	return &RoomResp{RoomID: roomID.String()}, nil
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
