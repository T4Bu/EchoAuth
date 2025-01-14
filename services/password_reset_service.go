package services

import (
	"EchoAuth/models"
	"EchoAuth/repositories"
	"EchoAuth/utils/validator"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"time"
)

type PasswordResetService struct {
	userRepo     repositories.UserRepository
	emailService EmailService
}

func NewPasswordResetService(userRepo repositories.UserRepository, emailService EmailService) *PasswordResetService {
	return &PasswordResetService{
		userRepo:     userRepo,
		emailService: emailService,
	}
}

// GenerateResetToken creates a reset token for the user with the given email
func (s *PasswordResetService) GenerateResetToken(email string) (string, error) {
	// Validate email format
	if err := validator.ValidateEmail(email); err != nil {
		return "", err
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Generate random token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	// Set token expiration (24 hours from now)
	expiresAt := time.Now().Add(24 * time.Hour)
	user.PasswordResetToken = token
	user.ResetTokenExpiresAt = expiresAt

	if err := s.userRepo.Update(user); err != nil {
		return "", err
	}

	// Send reset email
	if err := s.emailService.SendPasswordResetEmail(email, token); err != nil {
		// Log the error but don't return it to avoid revealing user existence
		log.Printf("Failed to send password reset email: %v", err)
	}

	return token, nil
}

// ValidateResetToken checks if the reset token is valid and not expired
func (s *PasswordResetService) ValidateResetToken(token string) (*models.User, error) {
	if token == "" {
		return nil, errors.New("invalid token")
	}

	user, err := s.userRepo.FindByResetToken(token)
	if err != nil {
		return nil, errors.New("invalid token")
	}

	if user.ResetTokenExpiresAt.IsZero() || user.ResetTokenExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	return user, nil
}

// ResetPassword changes the user's password and invalidates the reset token
func (s *PasswordResetService) ResetPassword(token, newPassword string) error {
	// Validate new password first
	if err := validator.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Then validate token
	user, err := s.ValidateResetToken(token)
	if err != nil {
		return err
	}

	if err := user.HashPassword(newPassword); err != nil {
		return err
	}

	// Clear reset token
	user.PasswordResetToken = ""
	user.ResetTokenExpiresAt = time.Time{}

	return s.userRepo.Update(user)
}
