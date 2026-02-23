package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

const (
	// HeaderCorrelationID is the HTTP header for request correlation.
	HeaderCorrelationID = "X-Correlation-ID"
)

// CorrelationID extracts or generates a correlation ID and injects it into the request context.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Header.Get(HeaderCorrelationID)
		if cid == "" {
			cid = uuid.Must(uuid.NewV7()).String()
		}

		w.Header().Set(HeaderCorrelationID, cid)

		ctx := context.WithValue(r.Context(), logger.ContextKeyCorrelationID, cid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
