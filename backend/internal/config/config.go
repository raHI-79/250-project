package config

import (
	"errors"
	"fmt"
	"os"
	"time"
	_ "time/tzdata"
)

const (
	defaultPort     = "8080"
	defaultAppEnv   = "development"
	defaultTimezone = "Asia/Dhaka"
)

var ErrMissingDatabaseURL = errors.New("config: DATABASE_URL is required (copy backend/.env.example to backend/.env and set your database password)")

type Config struct {
	Port            string
	AppEnv          string
	Timezone        string
	Location        *time.Location
	DatabaseURL     string
	TestDatabaseURL string
}

func Load() (*Config, error) {
	if err := LoadEnvFile(envFileName); err != nil {
		return nil, err
	}
	cfg := &Config{
		Port:            getEnv("PORT", defaultPort),
		AppEnv:          getEnv("APP_ENV", defaultAppEnv),
		Timezone:        getEnv("APP_TIMEZONE", defaultTimezone),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		TestDatabaseURL: os.Getenv("TEST_DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return nil, ErrMissingDatabaseURL
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("config: invalid APP_TIMEZONE %q: %w", cfg.Timezone, err)
	}
	cfg.Location = loc
	return cfg, nil
}

func (c *Config) Now() time.Time {
	return time.Now().In(c.Location)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
