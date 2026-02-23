package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	kafkago "github.com/segmentio/kafka-go"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
	"github.com/architeacher/nomos-technical-challenge/pkg/metrics"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/edifact"
	httpAdapter "github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/http"
	kafkaadapter "github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/outbound/kafka"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/repos"
	svcadapter "github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/services"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/infrastructure"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases/commands"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases/queries"
)

const (
	parserVersion = "1.0.0"

	httpReadHeaderTimeout = 10 * time.Second
	httpReadTimeout       = 30 * time.Second
	httpWriteTimeout      = 30 * time.Second
	httpIdleTimeout       = 120 * time.Second
)

func defaultOptions(ctx context.Context) []DependencyOption {
	return []DependencyOption{
		WithConfig(),
		WithLogger(),
		WithResource(ctx),
		WithMetrics(ctx),
		WithDatabase(ctx),
		WithKafkaWriter(),
		WithTracing(ctx),
		WithRepositories(),
		WithServices(),
		WithApplication(),
		WithHTTPServer(),
	}
}

// ── Infrastructure ──────────────────────────────────────────

// WithConfig returns a DependencyOption that loads the application configuration from environment variables.
func WithConfig() DependencyOption {
	return func(d *dependencies) error {
		if d.config != nil {
			return nil
		}

		cfg, err := config.FromEnv()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		d.config = cfg

		return nil
	}
}

// WithLogger returns a DependencyOption that configures the structured logger.
func WithLogger() DependencyOption {
	return func(d *dependencies) error {
		if d.infra.loggerSet {
			return nil
		}

		d.infra.logger = logger.New(d.config.Logging.Level, d.config.Logging.Format)
		d.infra.loggerSet = true

		return nil
	}
}

// WithResource returns a DependencyOption that creates the shared OTel resource used by all providers.
func WithResource(ctx context.Context) DependencyOption {
	return func(d *dependencies) error {
		if d.infra.resource != nil {
			return nil
		}

		res, err := infrastructure.NewResource(ctx, d.config.OTel)
		if err != nil {
			return fmt.Errorf("creating OTel resource: %w", err)
		}

		d.infra.resource = res

		return nil
	}
}

// WithMetrics returns a DependencyOption that configures the OTel MeterProvider and application metrics.
// The OTel Prometheus exporter (inside MeterProvider) registers with the default Prometheus registry,
// which is then used by the /metrics scrape endpoint.
func WithMetrics(ctx context.Context) DependencyOption {
	return func(d *dependencies) error {
		if d.infra.metrics != nil {
			return nil
		}

		mp, err := infrastructure.NewMeterProvider(ctx, d.config.OTel, d.infra.resource)
		if err != nil {
			return fmt.Errorf("initializing meter provider: %w", err)
		}

		d.infra.meterProv = mp

		m, err := metrics.NewMetrics("edifact", mp)
		if err != nil {
			return fmt.Errorf("creating metrics: %w", err)
		}

		d.infra.metrics = m
		d.infra.gatherer = prometheus.DefaultGatherer

		d.cleanupFuncs = append(d.cleanupFuncs, cleanupEntry{
			name: "meter provider",
			fn: func(ctx context.Context) error {
				return d.infra.meterProv.Shutdown(ctx)
			},
		})

		return nil
	}
}

// WithDatabase returns a DependencyOption that configures the PostgreSQL connection pool.
func WithDatabase(ctx context.Context) DependencyOption {
	return func(d *dependencies) error {
		if d.infra.db != nil {
			return nil
		}

		pool, err := infrastructure.NewPostgres(ctx, d.config.Database, d.config.OTel)
		if err != nil {
			return fmt.Errorf("initializing postgres: %w", err)
		}

		d.infra.db = pool

		d.cleanupFuncs = append(d.cleanupFuncs, cleanupEntry{
			name: "database",
			fn: func(_ context.Context) error { //nolint:unparam // Close() returns no error but cleanup signature requires one.
				d.infra.db.Close()

				return nil
			},
		})

		return nil
	}
}

// WithKafkaWriter returns a DependencyOption that configures the Kafka producer.
func WithKafkaWriter() DependencyOption {
	return func(d *dependencies) error {
		if d.infra.kafkaWriter != nil {
			return nil
		}

		d.infra.kafkaWriter = infrastructure.NewKafkaWriter(d.config.Kafka)

		d.cleanupFuncs = append(d.cleanupFuncs, cleanupEntry{
			name: "kafka writer",
			fn: func(_ context.Context) error {
				return d.infra.kafkaWriter.Close()
			},
		})

		return nil
	}
}

// WithTracing returns a DependencyOption that configures the OpenTelemetry tracer provider.
func WithTracing(ctx context.Context) DependencyOption {
	return func(d *dependencies) error {
		if d.infra.tracerProv != nil {
			return nil
		}

		tp, err := infrastructure.NewTracerProvider(ctx, d.config.OTel, d.infra.resource)
		if err != nil {
			return fmt.Errorf("initializing tracer: %w", err)
		}

		d.infra.tracerProv = tp

		d.cleanupFuncs = append(d.cleanupFuncs, cleanupEntry{
			name: "tracer",
			fn: func(ctx context.Context) error {
				return d.infra.tracerProv.Shutdown(ctx)
			},
		})

		return nil
	}
}

// ── Repositories ────────────────────────────────────────────

// WithRepositories returns a DependencyOption that configures all data-access repositories.
func WithRepositories() DependencyOption {
	return func(d *dependencies) error {
		if d.repos.interchange != nil {
			return nil
		}

		d.repos = repositories{
			interchange:  repos.NewInterchangeRepository(d.infra.db),
			version:      repos.NewProcessingVersionRepository(d.infra.db),
			message:      repos.NewMessageRepository(d.infra.db),
			subscription: repos.NewSubscriptionRepository(d.infra.db),
		}

		return nil
	}
}

// ── Services ────────────────────────────────────────────────

// WithServices returns a DependencyOption that configures the domain services.
func WithServices() DependencyOption {
	return func(d *dependencies) error {
		if d.services.ingestion != nil {
			return nil
		}

		eventPublisher := kafkaadapter.NewPublisherFromWriter(d.infra.kafkaWriter)
		parser := &edifactParser{}
		txMgr := repos.NewTransactionManager(d.infra.db)

		d.services = servicesDep{
			ingestion: svcadapter.NewIngestionService(
				d.repos.interchange,
				d.repos.version,
				d.repos.message,
				d.repos.subscription,
				eventPublisher,
				parser,
				txMgr,
				parserVersion,
			),
		}

		return nil
	}
}

// ── Application ─────────────────────────────────────────────

// WithApplication returns a DependencyOption that configures the CQRS application layer.
func WithApplication() DependencyOption {
	return func(d *dependencies) error {
		if d.apps.httpApp != nil {
			return nil
		}

		dbChecker := &dbHealthChecker{db: d.infra.db}
		kafkaChecker := &kafkaHealthChecker{writer: d.infra.kafkaWriter}

		ingestFileCmd := commands.NewIngestFileHandler(d.services.ingestion)

		getInterchangeQry := queries.NewGetInterchangeHandler(d.services.ingestion)
		getRawFileQry := queries.NewGetRawFileHandler(d.services.ingestion)
		getHealthQry := queries.NewFetchHealthHandler(dbChecker, kafkaChecker)
		getLivenessQry := queries.NewFetchLivenessHandler()
		getReadinessQry := queries.NewFetchReadinessHandler(dbChecker, kafkaChecker)
		listMessagesQry := queries.NewListMessagesHandler(d.services.ingestion)

		d.apps.httpApp = usecases.NewApplication(usecases.Dependencies{
			IngestFile:     ingestFileCmd,
			GetInterchange: getInterchangeQry,
			GetRawFile:     getRawFileQry,
			FetchHealth:    getHealthQry,
			FetchLiveness:  getLivenessQry,
			FetchReadiness: getReadinessQry,
			ListMessages:   listMessagesQry,
			Logger:         d.infra.logger,
			Metrics:        d.infra.metrics,
			Tracer:         d.infra.tracerProv.Tracer("edifact"),
		})

		return nil
	}
}

// ── HTTP Server ─────────────────────────────────────────────

// WithHTTPServer returns a DependencyOption that configures the chi-based HTTP server and routes.
func WithHTTPServer() DependencyOption {
	return func(d *dependencies) error {
		if d.infra.httpServer != nil {
			return nil
		}

		d.infra.httpServer = &http.Server{
			Addr: fmt.Sprintf(":%d", d.config.Server.Port),
			Handler: httpAdapter.NewRouter(httpAdapter.RouterConfig{
				App:      d.apps.httpApp,
				Logger:   d.infra.logger,
				Metrics:  d.infra.metrics,
				Gatherer: d.infra.gatherer,
			}),
			ReadHeaderTimeout: httpReadHeaderTimeout,
			ReadTimeout:       httpReadTimeout,
			WriteTimeout:      httpWriteTimeout,
			IdleTimeout:       httpIdleTimeout,
		}

		d.cleanupFuncs = append(d.cleanupFuncs, cleanupEntry{
			name: "HTTP server",
			fn: func(ctx context.Context) error {
				return d.infra.httpServer.Shutdown(ctx)
			},
		})

		return nil
	}
}

// ── Test Overrides ──────────────────────────────────────────

// WithExternalConfig returns a DependencyOption that injects a pre-built configuration for testing.
func WithExternalConfig(cfg *config.Config) DependencyOption {
	return func(d *dependencies) error {
		d.config = cfg

		return nil
	}
}

// WithExternalDB returns a DependencyOption that injects a pre-built database pool for testing.
func WithExternalDB(pool *pgxpool.Pool) DependencyOption {
	return func(d *dependencies) error {
		d.infra.db = pool

		return nil
	}
}

// WithExternalKafkaWriter returns a DependencyOption that injects a pre-built Kafka writer for testing.
func WithExternalKafkaWriter(w *kafkago.Writer) DependencyOption {
	return func(d *dependencies) error {
		d.infra.kafkaWriter = w

		return nil
	}
}

// ── Adapter Types ───────────────────────────────────────────

type edifactParser struct{}

func (p *edifactParser) Parse(data []byte) (*ports.ParseResult, error) {
	parser, err := edifact.NewParser(data)
	if err != nil {
		return nil, fmt.Errorf("creating EDIFACT parser: %w", err)
	}

	result, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing EDIFACT data: %w", err)
	}

	messages := make([]ports.ParsedMessage, len(result.Messages))
	for idx, msg := range result.Messages {
		messages[idx] = ports.ParsedMessage{
			MessageType:      msg.MessageType,
			MessageVersion:   msg.MessageVersion,
			MessageRelease:   msg.MessageRelease,
			MessageReference: msg.MessageReference,
			SubscriptionID:   msg.SubscriptionID,
			Segments:         msg.Segments,
		}
	}

	return &ports.ParseResult{
		SenderID:          result.SenderID,
		SenderQualifier:   result.SenderQualifier,
		ReceiverID:        result.ReceiverID,
		ReceiverQualifier: result.ReceiverQualifier,
		Reference:         result.Reference,
		PreparedAt:        result.PreparedAt,
		ContentHash:       result.ContentHash,
		Messages:          messages,
		Subscriptions:     result.Subscriptions,
	}, nil
}

type dbHealthChecker struct {
	db *pgxpool.Pool
}

func (h *dbHealthChecker) Ping(ctx context.Context) error {
	if err := h.db.Ping(ctx); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}

	return nil
}

type kafkaHealthChecker struct {
	writer *kafkago.Writer
}

// Ping verifies the Kafka writer is initialized. A full connectivity check is not possible
// with kafka-go's Writer API; actual write failures are handled at publish time.
func (h *kafkaHealthChecker) Ping(_ context.Context) error {
	if h.writer == nil {
		return errors.New("kafka writer is nil")
	}

	return nil
}
