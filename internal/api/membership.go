package api

import (
	"context"
	"time"
	"uuid"

	"github.com/phuslu/lru"

	"github.com/sanbei101/im/internal/store"
)

// membershipTTL bounds roster staleness: local mutations invalidate the room
// immediately and remote ones arrive through LISTEN/NOTIFY, so the TTL only
// backstops a lost notification.
const membershipTTL = 30 * time.Second

// Membership is the node-local roster cache in front of PostgreSQL. Every
// message-plane path - send admission, history authorization, reaction and
// read-cursor endpoints, push fan-out - resolves membership here, so the hot
// paths never wait on the database. The cached roster slice is shared between
// callers and must be treated as read-only.
type Membership struct {
	db    *store.DB
	cache *lru.TTLCache[uuid.UUID, []store.Member]
}

// NewMembership builds the node-local roster cache in front of db.
func NewMembership(db *store.DB) *Membership {
	return &Membership{
		db: db,
		cache: lru.NewTTLCache[uuid.UUID, []store.Member](
			4096,
			lru.WithShards[uuid.UUID, []store.Member](64),
		),
	}
}

// Members returns the room roster, empty for a room without members.
func (m *Membership) Members(ctx context.Context, roomID uuid.UUID) ([]store.Member, error) {
	if members, ok := m.cache.Get(roomID); ok {
		return members, nil
	}
	members, err := m.db.Members(ctx, roomID)
	if err != nil {
		return nil, err
	}
	m.cache.Set(roomID, members, membershipTTL)
	return members, nil
}

// Member returns one member's record, store.ErrNotFound for outsiders.
func (m *Membership) Member(ctx context.Context, roomID, userID uuid.UUID) (store.Member, error) {
	members, err := m.Members(ctx, roomID)
	if err != nil {
		return store.Member{}, err
	}
	for _, member := range members {
		if member.UserID == userID {
			return member, nil
		}
	}
	return store.Member{}, store.ErrNotFound
}

// Invalidate drops one room's roster; call after every local membership
// mutation so the next read refetches from PostgreSQL.
func (m *Membership) Invalidate(roomID uuid.UUID) {
	m.cache.Delete(roomID)
}
