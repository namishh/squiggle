package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := LoadConfig()
	if err != nil {
		logger.Error("[STARTUP]: invalid configuration", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv, err := NewServer(ctx, cfg, logger)
	if err != nil {
		logger.Error("[STARTUP]: failed to initialize server", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := srv.Close(); err != nil {
			logger.Error("[SHUTDOWN]: cleanup error", "err", err)
		}
	}()

	e := echo.New()
	srv.registerRoutes(e)

	sc := echo.StartConfig{
		Address:         ":" + cfg.Port,
		GracefulTimeout: 10 * time.Second,
	}
	if err := sc.Start(ctx, e); err != nil {
		logger.Error("[STARTUP]: server error", "err", err)
	}
}
