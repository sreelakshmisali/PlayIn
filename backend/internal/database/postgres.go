// Package database owns the PostgreSQL connection pool.
//
// It exposes a *pgxpool.Pool and nothing else. Query construction and row
// mapping belong to repositories in their own feature packages, not here.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orgmelethil/playhub/backend/internal/config"
)

// Pool is the application-wide PostgreSQL connection pool.
type Pool = pgxpool.Pool

// Connect opens a connection pool and verifies it with a ping.
// The caller owns the returned pool and must call Close on shutdown.
func Connect(ctx context.Context, cfg config.Database, logger *slog.Logger) (*Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse database dsn: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = cfg.ConnMaxLifetime / 2

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	logger.Info("database connected",
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("database", cfg.Name),
		slog.Int("max_conns", cfg.MaxOpenConns),
	)

	return pool, nil
}

// ConnectWithRetry calls Connect until it succeeds or ctx is done.
// PostgreSQL is often still starting when the API container comes up, so the
// process retries instead of crash-looping through the container runtime.
func ConnectWithRetry(ctx context.Context, cfg config.Database, logger *slog.Logger, attempts int, backoff time.Duration) (*Pool, error) {
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		pool, err := Connect(ctx, cfg, logger)
		if err == nil {
			return pool, nil
		}
		lastErr = err

		if attempt == attempts {
			break
		}

		logger.Warn("database connection failed, retrying",
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", attempts),
			slog.Duration("backoff", backoff),
			slog.String("error", err.Error()),
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return nil, fmt.Errorf("database unreachable after %d attempts: %w", attempts, lastErr)
}
