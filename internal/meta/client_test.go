package meta

import (
	"context"
	"net"
	"testing"
	"time"
	"uuid"

	"github.com/cloudwego/kitex/server"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/proto/pb/metaservice"
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

	address := freeAddr(t)
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	srv := metaservice.NewServer(NewServer(data), server.WithServiceAddr(tcpAddr))
	done := make(chan error, 1)
	go func() {
		done <- srv.Run()
	}()
	t.Cleanup(func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("stop meta server: %v", err)
		}
		if err := <-done; err != nil {
			t.Errorf("meta server exited: %v", err)
		}
	})

	client, err := NewClient(address)
	if err != nil {
		t.Fatal(err)
	}
	var users map[uuid.UUID]store.User
	deadline := time.Now().Add(2 * time.Second)
	for {
		users, err = client.UsersByIDs(context.Background(), []uuid.UUID{user.UserID})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("meta rpc did not become ready: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := users[user.UserID]; got.Username != "alice" {
		t.Fatalf("users = %+v", users)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}
