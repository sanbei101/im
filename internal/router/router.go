package router

import (
	"bytes"
	"encoding/json/v2"
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

// maxSingleChatBody bounds the one route whose room id must be derived from
// its JSON body: the peer id of a single-chat creation request.
const maxSingleChatBody = 4 << 10

// EdgeConfig describes the public HTTP edge. Every room-scoped request is
// forwarded to the owning API node; all other requests go to next.
type EdgeConfig struct {
	Slots     int
	HTTPAddrs []string
}

// Edge is the public HTTP entry used by the gateway. It never handles room
// requests locally; the owning API node does.
type Edge struct {
	slots   int
	next    http.Handler
	proxies []*httputil.ReverseProxy
}

func NewEdge(cfg EdgeConfig, next http.Handler) (*Edge, error) {
	if next == nil {
		return nil, errors.New("http edge next handler is nil")
	}
	if cfg.Slots < 1 || cfg.Slots > pkg.MaxSlots {
		return nil, fmt.Errorf("http edge slots must be within [1, %d]", pkg.MaxSlots)
	}
	if len(cfg.HTTPAddrs) == 0 {
		return nil, errors.New("http edge requires at least one api http address")
	}
	proxies := make([]*httputil.ReverseProxy, len(cfg.HTTPAddrs))
	for i, address := range cfg.HTTPAddrs {
		proxy, err := newProxy(address)
		if err != nil {
			return nil, fmt.Errorf("create api http proxy %q: %w", address, err)
		}
		proxies[i] = proxy
	}
	return &Edge{slots: cfg.Slots, next: next, proxies: proxies}, nil
}

func (e *Edge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	roomID, ok := roomIDFromRequest(r)
	if !ok {
		e.next.ServeHTTP(w, r)
		return
	}
	slot, err := pkg.RoomSlot(roomID, e.slots)
	if err != nil {
		e.next.ServeHTTP(w, r)
		return
	}
	owner, err := pkg.NodeIndex(slot, e.slots, len(e.proxies))
	if err != nil || owner < 0 || owner >= len(e.proxies) {
		e.next.ServeHTTP(w, r)
		return
	}
	e.proxies[owner].ServeHTTP(w, r)
}

// roomIDFromRequest finds the room id in the query or the path. Every
// room-scoped route carries the id in its path or as the room_id query
// parameter, so routing never buffers or parses a request body - except the
// single-chat creation route, whose room id is derived before the room exists.
func roomIDFromRequest(r *http.Request) (uuid.UUID, bool) {
	if raw := r.URL.Query().Get("room_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			return id, true
		}
	}
	if strings.TrimSuffix(r.URL.Path, "/") == "/api/v1/rooms/single" {
		return roomIDFromSingleChat(r)
	}
	return roomIDFromPath(r.URL.Path)
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

// roomIDFromSingleChat derives the deterministic one-to-one room id so a
// creation request can be routed before the room exists. The body is restored
// before forwarding so the upstream handler sees the original bytes.
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

	if r.Body == nil || r.Body == http.NoBody || r.ContentLength > maxSingleChatBody {
		return uuid.Nil(), false
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSingleChatBody+1))
	if err != nil || len(body) > maxSingleChatBody {
		return uuid.Nil(), false
	}
	if err := r.Body.Close(); err != nil {
		return uuid.Nil(), false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

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

func bearerToken(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}

// NewProxy exposes the same reverse proxy configuration to callers that only
// need a plain pass-through handler (for example the gateway's API fallback).
func NewProxy(address string) (http.Handler, error) {
	return newProxy(address)
}

func newProxy(address string) (*httputil.ReverseProxy, error) {
	target, err := parseTarget(address)
	if err != nil {
		return nil, err
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
		render.Error(w, http.StatusBadGateway, "api node unavailable: "+err.Error())
	}
	return proxy, nil
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
