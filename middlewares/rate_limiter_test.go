package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock rate limiter service
type mockRateLimiterService struct {
	mock.Mock
}

func (m *mockRateLimiterService) Allow(key string) (bool, error) {
	args := m.Called(key)
	return args.Bool(0), args.Error(1)
}

func (m *mockRateLimiterService) Reset(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func TestNewRateLimiter(t *testing.T) {
	limiter := &mockRateLimiterService{}
	rateLimiter := &RateLimiter{
		limiter: limiter,
	}

	assert.NotNil(t, rateLimiter)
	assert.NotNil(t, rateLimiter.limiter)
}

func TestRateLimiter_RateLimit(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mockRateLimiterService)
		remoteAddr     string
		expectedStatus int
	}{
		{
			name: "Request allowed",
			setupMock: func(m *mockRateLimiterService) {
				m.On("Allow", mock.Anything).Return(true, nil)
			},
			remoteAddr:     "192.168.1.1:1234",
			expectedStatus: http.StatusOK,
		},
		{
			name: "Rate limit exceeded",
			setupMock: func(m *mockRateLimiterService) {
				m.On("Allow", mock.Anything).Return(false, nil)
			},
			remoteAddr:     "192.168.1.1:1234",
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name: "Service error",
			setupMock: func(m *mockRateLimiterService) {
				m.On("Allow", mock.Anything).Return(false, assert.AnError)
			},
			remoteAddr:     "192.168.1.1:1234",
			expectedStatus: http.StatusOK, // Should pass through on error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &mockRateLimiterService{}
			tt.setupMock(limiter)

			rateLimiter := &RateLimiter{
				limiter: limiter,
			}

			handler := rateLimiter.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			limiter.AssertExpectations(t)
		})
	}
}

func TestRateLimiter_WithHeaders(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mockRateLimiterService)
		headers        map[string]string
		expectedStatus int
	}{
		{
			name: "X-Forwarded-For header",
			setupMock: func(m *mockRateLimiterService) {
				m.On("Allow", mock.Anything).Return(true, nil)
			},
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "X-Real-IP header",
			setupMock: func(m *mockRateLimiterService) {
				m.On("Allow", mock.Anything).Return(true, nil)
			},
			headers: map[string]string{
				"X-Real-IP": "10.0.0.1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Multiple X-Forwarded-For IPs",
			setupMock: func(m *mockRateLimiterService) {
				m.On("Allow", mock.Anything).Return(true, nil)
			},
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1, 192.168.1.1",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &mockRateLimiterService{}
			tt.setupMock(limiter)

			rateLimiter := &RateLimiter{
				limiter: limiter,
			}

			handler := rateLimiter.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			limiter.AssertExpectations(t)
		})
	}
}
