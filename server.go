package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type Server struct {
	cfg     Config
	db      *bun.DB
	redis   *redis.Client
	limiter *redis_rate.Limiter
	ai      *openrouter.OpenRouter
	logger  *slog.Logger
}

func NewServer(ctx context.Context, cfg Config, logger *slog.Logger) (*Server, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DatabaseURL)))
	db := bun.NewDB(sqldb, pgdialect.New())

	rc := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	if err := rc.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	ai := openrouter.New(openrouter.WithSecurity(cfg.OpenrouterKey))

	return &Server{
		cfg:     cfg,
		db:      db,
		redis:   rc,
		limiter: redis_rate.NewLimiter(rc),
		ai:      ai,
		logger:  logger,
	}, nil
}

func (s *Server) Close() error {
	return errors.Join(
		s.redis.Close(),
		s.db.Close(),
	)
}
