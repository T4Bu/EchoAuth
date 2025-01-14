package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRateLimiter struct {
	mock.Mock
}

func (m *mockRateLimiter) Allow(key string) (bool, error) {
	args := m.Called(key)
	return args.Bool(0), args.Error(1)
}

func (m *mockRateLimiter) Reset(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func TestRateLimitMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		setupLimiter   func(*mockRateLimiter)
		expectedStatus int
	}{
		{
			name: "Request allowed",
			setupRequest: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:1234"
			},
			setupLimiter: func(m *mockRateLimiter) {
				m.On("Allow", "rate_limit:192.168.1.1:/test").Return(true, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Rate limit exceeded",
			setupRequest: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:1234"
			},
			setupLimiter: func(m *mockRateLimiter) {
				m.On("Allow", "rate_limit:192.168.1.1:/test").Return(false, nil)
			},
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name: "Rate limiter error",
			setupRequest: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:1234"
			},
			setupLimiter: func(m *mockRateLimiter) {
				m.On("Allow", "rate_limit:192.168.1.1:/test").Return(false, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "X-Forwarded-For header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "10.0.0.1,192.168.1.1")
			},
			setupLimiter: func(m *mockRateLimiter) {
				m.On("Allow", "rate_limit:10.0.0.1:/test").Return(true, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "X-Real-IP header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "10.0.0.1")
			},
			setupLimiter: func(m *mockRateLimiter) {
				m.On("Allow", "rate_limit:10.0.0.1:/test").Return(true, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLimiter := new(mockRateLimiter)
			tt.setupLimiter(mockLimiter)

			handler := RateLimitMiddleware(mockLimiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockLimiter.AssertExpectations(t)
		})
	}
}

func TestGetIP(t *testing.T) {
	tests := []struct {
		name       string
		setupReq   func(*http.Request)
		expectedIP string
	}{
		{
			name: "X-Forwarded-For single IP",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "10.0.0.1")
			},
			expectedIP: "10.0.0.1",
		},
		{
			name: "X-Forwarded-For multiple IPs",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "10.0.0.1,192.168.1.1")
			},
			expectedIP: "10.0.0.1",
		},
		{
			name: "X-Real-IP",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "10.0.0.1")
			},
			expectedIP: "10.0.0.1",
		},
		{
			name: "RemoteAddr",
			setupReq: func(r *http.Request) {
				r.RemoteAddr = "10.0.0.1:1234"
			},
			expectedIP: "10.0.0.1",
		},
		{
			name: "Fallback order - prefer X-Forwarded-For",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "10.0.0.1")
				r.Header.Set("X-Real-IP", "192.168.1.1")
				r.RemoteAddr = "127.0.0.1:1234"
			},
			expectedIP: "10.0.0.1",
		},
		{
			name: "Fallback order - prefer X-Real-IP over RemoteAddr",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "192.168.1.1")
				r.RemoteAddr = "127.0.0.1:1234"
			},
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupReq(req)

			ip := getIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}
