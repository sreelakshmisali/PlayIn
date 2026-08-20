// Command api is the PlayHub HTTP API entry point.
//
// This file is the composition root: it is the only place that constructs
// concrete dependencies and wires them together. Everything below it receives
// what it needs as an argument.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/orgmelethil/playhub/backend/internal/admin"
	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/config"
	"github.com/orgmelethil/playhub/backend/internal/database"
	"github.com/orgmelethil/playhub/backend/internal/health"
	"github.com/orgmelethil/playhub/backend/internal/logging"
	"github.com/orgmelethil/playhub/backend/internal/owners"
	"github.com/orgmelethil/playhub/backend/internal/players"
	"github.com/orgmelethil/playhub/backend/internal/server"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "dev"

const (
	dbConnectAttempts = 10
	dbConnectBackoff  = 2 * time.Second
)

func main() {
	if err := run(); err != nil {
		// The logger may not exist yet, so failures here go straight to stderr.
		fmt.Fprintf(os.Stderr, "startup failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.App.Env, cfg.App.LogLevel)
	slog.SetDefault(logger)

	// env is already a base attribute on the logger, so it is not repeated here.
	logger.Info("starting playhub api", slog.String("version", version))

	// ctx is cancelled on SIGINT or SIGTERM, which starts a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.ConnectWithRetry(ctx, cfg.Database, logger, dbConnectAttempts, dbConnectBackoff)
	if err != nil {
		return err
	}
	defer pool.Close()

	healthHandler := health.NewHandler(health.NewService(pool, version, cfg.App.Env))

	// The auth service is shared: the auth handler exposes it, and the player
	// handler uses it to authenticate the requests it guards.
	authService := auth.NewService(
		auth.NewRepository(pool),
		auth.NewBcryptHasher(cfg.Auth.BcryptCost),
		auth.NewIssuer(cfg.Auth),
	)
	authHandler := auth.NewHandler(authService, logger)

	playersHandler := players.NewHandler(
		players.NewService(players.NewRepository(pool)),
		authService,
		logger,
	)

	// ownersService is also shared: the owners handler exposes it to owners,
	// and the admin handler calls its moderation methods directly rather than
	// duplicating turf persistence.
	ownersService := owners.NewService(owners.NewRepository(pool))
	ownersHandler := owners.NewHandler(ownersService, authService, logger)

	adminHandler := admin.NewHandler(ownersService, authService, authService, logger)

	router := server.NewRouter(server.Dependencies{
		Config:  cfg,
		Logger:  logger,
		Health:  healthHandler,
		Auth:    authHandler,
		Players: playersHandler,
		Owners:  ownersHandler,
		Admin:   adminHandler,
	})

	if err := server.New(cfg.HTTP, router, logger).Run(ctx); err != nil {
		return err
	}

	logger.Info("shutdown complete")
	return nil
}
