package models

import (
	"testing"
)

func TestUserHashPassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		wantErr     bool
		description string
	}{
		{
			name:        "Valid password",
			password:    "password123",
			wantErr:     false,
			description: "Should successfully hash a valid password",
		},
		{
			name:        "Empty password",
			password:    "",
			wantErr:     false,
			description: "Should handle empty password",
		},
		{
			name:        "Long password",
			password:    "verylongpasswordthatismorethan72characters123456789012345678901234567890",
			wantErr:     false,
			description: "Should handle long passwords",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{
				Password: tt.password,
			}

			err := u.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("User.HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify that the password was actually hashed
				if u.Password == tt.password {
					t.Error("Password was not hashed")
				}

				// Verify that the hashed password can be checked
				if !u.CheckPassword(tt.password) {
					t.Error("CheckPassword() failed to verify hashed password")
				}
			}
		})
	}
}

func TestUserCheckPassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		check       string
		wantErr     bool
		description string
	}{
		{
			name:        "Correct password",
			password:    "password123",
			check:       "password123",
			wantErr:     false,
			description: "Should verify correct password",
		},
		{
			name:        "Incorrect password",
			password:    "password123",
			check:       "wrongpassword",
			wantErr:     true,
			description: "Should reject incorrect password",
		},
		{
			name:        "Empty password check",
			password:    "password123",
			check:       "",
			wantErr:     true,
			description: "Should reject empty password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{
				Password: tt.password,
			}

			// First hash the password
			if err := u.HashPassword(tt.password); err != nil {
				t.Fatalf("Failed to hash password: %v", err)
			}

			// Then check the password
			result := u.CheckPassword(tt.check)
			if result != !tt.wantErr {
				t.Errorf("User.CheckPassword() result = %v, wantErr %v", result, tt.wantErr)
			}
		})
	}
}
