package middleware_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/http/middleware"
)

func TestPanicRecovery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		handler        http.HandlerFunc
		expectedStatus int
		expectedBody   string
		expectedCT     string
		shouldPanic    bool
	}{
		{
			name: "string panic returns JSON 500",
			handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("unexpected")
			}),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"code":"INTERNAL_ERROR","message":"internal server error"}`,
			expectedCT:     "application/json",
		},
		{
			name: "error panic returns JSON 500",
			handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic(errors.New("something broke"))
			}),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"code":"INTERNAL_ERROR","message":"internal server error"}`,
			expectedCT:     "application/json",
		},
		{
			name: "arbitrary panic type returns JSON 500",
			handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic(42)
			}),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"code":"INTERNAL_ERROR","message":"internal server error"}`,
			expectedCT:     "application/json",
		},
		{
			name: "passes through normally",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			expectedStatus: http.StatusOK,
		},
		{
			name: "re-panics http.ErrAbortHandler",
			handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic(http.ErrAbortHandler)
			}),
			shouldPanic: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			log := logger.NewTestLogger()
			mw := middleware.PanicRecovery(log)
			handler := mw(tc.handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			if tc.shouldPanic {
				require.Panics(t, func() {
					handler.ServeHTTP(rec, req)
				}, fmt.Sprintf("expected panic for %s", tc.name))

				return
			}

			handler.ServeHTTP(rec, req)

			require.Equal(t, tc.expectedStatus, rec.Code)

			if tc.expectedBody != "" {
				require.Equal(t, tc.expectedBody, rec.Body.String())
			}

			if tc.expectedCT != "" {
				require.Equal(t, tc.expectedCT, rec.Header().Get("Content-Type"))
			}
		})
	}
}
