package middleware

import (
	"net/http"
	"time"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

// RequestLogging logs method, path, status, and duration for each request.
func RequestLogging(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			ctxLog := log.WithContext(r.Context())
			ctxLog.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", sw.status).
				Int64("duration_ms", time.Since(start).Milliseconds()).
				Msg("request completed")
		})
	}
}
