package api

import (
	"context"
	"uuid"

	"github.com/sanbei101/im/internal/store"
)

// UserDirectory resolves user profiles from the authoritative user store.
// Work nodes use a remote Meta client; single-process tests use the local
// Pebble store directly.
type UserDirectory interface {
	UsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]store.User, error)
}
