package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/http/middleware"
)

func TestRequestLogging_LogsRequest(t *testing.T) {
	t.Parallel()

	log := logger.NewTestLogger()
	mw := middleware.RequestLogging(log)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
