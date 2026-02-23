package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

const (
	// HealthStatusHealthy indicates all dependencies are operational.
	HealthStatusHealthy = "healthy"

	// HealthStatusDegraded indicates some dependencies are failing.
	HealthStatusDegraded = "degraded"

	// HealthStatusUnhealthy indicates all dependencies are failing.
	HealthStatusUnhealthy = "unhealthy"
)

// FetchHealthQuery requests full health status of all dependencies.
type (
	FetchHealthQuery struct{}
)

// FetchLivenessQuery requests liveness status (is the process running).
type (
	FetchLivenessQuery struct{}
)

// FetchReadinessQuery requests readiness status (are all dependencies ready).
type (
	FetchReadinessQuery struct{}
)

// HealthCheck represents the status of a single dependency.
type (
	HealthCheck struct {
		Name    string
		Status  string
		Message string
	}
)

// HealthResponse holds the overall health and individual dependency checks.
type (
	HealthResponse struct {
		Status    string
		Timestamp time.Time
		Database  HealthCheck
		Kafka     HealthCheck
	}
)

// FetchHealthHandler checks all dependencies and returns aggregated health.
type (
	FetchHealthHandler struct {
		dbChecker    ports.HealthChecker
		kafkaChecker ports.HealthChecker
	}
)

// NewFetchHealthHandler creates a handler with health checkers for each dependency.
func NewFetchHealthHandler(dbChecker, kafkaChecker ports.HealthChecker) *FetchHealthHandler {
	return &FetchHealthHandler{
		dbChecker:    dbChecker,
		kafkaChecker: kafkaChecker,
	}
}

// Handle checks database and Kafka health and returns aggregated status.
func (h *FetchHealthHandler) Handle(ctx context.Context, _ FetchHealthQuery) (HealthResponse, error) {
	resp := HealthResponse{
		Timestamp: time.Now().UTC(),
	}

	resp.Database = checkDependency("database", h.dbChecker, ctx)
	resp.Kafka = checkDependency("kafka", h.kafkaChecker, ctx)

	resp.Status = aggregateStatus(resp.Database, resp.Kafka)

	return resp, nil
}

// FetchLivenessHandler always returns healthy — the process is running.
type (
	FetchLivenessHandler struct{}
)

// NewFetchLivenessHandler creates a liveness handler.
func NewFetchLivenessHandler() *FetchLivenessHandler {
	return &FetchLivenessHandler{}
}

// Handle returns a healthy response — if this code runs, the process is alive.
func (h *FetchLivenessHandler) Handle(_ context.Context, _ FetchLivenessQuery) (HealthResponse, error) {
	return HealthResponse{
		Status:    HealthStatusHealthy,
		Timestamp: time.Now().UTC(),
		Database:  HealthCheck{Name: "database", Status: HealthStatusHealthy},
		Kafka:     HealthCheck{Name: "kafka", Status: HealthStatusHealthy},
	}, nil
}

// FetchReadinessHandler checks that all dependencies are ready to serve.
type (
	FetchReadinessHandler struct {
		dbChecker    ports.HealthChecker
		kafkaChecker ports.HealthChecker
	}
)

// NewFetchReadinessHandler creates a readiness handler.
func NewFetchReadinessHandler(dbChecker, kafkaChecker ports.HealthChecker) *FetchReadinessHandler {
	return &FetchReadinessHandler{
		dbChecker:    dbChecker,
		kafkaChecker: kafkaChecker,
	}
}

// Handle returns healthy only when all dependencies are ready.
func (h *FetchReadinessHandler) Handle(ctx context.Context, _ FetchReadinessQuery) (HealthResponse, error) {
	resp := HealthResponse{
		Timestamp: time.Now().UTC(),
	}

	resp.Database = checkDependency("database", h.dbChecker, ctx)
	resp.Kafka = checkDependency("kafka", h.kafkaChecker, ctx)

	allHealthy := resp.Database.Status == HealthStatusHealthy &&
		resp.Kafka.Status == HealthStatusHealthy

	if allHealthy {
		resp.Status = HealthStatusHealthy
	} else {
		resp.Status = HealthStatusUnhealthy
	}

	return resp, nil
}

func checkDependency(name string, checker ports.HealthChecker, ctx context.Context) HealthCheck {
	if err := checker.Ping(ctx); err != nil {
		return HealthCheck{
			Name:    name,
			Status:  HealthStatusUnhealthy,
			Message: fmt.Sprintf("ping failed: %s", err.Error()),
		}
	}

	return HealthCheck{
		Name:   name,
		Status: HealthStatusHealthy,
	}
}

func aggregateStatus(checks ...HealthCheck) string {
	unhealthyCount := 0

	for _, check := range checks {
		if check.Status == HealthStatusUnhealthy {
			unhealthyCount++
		}
	}

	switch {
	case unhealthyCount == 0:
		return HealthStatusHealthy
	case unhealthyCount == len(checks):
		return HealthStatusUnhealthy
	default:
		return HealthStatusDegraded
	}
}
