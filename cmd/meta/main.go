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

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/api"
	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/logger"
)

const (
	httpReadHeaderTimeout = 10 * time.Second
	httpIdleTimeout       = time.Minute
	shutdownTimeout       = 5 * time.Second
)

func main() {
	logger.InitLogger()
	if err := run(config.NewMeta()); err != nil {
		log.Fatal().Err(err).Msg("meta server failed")
	}
}

func run(cfg *config.Config) error {
	if err := cfg.ValidateMeta(); err != nil {
		return fmt.Errorf("validate meta config: %w", err)
	}
	if err := jwt.Init(cfg.JWTSecret); err != nil {
		return fmt.Errorf("init jwt: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	data, err := store.Open(cfg.Meta.StorePath)
	if err != nil {
		return fmt.Errorf("open meta store: %w", err)
	}
	defer data.Close()

	handler := api.NewMetaRouter(data, cfg.JWTSecret)
	httpListener, err := net.Listen("tcp", cfg.Meta.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen meta http: %w", err)
	}
	pprofListener, err := net.Listen("tcp", cfg.Meta.PProfAddr)
	if err != nil {
		return errors.Join(fmt.Errorf("listen meta pprof: %w", err), httpListener.Close())
	}

	httpServer := &http.Server{
		Addr:              cfg.Meta.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		IdleTimeout:       httpIdleTimeout,
	}
	pprofServer := &http.Server{
		Addr:              cfg.Meta.PProfAddr,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}

	var tasks sync.WaitGroup
	tasks.Go(func() {
		every(ctx, time.Duration(cfg.Meta.BackupInterval)*time.Second, func(ctx context.Context, now time.Time) {
			dir := filepath.Join(cfg.Meta.BackupPath, now.Format("20060102-150405"))
			if err := data.Checkpoint(ctx, dir); err != nil {
				log.Error().Err(err).Str("dir", dir).Msg("meta checkpoint failed")
			}
		})
	})

	serveErr := make(chan error, 2)
	var servers sync.WaitGroup
	servers.Go(func() {
		log.Info().Str("addr", httpListener.Addr().String()).Msg("starting meta server")
		if err := httpServer.Serve(httpListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("meta http server: %w", err)
		}
	})
	servers.Go(func() {
		if err := pprofServer.Serve(pprofListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("meta pprof server: %w", err)
		}
	})

	var serveFailure error
	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down meta gracefully...")
	case serveFailure = <-serveErr:
		log.Error().Err(serveFailure).Msg("meta server failed, shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("meta http shutdown failed")
	}
	if err := pprofServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("meta pprof shutdown failed")
	}
	servers.Wait()
	tasks.Wait()
	if serveFailure != nil {
		return serveFailure
	}
	log.Info().Msg("meta server exited")
	return nil
}

// every runs fn on each tick until ctx is cancelled.
func every(ctx context.Context, interval time.Duration, fn func(context.Context, time.Time)) {
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
