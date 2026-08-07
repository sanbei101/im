package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/sanbei101/im/pkg/config"
)

const roomCacheTTL = 24 * time.Hour

type RoomMember struct {
	UserID   uuid.UUID
	JoinedAt time.Time
	IsMuted  bool
}

type RoomRef struct {
	RoomID uuid.UUID
	Score  int64
}

type RoomStore struct {
	client *redis.Client
}

func NewRoomStore(cfg *config.Config) *RoomStore {
	return &RoomStore{client: redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: 20,
	})}
}

func (s *RoomStore) Close() error {
	return s.client.Close()
}

func (s *RoomStore) CanSend(ctx context.Context, roomID, userID string) (bool, error) {
	room, err := uuid.Parse(roomID)
	if err != nil {
		return false, nil
	}
	user, err := uuid.Parse(userID)
	if err != nil {
		return false, nil
	}

	_, err = s.client.ZScore(ctx, membersKey(room), user.String()).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("check room member cache: %w", err)
	}
	muted, err := s.client.HGet(ctx, mutedKey(room), user.String()).Result()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("check room mute cache: %w", err)
	}
	return muted != "1", nil
}

func (s *RoomStore) CacheRoomMembers(ctx context.Context, roomID uuid.UUID, members []RoomMember) error {
	pipe := s.client.Pipeline()
	membersKey, mutedKey := membersKey(roomID), mutedKey(roomID)
	pipe.Del(ctx, membersKey, mutedKey)
	for _, member := range members {
		pipe.ZAdd(ctx, membersKey, redis.Z{Score: float64(member.JoinedAt.UnixMicro()), Member: member.UserID.String()})
		muted := "0"
		if member.IsMuted {
			muted = "1"
		}
		pipe.HSet(ctx, mutedKey, member.UserID.String(), muted)
	}
	pipe.Expire(ctx, membersKey, roomCacheTTL)
	pipe.Expire(ctx, mutedKey, roomCacheTTL)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("cache room members: %w", err)
	}
	return nil
}

func (s *RoomStore) CacheUserRooms(ctx context.Context, userID uuid.UUID, rooms []RoomRef) error {
	key := userRoomsKey(userID)
	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)
	for _, room := range rooms {
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(room.Score), Member: room.RoomID.String()})
	}
	pipe.Expire(ctx, key, roomCacheTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("cache user rooms: %w", err)
	}
	return nil
}

func (s *RoomStore) InvalidateRoom(ctx context.Context, roomID uuid.UUID, userIDs []uuid.UUID) error {
	keys := []string{membersKey(roomID), mutedKey(roomID)}
	for _, userID := range userIDs {
		keys = append(keys, userRoomsKey(userID))
	}
	if err := s.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("invalidate room cache: %w", err)
	}
	return nil
}

func membersKey(roomID uuid.UUID) string {
	return "room:" + roomID.String() + ":members"
}

func mutedKey(roomID uuid.UUID) string {
	return "room:" + roomID.String() + ":member_muted"
}

func userRoomsKey(userID uuid.UUID) string {
	return "user:" + userID.String() + ":rooms"
}
