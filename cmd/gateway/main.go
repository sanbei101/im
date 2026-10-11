package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/gateway"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/logger"
)

func main() {
	logger.InitLogger()
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("gateway server failed")
	}
}

func run() error {
	cfg := config.New()
	g := gateway.NewGateway(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	gatewayMux := http.NewServeMux()
	gatewayMux.HandleFunc("/ws", g.HandleUserMessage)

	srv := &http.Server{
		Addr:    cfg.Gateway.Addr,
		Handler: gatewayMux,
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := http.ListenAndServe(":6062", nil); err != nil {
			log.Error().Err(err).Msg("pprof server stopped")
		}
	})

	g.Start(ctx)

	serveErr := make(chan error, 1)
	wg.Go(func() {
		log.Info().Msg("starting gateway server on :8800...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("start gateway server: %w", err)
			return
		}
		log.Info().Msg("gateway server stopped")
	})

	var serveFailure error
	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down gracefully...")
	case err := <-serveErr:
		serveFailure = err
		log.Error().Err(err).Msg("gateway server failed, shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}
	wg.Wait()
	if serveFailure != nil {
		return serveFailure
	}
	log.Info().Msg("gateway exited completely")
	return nil
}
