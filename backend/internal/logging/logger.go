// Package logging builds the application's structured logger.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a structured logger.
// Production emits JSON for log collectors; other environments emit text,
// which is easier to read in a terminal.
func New(env, level string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(
		slog.String("service", "playhub-api"),
		slog.String("env", env),
	)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
