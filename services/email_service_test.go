package services

import (
	"errors"
	"strings"
	"testing"
)

var errMockSendFailed = errors.New("mock: failed to send email")

type mockEmailService struct {
	lastTo      string
	lastSubject string
	lastBody    string
	shouldFail  bool
}

func (m *mockEmailService) SendPasswordResetEmail(to, resetToken string) error {
	if m.shouldFail {
		return errMockSendFailed
	}
	m.lastTo = to
	m.lastSubject = "Password Reset Request"
	m.lastBody = resetToken
	return nil
}

func (m *mockEmailService) SendVerificationEmail(to, verificationToken string) error {
	if m.shouldFail {
		return errMockSendFailed
	}
	m.lastTo = to
	m.lastSubject = "Verify Your Email"
	m.lastBody = verificationToken
	return nil
}

func TestEmailService(t *testing.T) {
	mock := &mockEmailService{}

	tests := []struct {
		name      string
		to        string
		token     string
		shouldErr bool
		isReset   bool // true for password reset, false for verification
	}{
		{
			name:      "Valid password reset",
			to:        "test@example.com",
			token:     "reset-token",
			shouldErr: false,
			isReset:   true,
		},
		{
			name:      "Valid verification",
			to:        "test@example.com",
			token:     "verify-token",
			shouldErr: false,
			isReset:   false,
		},
		{
			name:      "Failed email",
			to:        "test@example.com",
			token:     "token",
			shouldErr: true,
			isReset:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.shouldFail = tt.shouldErr
			var err error
			if tt.isReset {
				err = mock.SendPasswordResetEmail(tt.to, tt.token)
				if !tt.shouldErr && !strings.Contains(mock.lastBody, tt.token) {
					t.Error("Password reset email should contain the token")
				}
			} else {
				err = mock.SendVerificationEmail(tt.to, tt.token)
				if !tt.shouldErr && !strings.Contains(mock.lastBody, tt.token) {
					t.Error("Verification email should contain the token")
				}
			}

			if (err != nil) != tt.shouldErr {
				t.Errorf("SendEmail() error = %v, shouldErr %v", err, tt.shouldErr)
			}

			if !tt.shouldErr {
				if mock.lastTo != tt.to {
					t.Errorf("Expected recipient %s, got %s", tt.to, mock.lastTo)
				}
				expectedSubject := "Password Reset Request"
				if !tt.isReset {
					expectedSubject = "Verify Your Email"
				}
				if mock.lastSubject != expectedSubject {
					t.Errorf("Expected subject %s, got %s", expectedSubject, mock.lastSubject)
				}
			}
		})
	}
}
