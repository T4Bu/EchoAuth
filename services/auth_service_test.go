package services

import (
	"EchoAuth/config"
	"EchoAuth/models"
	"EchoAuth/repositories"
	"EchoAuth/utils/validator"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt"
	"github.com/redis/go-redis/v9"
)

type mockUserRepository struct {
	users map[uint]*models.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[uint]*models.User),
	}
}

func (m *mockUserRepository) Create(user *models.User) error {
	if user.ID == 0 {
		user.ID = uint(len(m.users) + 1)
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) FindByEmail(email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (m *mockUserRepository) FindByID(id uint) (*models.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, repositories.ErrNotFound
}

func (m *mockUserRepository) FindByResetToken(token string) (*models.User, error) {
	return nil, repositories.ErrNotFound
}

func (m *mockUserRepository) Update(user *models.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return repositories.ErrNotFound
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(id uint) error {
	if _, exists := m.users[id]; !exists {
		return repositories.ErrNotFound
	}
	delete(m.users, id)
	return nil
}

func newMockRedis() *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	return redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
}

func newMockAccountLockoutService() *AccountLockoutService {
	return &AccountLockoutService{
		redis:         newMockRedis(),
		maxAttempts:   5,
		lockDuration:  15 * time.Minute,
		attemptExpiry: 1 * time.Hour,
	}
}

type mockTokenRepository struct {
	tokens map[string]*models.RefreshToken
}

func newMockTokenRepository() repositories.TokenRepositoryInterface {
	return &mockTokenRepository{
		tokens: make(map[string]*models.RefreshToken),
	}
}

func (m *mockTokenRepository) CreateRefreshToken(userID uint, token string, expiresAt time.Time, deviceInfo, ip string) (*models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{
		UserID:     userID,
		Token:      token,
		ExpiresAt:  expiresAt,
		DeviceInfo: deviceInfo,
		IP:         ip,
	}
	m.tokens[token] = refreshToken
	return refreshToken, nil
}

func (m *mockTokenRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	if t, exists := m.tokens[token]; exists {
		return t, nil
	}
	return nil, repositories.ErrNotFound
}

func (m *mockTokenRepository) RotateRefreshToken(oldToken *models.RefreshToken, newToken string, expiresAt time.Time) (*models.RefreshToken, error) {
	// Mark old token as used
	oldToken.Used = true
	m.tokens[oldToken.Token] = oldToken

	// Create new token
	refreshToken := &models.RefreshToken{
		UserID:     oldToken.UserID,
		Token:      newToken,
		ExpiresAt:  expiresAt,
		DeviceInfo: oldToken.DeviceInfo,
		IP:         oldToken.IP,
	}
	m.tokens[newToken] = refreshToken
	return refreshToken, nil
}

func (m *mockTokenRepository) RevokeRefreshToken(token string) error {
	if t, exists := m.tokens[token]; exists {
		now := time.Now()
		t.RevokedAt = &now
		m.tokens[token] = t
		return nil
	}
	return repositories.ErrNotFound
}

func (m *mockTokenRepository) RevokeAllUserTokens(userID uint) error {
	now := time.Now()
	for _, t := range m.tokens {
		if t.UserID == userID {
			t.RevokedAt = &now
			m.tokens[t.Token] = t
		}
	}
	return nil
}

func (m *mockTokenRepository) CleanupExpiredTokens() error {
	for token, t := range m.tokens {
		if t.ExpiresAt.Before(time.Now()) || t.Used || t.RevokedAt != nil {
			delete(m.tokens, token)
		}
	}
	return nil
}

func TestAuthServiceRegister(t *testing.T) {
	repo := newMockUserRepository()
	tokenRepo := repositories.NewTokenRepository(nil)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpiry: 24 * time.Hour}
	lockoutService := newMockAccountLockoutService()
	redisClient := newMockRedis()
	service := NewAuthService(repo, tokenRepo, cfg, lockoutService, redisClient)

	tests := []struct {
		name      string
		email     string
		password  string
		firstName string
		lastName  string
		wantErr   error
	}{
		{
			name:      "Valid registration",
			email:     "test@example.com",
			password:  "Password123!",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   nil,
		},
		{
			name:      "Empty email",
			email:     "",
			password:  "Password123!",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   validator.ErrEmailEmpty,
		},
		{
			name:      "Invalid email format",
			email:     "invalid-email",
			password:  "Password123!",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   validator.ErrEmailInvalid,
		},
		{
			name:      "Invalid email domain",
			email:     "test@.com",
			password:  "Password123!",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   validator.ErrDomainInvalid,
		},
		{
			name:      "Empty password",
			email:     "test@example.com",
			password:  "",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   validator.ErrPasswordTooShort,
		},
		{
			name:      "Weak password",
			email:     "test@example.com",
			password:  "password",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   validator.ErrPasswordTooSimple,
		},
		{
			name:      "Common password",
			email:     "test@example.com",
			password:  "password123",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   validator.ErrPasswordCommon,
		},
		{
			name:      "User already exists",
			email:     "existing@example.com",
			password:  "Password123!",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   ErrUserExists,
		},
	}

	// Create an existing user for the "User already exists" test
	existingUser := &models.User{
		Email:     "existing@example.com",
		Password:  "Password123!",
		FirstName: "John",
		LastName:  "Doe",
	}
	repo.Create(existingUser)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Register(tt.email, tt.password, tt.firstName, tt.lastName)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AuthService.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthServiceLogin(t *testing.T) {
	repo := newMockUserRepository()
	tokenRepo := repositories.NewTokenRepository(nil)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpiry: 24 * time.Hour}
	lockoutService := newMockAccountLockoutService()
	redisClient := newMockRedis()
	service := NewAuthService(repo, tokenRepo, cfg, lockoutService, redisClient)

	// Create a test user
	testUser := &models.User{
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "John",
		LastName:  "Doe",
	}
	testUser.HashPassword(testUser.Password)
	repo.Create(testUser)

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{
			name:     "Valid login",
			email:    "test@example.com",
			password: "Password123!",
			wantErr:  false,
		},
		{
			name:     "Invalid email",
			email:    "wrong@example.com",
			password: "Password123!",
			wantErr:  true,
		},
		{
			name:     "Invalid password",
			email:    "test@example.com",
			password: "WrongPassword123!",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.Login(context.Background(), tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("AuthService.Login() returned empty token for valid credentials")
			}
		})
	}
}

func TestAuthServiceValidateToken(t *testing.T) {
	repo := newMockUserRepository()
	tokenRepo := repositories.NewTokenRepository(nil)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpiry: 24 * time.Hour}
	lockoutService := newMockAccountLockoutService()
	redisClient := newMockRedis()
	service := NewAuthService(repo, tokenRepo, cfg, lockoutService, redisClient)

	// Create a valid token
	claims := &models.TokenClaims{
		UserID: 1,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	validToken, _ := token.SignedString([]byte(cfg.JWTSecret))

	// Create an expired token
	expiredClaims := &models.TokenClaims{
		UserID: 1,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(-time.Hour).Unix(),
			IssuedAt:  time.Now().Add(-time.Hour * 2).Unix(),
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString([]byte(cfg.JWTSecret))

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "Valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "Expired token",
			token:   expiredTokenString,
			wantErr: true,
		},
		{
			name:    "Invalid token",
			token:   "invalid-token",
			wantErr: true,
		},
		{
			name:    "Empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && claims == nil {
				t.Error("AuthService.ValidateToken() returned nil claims for valid token")
			}
		})
	}
}

func TestAuthServiceLoginWithRefresh(t *testing.T) {
	userRepo := newMockUserRepository()
	tokenRepo := newMockTokenRepository()
	cfg := &config.Config{
		JWTSecret: "test-secret",
		JWTExpiry: 24 * time.Hour,
	}
	lockoutService := newMockAccountLockoutService()
	redisClient := newMockRedis()
	service := NewAuthService(userRepo, tokenRepo, cfg, lockoutService, redisClient)

	// Create a test user
	testUser := &models.User{
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "John",
		LastName:  "Doe",
	}
	testUser.HashPassword(testUser.Password)
	userRepo.Create(testUser)

	tests := []struct {
		name       string
		email      string
		password   string
		deviceInfo string
		ip         string
		wantErr    bool
	}{
		{
			name:       "Valid login",
			email:      "test@example.com",
			password:   "Password123!",
			deviceInfo: "Chrome on macOS",
			ip:         "127.0.0.1",
			wantErr:    false,
		},
		{
			name:       "Invalid email",
			email:      "wrong@example.com",
			password:   "Password123!",
			deviceInfo: "Chrome on macOS",
			ip:         "127.0.0.1",
			wantErr:    true,
		},
		{
			name:       "Invalid password",
			email:      "test@example.com",
			password:   "WrongPassword123!",
			deviceInfo: "Chrome on macOS",
			ip:         "127.0.0.1",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessToken, refreshToken, err := service.LoginWithRefresh(tt.email, tt.password, tt.deviceInfo, tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.LoginWithRefresh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if accessToken == "" {
					t.Error("AuthService.LoginWithRefresh() returned empty access token for valid credentials")
				}
				if refreshToken == "" {
					t.Error("AuthService.LoginWithRefresh() returned empty refresh token for valid credentials")
				}
			}
		})
	}
}

func TestAuthServiceRefreshToken(t *testing.T) {
	userRepo := newMockUserRepository()
	tokenRepo := newMockTokenRepository()
	cfg := &config.Config{
		JWTSecret: "test-secret",
		JWTExpiry: 24 * time.Hour,
	}
	lockoutService := newMockAccountLockoutService()
	redisClient := newMockRedis()
	service := NewAuthService(userRepo, tokenRepo, cfg, lockoutService, redisClient)

	// Create a test user
	testUser := &models.User{
		ID:        1,
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "John",
		LastName:  "Doe",
	}
	userRepo.Create(testUser)

	// Create a valid refresh token
	validToken := &models.RefreshToken{
		UserID:     testUser.ID,
		Token:      "valid-refresh-token",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		DeviceInfo: "Chrome on macOS",
		IP:         "127.0.0.1",
	}
	tokenRepo.(*mockTokenRepository).tokens[validToken.Token] = validToken

	// Create an expired refresh token
	expiredToken := &models.RefreshToken{
		UserID:     testUser.ID,
		Token:      "expired-refresh-token",
		ExpiresAt:  time.Now().Add(-24 * time.Hour),
		DeviceInfo: "Chrome on macOS",
		IP:         "127.0.0.1",
	}
	tokenRepo.(*mockTokenRepository).tokens[expiredToken.Token] = expiredToken

	// Create a revoked refresh token
	now := time.Now()
	revokedToken := &models.RefreshToken{
		UserID:     testUser.ID,
		Token:      "revoked-refresh-token",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		DeviceInfo: "Chrome on macOS",
		IP:         "127.0.0.1",
		RevokedAt:  &now,
	}
	tokenRepo.(*mockTokenRepository).tokens[revokedToken.Token] = revokedToken

	tests := []struct {
		name       string
		token      string
		deviceInfo string
		ip         string
		wantErr    bool
	}{
		{
			name:       "Valid refresh token",
			token:      "valid-refresh-token",
			deviceInfo: "Chrome on macOS",
			ip:         "127.0.0.1",
			wantErr:    false,
		},
		{
			name:       "Invalid refresh token",
			token:      "invalid-refresh-token",
			deviceInfo: "Chrome on macOS",
			ip:         "127.0.0.1",
			wantErr:    true,
		},
		{
			name:       "Expired refresh token",
			token:      "expired-refresh-token",
			deviceInfo: "Chrome on macOS",
			ip:         "127.0.0.1",
			wantErr:    true,
		},
		{
			name:       "Revoked refresh token",
			token:      "revoked-refresh-token",
			deviceInfo: "Chrome on macOS",
			ip:         "127.0.0.1",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessToken, newRefreshToken, err := service.RefreshToken(tt.token, tt.deviceInfo, tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.RefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if accessToken == "" {
					t.Error("AuthService.RefreshToken() returned empty access token for valid refresh token")
				}
				if newRefreshToken == "" {
					t.Error("AuthService.RefreshToken() returned empty refresh token for valid refresh token")
				}
				if newRefreshToken == tt.token {
					t.Error("AuthService.RefreshToken() returned same refresh token instead of rotating it")
				}
			}
		})
	}
}

func TestAuthServiceRevokeToken(t *testing.T) {
	userRepo := newMockUserRepository()
	tokenRepo := newMockTokenRepository()
	cfg := &config.Config{
		JWTSecret: "test-secret",
		JWTExpiry: 24 * time.Hour,
	}
	lockoutService := newMockAccountLockoutService()
	redisClient := newMockRedis()
	service := NewAuthService(userRepo, tokenRepo, cfg, lockoutService, redisClient)

	// Create a valid refresh token
	validToken := &models.RefreshToken{
		UserID:     1,
		Token:      "valid-refresh-token",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		DeviceInfo: "Chrome on macOS",
		IP:         "127.0.0.1",
	}
	tokenRepo.(*mockTokenRepository).tokens[validToken.Token] = validToken

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "Valid token",
			token:   "valid-refresh-token",
			wantErr: false,
		},
		{
			name:    "Invalid token",
			token:   "invalid-refresh-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RevokeToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.RevokeToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				token, _ := tokenRepo.(*mockTokenRepository).GetRefreshToken(tt.token)
				if token.RevokedAt == nil {
					t.Error("AuthService.RevokeToken() did not revoke the token")
				}
			}
		})
	}
}
