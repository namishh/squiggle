package main

import (
	"context"
	"net/http"

	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/redis/go-redis/v9"
)

var rc *redis.Client
var rl *redis_rate.Limiter

func main() {
	e := echo.New()
	rc = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	if err := rc.Ping(context.Background()).Err(); err != nil {
		e.Logger.Error("redis connection failed", "error", err)
	}
	rl = redis_rate.NewLimiter(rc)
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(rateLimit)

	e.GET("/", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

func rateLimit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		res, err := rl.Allow(c.Request().Context(), c.RealIP(), redis_rate.PerMinute(10))
		if err != nil {
			return err
		}
		if res.Allowed == 0 {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"message": "rate limit exceeded"})
		}
		return next(c)
	}
}
