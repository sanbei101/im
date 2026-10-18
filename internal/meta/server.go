package meta

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"uuid"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/proto/pb"
)

// Server implements the internal Meta RPC service.
type Server struct {
	store *store.Store
}

func NewServer(data *store.Store) *Server {
	return &Server{store: data}
}

func (s *Server) GetUsers(ctx context.Context, req *pb.GetUsersReq) (*pb.GetUsersResp, error) {
	if len(req.GetUserIds()) > 100 {
		return nil, errors.New("too many user_ids (max 100)")
	}
	ids := make([]uuid.UUID, 0, len(req.GetUserIds()))
	for _, raw := range req.GetUserIds() {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid user_id %q: %w", raw, err)
		}
		ids = append(ids, id)
	}
	users, err := s.store.UsersByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load users: %w", err)
	}
	profiles := make([]*pb.UserProfile, 0, len(users))
	for _, user := range users {
		profiles = append(profiles, &pb.UserProfile{
			UserId:             user.UserID.String(),
			Username:           user.Username,
			Nickname:           user.Nickname,
			AvatarUrl:          user.AvatarURL,
			CreatedAtUnixMicro: user.CreatedAt.UnixMicro(),
		})
	}
	slices.SortFunc(profiles, func(a, b *pb.UserProfile) int {
		return strings.Compare(a.GetUserId(), b.GetUserId())
	})
	return &pb.GetUsersResp{Users: profiles}, nil
}

var _ pb.MetaService = (*Server)(nil)
