// Package config loads application configuration from environment variables.
//
// Configuration is read once at startup. Nothing else in the application reads
// os.Getenv directly; every component receives what it needs through
// dependency injection so behaviour stays explicit and testable.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the full application configuration.
type Config struct {
	App      App
	HTTP     HTTP
	Database Database
	Auth     Auth
}

// App holds process-wide settings.
type App struct {
	Env      string // development | staging | production
	LogLevel string // debug | info | warn | error
}

// HTTP holds HTTP server settings.
type HTTP struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

// Addr returns the listen address for the HTTP server.
func (h HTTP) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

// Database holds PostgreSQL connection settings.
type Database struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnectTimeout  time.Duration
}

// DSN builds a PostgreSQL connection string from the individual settings.
func (d Database) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:   d.Name,
	}
	q := url.Values{}
	q.Set("sslmode", d.SSLMode)
	q.Set("connect_timeout", strconv.Itoa(int(d.ConnectTimeout.Seconds())))
	u.RawQuery = q.Encode()
	return u.String()
}

// Auth holds token and password hashing settings.
type Auth struct {
	// JWTSecret signs and verifies access and refresh tokens. It has no
	// default: an application that falls back to a built-in signing key issues
	// forgeable tokens the moment it reaches an environment nobody configured.
	JWTSecret  string
	JWTIssuer  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	// BcryptCost is the work factor. Higher is slower to brute force and slower
	// to log in with; 12 is the current sensible middle.
	BcryptCost int
}

// minJWTSecretLength is 32 bytes, matching the HMAC-SHA256 block output. A
// shorter secret weakens the signature without saving anything worth having.
const minJWTSecretLength = 32

// Load reads configuration from the environment and validates it.
// It reports every problem it finds rather than failing on the first one, so a
// misconfigured deployment can be fixed in a single pass.
func Load() (*Config, error) {
	var problems []error
	collect := func(err error) {
		if err != nil {
			problems = append(problems, err)
		}
	}

	cfg := &Config{}

	cfg.App.Env = envString("APP_ENV", "development")
	cfg.App.LogLevel = envString("LOG_LEVEL", "info")

	cfg.HTTP.Host = envString("HTTP_HOST", "0.0.0.0")
	cfg.HTTP.Port = envInt("HTTP_PORT", 8080, collect)
	cfg.HTTP.ReadTimeout = envDuration("HTTP_READ_TIMEOUT", 10*time.Second, collect)
	cfg.HTTP.WriteTimeout = envDuration("HTTP_WRITE_TIMEOUT", 20*time.Second, collect)
	cfg.HTTP.IdleTimeout = envDuration("HTTP_IDLE_TIMEOUT", 120*time.Second, collect)
	cfg.HTTP.ShutdownTimeout = envDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second, collect)
	cfg.HTTP.AllowedOrigins = envStringSlice("HTTP_ALLOWED_ORIGINS", []string{"http://localhost:5173"})

	cfg.Database.Host = envString("DB_HOST", "localhost")
	cfg.Database.Port = envInt("DB_PORT", 5432, collect)
	cfg.Database.User = envString("DB_USER", "")
	cfg.Database.Password = envString("DB_PASSWORD", "")
	cfg.Database.Name = envString("DB_NAME", "")
	cfg.Database.SSLMode = envString("DB_SSLMODE", "disable")
	cfg.Database.MaxOpenConns = envInt("DB_MAX_OPEN_CONNS", 25, collect)
	cfg.Database.MaxIdleConns = envInt("DB_MAX_IDLE_CONNS", 5, collect)
	cfg.Database.ConnMaxLifetime = envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute, collect)
	cfg.Database.ConnectTimeout = envDuration("DB_CONNECT_TIMEOUT", 5*time.Second, collect)

	cfg.Auth.JWTSecret = envString("JWT_SECRET", "")
	cfg.Auth.JWTIssuer = envString("JWT_ISSUER", "playhub")
	cfg.Auth.AccessTTL = envDuration("JWT_ACCESS_TTL", 15*time.Minute, collect)
	cfg.Auth.RefreshTTL = envDuration("JWT_REFRESH_TTL", 720*time.Hour, collect)
	cfg.Auth.BcryptCost = envInt("BCRYPT_COST", 12, collect)

	collect(cfg.validate())

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid configuration: %w", errors.Join(problems...))
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var problems []error

	if !oneOf(c.App.Env, "development", "staging", "production") {
		problems = append(problems, fmt.Errorf("APP_ENV must be development, staging or production, got %q", c.App.Env))
	}
	if !oneOf(c.App.LogLevel, "debug", "info", "warn", "error") {
		problems = append(problems, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, got %q", c.App.LogLevel))
	}
	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		problems = append(problems, fmt.Errorf("HTTP_PORT must be between 1 and 65535, got %d", c.HTTP.Port))
	}
	if c.Database.User == "" {
		problems = append(problems, errors.New("DB_USER is required"))
	}
	if c.Database.Password == "" {
		problems = append(problems, errors.New("DB_PASSWORD is required"))
	}
	if c.Database.Name == "" {
		problems = append(problems, errors.New("DB_NAME is required"))
	}
	if !oneOf(c.Database.SSLMode, "disable", "allow", "prefer", "require", "verify-ca", "verify-full") {
		problems = append(problems, fmt.Errorf("DB_SSLMODE %q is not a valid libpq sslmode", c.Database.SSLMode))
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		problems = append(problems, errors.New("DB_MAX_IDLE_CONNS must not exceed DB_MAX_OPEN_CONNS"))
	}
	if len(c.Auth.JWTSecret) < minJWTSecretLength {
		problems = append(problems, fmt.Errorf("JWT_SECRET is required and must be at least %d characters", minJWTSecretLength))
	}
	if c.Auth.JWTIssuer == "" {
		problems = append(problems, errors.New("JWT_ISSUER must not be empty"))
	}
	if c.Auth.AccessTTL <= 0 {
		problems = append(problems, fmt.Errorf("JWT_ACCESS_TTL must be positive, got %v", c.Auth.AccessTTL))
	}
	if c.Auth.RefreshTTL <= c.Auth.AccessTTL {
		problems = append(problems, errors.New("JWT_REFRESH_TTL must be longer than JWT_ACCESS_TTL"))
	}
	// bcrypt rejects anything outside 4..31 itself; catching it here turns a
	// per-request failure into a refusal to start.
	if c.Auth.BcryptCost < 10 || c.Auth.BcryptCost > 31 {
		problems = append(problems, fmt.Errorf("BCRYPT_COST must be between 10 and 31, got %d", c.Auth.BcryptCost))
	}

	return errors.Join(problems...)
}

// IsProduction reports whether the process runs in the production environment.
func (c *Config) IsProduction() bool { return c.App.Env == "production" }

func envString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func envStringSlice(key string, fallback []string) []string {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func envInt(key string, fallback int, collect func(error)) int {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		collect(fmt.Errorf("%s must be an integer, got %q", key, raw))
		return fallback
	}
	return v
}

func envDuration(key string, fallback time.Duration, collect func(error)) time.Duration {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		collect(fmt.Errorf("%s must be a Go duration such as 10s or 2m, got %q", key, raw))
		return fallback
	}
	return v
}

func oneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}
