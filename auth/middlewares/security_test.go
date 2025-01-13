package middlewares

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSecurityMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		environment     string
		origin          string
		method          string
		requestMethod   string
		requestHeaders  string
		expectedStatus  int
		expectedHeaders map[string]string
	}{
		{
			name:           "development_allowed",
			environment:    "development",
			origin:         "http://localhost:3000",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "http://localhost:3000",
			},
		},
		{
			name:           "production_allowed",
			environment:    "production",
			origin:         "https://app.example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://app.example.com",
				"Strict-Transport-Security":   "max-age=31536000; includeSubDomains",
			},
		},
		{
			name:           "production_blocked",
			environment:    "production",
			origin:         "http://malicious.com",
			method:         "GET",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "valid_preflight",
			environment:    "development",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			requestMethod:  "POST",
			requestHeaders: "content-type, authorization",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "http://localhost:3000",
				"Access-Control-Max-Age":      "3600",
			},
		},
		{
			name:           "invalid_preflight_method",
			environment:    "development",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			requestMethod:  "INVALID",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid_preflight_headers",
			environment:    "development",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			requestMethod:  "POST",
			requestHeaders: "invalid-header",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ENV", tt.environment)
			if tt.environment == "production" {
				os.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")
			}
			defer os.Unsetenv("ENV")
			defer os.Unsetenv("ALLOWED_ORIGINS")

			config := NewSecurityConfig()
			handler := config.SecurityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.requestMethod != "" {
				req.Header.Set("Access-Control-Request-Method", tt.requestMethod)
			}
			if tt.requestHeaders != "" {
				req.Header.Set("Access-Control-Request-Headers", tt.requestHeaders)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			for header, expected := range tt.expectedHeaders {
				if got := w.Header().Get(header); got != expected {
					t.Errorf("expected header %s to be %s, got %s", header, expected, got)
				}
			}

			// Check security headers
			if tt.expectedStatus == http.StatusOK {
				securityHeaders := []string{
					"X-Content-Type-Options",
					"X-Frame-Options",
					"X-XSS-Protection",
				}
				for _, header := range securityHeaders {
					if got := w.Header().Get(header); got == "" {
						t.Errorf("missing security header: %s", header)
					}
				}
			}
		})
	}
}
