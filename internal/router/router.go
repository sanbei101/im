package router

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

const (
	// routedHeader marks a request already forwarded by another API node.
	// The receiving node handles it locally instead of forwarding it again.
	routedHeader = "X-Im-Room-Routed"
	// maxSniffBody bounds the body room_id probe; file uploads never pass
	// through this path, and room_id is always a short JSON string.
	maxSniffBody = 1 << 20
)

// Config describes the static API topology. HTTPAddrs is indexed by node,
// matching the gateway's api_addrs order.
type Config struct {
	Slots     int
	LocalNode int
	HTTPAddrs []string
}

// Router forwards HTTP requests that carry a room_id to the API node that
// owns the room's shard slot. Requests without a room_id stay local.
type Router struct {
	slots   int
	local   int
	next    http.Handler
	proxies []*httputil.ReverseProxy
}

func New(cfg Config, next http.Handler) (*Router, error) {
	if next == nil {
		return nil, errors.New("room router next handler is nil")
	}
	if cfg.Slots < 1 || cfg.Slots > pkg.MaxSlots {
		return nil, fmt.Errorf("room router slots must be within [1, %d]", pkg.MaxSlots)
	}
	if len(cfg.HTTPAddrs) == 0 {
		return nil, errors.New("room router requires at least one api http address")
	}
	if cfg.LocalNode < 0 || cfg.LocalNode >= len(cfg.HTTPAddrs) {
		return nil, errors.New("room router local node is outside api http addresses")
	}

	proxies := make([]*httputil.ReverseProxy, len(cfg.HTTPAddrs))
	for i, address := range cfg.HTTPAddrs {
		target, err := parseTarget(address)
		if err != nil {
			return nil, fmt.Errorf("parse api http address %q: %w", address, err)
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConnsPerHost:   64,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			render.Error(w, http.StatusBadGateway, "room owner unavailable: "+err.Error())
		}
		proxies[i] = proxy
	}

	return &Router{
		slots:   cfg.Slots,
		local:   cfg.LocalNode,
		next:    next,
		proxies: proxies,
	}, nil
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(routedHeader) != "" {
		r.Header.Del(routedHeader)
		rt.next.ServeHTTP(w, r)
		return
	}

	roomID, ok := roomIDFromRequest(r)
	if !ok {
		rt.next.ServeHTTP(w, r)
		return
	}

	slot, err := pkg.RoomSlot(roomID, rt.slots)
	if err != nil {
		rt.next.ServeHTTP(w, r)
		return
	}
	owner, err := pkg.NodeIndex(slot, rt.slots, len(rt.proxies))
	if err != nil || owner == rt.local || rt.proxies[owner] == nil {
		rt.next.ServeHTTP(w, r)
		return
	}

	r.Header.Set(routedHeader, "1")
	rt.proxies[owner].ServeHTTP(w, r)
}

// roomIDFromRequest finds the room id in the query, path, or JSON body. Body
// probing only runs for methods that can carry one, and the body is restored
// before forwarding so the upstream handler sees the original bytes.
func roomIDFromRequest(r *http.Request) (uuid.UUID, bool) {
	if raw := r.URL.Query().Get("room_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			return id, true
		}
	}
	if strings.TrimSuffix(r.URL.Path, "/") == "/api/v1/rooms/single" {
		return roomIDFromSingleChat(r)
	}
	if id, ok := roomIDFromPath(r.URL.Path); ok {
		return id, true
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return roomIDFromBody(r)
	default:
		return uuid.Nil(), false
	}
}

func roomIDFromPath(path string) (uuid.UUID, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "api" || parts[1] != "v1" {
		return uuid.Nil(), false
	}
	switch parts[2] {
	case "rooms", "conversations":
		switch parts[3] {
		case "", "group", "single", "list":
			return uuid.Nil(), false
		}
		id, err := uuid.Parse(parts[3])
		if err != nil {
			return uuid.Nil(), false
		}
		return id, true
	default:
		return uuid.Nil(), false
	}
}

func roomIDFromBody(r *http.Request) (uuid.UUID, bool) {
	body, ok := readBody(r)
	if !ok {
		return uuid.Nil(), false
	}
	var payload struct {
		RoomID string `json:"room_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return uuid.Nil(), false
	}
	id, err := uuid.Parse(payload.RoomID)
	if err != nil {
		return uuid.Nil(), false
	}
	return id, true
}

// roomIDFromSingleChat derives the deterministic one-to-one room id so a
// creation request can be routed before the room exists.
func roomIDFromSingleChat(r *http.Request) (uuid.UUID, bool) {
	token := bearerToken(r)
	if token == "" {
		return uuid.Nil(), false
	}
	userID, err := jwt.ParseToken(token)
	if err != nil {
		return uuid.Nil(), false
	}
	myID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil(), false
	}

	body, ok := readBody(r)
	if !ok {
		return uuid.Nil(), false
	}
	var payload struct {
		UserID2 string `json:"user_id_2"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return uuid.Nil(), false
	}
	otherID, err := uuid.Parse(payload.UserID2)
	if err != nil || otherID == myID {
		return uuid.Nil(), false
	}
	return pkg.SingleRoomID(myID, otherID), true
}

func readBody(r *http.Request) ([]byte, bool) {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength > maxSniffBody {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSniffBody+1))
	if err != nil {
		return nil, false
	}
	if err := r.Body.Close(); err != nil {
		return nil, false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) > maxSniffBody {
		return nil, false
	}
	return body, true
}

func bearerToken(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}

func parseTarget(address string) (*url.URL, error) {
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	target, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	if target.Host == "" {
		return nil, errors.New("missing host")
	}
	if target.Hostname() == "" {
		port := target.Port()
		if port == "" {
			return nil, errors.New("missing host and port")
		}
		target.Host = "127.0.0.1:" + port
	}
	return target, nil
}
