package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v5"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
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

	go runSentimentTest(context.Background())
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
	e.POST("/entry", handlePost)

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

func getSentimentScore(c context.Context, text string) (float64, error) {
	res, err := op.Chat.Send(c, components.ChatRequest{
		Model: openrouter.String("openai/gpt-oss-120b"),
		Messages: []components.ChatMessages{
			components.CreateChatMessagesUser(
				components.ChatUserMessage{
					Content: components.CreateChatUserMessageContentStr(
						"Return only a number from -1 (very negative) to 1 (very positive) for sentiment of a guesbook of a personal website. Little bit of criticism, sarcasm, consutructive criticism is allowed, do not be harsh on them: " + text,
					),
					Role: components.ChatUserMessageRoleUser,
				},
			),
		},
	}, nil)

	if err != nil {
		return 0, err
	}

	if len(res.ChatResult.Choices) == 0 {
		return 0, fmt.Errorf("No response from the model")
	}

	content := res.ChatResult.Choices[0].Message.Content
	if content.IsNull() {
		return 0, fmt.Errorf("No response from the model")
	}

	contentVal, ok := content.Get()
	if !ok || contentVal.Str == nil {
		return 0, fmt.Errorf("empty content in response")
	}
	var score float64
	_, err = fmt.Sscanf(strings.TrimSpace(*contentVal.Str), "%f", &score)
	return score, err
}

var testEntries = []string{
	"This site is amazing, love the design!",
	"The font is a bit hard to read on mobile.",
	"This is garbage, whoever made this is an idiot.",
	"Pretty decent overall, nice work.",
}

func runSentimentTest(ctx context.Context) {
	var wg sync.WaitGroup
	results := make([]float64, len(testEntries))
	errs := make([]error, len(testEntries))

	for i, entry := range testEntries {
		wg.Add(1)
		go func(i int, text string) {
			defer wg.Done()
			score, err := getSentimentScore(ctx, text)
			results[i] = score
			errs[i] = err
		}(i, entry)
	}

	wg.Wait()

	for i, entry := range testEntries {
		if errs[i] != nil {
			fmt.Printf("Entry: %q -> ERROR: %v\n", entry, errs[i])
			continue
		}
		fmt.Printf("Entry: %q -> Score: %.2f\n", entry, results[i])
	}
}

func handlePost(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "received a post request!"})
}
