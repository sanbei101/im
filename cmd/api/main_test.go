package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/config"
)

func TestEvery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticks := make(chan time.Time, 4)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		every(ctx, time.Millisecond, func(_ context.Context, now time.Time) {
			ticks <- now
		})
	}()

	for i := range 2 {
		select {
		case <-ticks:
		case <-time.After(time.Second):
			t.Fatalf("tick %d was not delivered", i)
		}
	}

	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("every did not stop after context cancellation")
	}
}

func TestServeHTTPGracefulShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := newHTTPServer(ln.Addr().String(), http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serveHTTP(ctx, srv, ln)
	}()

	response, err := http.Get("http://" + ln.Addr().String() + "/healthz")
	if err != nil {
		t.Fatalf("request health endpoint: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveHTTP: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveHTTP did not shut down")
	}
}

func TestServeHTTPWaitsForInFlightRequest(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	srv := newHTTPServer(ln.Addr().String(), http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			close(started)
			<-release
			w.WriteHeader(http.StatusNoContent)
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serveHTTP(ctx, srv, ln)
	}()

	requestDone := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + ln.Addr().String() + "/")
		if err == nil {
			err = response.Body.Close()
		}
		requestDone <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not reach the handler")
	}

	cancel()
	select {
	case err := <-done:
		t.Fatalf("serveHTTP returned before the in-flight request completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatalf("in-flight request: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request did not complete")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveHTTP: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveHTTP did not shut down after the request completed")
	}
}

func TestServeHTTPReportsServeError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	srv := newHTTPServer(ln.Addr().String(), http.NewServeMux())
	if err := serveHTTP(context.Background(), srv, ln); err == nil {
		t.Fatal("serveHTTP returned nil for a closed listener")
	}
}

func TestOwnsRoom(t *testing.T) {
	cfg := config.NewTest()
	cfg.Shard.Slots = 1024
	cfg.API.NodeCount = 2
	cfg.API.NodeIndex = 1

	owns := ownsRoom(cfg)
	half := uint16(cfg.Shard.Slots / 2)
	var sawOwned, sawForeign bool
	for range 10_000 {
		roomID := uuid.NewV7()
		slot, err := pkg.RoomSlot(roomID, cfg.Shard.Slots)
		if err != nil {
			t.Fatal(err)
		}
		if slot >= half {
			if !owns(roomID) {
				t.Fatalf("node 1 must own slot %d", slot)
			}
			sawOwned = true
		} else {
			if owns(roomID) {
				t.Fatalf("node 1 must not own slot %d", slot)
			}
			sawForeign = true
		}
		if sawOwned && sawForeign {
			break
		}
	}
	if !sawOwned || !sawForeign {
		t.Fatal("did not sample rooms from both partitions")
	}

	cfg.Shard.Slots = 0
	if ownsRoom(cfg)(uuid.NewV7()) {
		t.Fatal("an invalid topology must not own any room")
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	t.Run("missing http address", func(t *testing.T) {
		cfg := config.NewTest()
		cfg.API.HTTPAddr = ""
		if err := run(cfg); err == nil {
			t.Fatal("run accepted an empty api.http_addr")
		}
	})

	t.Run("missing jwt secret", func(t *testing.T) {
		cfg := config.NewTest()
		cfg.JWTSecret = ""
		if err := run(cfg); err == nil {
			t.Fatal("run accepted an empty jwt_secret")
		}
	})
}

func TestServeStartsAndStops(t *testing.T) {
	objectStore := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))
	defer objectStore.Close()

	cfg := config.NewTest()
	cfg.Store.Path = t.TempDir()
	cfg.Store.BackupPath = t.TempDir()
	cfg.API.Addr = freeAddr(t)
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.API.PProfAddr = freeAddr(t)
	cfg.Storage.Endpoint = strings.TrimPrefix(objectStore.URL, "http://")
	cfg.Storage.UseSSL = false
	cfg.Storage.PathStyle = true
	cfg.Storage.PublicBaseURL = "http://example.test"

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, cfg)
	}()

	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for {
		response, err := client.Get("http://" + cfg.API.HTTPAddr + "/healthz")
		if err == nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Fatalf("close health response: %v", closeErr)
			}
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("API did not become healthy")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop after context cancellation")
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}
