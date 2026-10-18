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
	"github.com/sanbei101/im/pkg/jwt"
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
	if err := jwt.Init(cfg.JWTSecret); err != nil {
		return fmt.Errorf("init jwt: %w", err)
	}
	g := gateway.NewGateway(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	gatewayMux := http.NewServeMux()
	gatewayMux.HandleFunc("/ws", g.HandleUserMessage)

	srv := &http.Server{
		Addr:              cfg.Gateway.Addr,
		Handler:           gatewayMux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       time.Minute,
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		debug := &http.Server{Addr: "127.0.0.1:6062", ReadHeaderTimeout: 10 * time.Second}
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

	g.Start(ctx)

	serveErr := make(chan error, 1)
	wg.Go(func() {
		log.Info().Str("addr", cfg.Gateway.Addr).Msg("starting gateway server")
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
		cancel() // the background streams must not outlive a failed server
		log.Error().Err(err).Msg("gateway server failed, shutting down")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}
	g.Stop(shutdownCtx)

	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-shutdownCtx.Done():
		log.Error().Msg("gateway shutdown timed out, giving up")
		return errors.New("gateway shutdown timed out")
	}
	if serveFailure != nil {
		return serveFailure
	}
	log.Info().Msg("gateway exited completely")
	return nil
}
