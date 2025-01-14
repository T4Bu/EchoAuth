package middlewares

import (
	"EchoAuth/models"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockAuthService struct{}

func (m *mockAuthService) Register(email, password, firstName, lastName string) error {
	return nil
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (string, error) {
	return "", nil
}

func (m *mockAuthService) ValidateToken(token string) (*models.TokenClaims, error) {
	if token == "valid-token" {
		return &models.TokenClaims{
			UserID: 1,
		}, nil
	}
	return nil, errors.New("invalid token")
}

func (m *mockAuthService) Logout(token string) error {
	if token == "valid-token" {
		return nil
	}
	return errors.New("invalid token")
}

func TestAuthMiddleware(t *testing.T) {
	mockService := &mockAuthService{}
	middleware := NewAuthMiddleware(mockService)
	handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "Valid token",
			token:          "valid-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid token",
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}
