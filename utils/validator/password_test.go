package validator

import (
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "Valid password",
			password: "SecurePass123!",
			wantErr:  nil,
		},
		{
			name:     "Password too short",
			password: "Pass1!",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "Password missing uppercase",
			password: "password123!",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Password missing lowercase",
			password: "PASSWORD123!",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Password missing number",
			password: "PasswordTest!",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Password missing special character",
			password: "SecurePass123",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Common password exact match",
			password: "password123",
			wantErr:  ErrPasswordCommon,
		},
		{
			name:     "Empty password",
			password: "",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "Password with spaces",
			password: "Secure Pass 123!",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if err != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidatePasswordComplexity tests various combinations of password complexity
func TestValidatePasswordComplexity(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "Only uppercase",
			password: "SECUREPASSWORD",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Only lowercase",
			password: "securepassword",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Only numbers",
			password: "12345678901",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Only special chars",
			password: "!@#$%^&*()",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Upper and lower only",
			password: "SecurePassword",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Upper and number only",
			password: "SECURE12345",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Upper and special only",
			password: "SECURE!@#$%",
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "All requirements met minimally",
			password: "Aa1!aaaa",
			wantErr:  nil,
		},
		{
			name:     "Complex password with all types",
			password: "Test123!@#Pass",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if err != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCommonPasswords specifically tests the common password rejection functionality
func TestCommonPasswords(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "Common password exact match",
			password: "password123",
			wantErr:  ErrPasswordCommon,
		},
		{
			name:     "Common password different case",
			password: "PASSWORD123",
			wantErr:  ErrPasswordCommon,
		},
		{
			name:     "Common password with addition",
			password: "password123!",
			wantErr:  ErrPasswordTooSimple, // Should fail complexity check
		},
		{
			name:     "Secure variation of common password",
			password: "Password123!",
			wantErr:  nil, // Valid because it's not an exact match and meets complexity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if err != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
