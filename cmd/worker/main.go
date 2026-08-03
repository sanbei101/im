package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"syscall"

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/mq"
	"github.com/sanbei101/im/internal/worker"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/logger"
)

func main() {
	logger.InitLogger()
	cfg := config.New()
	redisMQ := mq.NewRedisMQ(cfg)
	svc := worker.New(cfg, redisMQ)

	go func() {
		if err := http.ListenAndServe(":6063", nil); err != nil {
			log.Error().Err(err).Msg("pprof server stopped")
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc.Run(ctx)
	logger.Close()
}
