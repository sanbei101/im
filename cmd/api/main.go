package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/phuslu/log"
	"github.com/sanbei101/im/internal/api"
	"github.com/sanbei101/im/internal/api/handler"
	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger.InitLogger()
	cfg := config.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to ping postgres")
	}
	log.Info().Msg("connected to postgres")

	q := db.New(pool)
	userSvc := service.NewUserService(q)
	userHandler := handler.NewUserHandler(userSvc)

	r := api.SetupRouter(userHandler)

	srv := &http.Server{
		Addr:    ":8801",
		Handler: r,
	}

	go func() {
		log.Info().Msg("starting API server on :8801")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}
	log.Info().Msg("API server exited")
}
