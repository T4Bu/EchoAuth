package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		requestID    string
		userID       interface{}
		responseCode int
		responseBody string
		expectedSize int64
	}{
		{
			name:         "Successful GET request",
			method:       "GET",
			path:         "/test",
			responseCode: http.StatusOK,
			responseBody: "success",
			expectedSize: 7,
		},
		{
			name:         "Failed request",
			method:       "POST",
			path:         "/test",
			responseCode: http.StatusBadRequest,
			responseBody: "bad request",
			expectedSize: 10,
		},
		{
			name:         "Request with ID",
			method:       "GET",
			path:         "/test",
			requestID:    "test-request-id",
			responseCode: http.StatusOK,
			responseBody: "success",
			expectedSize: 7,
		},
		{
			name:         "Authenticated request",
			method:       "GET",
			path:         "/test",
			userID:       123,
			responseCode: http.StatusOK,
			responseBody: "success",
			expectedSize: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that writes the specified response
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.responseBody))
			})

			// Wrap the handler with the logger middleware
			wrappedHandler := LoggerMiddleware(handler)

			// Create a test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.requestID != "" {
				req.Header.Set("X-Request-ID", tt.requestID)
			}

			// Add user ID to context if specified
			if tt.userID != nil {
				ctx := req.Context()
				ctx = context.WithValue(ctx, userIDKey, tt.userID)
				req = req.WithContext(ctx)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Process the request
			wrappedHandler.ServeHTTP(rr, req)

			// Verify response
			assert.Equal(t, tt.responseCode, rr.Code)
			assert.Equal(t, tt.responseBody, rr.Body.String())
		})
	}
}

func TestResponseWriter(t *testing.T) {
	tests := []struct {
		name           string
		writeHeader    bool
		writeBody      bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Write body without explicit header",
			writeHeader:    false,
			writeBody:      true,
			expectedStatus: http.StatusOK,
			expectedBody:   "test body",
		},
		{
			name:           "Write header then body",
			writeHeader:    true,
			writeBody:      true,
			expectedStatus: http.StatusCreated,
			expectedBody:   "test body",
		},
		{
			name:           "Write header only",
			writeHeader:    true,
			writeBody:      false,
			expectedStatus: http.StatusCreated,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base response writer
			rr := httptest.NewRecorder()
			w := &responseWriter{
				ResponseWriter: rr,
				status:         200,
				written:        0,
				wroteHeader:    false,
			}

			// Write header if specified
			if tt.writeHeader {
				w.WriteHeader(http.StatusCreated)
				assert.True(t, w.wroteHeader)
				assert.Equal(t, http.StatusCreated, w.status)
			}

			// Write body if specified
			if tt.writeBody {
				n, err := w.Write([]byte("test body"))
				assert.NoError(t, err)
				assert.Equal(t, 9, n)
				assert.Equal(t, int64(9), w.written)
			}

			// Verify final state
			assert.Equal(t, tt.expectedStatus, w.status)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}
