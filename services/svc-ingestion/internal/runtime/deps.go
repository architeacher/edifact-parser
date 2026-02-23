package runtime

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	kafkago "github.com/segmentio/kafka-go"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
	"github.com/architeacher/nomos-technical-challenge/pkg/metrics"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases"
)

// cleanupEntry pairs a resource name with its shutdown function for ordered cleanup.
type cleanupEntry struct {
	name string
	fn   func(ctx context.Context) error
}

type (
	infrastructureDep struct {
		httpServer  *http.Server
		db          *pgxpool.Pool
		kafkaWriter *kafkago.Writer
		resource    *resource.Resource
		meterProv   *sdkmetric.MeterProvider
		tracerProv  *sdktrace.TracerProvider
		logger      logger.Logger
		loggerSet   bool
		metrics     *metrics.Metrics
		gatherer    prometheus.Gatherer
	}

	repositories struct {
		interchange  ports.InterchangeRepository
		version      ports.ProcessingVersionRepository
		message      ports.MessageRepository
		subscription ports.SubscriptionRepository
	}

	servicesDep struct {
		ingestion ports.IngestionService
	}

	applications struct {
		httpApp *usecases.Application
	}

	dependencies struct {
		config   *config.Config
		infra    infrastructureDep
		repos    repositories
		services servicesDep
		apps     applications

		// cleanupFuncs stores shutdown functions in registration order; cleanup iterates in reverse (LIFO).
		cleanupFuncs []cleanupEntry
	}

	// DependencyOption configures a single dependency during initialization.
	DependencyOption func(*dependencies) error
)

func initializeDependencies(ctx context.Context, opts ...DependencyOption) (*dependencies, error) {
	deps := &dependencies{}

	// Overrides run first so their nil-check guards in defaultOptions skip already-provided dependencies.
	allOpts := slices.Concat(opts, defaultOptions(ctx))

	for _, opt := range allOpts {
		if err := opt(deps); err != nil {
			return nil, fmt.Errorf("applying dependency option: %w", err)
		}
	}

	return deps, nil
}
