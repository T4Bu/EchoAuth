package services

import (
	"auth/models"
	"auth/repositories"
	"auth/utils/validator"
	"context"
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
	Login(ctx context.Context, email, password string) (string, error)
	ValidateToken(token string) (*models.TokenClaims, error)
	Logout(token string) error
}

type AuthService struct {
	userRepo   repositories.UserRepository
	jwtSecret  []byte
	lockoutSvc *AccountLockoutService
}

func NewAuthService(userRepo repositories.UserRepository, jwtSecret []byte, lockoutSvc *AccountLockoutService) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtSecret:  jwtSecret,
		lockoutSvc: lockoutSvc,
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

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*models.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
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
