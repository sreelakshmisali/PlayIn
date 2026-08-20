package database

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/orgmelethil/playhub/backend/internal/config"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// unreachable points at a port nothing listens on, so Connect fails fast
// without needing a PostgreSQL instance.
func unreachable() config.Database {
	return config.Database{
		Host: "127.0.0.1", Port: 1,
		User: "playhub", Password: "secret", Name: "playhub",
		SSLMode:        "disable",
		MaxOpenConns:   2,
		MaxIdleConns:   0,
		ConnectTimeout: time.Second,
	}
}

func TestConnectRejectsInvalidDSN(t *testing.T) {
	cfg := unreachable()
	cfg.SSLMode = "not-a-mode"

	_, err := Connect(context.Background(), cfg, discardLogger())
	if err == nil {
		t.Fatal("Connect() returned nil error, want a parse failure")
	}
	if !strings.Contains(err.Error(), "parse database dsn") {
		t.Errorf("error = %q, want a dsn parse error", err.Error())
	}
}

func TestConnectFailsWhenServerUnreachable(t *testing.T) {
	_, err := Connect(context.Background(), unreachable(), discardLogger())
	if err == nil {
		t.Fatal("Connect() returned nil error, want a ping failure")
	}
	if !strings.Contains(err.Error(), "ping database") {
		t.Errorf("error = %q, want a ping error", err.Error())
	}
}

func TestConnectWithRetryGivesUpAfterAttempts(t *testing.T) {
	start := time.Now()

	_, err := ConnectWithRetry(context.Background(), unreachable(), discardLogger(), 2, 10*time.Millisecond)
	if err == nil {
		t.Fatal("ConnectWithRetry() returned nil error, want failure")
	}
	if !strings.Contains(err.Error(), "after 2 attempts") {
		t.Errorf("error = %q, want it to report the attempt count", err.Error())
	}
	if elapsed := time.Since(start); elapsed < 10*time.Millisecond {
		t.Errorf("elapsed = %v, want at least one backoff pause", elapsed)
	}
}

func TestConnectWithRetryStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := ConnectWithRetry(ctx, unreachable(), discardLogger(), 10, time.Second)
	if err == nil {
		t.Fatal("ConnectWithRetry() returned nil error, want cancellation")
	}
}
