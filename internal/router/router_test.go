package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"uuid"

	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/jwt"
)

func TestMain(m *testing.M) {
	if err := jwt.Init("test-secret"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestEdgeRoutesRoomRequests(t *testing.T) {
	const slots, nodes = 1024, 2

	t.Run("path", func(t *testing.T) {
		roomID := roomForNode(t, slots, nodes, 1)
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeBody(t, w, "upstream")
		}))
		defer upstream.Close()

		edge := newEdge(t, upstream.Listener.Addr().String(), localHandler(t, "next"))
		recorder := httptest.NewRecorder()
		edge.ServeHTTP(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/rooms/"+roomID.String()+"/members",
			nil,
		))
		if recorder.Code != http.StatusOK || recorder.Body.String() != "upstream" {
			t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("query", func(t *testing.T) {
		roomID := roomForNode(t, slots, nodes, 1)
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeBody(t, w, "upstream")
		}))
		defer upstream.Close()

		edge := newEdge(t, upstream.Listener.Addr().String(), localHandler(t, "next"))
		recorder := httptest.NewRecorder()
		edge.ServeHTTP(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/messages/history?room_id="+roomID.String(),
			nil,
		))
		if recorder.Code != http.StatusOK || recorder.Body.String() != "upstream" {
			t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("body is preserved", func(t *testing.T) {
		roomID := roomForNode(t, slots, nodes, 1)
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
				return
			}
			writeBody(t, w, string(body))
		}))
		defer upstream.Close()

		edge := newEdge(t, upstream.Listener.Addr().String(), localHandler(t, "next"))
		body := `{"room_id":"` + roomID.String() + `","msg_id":"00000000-0000-0000-0000-000000000000"}`
		recorder := httptest.NewRecorder()
		edge.ServeHTTP(recorder, httptest.NewRequest(
			http.MethodPost,
			"/api/v1/messages/recall",
			strings.NewReader(body),
		))
		if recorder.Code != http.StatusOK || recorder.Body.String() != body {
			t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("non-room request", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Error("non-room request was forwarded to a room owner")
		}))
		defer upstream.Close()

		edge := newEdge(t, upstream.Listener.Addr().String(), localHandler(t, "next"))
		recorder := httptest.NewRecorder()
		edge.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil))
		if recorder.Code != http.StatusOK || recorder.Body.String() != "next" {
			t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
		}
	})
}

func TestEdgeRoutesSingleChatCreation(t *testing.T) {
	const slots, nodes = 1024, 2
	user1, user2 := singleChatPairForNode(t, slots, nodes, 1)
	token, err := jwt.GenerateToken(user1.String())
	if err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read body: %v", readErr)
			return
		}
		if !strings.Contains(string(body), user2.String()) {
			t.Errorf("upstream body = %q", body)
		}
		writeBody(t, w, "upstream")
	}))
	defer upstream.Close()

	edge := newEdge(t, upstream.Listener.Addr().String(), localHandler(t, "next"))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/rooms/single",
		strings.NewReader(`{"user_id_2":"`+user2.String()+`"}`),
	)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	edge.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "upstream" {
		t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestEdgeRejectsInvalidConfig(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, cfg := range []EdgeConfig{
		{Slots: 0, HTTPAddrs: []string{"127.0.0.1:1"}},
		{Slots: 1, HTTPAddrs: nil},
		{Slots: 1, HTTPAddrs: []string{"://bad"}},
	} {
		if _, err := NewEdge(cfg, next); err == nil {
			t.Fatalf("expected error for edge config %+v", cfg)
		}
	}
	if _, err := NewEdge(EdgeConfig{Slots: 1, HTTPAddrs: []string{"127.0.0.1:1"}}, nil); err == nil {
		t.Fatal("expected error for nil edge next handler")
	}
}

func newEdge(t *testing.T, upstream string, next http.Handler) *Edge {
	t.Helper()
	edge, err := NewEdge(EdgeConfig{
		Slots:     1024,
		HTTPAddrs: []string{"127.0.0.1:1", upstream},
	}, next)
	if err != nil {
		t.Fatal(err)
	}
	return edge
}

func localHandler(t *testing.T, body string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeBody(t, w, body)
	})
}

func writeBody(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func singleChatPairForNode(t *testing.T, slots, nodes, want int) (uuid.UUID, uuid.UUID) {
	t.Helper()
	for range 100_000 {
		a, b := uuid.NewV7(), uuid.NewV7()
		roomID := pkg.SingleRoomID(a, b)
		slot, err := pkg.RoomSlot(roomID, slots)
		if err != nil {
			t.Fatal(err)
		}
		owner, err := pkg.NodeIndex(slot, slots, nodes)
		if err != nil {
			t.Fatal(err)
		}
		if owner == want {
			return a, b
		}
	}
	t.Fatalf("could not find a single chat owned by node %d", want)
	return uuid.Nil(), uuid.Nil()
}

func roomForNode(t *testing.T, slots, nodes, want int) uuid.UUID {
	t.Helper()
	for range 100_000 {
		roomID := uuid.NewV7()
		slot, err := pkg.RoomSlot(roomID, slots)
		if err != nil {
			t.Fatal(err)
		}
		owner, err := pkg.NodeIndex(slot, slots, nodes)
		if err != nil {
			t.Fatal(err)
		}
		if owner == want {
			return roomID
		}
	}
	t.Fatalf("could not find a room owned by node %d", want)
	return uuid.Nil()
}
