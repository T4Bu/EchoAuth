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
	"time"

	"github.com/golang-jwt/jwt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserExists         = errors.New("user already exists")
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
}

func NewAuthService(userRepo repositories.UserRepository, tokenRepo repositories.TokenRepositoryInterface, cfg *config.Config, lockoutSvc *AccountLockoutService) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		jwtExpiry:     cfg.JWTExpiry,
		refreshExpiry: 30 * 24 * time.Hour, // 30 days
		jwtSecret:     cfg.JWTSecret,
		lockoutSvc:    lockoutSvc,
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
	// Validate token first
	claims, err := s.ValidateToken(token)
	if err != nil {
		return err
	}

	// Find user to get email
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return err
	}

	// Reset attempts on logout
	return s.lockoutSvc.ResetAttempts(context.Background(), user.Email)
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
