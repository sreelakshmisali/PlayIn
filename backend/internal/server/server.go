package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/config"
)

// Server owns the HTTP listener and its lifecycle.
type Server struct {
	http   *http.Server
	logger *slog.Logger
	cfg    config.HTTP
}

// New wires an HTTP server with the configured timeouts. Timeouts are set
// explicitly because net/http defaults to none, which leaves the process open
// to slow-client resource exhaustion.
func New(cfg config.HTTP, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		http: &http.Server{
			Addr:              cfg.Addr(),
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
		logger: logger,
		cfg:    cfg,
	}
}

// Run serves until ctx is cancelled, then drains in-flight requests within the
// configured shutdown timeout.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("http server listening", slog.String("addr", s.cfg.Addr()))
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	s.logger.Info("http server shutting down", slog.Duration("timeout", s.cfg.ShutdownTimeout))

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	s.logger.Info("http server stopped")
	return nil
}
