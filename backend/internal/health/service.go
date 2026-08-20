// Package health reports whether the API and its dependencies are usable.
//
// It is the reference shape for every feature package added later: a Service
// holding the logic, a Handler holding only HTTP concerns, and dependencies
// injected as narrow interfaces.
package health

import (
	"context"
	"time"
)

// Status is the outcome of a single check or of the service as a whole.
type Status string

const (
	// StatusOK means the component is fully usable.
	StatusOK Status = "ok"
	// StatusDegraded means a dependency is unreachable.
	StatusDegraded Status = "degraded"
)

// Pinger is the database behaviour health needs. *pgxpool.Pool satisfies it.
// Depending on the interface rather than the pool keeps this package testable
// without a live PostgreSQL instance.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Check is the result of probing one dependency.
type Check struct {
	Status    Status `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// Report is the aggregate health of the service.
type Report struct {
	Status    Status           `json:"status"`
	Service   string           `json:"service"`
	Version   string           `json:"version"`
	Env       string           `json:"env"`
	Timestamp time.Time        `json:"timestamp"`
	Checks    map[string]Check `json:"checks"`
}

// Healthy reports whether every dependency responded.
func (r Report) Healthy() bool { return r.Status == StatusOK }

// Service produces health reports.
type Service struct {
	db      Pinger
	version string
	env     string
	timeout time.Duration
	now     func() time.Time
}

// NewService wires a Service. The probe timeout is deliberately short: a health
// endpoint that hangs is worse than one that reports a failure.
func NewService(db Pinger, version, env string) *Service {
	return &Service{
		db:      db,
		version: version,
		env:     env,
		timeout: 2 * time.Second,
		now:     time.Now,
	}
}

// Check probes every dependency and returns the aggregate report.
func (s *Service) Check(ctx context.Context) Report {
	report := Report{
		Status:    StatusOK,
		Service:   "playhub-api",
		Version:   s.version,
		Env:       s.env,
		Timestamp: s.now().UTC(),
		Checks:    make(map[string]Check, 1),
	}

	dbCheck := s.checkDatabase(ctx)
	report.Checks["database"] = dbCheck

	if dbCheck.Status != StatusOK {
		report.Status = StatusDegraded
	}

	return report
}

func (s *Service) checkDatabase(ctx context.Context) Check {
	if s.db == nil {
		return Check{Status: StatusDegraded, Error: "database is not configured"}
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	start := s.now()
	err := s.db.Ping(ctx)
	latency := s.now().Sub(start).Milliseconds()

	if err != nil {
		return Check{Status: StatusDegraded, LatencyMS: latency, Error: err.Error()}
	}
	return Check{Status: StatusOK, LatencyMS: latency}
}
