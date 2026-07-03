package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v5"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/labstack/echo/v5/middleware"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var rc *redis.Client
var rl *redis_rate.Limiter
var op *openrouter.OpenRouter
var db *gorm.DB

func main() {
	e := echo.New()
	var dberr error
	db, dberr = gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if dberr != nil {
		log.Fatalln(dberr)
	}

	op = openrouter.New(openrouter.WithSecurity(os.Getenv("OPENROUTER_KEY")))

	rc = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
	if err := rc.Ping(context.Background()).Err(); err != nil {
		e.Logger.Error("redis connection failed", "error", err)
	}
	rl = redis_rate.NewLimiter(rc)
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))
	e.Use(rateLimit)

	e.GET("/", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

// ::MIDDLEWARES
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

// ::HANDLERS
