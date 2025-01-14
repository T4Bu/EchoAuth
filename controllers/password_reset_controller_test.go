package controllers

import (
	"EchoAuth/models"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockPasswordResetService is a mock implementation of the password reset service
type mockPasswordResetService struct {
	generateTokenFunc func(email string) (string, error)
	validateTokenFunc func(token string) (*models.User, error)
	resetPasswordFunc func(token, newPassword string) error
}

func (m *mockPasswordResetService) GenerateResetToken(email string) (string, error) {
	return m.generateTokenFunc(email)
}

func (m *mockPasswordResetService) ValidateResetToken(token string) (*models.User, error) {
	return m.validateTokenFunc(token)
}

func (m *mockPasswordResetService) ResetPassword(token, newPassword string) error {
	return m.resetPasswordFunc(token, newPassword)
}

func TestNewPasswordResetController(t *testing.T) {
	mockService := &mockPasswordResetService{
		generateTokenFunc: func(email string) (string, error) { return "", nil },
		validateTokenFunc: func(token string) (*models.User, error) { return nil, nil },
		resetPasswordFunc: func(token, newPassword string) error { return nil },
	}
	controller := NewPasswordResetController(mockService)

	if controller == nil {
		t.Fatal("Expected non-nil controller")
	}
	if controller.resetService == nil {
		t.Error("Expected controller to have a non-nil service")
	}
}

func TestPasswordResetController_RequestReset(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(m *mockPasswordResetService)
		wantStatusCode int
		wantResponse   map[string]string
		description    string
		contentType    string
	}{
		{
			name: "Valid request",
			requestBody: RequestResetRequest{
				Email: "test@example.com",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.generateTokenFunc = func(email string) (string, error) {
					return "valid-token", nil
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: map[string]string{
				"token":   "valid-token",
				"message": "Reset token generated successfully",
			},
			description: "Should successfully generate a reset token",
			contentType: "application/json",
		},
		{
			name:        "Invalid request body",
			requestBody: "invalid json",
			setupMock: func(m *mockPasswordResetService) {
				m.generateTokenFunc = func(email string) (string, error) {
					return "", nil
				}
			},
			wantStatusCode: http.StatusBadRequest,
			description:    "Should handle invalid request body",
			contentType:    "application/json",
		},
		{
			name: "Empty email",
			requestBody: RequestResetRequest{
				Email: "",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.generateTokenFunc = func(email string) (string, error) {
					return "", errors.New("email is required")
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: map[string]string{
				"message": "If your email is registered, you will receive a reset link shortly",
			},
			description: "Should handle empty email gracefully",
			contentType: "application/json",
		},
		{
			name: "Invalid email format",
			requestBody: RequestResetRequest{
				Email: "invalid-email",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.generateTokenFunc = func(email string) (string, error) {
					return "", errors.New("invalid email format")
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: map[string]string{
				"message": "If your email is registered, you will receive a reset link shortly",
			},
			description: "Should handle invalid email format gracefully",
			contentType: "application/json",
		},
		{
			name: "Service error",
			requestBody: RequestResetRequest{
				Email: "test@example.com",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.generateTokenFunc = func(email string) (string, error) {
					return "", errors.New("service error")
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: map[string]string{
				"message": "If your email is registered, you will receive a reset link shortly",
			},
			description: "Should handle service error gracefully",
			contentType: "application/json",
		},
		{
			name: "Invalid content type",
			requestBody: RequestResetRequest{
				Email: "test@example.com",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.generateTokenFunc = func(email string) (string, error) {
					return "", nil
				}
			},
			wantStatusCode: http.StatusBadRequest,
			description:    "Should reject non-JSON content type",
			contentType:    "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPasswordResetService{
				generateTokenFunc: func(email string) (string, error) { return "", nil },
				validateTokenFunc: func(token string) (*models.User, error) { return nil, nil },
				resetPasswordFunc: func(token, newPassword string) error { return nil },
			}
			tt.setupMock(mockService)
			controller := NewPasswordResetController(mockService)

			var body bytes.Buffer
			if err := json.NewEncoder(&body).Encode(tt.requestBody); err != nil {
				t.Fatalf("Failed to encode request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/reset-password", &body)
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()

			controller.RequestReset(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("RequestReset() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			if tt.wantResponse != nil {
				var got map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if got["message"] != tt.wantResponse["message"] {
					t.Errorf("RequestReset() message = %v, want %v", got["message"], tt.wantResponse["message"])
				}
				if token, exists := tt.wantResponse["token"]; exists && got["token"] != token {
					t.Errorf("RequestReset() token = %v, want %v", got["token"], token)
				}
			}
		})
	}
}

func TestPasswordResetController_ResetPassword(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(m *mockPasswordResetService)
		wantStatusCode int
		wantResponse   map[string]string
		wantErrorMsg   string
		description    string
		contentType    string
	}{
		{
			name: "Valid reset",
			requestBody: ResetPasswordRequest{
				Token:       "valid-token",
				NewPassword: "newpassword123",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.resetPasswordFunc = func(token, newPassword string) error {
					return nil
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: map[string]string{
				"message": "Password reset successfully",
			},
			description: "Should successfully reset password",
			contentType: "application/json",
		},
		{
			name:        "Invalid request body",
			requestBody: "invalid json",
			setupMock: func(m *mockPasswordResetService) {
				m.resetPasswordFunc = func(token, newPassword string) error {
					return nil
				}
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrorMsg:   "Invalid request body",
			description:    "Should handle invalid request body",
			contentType:    "application/json",
		},
		{
			name: "Empty password",
			requestBody: ResetPasswordRequest{
				Token:       "valid-token",
				NewPassword: "",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.resetPasswordFunc = func(token, newPassword string) error {
					return nil
				}
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrorMsg:   "New password is required",
			description:    "Should handle empty password",
			contentType:    "application/json",
		},
		{
			name: "Missing token",
			requestBody: ResetPasswordRequest{
				Token:       "",
				NewPassword: "newpassword123",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.resetPasswordFunc = func(token, newPassword string) error {
					return errors.New("token is required")
				}
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrorMsg:   "token is required",
			description:    "Should handle missing token",
			contentType:    "application/json",
		},
		{
			name: "Invalid token",
			requestBody: ResetPasswordRequest{
				Token:       "invalid-token",
				NewPassword: "newpassword123",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.resetPasswordFunc = func(token, newPassword string) error {
					return errors.New("invalid or expired token")
				}
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrorMsg:   "invalid or expired token",
			description:    "Should handle invalid token",
			contentType:    "application/json",
		},
		{
			name: "Invalid content type",
			requestBody: ResetPasswordRequest{
				Token:       "valid-token",
				NewPassword: "newpassword123",
			},
			setupMock: func(m *mockPasswordResetService) {
				m.resetPasswordFunc = func(token, newPassword string) error {
					return nil
				}
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrorMsg:   "Invalid request body",
			description:    "Should reject non-JSON content type",
			contentType:    "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPasswordResetService{
				generateTokenFunc: func(email string) (string, error) { return "", nil },
				validateTokenFunc: func(token string) (*models.User, error) { return nil, nil },
				resetPasswordFunc: func(token, newPassword string) error { return nil },
			}
			tt.setupMock(mockService)
			controller := NewPasswordResetController(mockService)

			var body bytes.Buffer
			if err := json.NewEncoder(&body).Encode(tt.requestBody); err != nil {
				t.Fatalf("Failed to encode request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/reset-password", &body)
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()

			controller.ResetPassword(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("ResetPassword() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			if tt.wantResponse != nil {
				var got map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if got["message"] != tt.wantResponse["message"] {
					t.Errorf("ResetPassword() message = %v, want %v", got["message"], tt.wantResponse["message"])
				}
			}

			if tt.wantErrorMsg != "" {
				gotBody := rec.Body.String()
				if !strings.Contains(gotBody, tt.wantErrorMsg) {
					t.Errorf("ResetPassword() error message = %v, want to contain %v", gotBody, tt.wantErrorMsg)
				}
			}
		})
	}
}
