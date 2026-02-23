package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/http/middleware"
)

func TestCorrelationID_GeneratesWhenMissing(t *testing.T) {
	t.Parallel()

	handler := middleware.CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid, ok := r.Context().Value(logger.ContextKeyCorrelationID).(string)
		require.True(t, ok)
		require.NotEmpty(t, cid)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.NotEmpty(t, rec.Header().Get(middleware.HeaderCorrelationID))
}

func TestCorrelationID_PreservesExisting(t *testing.T) {
	t.Parallel()

	handler := middleware.CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid, ok := r.Context().Value(logger.ContextKeyCorrelationID).(string)
		require.True(t, ok)
		require.Equal(t, "my-cid-123", cid)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(middleware.HeaderCorrelationID, "my-cid-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, "my-cid-123", rec.Header().Get(middleware.HeaderCorrelationID))
}
