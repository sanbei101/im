package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
)

const internalTokenHeader = "X-Internal-Token"

// Client reads authoritative user profiles from the Meta service.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(address, token string) (*Client, error) {
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("meta address is empty")
	}
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parse meta address: %w", err)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("meta address %q has no host", address)
	}
	return &Client{
		baseURL: strings.TrimRight(parsed.String(), "/"),
		token:   token,
		http: &http.Client{
			Timeout: 5 * time.Second,
		},
	}, nil
}

func (c *Client) UsersByIDs(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]store.User, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]store.User{}, nil
	}
	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = id.String()
	}
	endpoint := c.baseURL + "/internal/users?ids=" + url.QueryEscape(strings.Join(values, ","))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create meta request: %w", err)
	}
	request.Header.Set(internalTokenHeader, c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request meta users: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			return nil, fmt.Errorf("meta users returned %d; read body: %w", response.StatusCode, readErr)
		}
		return nil, fmt.Errorf("meta users returned %d: %s", response.StatusCode, body)
	}
	var payload render.Response[map[uuid.UUID]store.User]
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode meta users: %w", err)
	}
	if payload.Code != http.StatusOK {
		return nil, fmt.Errorf("meta users error %d: %s", payload.Code, payload.Msg)
	}
	if payload.Data == nil {
		return map[uuid.UUID]store.User{}, nil
	}
	return payload.Data, nil
}
