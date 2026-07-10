package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

const adminChannel = "admin:events"

func (s *Server) publish(ctx context.Context, eventType string, payload map[string]any) {

	payload["type"] = eventType
	body, err := json.Marshal(payload)

	if err != nil {
		s.logger.Warn("[ADMIN SSE] marshal failed", "err", err)
		return
	}
	if err := s.redis.Publish(ctx, adminChannel, body).Err(); err != nil {
		s.logger.Warn("[ADMIN SSE] publish failed", "err", err)
	}
}

func (s *Server) stream(c *echo.Context) error {
	ctx := c.Request().Context()
	w := c.Response()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	sub := s.redis.Subscribe(ctx, adminChannel)
	defer sub.Close()
	ch := sub.Channel()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			fmt.Fprintf(w, "event: update\ndata: %s\n\n", msg.Payload)
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
		}
	}
}
