package config

import (
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

type Config struct {
	Port     string
	AppEnv   string
	Timezone string
	Location *time.Location
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:     getEnv("PORT", defaultPort),
		AppEnv:   getEnv("APP_ENV", defaultAppEnv),
		Timezone: getEnv("APP_TIMEZONE", defaultTimezone),
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
