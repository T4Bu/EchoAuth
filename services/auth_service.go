package services

import (
	"EchoAuth/config"
	"EchoAuth/models"
	"EchoAuth/repositories"
	"EchoAuth/utils/validator"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserExists         = errors.New("user already exists")
	ErrTokenBlacklisted   = errors.New("token is blacklisted")
)

type AuthServiceInterface interface {
	Register(email, password, firstName, lastName string) error
	LoginWithRefresh(email, password, deviceInfo, ip string) (string, string, error)
	Logout(token string) error
	ValidateToken(token string) (*models.TokenClaims, error)
	RefreshToken(refreshToken, deviceInfo, ip string) (string, string, error)
	GetJWTExpiry() time.Duration
	GetUserByEmail(email string) (*models.User, error)
}

type AuthService struct {
	userRepo      repositories.UserRepository
	tokenRepo     repositories.TokenRepositoryInterface
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
	jwtSecret     string
	lockoutSvc    *AccountLockoutService
	redisClient   *redis.Client
}

func NewAuthService(userRepo repositories.UserRepository, tokenRepo repositories.TokenRepositoryInterface, cfg *config.Config, lockoutSvc *AccountLockoutService, redisClient *redis.Client) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		jwtExpiry:     cfg.JWTExpiry,
		refreshExpiry: 30 * 24 * time.Hour, // 30 days
		jwtSecret:     cfg.JWTSecret,
		lockoutSvc:    lockoutSvc,
		redisClient:   redisClient,
	}
}

func (s *AuthService) Register(email, password, firstName, lastName string) error {
	// Validate email
	if err := validator.ValidateEmail(email); err != nil {
		return err
	}

	// Validate password complexity
	if err := validator.ValidatePassword(password); err != nil {
		return err
	}

	// Check if user exists
	existingUser, err := s.userRepo.FindByEmail(email)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return err
	}
	if existingUser != nil {
		return ErrUserExists
	}

	// Create new user
	user := &models.User{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}

	// Hash password
	if err := user.HashPassword(password); err != nil {
		return err
	}

	return s.userRepo.Create(user)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	// Check if account is locked
	locked, err := s.lockoutSvc.IsLocked(ctx, email)
	if err != nil {
		return "", err
	}
	if locked {
		return "", ErrAccountLocked
	}

	// Find user
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			// Record failed attempt even if user doesn't exist
			_ = s.lockoutSvc.RecordFailedAttempt(ctx, email)
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	// Check password
	if !user.CheckPassword(password) {
		// Record failed attempt
		err = s.lockoutSvc.RecordFailedAttempt(ctx, email)
		if err != nil {
			return "", err
		}
		return "", ErrInvalidCredentials
	}

	// Reset failed attempts on successful login
	err = s.lockoutSvc.ResetAttempts(ctx, email)
	if err != nil {
		return "", err
	}

	// Generate JWT token
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.ID
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	claims["iat"] = time.Now().Unix()

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*models.TokenClaims, error) {
	// First check if token is blacklisted
	ctx := context.Background()
	exists, err := s.redisClient.Exists(ctx, fmt.Sprintf("blacklist:%s", tokenString)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if exists == 1 {
		return nil, ErrTokenBlacklisted
	}

	token, err := jwt.ParseWithClaims(tokenString, &models.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *AuthService) Logout(token string) error {
	// First validate the token
	claims, err := s.ValidateToken(token)
	if err != nil {
		return err
	}

	// Calculate token expiry
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // Token is already expired
	}

	// Add token to blacklist with TTL matching token expiry
	ctx := context.Background()
	key := fmt.Sprintf("blacklist:%s", token)
	err = s.redisClient.Set(ctx, key, true, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

func (s *AuthService) generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *AuthService) LoginWithRefresh(email, password string, deviceInfo, ip string) (string, string, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return "", "", err
	}

	if !user.CheckPassword(password) {
		return "", "", errors.New("invalid credentials")
	}

	// Generate access token
	accessToken, err := s.GenerateToken(user.ID)
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	// Store refresh token
	_, err = s.tokenRepo.CreateRefreshToken(
		user.ID,
		refreshToken,
		time.Now().Add(s.refreshExpiry),
		deviceInfo,
		ip,
	)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) RefreshToken(refreshToken, deviceInfo, ip string) (string, string, error) {
	// Get existing refresh token
	token, err := s.tokenRepo.GetRefreshToken(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	// Validate token
	if !token.IsValid() {
		return "", "", errors.New("refresh token is expired or revoked")
	}

	// Generate new tokens
	accessToken, err := s.GenerateToken(token.UserID)
	if err != nil {
		return "", "", err
	}

	newRefreshToken, err := s.generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	// Rotate refresh token
	_, err = s.tokenRepo.RotateRefreshToken(
		token,
		newRefreshToken,
		time.Now().Add(s.refreshExpiry),
	)
	if err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

func (s *AuthService) RevokeToken(refreshToken string) error {
	return s.tokenRepo.RevokeRefreshToken(refreshToken)
}

func (s *AuthService) RevokeAllUserTokens(userID uint) error {
	return s.tokenRepo.RevokeAllUserTokens(userID)
}

func (s *AuthService) LogoutWithRefresh(refreshToken string) error {
	return s.RevokeToken(refreshToken)
}

func (s *AuthService) GenerateToken(userID uint) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = userID
	claims["exp"] = time.Now().Add(s.jwtExpiry).Unix()
	claims["iat"] = time.Now().Unix()

	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) GetJWTExpiry() time.Duration {
	return s.jwtExpiry
}

func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
	return s.userRepo.FindByEmail(email)
}
