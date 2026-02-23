package middleware

import (
	"net/http"
)

// statusWriter wraps http.ResponseWriter to capture the status code.
type (
	statusWriter struct {
		http.ResponseWriter
		status int
	}
)

// WriteHeader captures the status code before writing it.
func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}
