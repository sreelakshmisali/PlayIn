package config

import (
	"strings"
	"testing"
	"time"
)

// setRequired sets the variables that have no default.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DB_USER", "playhub")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "playhub")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
}

func TestLoadAppliesDefaults(t *testing.T) {
	setRequired(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.App.Env != "development" {
		t.Errorf("App.Env = %q, want development", cfg.App.Env)
	}
	if cfg.HTTP.Port != 8080 {
		t.Errorf("HTTP.Port = %d, want 8080", cfg.HTTP.Port)
	}
	if cfg.HTTP.Addr() != "0.0.0.0:8080" {
		t.Errorf("HTTP.Addr() = %q, want 0.0.0.0:8080", cfg.HTTP.Addr())
	}
	if cfg.Database.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 30m", cfg.Database.ConnMaxLifetime)
	}
	if got := cfg.HTTP.AllowedOrigins; len(got) != 1 || got[0] != "http://localhost:5173" {
		t.Errorf("AllowedOrigins = %v, want [http://localhost:5173]", got)
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() = true, want false")
	}
	if cfg.Auth.JWTIssuer != "playhub" {
		t.Errorf("Auth.JWTIssuer = %q, want playhub", cfg.Auth.JWTIssuer)
	}
	if cfg.Auth.AccessTTL != 15*time.Minute {
		t.Errorf("Auth.AccessTTL = %v, want 15m", cfg.Auth.AccessTTL)
	}
	if cfg.Auth.RefreshTTL != 720*time.Hour {
		t.Errorf("Auth.RefreshTTL = %v, want 720h", cfg.Auth.RefreshTTL)
	}
	if cfg.Auth.BcryptCost != 12 {
		t.Errorf("Auth.BcryptCost = %d, want 12", cfg.Auth.BcryptCost)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	setRequired(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("HTTP_READ_TIMEOUT", "3s")
	t.Setenv("HTTP_ALLOWED_ORIGINS", "https://playhub.app, https://admin.playhub.app ,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if !cfg.IsProduction() {
		t.Error("IsProduction() = false, want true")
	}
	if cfg.HTTP.Port != 9090 {
		t.Errorf("HTTP.Port = %d, want 9090", cfg.HTTP.Port)
	}
	if cfg.HTTP.ReadTimeout != 3*time.Second {
		t.Errorf("ReadTimeout = %v, want 3s", cfg.HTTP.ReadTimeout)
	}
	if len(cfg.HTTP.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins = %v, want 2 entries", cfg.HTTP.AllowedOrigins)
	}
	if cfg.HTTP.AllowedOrigins[1] != "https://admin.playhub.app" {
		t.Errorf("AllowedOrigins[1] = %q, want trimmed value", cfg.HTTP.AllowedOrigins[1])
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantSub string
	}{
		{"missing credentials", map[string]string{}, "DB_USER is required"},
		{"bad env", map[string]string{"APP_ENV": "staging-2"}, "APP_ENV must be"},
		{"bad log level", map[string]string{"LOG_LEVEL": "verbose"}, "LOG_LEVEL must be"},
		{"non numeric port", map[string]string{"HTTP_PORT": "eighty"}, "HTTP_PORT must be an integer"},
		{"port out of range", map[string]string{"HTTP_PORT": "70000"}, "HTTP_PORT must be between"},
		{"bad duration", map[string]string{"HTTP_READ_TIMEOUT": "10 seconds"}, "must be a Go duration"},
		{"bad sslmode", map[string]string{"DB_SSLMODE": "maybe"}, "not a valid libpq sslmode"},
		{"idle exceeds open", map[string]string{"DB_MAX_OPEN_CONNS": "2", "DB_MAX_IDLE_CONNS": "5"}, "must not exceed"},
		{"missing jwt secret", map[string]string{"JWT_SECRET": ""}, "JWT_SECRET is required"},
		{"short jwt secret", map[string]string{"JWT_SECRET": "too-short"}, "at least 32 characters"},
		{"refresh shorter than access", map[string]string{
			"JWT_ACCESS_TTL": "24h", "JWT_REFRESH_TTL": "1h",
		}, "JWT_REFRESH_TTL must be longer"},
		{"bcrypt cost too low", map[string]string{"BCRYPT_COST": "4"}, "BCRYPT_COST must be between"},
		{"bcrypt cost too high", map[string]string{"BCRYPT_COST": "32"}, "BCRYPT_COST must be between"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "missing credentials" {
				setRequired(t)
			} else {
				t.Setenv("DB_USER", "")
				t.Setenv("DB_PASSWORD", "")
				t.Setenv("DB_NAME", "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			_, err := Load()
			if err == nil {
				t.Fatal("Load() returned nil error, want failure")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestDatabaseDSN(t *testing.T) {
	db := Database{
		Host: "db", Port: 5432,
		User: "playhub", Password: "p@ss word",
		Name: "playhub", SSLMode: "disable",
		ConnectTimeout: 5 * time.Second,
	}

	dsn := db.DSN()

	for _, want := range []string{
		"postgres://",
		"playhub:p%40ss%20word@db:5432/playhub",
		"sslmode=disable",
		"connect_timeout=5",
	} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN() = %q, want it to contain %q", dsn, want)
		}
	}
}
