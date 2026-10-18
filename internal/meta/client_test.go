package meta

import (
	"context"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/sanbei101/im/internal/api"
	"github.com/sanbei101/im/internal/store"
)

func TestClientUsersByIDs(t *testing.T) {
	data, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := data.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	user, err := data.CreateUser(context.Background(), "alice", "hash")
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(api.NewMetaRouter(data, "internal-secret"))
	defer server.Close()

	client, err := NewClient(server.URL, "internal-secret")
	if err != nil {
		t.Fatal(err)
	}
	users, err := client.UsersByIDs(context.Background(), []uuid.UUID{user.UserID})
	if err != nil {
		t.Fatal(err)
	}
	if got := users[user.UserID]; got.Username != "alice" {
		t.Fatalf("users = %+v", users)
	}

	wrongToken, err := NewClient(server.URL, "wrong")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrongToken.UsersByIDs(context.Background(), []uuid.UUID{user.UserID}); err == nil {
		t.Fatal("wrong internal token was accepted")
	}
}
