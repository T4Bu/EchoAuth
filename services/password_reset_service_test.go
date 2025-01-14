package services

import (
	"EchoAuth/models"
	"EchoAuth/utils/validator"
	"errors"
	"testing"
	"time"
)

type mockResetRepo struct {
	users map[string]*models.User
}

func newMockResetRepo() *mockResetRepo {
	return &mockResetRepo{
		users: make(map[string]*models.User),
	}
}

func (m *mockResetRepo) Create(user *models.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockResetRepo) FindByEmail(email string) (*models.User, error) {
	if user, exists := m.users[email]; exists {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockResetRepo) FindByID(id uint) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockResetRepo) FindByResetToken(token string) (*models.User, error) {
	for _, user := range m.users {
		if user.PasswordResetToken == token {
			return user, nil
		}
	}
	return nil, errors.New("token not found")
}

func (m *mockResetRepo) Update(user *models.User) error {
	if _, exists := m.users[user.Email]; !exists {
		return errors.New("user not found")
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockResetRepo) Delete(id uint) error {
	for email, user := range m.users {
		if user.ID == id {
			delete(m.users, email)
			return nil
		}
	}
	return errors.New("user not found")
}

func TestPasswordResetService_GenerateResetToken(t *testing.T) {
	repo := newMockResetRepo()
	service := NewPasswordResetService(repo, &mockEmailService{})

	// Create test user
	user := &models.User{
		Email:    "test@example.com",
		Password: "old_password",
	}
	repo.Create(user)

	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "Valid email",
			email:   "test@example.com",
			wantErr: nil,
		},
		{
			name:    "Empty email",
			email:   "",
			wantErr: validator.ErrEmailEmpty,
		},
		{
			name:    "Invalid email format",
			email:   "invalid-email",
			wantErr: validator.ErrEmailInvalid,
		},
		{
			name:    "Invalid email domain",
			email:   "test@.com",
			wantErr: validator.ErrDomainInvalid,
		},
		{
			name:    "Non-existent email",
			email:   "nonexistent@example.com",
			wantErr: errors.New("user not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.GenerateResetToken(tt.email)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("GenerateResetToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == nil && token == "" {
				t.Error("GenerateResetToken() returned empty token for valid email")
			}
		})
	}
}

func TestPasswordResetService_ValidateResetToken(t *testing.T) {
	repo := newMockResetRepo()
	service := NewPasswordResetService(repo, &mockEmailService{})

	// Create test user with valid token
	user := &models.User{
		Email:               "test@example.com",
		Password:            "old_password",
		PasswordResetToken:  "valid-token",
		ResetTokenExpiresAt: time.Now().Add(time.Hour),
	}
	repo.Create(user)

	// Create user with expired token
	expiredUser := &models.User{
		Email:               "expired@example.com",
		Password:            "old_password",
		PasswordResetToken:  "expired-token",
		ResetTokenExpiresAt: time.Now().Add(-time.Hour),
	}
	repo.Create(expiredUser)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "Valid token",
			token:   "valid-token",
			wantErr: false,
		},
		{
			name:    "Expired token",
			token:   "expired-token",
			wantErr: true,
		},
		{
			name:    "Non-existent token",
			token:   "non-existent",
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
			user, err := service.ValidateResetToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResetToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && user == nil {
				t.Error("Expected user for valid token")
			}
		})
	}
}

func TestPasswordResetService_ResetPassword(t *testing.T) {
	repo := newMockResetRepo()
	service := NewPasswordResetService(repo, &mockEmailService{})

	// Create test user with valid token
	user := &models.User{
		Email:               "test@example.com",
		Password:            "old_password",
		PasswordResetToken:  "valid-token",
		ResetTokenExpiresAt: time.Now().Add(time.Hour),
	}
	repo.Create(user)

	tests := []struct {
		name        string
		token       string
		newPassword string
		wantErr     error
	}{
		{
			name:        "Valid reset",
			token:       "valid-token",
			newPassword: "NewPassword123!",
			wantErr:     nil,
		},
		{
			name:        "Invalid token",
			token:       "invalid-token",
			newPassword: "NewPassword123!",
			wantErr:     errors.New("invalid token"),
		},
		{
			name:        "Empty password",
			token:       "valid-token",
			newPassword: "",
			wantErr:     validator.ErrPasswordTooShort,
		},
		{
			name:        "Weak password",
			token:       "valid-token",
			newPassword: "weak",
			wantErr:     validator.ErrPasswordTooShort,
		},
		{
			name:        "Common password",
			token:       "valid-token",
			newPassword: "password123",
			wantErr:     validator.ErrPasswordCommon,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ResetPassword(tt.token, tt.newPassword)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("ResetPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
