package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/pkg/metrics"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/http/middleware"
)

func TestMetricsMiddleware_RecordsMetrics(t *testing.T) {
	t.Parallel()

	m := metrics.NewNoopMetrics()
	mw := middleware.MetricsMiddleware(m)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/interchanges", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}
