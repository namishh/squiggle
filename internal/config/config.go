package config

import (
	"fmt"
	"os"
)

type Config struct {
	Environment string
	Port        string

	DatabaseURL string
	RedisURL    string

	OpenrouterKey   string
	OpenrouterModel string

	AllowedOrigin string

	TurnstileSecret  string
	TurnstileDemoKey string

	IPSalt string

	SpamThreshold int
	HideThreshold int
}

func getEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}

func LoadConfig() (Config, error) {
	cfg := Config{
		Environment:      getEnvOr("ENVIRONTMENT", "dev"),
		Port:             getEnvOr("ADDR", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RedisURL:         os.Getenv("REDIS_URL"),
		OpenrouterKey:    os.Getenv("OPENROUTER_KEY"),
		OpenrouterModel:  getEnvOr("OPENROUTER_MODEL", "openai/gpt-oss-20b"),
		AllowedOrigin:    os.Getenv("ALLOWED_ORIGIN"),
		TurnstileSecret:  os.Getenv("TURNSTILE_SECRET"),
		TurnstileDemoKey: os.Getenv("TURNSTILE_DEMO"),
		IPSalt:           os.Getenv("IP_SALT"),
		SpamThreshold:    3,
		HideThreshold:    6,
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.RedisURL == "" {
		return cfg, fmt.Errorf("REDIS_URL is required")
	}

	if cfg.OpenrouterKey == "" {
		return cfg, fmt.Errorf("OPENROUTER_KEY is required")
	}

	if cfg.IPSalt == "" {
		return cfg, fmt.Errorf("IP_SALT is required")
	}
	if cfg.Environment == "prod" && cfg.AllowedOrigin == "" {
		return cfg, fmt.Errorf("ALLOWED_ORIGIN is required when ENVIRONMENT=prod")
	}
	if cfg.Environment == "prod" && cfg.TurnstileSecret == "" {
		return cfg, fmt.Errorf("TURNSTILE_SECRET is required when ENVIRONMENT=prod")
	}

	return cfg, nil
}

func (c Config) TurnstileSecretKey() string {
	if c.Environment == "dev" {
		return c.TurnstileDemoKey
	}
	return c.TurnstileSecret
}
