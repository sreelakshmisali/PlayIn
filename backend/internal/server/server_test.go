package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/orgmelethil/playhub/backend/internal/config"
)

// freePort asks the kernel for an unused port so parallel test runs do not
// collide on a hard-coded one.
func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func testHTTPConfig(t *testing.T) config.HTTP {
	t.Helper()

	return config.HTTP{
		Host:            "127.0.0.1",
		Port:            freePort(t),
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: 2 * time.Second,
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServerServesAndShutsDownGracefully(t *testing.T) {
	cfg := testHTTPConfig(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := New(cfg, handler, discardLogger())

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()

	waitForListener(t, cfg.Addr())

	resp, err := http.Get("http://" + cfg.Addr() + "/ping")
	if err != nil {
		t.Fatalf("request to running server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Run() returned %v, want nil after graceful shutdown", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

func TestServerReportsListenFailure(t *testing.T) {
	cfg := testHTTPConfig(t)

	// Hold the port so the server cannot bind it.
	blocker, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer blocker.Close()

	err = New(cfg, http.NotFoundHandler(), discardLogger()).Run(context.Background())
	if err == nil {
		t.Fatal("Run() returned nil, want a bind error")
	}
}

func waitForListener(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server never started listening on %s", addr)
}
