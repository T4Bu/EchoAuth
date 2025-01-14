package controllers

import (
	"auth/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockAuthService struct {
	registerError error
	loginToken    string
	loginError    error
}

func (m *mockAuthService) Register(email, password, firstName, lastName string) error {
	return m.registerError
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (string, error) {
	if m.loginError != nil {
		return "", m.loginError
	}
	return m.loginToken, nil
}

func (m *mockAuthService) Logout(token string) error {
	return nil
}

func (m *mockAuthService) ValidateToken(token string) (*models.TokenClaims, error) {
	return nil, nil
}

func (m *mockAuthService) GetJWTExpiry() time.Duration {
	return 24 * time.Hour
}

func (m *mockAuthService) GetUserByEmail(email string) (*models.User, error) {
	return &models.User{
		Email: email,
		ID:    1,
	}, nil
}

func (m *mockAuthService) LoginWithRefresh(email, password, deviceInfo, ip string) (string, string, error) {
	if m.loginError != nil {
		return "", "", m.loginError
	}
	return m.loginToken, "refresh-token", nil
}

func (m *mockAuthService) RefreshToken(refreshToken, deviceInfo, ip string) (string, string, error) {
	if m.loginError != nil {
		return "", "", m.loginError
	}
	return m.loginToken, "new-refresh-token", nil
}

func TestAuthControllerRegister(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    RegisterRequest
		serviceError   error
		expectedStatus int
	}{
		{
			name: "Successful registration",
			requestBody: RegisterRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			serviceError:   nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid email",
			requestBody: RegisterRequest{
				Email:     "",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			serviceError:   nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockAuthService{registerError: tt.serviceError}
			controller := NewAuthController(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			controller.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAuthControllerLogin(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    LoginRequest
		mockToken      string
		mockError      error
		expectedStatus int
	}{
		{
			name: "Successful login",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockToken:      "valid-token",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid credentials",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockToken:      "",
			mockError:      errors.New("invalid credentials"),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockAuthService{
				loginToken: tt.mockToken,
				loginError: tt.mockError,
			}
			controller := NewAuthController(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			controller.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.mockError == nil {
				var response LoginResponse
				json.NewDecoder(w.Body).Decode(&response)
				if response.AccessToken != tt.mockToken {
					t.Errorf("Expected token %s, got %s", tt.mockToken, response.AccessToken)
				}
			}
		})
	}
}

func TestAuthControllerLogout(t *testing.T) {
	mockService := &mockAuthService{}
	controller := NewAuthController(mockService)

	// Create request body with refresh token
	body, _ := json.Marshal(map[string]string{
		"refresh_token": "test-refresh-token",
	})
	req := httptest.NewRequest("POST", "/api/auth/logout", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}
