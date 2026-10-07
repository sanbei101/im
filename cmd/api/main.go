package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cloudwego/kitex/server"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/api"
	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/logger"
	"github.com/sanbei101/im/proto/pb/gatewayservice"
)

func main() {
	logger.InitLogger()
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("api server failed")
	}
}

func run() error {
	cfg := config.New()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	data, err := store.Open(cfg.Store.Path)
	if err != nil {
		return fmt.Errorf("open pebble store: %w", err)
	}
	defer data.Close()
	r := api.NewRouter(data)
	streamHandler := api.NewStreamHandler(data, cfg.API.NodeID, cfg.Shard.Slots, cfg.API.NodeIndex, cfg.API.NodeCount)
	listenAddr, err := net.ResolveTCPAddr("tcp", cfg.API.Addr)
	if err != nil {
		return fmt.Errorf("resolve api stream address: %w", err)
	}
	kitexServer := gatewayservice.NewServer(streamHandler, server.WithServiceAddr(listenAddr))

	srv := &http.Server{
		Addr:    ":8801",
		Handler: r,
	}

	go func() {
		if err := http.ListenAndServe(":6061", nil); err != nil {
			log.Error().Err(err).Msg("pprof server stopped")
		}
	}()
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.Store.BackupInterval) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				dir := filepath.Join(cfg.Store.BackupPath, now.Format("20060102-150405"))
				if err := data.Checkpoint(ctx, dir); err != nil {
					log.Error().Err(err).Str("dir", dir).Msg("pebble checkpoint failed")
				}
			}
		}
	}()

	go func() {
		log.Info().Msg("starting API server on :8801")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cancel()
			log.Fatal().Err(err).Msg("failed to start API server")
		}
		log.Info().Msg("API server stopped")
	}()
	go func() {
		log.Info().Str("addr", cfg.API.Addr).Msg("starting api stream server")
		if err := kitexServer.Run(); err != nil {
			log.Error().Err(err).Msg("api stream server stopped")
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("API server forced to shutdown")
	}
	if err := kitexServer.Stop(); err != nil {
		log.Error().Err(err).Msg("api stream server shutdown failed")
	}
	log.Info().Msg("API server exited")
	return nil
}
