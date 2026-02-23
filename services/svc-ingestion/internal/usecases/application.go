package usecases

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/architeacher/nomos-technical-challenge/pkg/decorator"
	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases/commands"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases/queries"
)

// Application composes all command and query handlers with cross-cutting decorators applied.
type (
	Application struct {
		Commands ApplicationCommands
		Queries  ApplicationQueries
	}

	// ApplicationCommands groups all command handlers.
	ApplicationCommands struct {
		IngestFile decorator.CommandHandler[commands.IngestFileCommand, *commands.IngestFileResult]
	}

	// ApplicationQueries groups all query handlers.
	ApplicationQueries struct {
		GetInterchange decorator.QueryHandler[queries.GetInterchangeQuery, queries.GetInterchangeResult]
		GetRawFile     decorator.QueryHandler[queries.GetRawFileQuery, queries.GetRawFileResult]
		FetchHealth    decorator.QueryHandler[queries.FetchHealthQuery, queries.HealthResponse]
		FetchLiveness  decorator.QueryHandler[queries.FetchLivenessQuery, queries.HealthResponse]
		FetchReadiness decorator.QueryHandler[queries.FetchReadinessQuery, queries.HealthResponse]
		ListMessages   decorator.QueryHandler[queries.ListMessagesQuery, queries.ListMessagesResult]
	}

	// Dependencies carries the raw handlers and infrastructure needed to build the Application.
	Dependencies struct {
		IngestFile     decorator.CommandHandler[commands.IngestFileCommand, *commands.IngestFileResult]
		GetInterchange decorator.QueryHandler[queries.GetInterchangeQuery, queries.GetInterchangeResult]
		GetRawFile     decorator.QueryHandler[queries.GetRawFileQuery, queries.GetRawFileResult]
		FetchHealth    decorator.QueryHandler[queries.FetchHealthQuery, queries.HealthResponse]
		FetchLiveness  decorator.QueryHandler[queries.FetchLivenessQuery, queries.HealthResponse]
		FetchReadiness decorator.QueryHandler[queries.FetchReadinessQuery, queries.HealthResponse]
		ListMessages   decorator.QueryHandler[queries.ListMessagesQuery, queries.ListMessagesResult]
		Logger         logger.Logger
		Metrics        decorator.MetricsRecorder
		Tracer         trace.Tracer
	}
)

// NewApplication creates an Application with all handlers wrapped in logging, metrics, and tracing decorators.
// Decorator order (outermost to innermost): Logging -> Metrics -> Tracing -> Handler.
// Panics with a descriptive message if any dependency is nil.
func NewApplication(deps Dependencies) *Application {
	validateDependencies(deps)

	return &Application{
		Commands: ApplicationCommands{
			IngestFile: decorator.ApplyCommandDecorators(
				deps.IngestFile,
				deps.Logger,
				deps.Metrics,
				deps.Tracer,
			),
		},
		Queries: ApplicationQueries{
			GetInterchange: decorator.ApplyQueryDecorators(
				deps.GetInterchange,
				deps.Logger,
				deps.Metrics,
				deps.Tracer,
			),
			GetRawFile: decorator.ApplyQueryDecorators(
				deps.GetRawFile,
				deps.Logger,
				deps.Metrics,
				deps.Tracer,
			),
			FetchHealth: decorator.ApplyQueryDecorators(
				deps.FetchHealth,
				deps.Logger,
				deps.Metrics,
				deps.Tracer,
			),
			FetchLiveness: decorator.ApplyQueryDecorators(
				deps.FetchLiveness,
				deps.Logger,
				deps.Metrics,
				deps.Tracer,
			),
			FetchReadiness: decorator.ApplyQueryDecorators(
				deps.FetchReadiness,
				deps.Logger,
				deps.Metrics,
				deps.Tracer,
			),
			ListMessages: decorator.ApplyQueryDecorators(
				deps.ListMessages,
				deps.Logger,
				deps.Metrics,
				deps.Tracer,
			),
		},
	}
}

// validateDependencies panics with a descriptive message if any required dependency is nil.
func validateDependencies(deps Dependencies) {
	if deps.IngestFile == nil {
		panic("IngestFile handler must not be nil")
	}

	if deps.GetInterchange == nil {
		panic("GetInterchange handler must not be nil")
	}

	if deps.GetRawFile == nil {
		panic("GetRawFile handler must not be nil")
	}

	if deps.FetchHealth == nil {
		panic("FetchHealth handler must not be nil")
	}

	if deps.FetchLiveness == nil {
		panic("FetchLiveness handler must not be nil")
	}

	if deps.FetchReadiness == nil {
		panic("FetchReadiness handler must not be nil")
	}

	if deps.ListMessages == nil {
		panic("ListMessages handler must not be nil")
	}

	if deps.Metrics == nil {
		panic("Metrics must not be nil")
	}

	if deps.Tracer == nil {
		panic("Tracer must not be nil")
	}
}
