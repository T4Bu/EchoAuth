package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"EchoAuth/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) Register(email, password, firstName, lastName string) error {
	args := m.Called(email, password, firstName, lastName)
	return args.Error(0)
}

func (m *mockAuthService) LoginWithRefresh(email, password, deviceInfo, ip string) (string, string, error) {
	args := m.Called(email, password, deviceInfo, ip)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockAuthService) Logout(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *mockAuthService) ValidateToken(token string) (*models.TokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TokenClaims), args.Error(1)
}

func (m *mockAuthService) RefreshToken(refreshToken, deviceInfo, ip string) (string, string, error) {
	args := m.Called(refreshToken, deviceInfo, ip)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockAuthService) GetJWTExpiry() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *mockAuthService) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockAuthService) LogoutWithRefresh(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func TestAuthControllerRegister(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(mockService *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid registration",
			requestBody: RegisterRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User",
			},
			setupMock: func(mockService *mockAuthService) {
				mockService.On("Register", "test@example.com", "password123", "Test", "User").
					Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"message":"User registered successfully"}`,
		},
		{
			name:           "Invalid request body",
			requestBody:    "invalid json",
			setupMock:      func(mockService *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid request body"}`,
		},
		{
			name: "Missing required fields",
			requestBody: RegisterRequest{
				Email:     "",
				Password:  "",
				FirstName: "",
				LastName:  "",
			},
			setupMock:      func(mock *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag\nKey: 'RegisterRequest.Password' Error:Field validation for 'Password' failed on the 'required' tag"}`,
		},
		{
			name: "User already exists",
			requestBody: RegisterRequest{
				Email:     "existing@example.com",
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User",
			},
			setupMock: func(mockService *mockAuthService) {
				mockService.On("Register", "existing@example.com", "password123", "Test", "User").
					Return(models.ErrUserExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"user already exists"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mockAuthService)
			tt.setupMock(mockService)
			controller := NewAuthController(mockService)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			w := httptest.NewRecorder()

			controller.Register(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthControllerLogin(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(mockService *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid credentials",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMock: func(mockService *mockAuthService) {
				mockService.On("LoginWithRefresh", "test@example.com", "password123", "test-user-agent", "127.0.0.1").
					Return("access-token", "refresh-token", nil)
				mockService.On("GetJWTExpiry").Return(time.Hour * 24)
				mockService.On("GetUserByEmail", "test@example.com").
					Return(&models.User{
						ID:        1,
						Email:     "test@example.com",
						FirstName: "Test",
						LastName:  "User",
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"access_token":"access-token","refresh_token":"refresh-token","token_type":"Bearer","expires_in":86400,"user":{"id":1,"email":"test@example.com","first_name":"Test","last_name":"User","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}}`,
		},
		{
			name:           "Invalid request body",
			requestBody:    "invalid json",
			setupMock:      func(mockService *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid request body"}`,
		},
		{
			name: "Invalid credentials",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpass",
			},
			setupMock: func(mockService *mockAuthService) {
				mockService.On("LoginWithRefresh", "test@example.com", "wrongpass", "test-user-agent", "127.0.0.1").
					Return("", "", errors.New("invalid credentials"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid credentials"}`,
		},
		{
			name: "Missing email",
			requestBody: LoginRequest{
				Password: "password123",
			},
			setupMock:      func(mockService *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"}`,
		},
		{
			name: "Invalid email format",
			requestBody: LoginRequest{
				Email:    "invalid-email",
				Password: "password123",
			},
			setupMock:      func(mockService *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mockAuthService)
			tt.setupMock(mockService)
			controller := NewAuthController(mockService)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("User-Agent", "test-user-agent")
			req.RemoteAddr = "127.0.0.1"
			w := httptest.NewRecorder()

			controller.Login(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthControllerLogout(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupAuth      string
		setupMock      func(mockService *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid logout",
			requestBody: RefreshTokenRequest{
				RefreshToken: "test-refresh-token",
			},
			setupAuth: "Bearer test-access-token",
			setupMock: func(mockService *mockAuthService) {
				mockService.On("Logout", "test-access-token").Return(nil)
				mockService.On("LogoutWithRefresh", "test-refresh-token").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Successfully logged out"}`,
		},
		{
			name:           "Invalid request body",
			requestBody:    "invalid json",
			setupAuth:      "Bearer test-access-token",
			setupMock:      func(mockService *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid request body"}`,
		},
		{
			name: "Missing auth header",
			requestBody: RefreshTokenRequest{
				RefreshToken: "test-refresh-token",
			},
			setupAuth:      "",
			setupMock:      func(mockService *mockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authorization header required"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mockAuthService)
			tt.setupMock(mockService)
			controller := NewAuthController(mockService)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
			if tt.setupAuth != "" {
				req.Header.Set("Authorization", tt.setupAuth)
			}
			w := httptest.NewRecorder()

			controller.Logout(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthControllerRefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(mockService *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid refresh token",
			requestBody: RefreshTokenRequest{
				RefreshToken: "valid_refresh_token",
			},
			setupMock: func(mockService *mockAuthService) {
				mockService.On("RefreshToken", "valid_refresh_token", "test-user-agent", "127.0.0.1").
					Return("new_access_token", "new_refresh_token", nil)
				mockService.On("GetJWTExpiry").Return(time.Hour * 24)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"access_token":"new_access_token","refresh_token":"new_refresh_token","token_type":"Bearer","expires_in":86400}`,
		},
		{
			name:           "Invalid request body",
			requestBody:    "invalid json",
			setupMock:      func(mockService *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid request body"}`,
		},
		{
			name: "Missing refresh token",
			requestBody: map[string]string{
				"refresh_token": "",
			},
			setupMock:      func(mockService *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Key: 'RefreshTokenRequest.RefreshToken' Error:Field validation for 'RefreshToken' failed on the 'required' tag"}`,
		},
		{
			name: "Invalid refresh token",
			requestBody: RefreshTokenRequest{
				RefreshToken: "invalid_token",
			},
			setupMock: func(mockService *mockAuthService) {
				mockService.On("RefreshToken", "invalid_token", "test-user-agent", "127.0.0.1").
					Return("", "", errors.New("invalid token"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid or expired refresh token"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mockAuthService)
			tt.setupMock(mockService)
			controller := NewAuthController(mockService)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
			req.Header.Set("User-Agent", "test-user-agent")
			req.RemoteAddr = "127.0.0.1"
			w := httptest.NewRecorder()

			controller.RefreshToken(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}
