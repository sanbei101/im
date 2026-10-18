package meta

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	kitexclient "github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/transport"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/proto/pb"
	"github.com/sanbei101/im/proto/pb/metaservice"
)

const metaServiceName = "im-meta"

// Client reads authoritative user profiles from the Meta Kitex service.
type Client struct {
	rpc metaservice.Client
}

func NewClient(address string) (*Client, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, errors.New("meta rpc address is empty")
	}
	address = strings.TrimPrefix(address, "http://")
	address = strings.TrimPrefix(address, "https://")
	address = strings.TrimRight(address, "/")
	rpc, err := metaservice.NewClient(
		metaServiceName,
		kitexclient.WithHostPorts(address),
		kitexclient.WithTransportProtocol(transport.GRPC),
	)
	if err != nil {
		return nil, fmt.Errorf("create meta rpc client: %w", err)
	}
	return &Client{rpc: rpc}, nil
}

func (c *Client) UsersByIDs(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]store.User, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]store.User{}, nil
	}
	rawIDs := make([]string, len(ids))
	for i, id := range ids {
		rawIDs[i] = id.String()
	}
	response, err := c.rpc.GetUsers(ctx, &pb.GetUsersReq{UserIds: rawIDs})
	if err != nil {
		return nil, fmt.Errorf("get meta users: %w", err)
	}
	users := make(map[uuid.UUID]store.User, len(response.GetUsers()))
	for _, profile := range response.GetUsers() {
		id, parseErr := uuid.Parse(profile.GetUserId())
		if parseErr != nil {
			return nil, fmt.Errorf("parse meta user id %q: %w", profile.GetUserId(), parseErr)
		}
		users[id] = store.User{
			UserID:    id,
			Username:  profile.GetUsername(),
			Nickname:  profile.GetNickname(),
			AvatarURL: profile.GetAvatarUrl(),
			CreatedAt: time.UnixMicro(profile.GetCreatedAtUnixMicro()),
		}
	}
	return users, nil
}
