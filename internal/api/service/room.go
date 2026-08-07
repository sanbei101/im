package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/cache"
	"github.com/sanbei101/im/internal/db"
)

type RoomService struct {
	query *db.Queries
	db    *pgxpool.Pool
	cache *cache.RoomStore
}

func NewRoomService(query *db.Queries, dbPool *pgxpool.Pool, roomCache *cache.RoomStore) *RoomService {
	return &RoomService{query: query, db: dbPool, cache: roomCache}
}

type CreateRoomReq struct {
	UserID2 string `json:"user_id_2" validate:"required,uuid"`
}

type CreateGroupRoomReq struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids" validate:"required,min=1"`
}

type RoomResp struct {
	RoomID string `json:"room_id"`
}

type ListRoomsResp struct {
	Rooms []RoomInfo `json:"rooms"`
}

type RoomInfo struct {
	RoomID                string `json:"room_id"`
	ChatType              string `json:"chat_type"`
	Name                  string `json:"name"`
	AvatarURL             string `json:"avatar_url"`
	IsHidden              bool   `json:"is_hidden"`
	IsMuted               bool   `json:"is_muted"`
	LastMessageServerTime int64  `json:"last_message_server_time"`
	UnreadCount           int64  `json:"unread_count"`
}

var ErrUsersNotFriends = errors.New("users are not friends")

func (s *RoomService) ListRooms(ctx context.Context, userID string) (*ListRoomsResp, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	rooms, err := s.query.GetUserRooms(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	result := make([]RoomInfo, len(rooms))
	for i, r := range rooms {
		result[i] = RoomInfo{
			RoomID:                r.RoomID.String(),
			ChatType:              string(r.ChatType),
			Name:                  r.Name,
			AvatarURL:             r.AvatarUrl,
			IsHidden:              r.IsHidden,
			IsMuted:               r.IsMuted,
			LastMessageServerTime: r.LastMessageServerTime,
			UnreadCount:           r.UnreadCount,
		}
	}
	if s.cache != nil {
		refs := make([]cache.RoomRef, 0, len(rooms))
		roomIDs := make([]uuid.UUID, 0, len(rooms))
		for _, room := range rooms {
			score := room.LastMessageServerTime
			if score == 0 {
				score = room.CreatedAt.UnixMicro()
			}
			refs = append(refs, cache.RoomRef{RoomID: room.RoomID, Score: score})
			roomIDs = append(roomIDs, room.RoomID)
		}
		if err := s.cache.CacheUserRooms(ctx, userUUID, refs); err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("cache user rooms failed")
		}
		s.cacheRoomMembers(ctx, roomIDs)
	}

	return &ListRoomsResp{Rooms: result}, nil
}

func (s *RoomService) CreateOrGetSingleChatRoom(
	ctx context.Context,
	userID1 string,
	req CreateRoomReq,
) (*RoomResp, error) {
	user1, err := uuid.Parse(userID1)
	if err != nil {
		return nil, err
	}
	user2, err := uuid.Parse(req.UserID2)
	if err != nil {
		return nil, err
	}

	if user1 == user2 {
		return nil, ErrCannotRelateSelf
	}
	if _, err := s.query.GetUserByID(ctx, user2); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get single-chat user: %w", err)
	}
	blocked, err := s.query.IsBlockedBetween(ctx, db.IsBlockedBetweenParams{UserA: user1, UserB: user2})
	if err != nil {
		return nil, fmt.Errorf("check single-chat blocks: %w", err)
	}
	if blocked {
		return nil, ErrUserBlocked
	}
	low, high := orderedUsers(user1, user2)
	friends, err := s.query.AreFriends(ctx, db.AreFriendsParams{UserIDLow: low, UserIDHigh: high})
	if err != nil {
		return nil, fmt.Errorf("check single-chat friendship: %w", err)
	}
	if !friends {
		return nil, ErrUsersNotFriends
	}

	hash := computeSingleChatHash(user1, user2)
	room, err := s.query.GetRoomByHash(ctx, hash)
	if err == nil {
		return &RoomResp{RoomID: room.RoomID.String()}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get single-chat room: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(ctx); err != nil {
				log.Error().Err(err).Msg("failed to rollback transaction")
			}
		}
	}()
	txQuery := s.query.WithTx(tx)
	roomUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate room id: %w", err)
	}
	roomName, roomAvatar := generateRoomInfo(roomUUID)
	_, err = txQuery.CreateRoom(ctx, db.CreateRoomParams{
		RoomID:         roomUUID,
		ChatType:       db.ChatTypeSingle,
		Name:           roomName,
		AvatarUrl:      roomAvatar,
		SingleChatHash: hash,
	})
	if err != nil {
		return nil, err
	}

	err = txQuery.AddRoomMember(ctx, db.AddRoomMemberParams{
		RoomID: roomUUID,
		UserID: user1,
		Role:   db.MemberRoleMember,
	})
	if err != nil {
		return nil, err
	}

	err = txQuery.AddRoomMember(ctx, db.AddRoomMemberParams{
		RoomID: roomUUID,
		UserID: user2,
		Role:   db.MemberRoleMember,
	})
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	committed = true
	s.invalidateRoomCache(ctx, roomUUID, []uuid.UUID{user1, user2})

	return &RoomResp{RoomID: roomUUID.String()}, nil
}

func (s *RoomService) CreateGroupRoom(ctx context.Context, ownerID string, req CreateGroupRoomReq) (*RoomResp, error) {
	owner, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, err
	}
	if len(req.MemberIDs) < 1 {
		return nil, errors.New("group room requires at least 1 member")
	}

	memberUUIDs := make([]uuid.UUID, 0, len(req.MemberIDs))
	seen := map[uuid.UUID]struct{}{owner: {}}
	for _, id := range req.MemberIDs {
		u, err := uuid.Parse(id)
		if err != nil {
			return nil, ErrInvalidInput
		}
		if _, ok := seen[u]; ok {
			continue
		}
		if _, err := s.query.GetUserByID(ctx, u); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			return nil, fmt.Errorf("get group member: %w", err)
		}
		seen[u] = struct{}{}
		memberUUIDs = append(memberUUIDs, u)
	}
	if len(memberUUIDs) == 0 {
		return nil, errors.New("group room requires another member")
	}

	roomUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate room id: %w", err)
	}
	roomName, roomURL := generateRoomInfo(roomUUID)
	if req.Name != "" {
		roomName = req.Name
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin room transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(ctx); err != nil {
				log.Error().Err(err).Str("room_id", roomUUID.String()).Msg("failed to rollback room transaction")
			}
		}
	}()

	txQuery := s.query.WithTx(tx)
	if _, err := txQuery.CreateGroupRoom(ctx, db.CreateGroupRoomParams{
		RoomID: roomUUID, Name: roomName, AvatarUrl: roomURL,
	}); err != nil {
		return nil, fmt.Errorf("create group room: %w", err)
	}
	if err := txQuery.AddRoomMember(ctx, db.AddRoomMemberParams{
		RoomID: roomUUID, UserID: owner, Role: db.MemberRoleOwner,
	}); err != nil {
		return nil, fmt.Errorf("add room owner: %w", err)
	}
	for _, member := range memberUUIDs {
		if err := txQuery.AddRoomMember(ctx, db.AddRoomMemberParams{
			RoomID: roomUUID, UserID: member, Role: db.MemberRoleMember,
		}); err != nil {
			return nil, fmt.Errorf("add room member: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit room transaction: %w", err)
	}
	committed = true
	users := append([]uuid.UUID{owner}, memberUUIDs...)
	s.invalidateRoomCache(ctx, roomUUID, users)
	return &RoomResp{RoomID: roomUUID.String()}, nil
}

var (
	adjectives = []string{"快乐的", "神秘的", "热情的", "冷静的", "勇敢的", "温柔的", "酷炫的", "安静的"}
	nouns      = []string{"会议室", "小屋", "角落", "广场", "花园", "沙龙", "茶馆", "驿站"}
)

func generateRoomInfo(roomID uuid.UUID) (name, avatarURL string) {
	adj := adjectives[rand.IntN(len(adjectives))]
	noun := nouns[rand.IntN(len(nouns))]
	name = adj + noun

	avatarURL = "https://api.dicebear.com/7.x/identicon/svg?seed=" + roomID.String()
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

var (
	ErrRoomNotFound         = errors.New("room not found")
	ErrRoomDenied           = errors.New("room permission denied")
	ErrRoomNotGroup         = errors.New("operation requires group room")
	ErrRoomOwnerRequired    = errors.New("owner permission required")
	ErrRoomMemberNotFound   = errors.New("room member not found")
	ErrRoomOwnerCannotLeave = errors.New("owner must transfer ownership before leaving")
)

type RoomDetailResp struct {
	RoomID    string            `json:"room_id"`
	ChatType  string            `json:"chat_type"`
	Name      string            `json:"name"`
	AvatarURL string            `json:"avatar_url"`
	Members   []*RoomMemberResp `json:"members"`
}

type RoomMemberResp struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Bio         string `json:"bio"`
	Role        string `json:"role"`
	IsHidden    bool   `json:"is_hidden"`
	IsMuted     bool   `json:"is_muted"`
}

type RoomMemberReq struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}
type RoomRoleReq struct {
	Role string `json:"role" validate:"required"`
}
type RoomSettingsReq struct {
	IsHidden *bool `json:"is_hidden"`
	IsMuted  *bool `json:"is_muted"`
}
type RoomTransferReq struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

func (s *RoomService) roomAndRole(
	ctx context.Context,
	roomID, userID string,
) (uuid.UUID, uuid.UUID, db.MemberRole, error) {
	room, err := uuid.Parse(roomID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", ErrInvalidInput
	}
	user, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", ErrInvalidInput
	}
	role, err := s.query.GetRoomMemberRole(ctx, db.GetRoomMemberRoleParams{RoomID: room, UserID: user})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, "", ErrRoomAccessDenied
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("get room role: %w", err)
	}
	return room, user, role, nil
}

func (s *RoomService) GetRoom(ctx context.Context, userID, roomID string) (*RoomDetailResp, error) {
	room, _, _, err := s.roomAndRole(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	r, err := s.query.GetRoomByID(ctx, room)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	members, err := s.query.ListRoomMembers(ctx, room)
	if err != nil {
		return nil, fmt.Errorf("list room members: %w", err)
	}
	s.cacheRoomMemberRows(ctx, room, members)
	resp := &RoomDetailResp{
		RoomID:    r.RoomID.String(),
		ChatType:  string(r.ChatType),
		Name:      r.Name,
		AvatarURL: r.AvatarUrl,
		Members:   make([]*RoomMemberResp, 0, len(members)),
	}
	for _, m := range members {
		resp.Members = append(resp.Members, roomMemberResponse(m))
	}
	return resp, nil
}

func (s *RoomService) ListMembers(ctx context.Context, userID, roomID string) ([]*RoomMemberResp, error) {
	room, _, _, err := s.roomAndRole(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.query.ListRoomMembers(ctx, room)
	if err != nil {
		return nil, fmt.Errorf("list room members: %w", err)
	}
	s.cacheRoomMemberRows(ctx, room, rows)
	result := make([]*RoomMemberResp, 0, len(rows))
	for _, m := range rows {
		result = append(result, roomMemberResponse(m))
	}
	return result, nil
}

func roomMemberResponse(m *db.ListRoomMembersRow) *RoomMemberResp {
	return &RoomMemberResp{
		UserID:      m.UserID.String(),
		Username:    m.Username,
		DisplayName: m.DisplayName,
		AvatarURL:   m.AvatarUrl,
		Bio:         m.Bio,
		Role:        string(m.Role),
		IsHidden:    m.IsHidden,
		IsMuted:     m.IsMuted,
	}
}

func (s *RoomService) AddMember(ctx context.Context, actorID, roomID string, req RoomMemberReq) error {
	room, actor, role, err := s.roomAndRole(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if role != db.MemberRoleOwner && role != db.MemberRoleAdmin {
		return ErrRoomDenied
	}
	if _, err := s.query.GetRoomByID(ctx, room); errors.Is(err, pgx.ErrNoRows) {
		return ErrRoomNotFound
	} else if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	member, err := uuid.Parse(req.UserID)
	if err != nil || member == actor {
		return ErrInvalidInput
	}
	if _, err := s.query.GetUserByID(ctx, member); errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	} else if err != nil {
		return fmt.Errorf("get member user: %w", err)
	}
	if err := s.query.AddRoomMember(
		ctx,
		db.AddRoomMemberParams{RoomID: room, UserID: member, Role: db.MemberRoleMember},
	); err != nil {
		return fmt.Errorf("add room member: %w", err)
	}
	s.invalidateRoomCache(ctx, room, []uuid.UUID{actor, member})
	return nil
}

func (s *RoomService) KickMember(ctx context.Context, actorID, roomID, memberID string) error {
	room, actor, role, err := s.roomAndRole(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	member, err := uuid.Parse(memberID)
	if err != nil {
		return ErrInvalidInput
	}
	targetRole, err := s.query.GetRoomMemberRole(ctx, db.GetRoomMemberRoleParams{RoomID: room, UserID: member})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRoomMemberNotFound
	}
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if member == actor || targetRole == db.MemberRoleOwner ||
		(role == db.MemberRoleAdmin && targetRole == db.MemberRoleAdmin) {
		return ErrRoomDenied
	}
	if role != db.MemberRoleOwner && role != db.MemberRoleAdmin {
		return ErrRoomDenied
	}
	if err := s.query.DeleteRoomMember(ctx, db.DeleteRoomMemberParams{RoomID: room, UserID: member}); err != nil {
		return fmt.Errorf("kick room member: %w", err)
	}
	s.invalidateRoomCache(ctx, room, []uuid.UUID{actor, member})
	return nil
}

func (s *RoomService) Leave(ctx context.Context, userID, roomID string) error {
	room, user, role, err := s.roomAndRole(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if role == db.MemberRoleOwner {
		return ErrRoomOwnerCannotLeave
	}
	if err := s.query.DeleteRoomMember(ctx, db.DeleteRoomMemberParams{RoomID: room, UserID: user}); err != nil {
		return fmt.Errorf("leave room: %w", err)
	}
	s.invalidateRoomCache(ctx, room, []uuid.UUID{user})
	return nil
}

func (s *RoomService) Dissolve(ctx context.Context, userID, roomID string) error {
	room, _, role, err := s.roomAndRole(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if role != db.MemberRoleOwner {
		return ErrRoomOwnerRequired
	}
	members, err := s.query.ListRoomMembersByRoomIDs(ctx, []uuid.UUID{room})
	if err != nil {
		return fmt.Errorf("list members before dissolve: %w", err)
	}
	if err := s.query.DeleteRoom(ctx, room); err != nil {
		return fmt.Errorf("dissolve room: %w", err)
	}
	userIDs := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}
	s.invalidateRoomCache(ctx, room, userIDs)
	return nil
}

func (s *RoomService) TransferOwnership(ctx context.Context, userID, roomID string, req RoomTransferReq) error {
	room, actor, role, err := s.roomAndRole(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if role != db.MemberRoleOwner {
		return ErrRoomOwnerRequired
	}
	target, err := uuid.Parse(req.UserID)
	if err != nil || target == actor {
		return ErrInvalidInput
	}
	targetRole, err := s.query.GetRoomMemberRole(ctx, db.GetRoomMemberRoleParams{RoomID: room, UserID: target})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRoomMemberNotFound
	}
	if err != nil {
		return fmt.Errorf("get transfer target: %w", err)
	}
	if targetRole == db.MemberRoleOwner {
		return ErrInvalidInput
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin ownership transfer: %w", err)
	}
	committed := false
	defer rollbackTx(ctx, tx, &committed, "transfer ownership")
	q := s.query.WithTx(tx)
	if err := q.UpdateRoomMemberRole(
		ctx,
		db.UpdateRoomMemberRoleParams{RoomID: room, UserID: actor, Role: db.MemberRoleAdmin},
	); err != nil {
		return fmt.Errorf("demote owner: %w", err)
	}
	if err := q.UpdateRoomMemberRole(
		ctx,
		db.UpdateRoomMemberRoleParams{RoomID: room, UserID: target, Role: db.MemberRoleOwner},
	); err != nil {
		return fmt.Errorf("promote owner: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit ownership transfer: %w", err)
	}
	committed = true
	s.invalidateRoomCache(ctx, room, []uuid.UUID{actor, target})
	return nil
}

func (s *RoomService) SetRole(ctx context.Context, userID, roomID, memberID string, req RoomRoleReq) error {
	room, _, role, err := s.roomAndRole(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if role != db.MemberRoleOwner {
		return ErrRoomOwnerRequired
	}
	member, err := uuid.Parse(memberID)
	if err != nil {
		return ErrInvalidInput
	}
	if req.Role != string(db.MemberRoleAdmin) && req.Role != string(db.MemberRoleMember) {
		return ErrInvalidInput
	}
	if _, err := s.query.GetRoomMemberRole(
		ctx,
		db.GetRoomMemberRoleParams{RoomID: room, UserID: member},
	); errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrRoomMemberNotFound
	} else if err != nil {
		return fmt.Errorf("get role target: %w", err)
	}
	if err := s.query.UpdateRoomMemberRole(
		ctx,
		db.UpdateRoomMemberRoleParams{RoomID: room, UserID: member, Role: db.MemberRole(req.Role)},
	); err != nil {
		return fmt.Errorf("set room role: %w", err)
	}
	s.invalidateRoomCache(ctx, room, []uuid.UUID{member})
	return nil
}

func (s *RoomService) UpdateSettings(ctx context.Context, userID, roomID string, req RoomSettingsReq) error {
	room, user, _, err := s.roomAndRole(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if req.IsHidden == nil && req.IsMuted == nil {
		return ErrInvalidInput
	}
	current, err := s.query.GetRoomMemberSettings(ctx, db.GetRoomMemberSettingsParams{RoomID: room, UserID: user})
	if err != nil {
		return fmt.Errorf("get room settings: %w", err)
	}
	if req.IsHidden != nil {
		current.IsHidden = *req.IsHidden
	}
	if req.IsMuted != nil {
		current.IsMuted = *req.IsMuted
	}
	if err := s.query.UpdateRoomMemberSettings(
		ctx,
		db.UpdateRoomMemberSettingsParams{
			RoomID:   room,
			UserID:   user,
			IsHidden: current.IsHidden,
			IsMuted:  current.IsMuted,
		},
	); err != nil {
		return fmt.Errorf("update room settings: %w", err)
	}
	s.invalidateRoomCache(ctx, room, []uuid.UUID{user})
	return nil
}

func (s *RoomService) cacheRoomMembers(ctx context.Context, roomIDs []uuid.UUID) {
	if s.cache == nil || len(roomIDs) == 0 {
		return
	}
	rows, err := s.query.ListRoomMembersByRoomIDs(ctx, roomIDs)
	if err != nil {
		log.Error().Err(err).Msg("load room members for cache failed")
		return
	}
	byRoom := make(map[uuid.UUID][]cache.RoomMember, len(roomIDs))
	for _, row := range rows {
		byRoom[row.RoomID] = append(byRoom[row.RoomID], cache.RoomMember{
			UserID: row.UserID, JoinedAt: row.JoinedAt, IsMuted: row.IsMuted,
		})
	}
	for _, roomID := range roomIDs {
		s.cacheRoomMembersByID(ctx, roomID, byRoom[roomID])
	}
}

func (s *RoomService) cacheRoomMemberRows(ctx context.Context, roomID uuid.UUID, rows []*db.ListRoomMembersRow) {
	if s.cache == nil {
		return
	}
	members := make([]cache.RoomMember, 0, len(rows))
	for _, row := range rows {
		members = append(members, cache.RoomMember{
			UserID: row.UserID, JoinedAt: row.JoinedAt, IsMuted: row.IsMuted,
		})
	}
	s.cacheRoomMembersByID(ctx, roomID, members)
}

func (s *RoomService) cacheRoomMembersByID(ctx context.Context, roomID uuid.UUID, members []cache.RoomMember) {
	if err := s.cache.CacheRoomMembers(ctx, roomID, members); err != nil {
		log.Error().Err(err).Str("room_id", roomID.String()).Msg("cache room members failed")
	}
}

func (s *RoomService) invalidateRoomCache(ctx context.Context, roomID uuid.UUID, userIDs []uuid.UUID) {
	if s.cache == nil {
		return
	}
	if err := s.cache.InvalidateRoom(ctx, roomID, userIDs); err != nil {
		log.Error().Err(err).Str("room_id", roomID.String()).Msg("invalidate room cache failed")
	}
}
