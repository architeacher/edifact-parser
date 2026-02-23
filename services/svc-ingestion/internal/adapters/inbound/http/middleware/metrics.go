package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/architeacher/nomos-technical-challenge/pkg/metrics"
)

// MetricsMiddleware records request latency and status code counters.
func MetricsMiddleware(metrics *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			durationMs := float64(time.Since(start).Milliseconds())
			operation := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			metrics.RecordLatency(operation, durationMs)

			status := "success"
			if sw.status >= http.StatusBadRequest {
				status = "error"
			}

			metrics.IncrementCounter(operation, status)
		})
	}
}
