package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

const (
	connectTimeout    = 10 * time.Second
	DefaultMaxConns   = 10
	minConns          = 1
	maxConnLifetime   = time.Hour
	maxConnIdleTime   = 30 * time.Minute
	healthCheckPeriod = time.Minute
	ConnectWindow     = 60 * time.Second
	retryPause        = 3 * time.Second
)

func Open(ctx context.Context, databaseURL string, maxConns int) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: DATABASE_URL could not be parsed: %w", err)
	}
	if maxConns < minConns {
		maxConns = DefaultMaxConns
	}
	poolConfig.MaxConns = int32(maxConns)
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = maxConnLifetime
	poolConfig.MaxConnIdleTime = maxConnIdleTime
	poolConfig.HealthCheckPeriod = healthCheckPeriod
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("db: could not create the connection pool: %w", err)
	}
	pingContext, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := pool.Ping(pingContext); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: could not connect to database %q: %w", poolConfig.ConnConfig.Database, err)
	}
	return pool, nil
}

func Connect(ctx context.Context, databaseURL string, maxConns int, within time.Duration) (*pgxpool.Pool, error) {
	if within <= 0 {
		within = ConnectWindow
	}
	deadline := time.Now().Add(within)
	attempt := 0
	for {
		attempt++
		pool, err := Open(ctx, databaseURL, maxConns)
		if err == nil {
			if attempt > 1 {
				log.Printf("db: connected after %d attempts", attempt)
			}
			return pool, nil
		}
		if ctx.Err() != nil || !time.Now().Add(retryPause).Before(deadline) {
			return nil, err
		}
		log.Printf("db: the database is not ready yet, trying again in %s", retryPause)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryPause):
		}
	}
}
