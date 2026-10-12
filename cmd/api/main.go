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

func main() {
	logger.InitLogger()
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("api server failed")
	}
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

func run() error {
	cfg := config.New()
	jwt.Init(cfg.JWTSecret)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	data, err := store.Open(cfg.Store.Path)
	if err != nil {
		cancel()
		return fmt.Errorf("open pebble store: %w", err)
	}
	defer data.Close()
	defer cancel()

	objects, err := store.NewMinioObjectStore(
		cfg.Storage.Endpoint, cfg.Storage.Bucket,
		cfg.Storage.AccessKeyID, cfg.Storage.SecretAccessKey, cfg.Storage.UseSSL,
	)
	if err != nil {
		return fmt.Errorf("create object store: %w", err)
	}
	if err := objects.EnsureBucket(ctx); err != nil {
		return fmt.Errorf("ensure archive bucket: %w", err)
	}
	if err := data.AttachRemote(objects, filepath.Join(cfg.Store.Path, "archive-cache"), 512<<20); err != nil {
		return fmt.Errorf("attach remote archive: %w", err)
	}

	streamHandler := api.NewStreamHandler(data, cfg.API.NodeID, cfg.Shard.Slots, cfg.API.NodeIndex, cfg.API.NodeCount)
	r := api.NewRouter(data, streamHandler, objects, cfg.Storage.PublicURLPrefix)
	listenAddr, err := net.ResolveTCPAddr("tcp", cfg.API.Addr)
	if err != nil {
		return fmt.Errorf("resolve api stream address: %w", err)
	}
	kitexServer := gatewayservice.NewServer(streamHandler,
		server.WithServiceAddr(listenAddr),
		server.WithExitWaitTime(5*time.Second),
	)
	srv := &http.Server{
		Addr:              ":8801",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       time.Minute,
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		debug := &http.Server{Addr: "127.0.0.1:6061", ReadHeaderTimeout: 10 * time.Second}
		go func() {
			<-ctx.Done()
			if err := debug.Shutdown(context.Background()); err != nil {
				log.Error().Err(err).Msg("pprof server forced to shutdown")
			}
		}()
		if err := debug.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("pprof server stopped")
		}
	})
	wg.Go(func() {
		every(ctx, time.Duration(cfg.Store.BackupInterval)*time.Second, func(ctx context.Context, now time.Time) {
			dir := filepath.Join(cfg.Store.BackupPath, now.Format("20060102-150405"))
			if err := data.Checkpoint(ctx, dir); err != nil {
				log.Error().Err(err).Str("dir", dir).Msg("pebble checkpoint failed")
			}
		})
	})

	wg.Go(func() {
		params := store.DefaultArchiveParams()
		params.Owns = func(roomID uuid.UUID) bool {
			slot, err := pkg.RoomSlot(roomID, cfg.Shard.Slots)
			if err != nil {
				return false
			}
			index, err := pkg.NodeIndex(slot, cfg.Shard.Slots, cfg.API.NodeCount)
			return err == nil && index == cfg.API.NodeIndex
		}
		every(ctx, time.Minute, func(ctx context.Context, _ time.Time) {
			if err := data.ArchiveOnce(ctx, objects, params); err != nil {
				log.Error().Err(err).Msg("message archive failed")
			}
		})
	})

	serveErr := make(chan error, 2)
	go func() {
		log.Info().Msg("starting API server on :8801")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("start api server: %w", err)
			return
		}
		log.Info().Msg("API server stopped")
	}()
	go func() {
		log.Info().Str("addr", cfg.API.Addr).Msg("starting api stream server")
		if err := kitexServer.Run(); err != nil {
			serveErr <- fmt.Errorf("api stream server: %w", err)
			return
		}
	}()

	var serveFailure error
	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down gracefully...")
	case err := <-serveErr:
		serveFailure = err
		cancel()
		log.Error().Err(err).Msg("api server failed, shutting down")
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("API server forced to shutdown")
	}
	wg.Wait()
	if err := kitexServer.Stop(); err != nil {
		log.Error().Err(err).Msg("api stream server shutdown failed")
	}
	if serveFailure != nil {
		return serveFailure
	}
	log.Info().Msg("API server exited")
	return nil
}
