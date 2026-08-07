package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/api"
	"github.com/sanbei101/im/internal/api/handler"
	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/internal/cache"
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/internal/pgxuuid"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/logger"
)

func main() {
	logger.InitLogger()
	cfg := config.New()
	if err := jwt.Configure(cfg.Auth.JWTSecret); err != nil {
		log.Fatal().Err(err).Msg("failed to configure JWT")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	pgxCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		cancel()
		log.Fatal().Err(err).Msg("failed to parse postgres config")
	}
	pgxCfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		cancel()
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		cancel()
		log.Fatal().Err(err).Msg("failed to ping postgres")
	}
	log.Info().Msg("connected to postgres")

	query := db.New(pool)
	roomCache := cache.NewRoomStore(cfg)
	defer func() {
		if err := roomCache.Close(); err != nil {
			log.Error().Err(err).Msg("close room cache failed")
		}
	}()
	userSvc := service.NewUserService(query, pool)
	userHandler := handler.NewUserHandler(userSvc)
	messageSvc := service.NewMessageService(query)
	messageHandler := handler.NewMessageHandler(messageSvc)
	roomSvc := service.NewRoomService(query, pool, roomCache)
	roomHandler := handler.NewRoomHandler(roomSvc)
	friendSvc := service.NewFriendService(query, pool)
	friendHandler := handler.NewFriendHandler(friendSvc)

	benchSvc := service.NewBenchMockService(query)
	benchHandler := handler.NewBenchMockHandler(benchSvc)

	r := api.SetupRouter(userHandler, messageHandler, roomHandler, friendHandler, benchHandler, cfg.API.AllowedOrigins)

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
		log.Info().Msg("starting API server on :8801")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			pool.Close()
			cancel()
			log.Fatal().Err(err).Msg("failed to start API server")
		}
		log.Info().Msg("API server stopped")
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("API server forced to shutdown")
	}
	pool.Close()
	logger.Close()
	log.Info().Msg("API server exited")
}
