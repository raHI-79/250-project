package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
	_ "time/tzdata"
)

const (
	defaultPort           = "8080"
	defaultAppEnv         = "development"
	defaultTimezone       = "Asia/Dhaka"
	defaultJWTTTL         = "24h"
	defaultFrontendOrigin = "http://localhost:5173"
	defaultAutoCancel     = "5m"
	defaultUploadDir      = "uploads"
	defaultMaxUploadMB    = 10
	minJWTSecretLength    = 32
)

var (
	ErrMissingDatabaseURL = errors.New("config: DATABASE_URL is required (copy backend/.env.example to backend/.env and set your database password)")
	ErrMissingJWTSecret   = errors.New("config: JWT_SECRET is required (set a long random value in backend/.env)")
	ErrShortJWTSecret     = fmt.Errorf("config: JWT_SECRET must be at least %d characters long", minJWTSecretLength)
)

type Config struct {
	Port            string
	AppEnv          string
	Timezone        string
	Location        *time.Location
	DatabaseURL     string
	TestDatabaseURL string
	JWTSecret       string
	JWTTTL          time.Duration
	FrontendOrigin  string
	AutoCancel      time.Duration
	UploadDir       string
	MaxUploadMB     int
	AdminIdentifier string
	AdminPassword   string
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
		JWTSecret:       os.Getenv("JWT_SECRET"),
		FrontendOrigin:  getEnv("FRONTEND_ORIGIN", defaultFrontendOrigin),
		UploadDir:       getEnv("UPLOAD_DIR", defaultUploadDir),
		AdminIdentifier: os.Getenv("ADMIN_IDENTIFIER"),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
	}
	if cfg.DatabaseURL == "" {
		return nil, ErrMissingDatabaseURL
	}
	if cfg.JWTSecret == "" {
		return nil, ErrMissingJWTSecret
	}
	if len(cfg.JWTSecret) < minJWTSecretLength {
		return nil, ErrShortJWTSecret
	}
	rawTTL := getEnv("JWT_TTL", defaultJWTTTL)
	ttl, err := time.ParseDuration(rawTTL)
	if err != nil {
		return nil, fmt.Errorf("config: invalid JWT_TTL %q: %w", rawTTL, err)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("config: JWT_TTL must be a positive duration, got %q", rawTTL)
	}
	cfg.JWTTTL = ttl
	rawAutoCancel := getEnv("AUTO_CANCEL_INTERVAL", defaultAutoCancel)
	autoCancel, err := time.ParseDuration(rawAutoCancel)
	if err != nil {
		return nil, fmt.Errorf("config: invalid AUTO_CANCEL_INTERVAL %q: %w", rawAutoCancel, err)
	}
	if autoCancel <= 0 {
		return nil, fmt.Errorf("config: AUTO_CANCEL_INTERVAL must be a positive duration, got %q", rawAutoCancel)
	}
	cfg.AutoCancel = autoCancel
	rawMaxUpload := getEnv("MAX_UPLOAD_MB", strconv.Itoa(defaultMaxUploadMB))
	maxUpload, err := strconv.Atoi(rawMaxUpload)
	if err != nil {
		return nil, fmt.Errorf("config: invalid MAX_UPLOAD_MB %q: %w", rawMaxUpload, err)
	}
	if maxUpload <= 0 {
		return nil, fmt.Errorf("config: MAX_UPLOAD_MB must be a positive number of megabytes, got %q", rawMaxUpload)
	}
	cfg.MaxUploadMB = maxUpload
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

func (c *Config) HasAdminSeed() bool {
	return c.AdminIdentifier != "" && c.AdminPassword != ""
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
