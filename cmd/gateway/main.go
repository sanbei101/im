package main

import (
	"context"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/phuslu/log"
	"github.com/sanbei101/im/internal/gateway"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/logger"
)

var wg sync.WaitGroup

func main() {
	logger.InitLogger()
	config := config.New()
	g := gateway.New(config)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    ":8800",
		Handler: nil,
	}
	http.HandleFunc("/ws", g.HandleUserMessage)

	wg.Go(func() {
		log.Info().Msg("starting handle worker messages...")
		g.HandleWorkerMessages(ctx)
		log.Info().Msg("handle worker messages stopped")
	})

	wg.Go(func() {
		log.Info().Msg("starting http server on :8800...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start HTTP server")
		}
		log.Info().Msg("http server stopped")
	})

	<-ctx.Done()
	log.Info().Msg("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}
	wg.Wait()
	log.Info().Msg("gateway exited completely")
}
