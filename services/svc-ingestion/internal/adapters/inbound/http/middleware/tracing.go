package middleware

import (
	"net/http"
)

// Tracing creates an OpenTelemetry span for each request.
func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tracing is handled by otelhttp or the tracer provider;
		// this middleware is a placeholder for manual span creation if needed.
		next.ServeHTTP(w, r)
	})
}
