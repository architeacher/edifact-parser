package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oapi-codegen/nullable"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
	"github.com/architeacher/nomos-technical-challenge/pkg/metrics"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/http/handlers/public"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/http/middleware"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases/commands"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases/queries"
)

const (
	baseURL     = "/v1"
	maxFileSize = 10 * 1024 * 1024
)

// RouterConfig holds the dependencies required to construct the HTTP router.
type RouterConfig struct {
	App      *usecases.Application
	Logger   logger.Logger
	Metrics  *metrics.Metrics
	Gatherer prometheus.Gatherer
}

// NewRouter creates an http.Handler with middleware, metrics endpoint, and OpenAPI-generated routes.
func NewRouter(cfg RouterConfig) http.Handler {
	router := chi.NewRouter()

	// Global middlewares — safety-critical, applied to every route including /metrics.
	router.Use(
		middleware.CorrelationID,
		middleware.PanicRecovery(cfg.Logger),
	)

	router.Handle("/metrics", promhttp.HandlerFor(cfg.Gatherer, promhttp.HandlerOpts{}))

	impl := newStrictServerImplementation(cfg.App)
	handler := public.NewStrictHandler(impl, []public.StrictMiddlewareFunc{})

	// Per-route middlewares are applied only to the OpenAPI-generated routes.
	return public.HandlerWithOptions(handler, public.ChiServerOptions{
		BaseRouter:  router,
		BaseURL:     baseURL,
		Middlewares: initMiddlewares(cfg),
	})
}

// initMiddlewares returns per-route middlewares for the OpenAPI-generated API routes.
// These are intentionally not applied globally to avoid polluting logs and metrics
// with Prometheus scrape traffic on /metrics.
func initMiddlewares(cfg RouterConfig) []public.MiddlewareFunc {
	return []public.MiddlewareFunc{
		middleware.Tracing,
		middleware.RequestLogging(cfg.Logger),
		middleware.MetricsMiddleware(cfg.Metrics),
	}
}

// strictServerImplementation implements the StrictServerInterface.
type strictServerImplementation struct {
	*usecases.Application
}

// newStrictServerImplementation creates a new HTTP handler implementation.
func newStrictServerImplementation(app *usecases.Application) *strictServerImplementation {
	return &strictServerImplementation{
		Application: app,
	}
}

// GetHealth returns the combined health status of all dependencies.
func (s *strictServerImplementation) GetHealth(
	ctx context.Context,
	_ public.GetHealthRequestObject,
) (public.GetHealthResponseObject, error) {
	result, err := s.Queries.FetchHealth.Handle(ctx, queries.FetchHealthQuery{})
	if err != nil {
		// Return 200 with unhealthy status on error
		return public.GetHealth200JSONResponse(
			public.Health{
				Status: public.Unhealthy,
			},
		), nil
	}

	resp := toHealthResponse(result)

	// Return 503 for unhealthy status, 200 for healthy/degraded
	if result.Status == queries.HealthStatusUnhealthy {
		return public.GetHealth503JSONResponse(resp), nil
	}

	return public.GetHealth200JSONResponse(resp), nil
}

// GetLiveness returns the liveness status (process is running).
func (s *strictServerImplementation) GetLiveness(
	ctx context.Context,
	_ public.GetLivenessRequestObject,
) (public.GetLivenessResponseObject, error) {
	result, err := s.Queries.FetchLiveness.Handle(ctx, queries.FetchLivenessQuery{})
	if err != nil {
		// Return 503 on error
		return public.GetLiveness503Response{}, nil
	}

	if result.Status == queries.HealthStatusUnhealthy {
		return public.GetLiveness503Response{}, nil
	}

	return public.GetLiveness200Response{}, nil
}

// GetReadiness returns the readiness status (all dependencies ready).
func (s *strictServerImplementation) GetReadiness(
	ctx context.Context,
	_ public.GetReadinessRequestObject,
) (public.GetReadinessResponseObject, error) {
	result, err := s.Queries.FetchReadiness.Handle(ctx, queries.FetchReadinessQuery{})
	if err != nil {
		// Return 503 on error
		return public.GetReadiness503Response{}, nil
	}

	if result.Status == queries.HealthStatusUnhealthy {
		return public.GetReadiness503Response{}, nil
	}

	return public.GetReadiness200Response{}, nil
}

// IngestFile handles file ingestion.
func (s *strictServerImplementation) IngestFile(
	ctx context.Context,
	request public.IngestFileRequestObject,
) (public.IngestFileResponseObject, error) {
	// Extract file from multipart form
	fileData, contentType, err := extractFile(request.Body)
	if err != nil {
		return public.IngestFile500JSONResponse{
			ServerErrorJSONResponse: public.ServerErrorJSONResponse(
				public.Error{
					Code:    public.BADREQUEST,
					Message: fmt.Sprintf("Failed to read file: %v", err),
				},
			),
		}, nil
	}

	// Validate file size
	if len(fileData) > maxFileSize {
		return public.IngestFile413JSONResponse(
			public.Error{
				Code:    public.FILETOOLARGE,
				Message: fmt.Sprintf("File exceeds maximum size of %d bytes", maxFileSize),
			},
		), nil
	}

	// Call the ingest command
	result, err := s.Commands.IngestFile.Handle(ctx, commands.IngestFileCommand{
		FileData:    fileData,
		ContentType: contentType,
	})

	// Handle command errors
	if err != nil {
		if errors.Is(err, model.ErrDuplicateInterchange) && result != nil {
			return public.IngestFile200JSONResponse(public.Ingest{
				InterchangeId:  result.InterchangeID,
				Status:         mapStatus(result.Status),
				MessageSummary: public.MessageSummary{Total: result.MessageCount, ByType: make(map[string]int)},
			}), nil
		}

		if strings.Contains(err.Error(), "file exceeds") {
			return public.IngestFile413JSONResponse(
				public.Error{
					Code:    public.FILETOOLARGE,
					Message: err.Error(),
				},
			), nil
		}

		return public.IngestFile500JSONResponse{
			ServerErrorJSONResponse: public.ServerErrorJSONResponse(
				public.Error{
					Code:    public.INTERNALERROR,
					Message: err.Error(),
				},
			),
		}, nil
	}

	// Map result to appropriate response
	msgSummary := public.MessageSummary{
		Total:  result.MessageCount,
		ByType: make(map[string]int),
	}

	resp := public.Ingest{
		InterchangeId:  result.InterchangeID,
		Status:         mapStatus(result.Status),
		MessageSummary: msgSummary,
	}

	// Return 201 for completed, 422 for failed
	if result.Status == model.StatusCompleted {
		return public.IngestFile201JSONResponse(resp), nil
	}

	return public.IngestFile422JSONResponse(
		public.Error{
			Code:    public.INVALIDEDIFACT,
			Message: "EDIFACT file parsing failed",
		},
	), nil
}

// GetInterchange retrieves an interchange by ID.
func (s *strictServerImplementation) GetInterchange(
	ctx context.Context,
	request public.GetInterchangeRequestObject,
) (public.GetInterchangeResponseObject, error) {
	// Parse UUID from request
	id, err := uuid.Parse(request.Id.String())
	if err != nil {
		return public.GetInterchange404JSONResponse{
			NotFoundJSONResponse: public.NotFoundJSONResponse(
				public.Error{
					Code:    public.NOTFOUND,
					Message: "Invalid interchange ID format",
				},
			),
		}, nil
	}

	// Call the query handler
	result, err := s.Queries.GetInterchange.Handle(ctx, queries.GetInterchangeQuery{
		InterchangeID: id,
	})

	// Handle errors
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return public.GetInterchange404JSONResponse{
				NotFoundJSONResponse: public.NotFoundJSONResponse(
					public.Error{
						Code:    public.NOTFOUND,
						Message: fmt.Sprintf("Interchange not found: %v", id),
					},
				),
			}, nil
		}

		return public.GetInterchange500JSONResponse{
			ServerErrorJSONResponse: public.ServerErrorJSONResponse(
				public.Error{
					Code:    public.INTERNALERROR,
					Message: err.Error(),
				},
			),
		}, nil
	}

	// Build response
	interchangeResp := public.InterchangeDetail{
		Id:         result.ID,
		SenderId:   result.SenderID,
		ReceiverId: result.ReceiverID,
		Reference:  result.Reference,
		PreparedAt: result.PreparedAt,
		CreatedAt:  result.PreparedAt, // Use prepared time as created for now
		Status:     mapInterchangeDetailStatus(result.Status),
	}

	if result.ActiveVersion != nil {
		interchangeResp.ActiveVersion = public.ProcessingVersionSummary{
			Id:            result.ActiveVersion.ID,
			VersionNumber: result.ActiveVersion.VersionNumber,
			ParserVersion: result.ActiveVersion.ParserVersion,
			FormatVersion: result.ActiveVersion.FormatVersion,
			CreatedAt:     result.ActiveVersion.CreatedAt,
		}
	}

	if result.MessageCount > 0 {
		interchangeResp.MessageSummary = &public.MessageSummary{
			Total:  int(result.MessageCount),
			ByType: make(map[string]int),
		}
	}

	return public.GetInterchange200JSONResponse(interchangeResp), nil
}

// GetRawFile returns the raw EDIFACT file content.
func (s *strictServerImplementation) GetRawFile(
	ctx context.Context,
	request public.GetRawFileRequestObject,
) (public.GetRawFileResponseObject, error) {
	// Parse UUID from request
	id, err := uuid.Parse(request.Id.String())
	if err != nil {
		return public.GetRawFile404JSONResponse{
			NotFoundJSONResponse: public.NotFoundJSONResponse(
				public.Error{
					Code:    public.NOTFOUND,
					Message: "Invalid interchange ID format",
				},
			),
		}, nil
	}

	// Call the query handler
	result, err := s.Queries.GetRawFile.Handle(ctx, queries.GetRawFileQuery{
		InterchangeID: id,
	})

	// Handle errors
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return public.GetRawFile404JSONResponse{
				NotFoundJSONResponse: public.NotFoundJSONResponse(
					public.Error{
						Code:    public.NOTFOUND,
						Message: fmt.Sprintf("Interchange not found: %v", id),
					},
				),
			}, nil
		}

		return public.GetRawFile500JSONResponse{
			ServerErrorJSONResponse: public.ServerErrorJSONResponse(
				public.Error{
					Code:    public.INTERNALERROR,
					Message: err.Error(),
				},
			),
		}, nil
	}

	// Return file content with proper headers
	return public.GetRawFile200ApplicationoctetStreamResponse{
		Body:          io.NopCloser(bytes.NewReader(result.Content)),
		ContentLength: int64(len(result.Content)),
	}, nil
}

// ListMessages lists messages with optional filtering.
func (s *strictServerImplementation) ListMessages(
	ctx context.Context,
	request public.ListMessagesRequestObject,
) (public.ListMessagesResponseObject, error) {
	// Build filter from request parameters
	filter := &model.MessageFilter{
		Limit: model.DefaultPageLimit,
	}

	if request.Params.SubscriptionId != nil {
		id := *request.Params.SubscriptionId
		filter.SubscriptionID = &id
	}

	if request.Params.MessageType != nil {
		filter.MessageType = request.Params.MessageType
	}

	if request.Params.InterchangeId != nil {
		id := *request.Params.InterchangeId
		filter.InterchangeID = &id
	}

	if request.Params.ProcessingVersionId != nil {
		id := *request.Params.ProcessingVersionId
		filter.ProcessingVersionID = &id
	}

	if request.Params.Limit != nil && *request.Params.Limit > 0 {
		filter.Limit = *request.Params.Limit
	}

	if request.Params.Cursor != nil {
		filter.Cursor = request.Params.Cursor
	}

	// Call the query handler
	result, err := s.Queries.ListMessages.Handle(ctx, queries.ListMessagesQuery{
		Filter: filter,
	})

	// Handle errors
	if err != nil {
		return public.ListMessages500JSONResponse{
			ServerErrorJSONResponse: public.ServerErrorJSONResponse(
				public.Error{
					Code:    public.INTERNALERROR,
					Message: err.Error(),
				},
			),
		}, nil
	}

	// Build response
	messages := make([]public.Message, 0, len(result.Messages))
	for _, msg := range result.Messages {
		genMsg := public.Message{
			Id:                  msg.ID,
			InterchangeId:       msg.InterchangeID,
			ProcessingVersionId: msg.ProcessingVersionID,
			MessageType:         msg.MessageType,
			MessageReference:    msg.MessageReference,
			CreatedAt:           msg.CreatedAt,
		}

		if msg.SubscriptionID != nil {
			genMsg.SubscriptionId = nullable.NewNullableWithValue(*msg.SubscriptionID)
		}

		if msg.Segments != nil {
			genMsg.Segments = &msg.Segments
		}

		messages = append(messages, genMsg)
	}

	resp := public.MessageList{
		Items: messages,
		Total: len(messages),
	}

	if result.NextCursor != nil {
		resp.NextCursor = nullable.NewNullableWithValue(*result.NextCursor)
	}

	return public.ListMessages200JSONResponse(resp), nil
}

// ReprocessInterchanges handles bulk reprocessing (Phase 4 - stub).
func (s *strictServerImplementation) ReprocessInterchanges(
	_ context.Context,
	_ public.ReprocessInterchangesRequestObject,
) (public.ReprocessInterchangesResponseObject, error) {
	return public.ReprocessInterchanges400JSONResponse{
		BadRequestJSONResponse: public.BadRequestJSONResponse(
			public.Error{
				Code:    public.BADREQUEST,
				Message: "Not implemented - Phase 4",
			},
		),
	}, nil
}

// RollbackVersion handles rollback to previous version (Phase 4 - stub).
func (s *strictServerImplementation) RollbackVersion(
	_ context.Context,
	_ public.RollbackVersionRequestObject,
) (public.RollbackVersionResponseObject, error) {
	return public.RollbackVersion500JSONResponse{
		ServerErrorJSONResponse: public.ServerErrorJSONResponse(
			public.Error{
				Code:    public.INTERNALERROR,
				Message: "Not implemented - Phase 4",
			},
		),
	}, nil
}

// extractFile reads the multipart file from the request.
func extractFile(reader *multipart.Reader) ([]byte, string, error) {
	// Read the first form part
	part, err := reader.NextPart()
	if err != nil {
		return nil, "", fmt.Errorf("reading multipart form: %w", err)
	}
	defer part.Close()

	// Read the file data
	data, err := io.ReadAll(part)
	if err != nil {
		return nil, "", fmt.Errorf("reading file data: %w", err)
	}

	contentType := part.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return data, contentType, nil
}

// mapStatus converts domain status to generated status type.
func mapStatus(status model.InterchangeStatus) public.IngestStatus {
	switch status {
	case model.StatusCompleted:
		return public.IngestStatusCompleted
	case model.StatusFailed:
		return public.IngestStatusFailed
	default:
		return public.IngestStatusFailed
	}
}

// mapInterchangeDetailStatus converts domain status to generated interchange detail status type.
func mapInterchangeDetailStatus(status model.InterchangeStatus) public.InterchangeDetailStatus {
	switch status {
	case model.StatusCompleted:
		return public.InterchangeDetailStatusCompleted
	case model.StatusFailed:
		return public.InterchangeDetailStatusFailed
	default:
		return public.InterchangeDetailStatusFailed
	}
}

// toHealthResponse converts domain health response to generated health response.
func toHealthResponse(health queries.HealthResponse) public.Health {
	resp := public.Health{
		Status: mapHealthStatus(health.Status),
	}

	if health.Database.Status != "" {
		checkStatus := health.Database.Status
		if checkStatus == queries.HealthStatusHealthy {
			ok := public.HealthChecksPostgresOk
			resp.Checks.Postgres = &ok
		} else {
			err := public.HealthChecksPostgresError
			resp.Checks.Postgres = &err
		}
	}

	if health.Kafka.Status != "" {
		checkStatus := health.Kafka.Status
		if checkStatus == queries.HealthStatusHealthy {
			ok := public.HealthChecksKafkaOk
			resp.Checks.Kafka = &ok
		} else {
			err := public.HealthChecksKafkaError
			resp.Checks.Kafka = &err
		}
	}

	return resp
}

// mapHealthStatus converts domain health status to generated health status.
func mapHealthStatus(status string) public.HealthStatus {
	switch status {
	case queries.HealthStatusHealthy:
		return public.Healthy
	case queries.HealthStatusDegraded:
		return public.Degraded
	case queries.HealthStatusUnhealthy:
		return public.Unhealthy
	default:
		return public.Unhealthy
	}
}
