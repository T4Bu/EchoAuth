package models

import (
	"testing"
	"time"
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

				// Verify that the hash is valid by checking it
				if !u.CheckPassword(tt.password) {
					t.Error("Password hash verification failed")
				}
			}
		})
	}
}

func TestUserCheckPassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		checkPass   string
		want        bool
		description string
	}{
		{
			name:        "Correct password",
			password:    "password123",
			checkPass:   "password123",
			want:        true,
			description: "Should return true for correct password",
		},
		{
			name:        "Incorrect password",
			password:    "password123",
			checkPass:   "wrongpassword",
			want:        false,
			description: "Should return false for incorrect password",
		},
		{
			name:        "Empty password",
			password:    "password123",
			checkPass:   "",
			want:        false,
			description: "Should return false for empty password",
		},
		{
			name:        "Case sensitive check",
			password:    "Password123",
			checkPass:   "password123",
			want:        false,
			description: "Should be case sensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{
				Password: tt.password,
			}

			// Hash the initial password
			err := u.HashPassword(tt.password)
			if err != nil {
				t.Fatalf("Failed to hash password: %v", err)
			}

			// Check the password
			if got := u.CheckPassword(tt.checkPass); got != tt.want {
				t.Errorf("User.CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserValidation(t *testing.T) {
	tests := []struct {
		name        string
		user        User
		wantErr     bool
		description string
	}{
		{
			name: "Valid user",
			user: User{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr:     false,
			description: "Should pass validation for valid user",
		},
		{
			name: "Missing email",
			user: User{
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr:     true,
			description: "Should fail validation when email is missing",
		},
		{
			name: "Invalid email format",
			user: User{
				Email:     "invalid-email",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr:     true,
			description: "Should fail validation for invalid email format",
		},
		{
			name: "Missing password",
			user: User{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr:     true,
			description: "Should fail validation when password is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserTimestamps(t *testing.T) {
	u := &User{}

	// Test CreatedAt is set
	if !u.CreatedAt.IsZero() {
		t.Error("CreatedAt should be zero before save")
	}

	// Simulate database save
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	if u.CreatedAt != now {
		t.Error("CreatedAt was not set correctly")
	}

	if u.UpdatedAt != now {
		t.Error("UpdatedAt was not set correctly")
	}
}
