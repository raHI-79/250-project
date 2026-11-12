package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	connectTimeout    = 10 * time.Second
	DefaultMaxConns   = 10
	minConns          = 1
	maxConnLifetime   = time.Hour
	maxConnIdleTime   = 4 * time.Minute
	IdleCheckAfter    = 30 * time.Second
	idleCheckTimeout  = 3 * time.Second
	healthCheckPeriod = time.Minute
	ConnectWindow     = 60 * time.Second
	retryPause        = 3 * time.Second
)

func GuardIdleConnections(poolConfig *pgxpool.Config, idleFor time.Duration) {
	var released sync.Map
	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		released.Store(conn, time.Now())
		return true
	}
	poolConfig.BeforeClose = func(conn *pgx.Conn) {
		released.Delete(conn)
	}
	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		at, ok := released.Load(conn)
		if !ok || time.Since(at.(time.Time)) < idleFor {
			return true
		}
		check, cancel := context.WithTimeout(ctx, idleCheckTimeout)
		defer cancel()
		if err := conn.Ping(check); err != nil {
			log.Printf("db: dropped a connection that went away while idle: %v", err)
			return false
		}
		return true
	}
}

func Open(ctx context.Context, databaseURL string, maxConns int, timezone string) (*pgxpool.Pool, error) {
	return OpenGuarded(ctx, databaseURL, maxConns, timezone, IdleCheckAfter)
}

func OpenGuarded(ctx context.Context, databaseURL string, maxConns int, timezone string,
	idleFor time.Duration) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: DATABASE_URL could not be parsed: %w", err)
	}
	if zone := strings.TrimSpace(timezone); zone != "" {
		if poolConfig.ConnConfig.RuntimeParams == nil {
			poolConfig.ConnConfig.RuntimeParams = make(map[string]string, 1)
		}
		poolConfig.ConnConfig.RuntimeParams["timezone"] = zone
	}
	if maxConns < minConns {
		maxConns = DefaultMaxConns
	}
	poolConfig.MaxConns = int32(maxConns)
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = maxConnLifetime
	poolConfig.MaxConnIdleTime = maxConnIdleTime
	poolConfig.HealthCheckPeriod = healthCheckPeriod
	if idleFor >= 0 {
		GuardIdleConnections(poolConfig, idleFor)
	}
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

func Connect(ctx context.Context, databaseURL string, maxConns int, timezone string,
	within time.Duration) (*pgxpool.Pool, error) {
	if within <= 0 {
		within = ConnectWindow
	}
	deadline := time.Now().Add(within)
	attempt := 0
	for {
		attempt++
		pool, err := Open(ctx, databaseURL, maxConns, timezone)
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
