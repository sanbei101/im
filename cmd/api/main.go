package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"uuid"

	"github.com/cloudwego/kitex/server"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/api"
	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/logger"
	"github.com/sanbei101/im/proto/pb/gatewayservice"
)

const (
	httpReadHeaderTimeout = 10 * time.Second
	httpIdleTimeout       = time.Minute
	shutdownTimeout       = 5 * time.Second
)

func main() {
	logger.InitLogger()
	if err := run(config.New()); err != nil {
		log.Fatal().Err(err).Msg("api server failed")
	}
}

// run validates the configuration and serves until the process receives
// SIGINT/SIGTERM. It takes the config explicitly so tests can exercise the
// startup path without touching the working directory.
func run(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	if err := jwt.Init(cfg.JWTSecret); err != nil {
		return fmt.Errorf("init jwt: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return serve(ctx, cfg)
}

// serve owns every long-lived component. All loops derive from ctx, so a
// failure in one component shuts the rest down before serve returns.
func serve(ctx context.Context, cfg *config.Config) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := store.OpenDB(runCtx, cfg.Postgres.URL)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()

	data, err := store.Open(cfg.Store.Path)
	if err != nil {
		return fmt.Errorf("open pebble store: %w", err)
	}
	defer data.Close()
	data.AttachDB(db)

	objects, err := store.NewMinioObjectStore(store.ObjectStoreConfig{
		Endpoint:  cfg.Storage.Endpoint,
		Bucket:    cfg.Storage.Bucket,
		AccessKey: cfg.Storage.AccessKey,
		SecretKey: cfg.Storage.SecretKey,
		Region:    cfg.Storage.Region,
		UseSSL:    cfg.Storage.UseSSL,
		PathStyle: cfg.Storage.PathStyle,
	})
	if err != nil {
		return fmt.Errorf("create object store: %w", err)
	}
	if err := objects.EnsureBucket(runCtx); err != nil {
		return fmt.Errorf("ensure archive bucket: %w", err)
	}
	if err := data.AttachRemote(objects, filepath.Join(cfg.Store.Path, "archive-cache"), 512<<20); err != nil {
		return fmt.Errorf("attach remote archive: %w", err)
	}

	streamHandler := api.NewStreamHandler(data, db, cfg.Shard.Slots, cfg.API.NodeIndex, cfg.API.NodeCount)
	notifyDone := make(chan struct{})
	go func() {
		defer close(notifyDone)
		db.ListenNotifications(runCtx, func(notification store.Notification) {
			if notification.Origin == cfg.API.NodeID {
				return // the publishing node already delivered it synchronously
			}
			streamHandler.NotifyRaw(notification.Event, notification.UserIDs, notification.Data)
		})
	}()
	handler := api.NewRouter(db, data, streamHandler, objects, cfg.Storage.PublicBaseURL, cfg.API.NodeID)
	streamAddr, err := net.ResolveTCPAddr("tcp", cfg.API.Addr)
	if err != nil {
		return fmt.Errorf("resolve api stream address: %w", err)
	}
	kitexServer := gatewayservice.NewServer(streamHandler,
		server.WithServiceAddr(streamAddr),
		server.WithExitWaitTime(shutdownTimeout),
	)

	httpListener, err := net.Listen("tcp", cfg.API.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen api http: %w", err)
	}
	pprofListener, err := net.Listen("tcp", cfg.API.PProfAddr)
	if err != nil {
		// The HTTP listener is not served yet, so close it before returning.
		return errors.Join(fmt.Errorf("listen pprof: %w", err), httpListener.Close())
	}

	var tasks sync.WaitGroup
	tasks.Go(func() {
		every(runCtx, time.Duration(cfg.Store.BackupInterval)*time.Second, func(ctx context.Context, now time.Time) {
			dir := filepath.Join(cfg.Store.BackupPath, now.Format("20060102-150405"))
			if err := data.Checkpoint(ctx, dir); err != nil {
				log.Error().Err(err).Str("dir", dir).Msg("pebble checkpoint failed")
			}
		})
	})
	tasks.Go(func() {
		params := store.DefaultArchiveParams()
		params.Owns = ownsRoom(cfg)
		every(runCtx, time.Minute, func(ctx context.Context, _ time.Time) {
			if err := data.ArchiveOnce(ctx, objects, params); err != nil {
				log.Error().Err(err).Msg("message archive failed")
			}
		})
	})

	serveErr := make(chan error, 3)
	var httpServers sync.WaitGroup
	httpServers.Go(func() {
		err := serveHTTP(runCtx, newHTTPServer(cfg.API.HTTPAddr, handler), httpListener)
		if err == nil && runCtx.Err() == nil {
			err = errors.New("api http server stopped unexpectedly")
		}
		if err != nil {
			serveErr <- err
		}
	})
	httpServers.Go(func() {
		err := serveHTTP(runCtx, newPProfServer(cfg.API.PProfAddr), pprofListener)
		if err == nil && runCtx.Err() == nil {
			err = errors.New("pprof server stopped unexpectedly")
		}
		if err != nil {
			serveErr <- err
		}
	})

	kitexDone := make(chan struct{})
	go func() {
		defer close(kitexDone)
		log.Info().Str("addr", cfg.API.Addr).Msg("starting api stream server")
		err := kitexServer.Run()
		if err == nil && runCtx.Err() == nil {
			err = errors.New("api stream server stopped unexpectedly")
		}
		if err != nil {
			serveErr <- fmt.Errorf("api stream server: %w", err)
		}
	}()

	log.Info().Str("addr", httpListener.Addr().String()).Msg("starting API server")
	var serveFailure error
	select {
	case <-runCtx.Done():
		log.Info().Msg("shutting down gracefully...")
	case serveFailure = <-serveErr:
		log.Error().Err(serveFailure).Msg("api server failed, shutting down")
		cancel()
	}

	httpServers.Wait()
	tasks.Wait()
	cancel()
	<-notifyDone
	if err := kitexServer.Stop(); err != nil {
		log.Error().Err(err).Msg("api stream server shutdown failed")
	}
	<-kitexDone
	if serveFailure != nil {
		return serveFailure
	}
	log.Info().Msg("API server exited")
	return nil
}

// every runs fn on each tick of interval until ctx is cancelled.
func every(ctx context.Context, interval time.Duration, fn func(ctx context.Context, now time.Time)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			fn(ctx, now)
		}
	}
}

// ownsRoom reports whether this API node owns the room's shard slot. It is
// used by the archiver so a node only deletes history it is responsible for.
func ownsRoom(cfg *config.Config) func(uuid.UUID) bool {
	return func(roomID uuid.UUID) bool {
		slot, err := pkg.RoomSlot(roomID, cfg.Shard.Slots)
		if err != nil {
			return false
		}
		index, err := pkg.NodeIndex(slot, cfg.Shard.Slots, cfg.API.NodeCount)
		return err == nil && index == cfg.API.NodeIndex
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		IdleTimeout:       httpIdleTimeout,
	}
}

func newPProfServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}
}

// serveHTTP runs srv on ln and shuts it down when ctx is cancelled. It returns
// nil for a graceful shutdown and the serving error otherwise.
func serveHTTP(ctx context.Context, srv *http.Server, ln net.Listener) error {
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server %s: %w", srv.Addr, err)
		}
		return nil
	case err := <-serveDone:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server %s: %w", srv.Addr, err)
	}
}
