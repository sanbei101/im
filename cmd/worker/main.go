package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/sanbei101/im/internal/worker"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/logger"
)

func main() {
	logger.InitLogger()
	cfg := config.New()
	svc := worker.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc.Run(ctx)
}
