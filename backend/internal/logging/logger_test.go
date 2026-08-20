package logging

import (
	"log/slog"
	"testing"
)

func TestNewAttachesServiceContext(t *testing.T) {
	logger := New("development", "info")

	if logger == nil {
		t.Fatal("New() returned nil")
	}
	if !logger.Enabled(t.Context(), slog.LevelInfo) {
		t.Error("info level is disabled, want it enabled")
	}
	if logger.Enabled(t.Context(), slog.LevelDebug) {
		t.Error("debug level is enabled at info threshold, want it disabled")
	}
}

func TestNewHonoursLevel(t *testing.T) {
	tests := []struct {
		level     string
		wantLevel slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"  DEBUG  ", slog.LevelDebug},
		{"nonsense", slog.LevelInfo},
	}

	for _, tc := range tests {
		t.Run(tc.level, func(t *testing.T) {
			if got := parseLevel(tc.level); got != tc.wantLevel {
				t.Errorf("parseLevel(%q) = %v, want %v", tc.level, got, tc.wantLevel)
			}

			logger := New("production", tc.level)
			if !logger.Enabled(t.Context(), tc.wantLevel) {
				t.Errorf("logger for %q does not accept %v", tc.level, tc.wantLevel)
			}
		})
	}
}
