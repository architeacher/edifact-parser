package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

// PanicRecovery catches panics, returns 500, and logs with correlation ID.
func PanicRecovery(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rvr := recover()
				if rvr == nil {
					return
				}

				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}

				var errMsg string
				switch v := rvr.(type) {
				case string:
					errMsg = v
				case error:
					errMsg = v.Error()
				default:
					errMsg = fmt.Sprintf("%v", v)
				}

				ctxLog := log.WithContext(r.Context())
				ctxLog.Error().
					Err(fmt.Errorf("panic: %s", errMsg)).
					Str("stack", string(debug.Stack())).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("panic recovered")

				w.Header().Set("Content-Type", "application/json")

				if r.Header.Get("Connection") != "Upgrade" {
					w.WriteHeader(http.StatusInternalServerError)
				}

				_, _ = w.Write([]byte(`{"code":"INTERNAL_ERROR","message":"internal server error"}`))
			}()

			next.ServeHTTP(w, r)
		})
	}
}
