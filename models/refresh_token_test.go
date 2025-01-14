package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRefreshTokenIsValid(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		token       RefreshToken
		want        bool
		description string
	}{
		{
			name: "Valid token",
			token: RefreshToken{
				Token:     "valid-token",
				ExpiresAt: now.Add(24 * time.Hour),
				Used:      false,
				RevokedAt: nil,
			},
			want:        true,
			description: "Token that is not expired, used, or revoked should be valid",
		},
		{
			name: "Expired token",
			token: RefreshToken{
				Token:     "expired-token",
				ExpiresAt: now.Add(-24 * time.Hour),
				Used:      false,
				RevokedAt: nil,
			},
			want:        false,
			description: "Expired token should be invalid",
		},
		{
			name: "Used token",
			token: RefreshToken{
				Token:     "used-token",
				ExpiresAt: now.Add(24 * time.Hour),
				Used:      true,
				RevokedAt: nil,
			},
			want:        false,
			description: "Used token should be invalid",
		},
		{
			name: "Revoked token",
			token: RefreshToken{
				Token:     "revoked-token",
				ExpiresAt: now.Add(24 * time.Hour),
				Used:      false,
				RevokedAt: &now,
			},
			want:        false,
			description: "Revoked token should be invalid",
		},
		{
			name: "Multiple invalid conditions",
			token: RefreshToken{
				Token:     "invalid-token",
				ExpiresAt: now.Add(-24 * time.Hour),
				Used:      true,
				RevokedAt: &now,
			},
			want:        false,
			description: "Token with multiple invalid conditions should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsValid(); got != tt.want {
				t.Errorf("RefreshToken.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefreshTokenChain(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		setup       func() *RefreshToken
		description string
	}{
		{
			name: "Token chain",
			setup: func() *RefreshToken {
				// Create IDs for the token chain
				id2 := uuid.New()
				id3 := uuid.New()

				// Create the final token in the chain
				return &RefreshToken{
					ID:         id3,
					Token:      "token-3",
					ExpiresAt:  now.Add(72 * time.Hour),
					PreviousID: &id2,
				}
			},
			description: "Test token chain relationships",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setup()

			// Verify token has previous ID
			if token.PreviousID == nil {
				t.Error("Expected token to have previous ID")
			}

			// Verify token ID is not zero
			if token.ID == uuid.Nil {
				t.Error("Expected token to have non-zero ID")
			}

			// Verify expiry is in the future
			if !token.ExpiresAt.After(now) {
				t.Error("Expected token expiry to be in the future")
			}
		})
	}
}

func TestRefreshTokenMetadata(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		token       RefreshToken
		description string
	}{
		{
			name: "Token with metadata",
			token: RefreshToken{
				UserID:     1,
				Token:      "test-token",
				ExpiresAt:  now.Add(24 * time.Hour),
				DeviceInfo: "Chrome on macOS",
				IP:         "127.0.0.1",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			description: "Test token metadata fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify user ID
			if tt.token.UserID == 0 {
				t.Error("Expected non-zero user ID")
			}

			// Verify token string
			if tt.token.Token == "" {
				t.Error("Expected non-empty token string")
			}

			// Verify device info
			if tt.token.DeviceInfo == "" {
				t.Error("Expected non-empty device info")
			}

			// Verify IP address
			if tt.token.IP == "" {
				t.Error("Expected non-empty IP address")
			}

			// Verify timestamps
			if tt.token.CreatedAt.IsZero() {
				t.Error("Expected non-zero created at timestamp")
			}
			if tt.token.UpdatedAt.IsZero() {
				t.Error("Expected non-zero updated at timestamp")
			}
		})
	}
}
