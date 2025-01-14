package models

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
)

func TestTokenClaims(t *testing.T) {
	tests := []struct {
		name        string
		claims      TokenClaims
		wantValid   bool
		description string
	}{
		{
			name: "Valid claims",
			claims: TokenClaims{
				UserID: 1,
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: time.Now().Add(time.Hour).Unix(),
					IssuedAt:  time.Now().Unix(),
				},
			},
			wantValid:   true,
			description: "Claims with future expiry should be valid",
		},
		{
			name: "Expired claims",
			claims: TokenClaims{
				UserID: 1,
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: time.Now().Add(-time.Hour).Unix(),
					IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
				},
			},
			wantValid:   false,
			description: "Claims with past expiry should be invalid",
		},
		{
			name: "Future issued claims",
			claims: TokenClaims{
				UserID: 1,
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: time.Now().Add(2 * time.Hour).Unix(),
					IssuedAt:  time.Now().Add(time.Hour).Unix(),
				},
			},
			wantValid:   false,
			description: "Claims with future issuance should be invalid",
		},
		{
			name: "Zero user ID",
			claims: TokenClaims{
				UserID: 0,
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: time.Now().Add(time.Hour).Unix(),
					IssuedAt:  time.Now().Unix(),
				},
			},
			wantValid:   false,
			description: "Claims with zero user ID should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate using jwt.Valid() which internally calls Valid()
			err := tt.claims.Valid()
			isValid := err == nil

			if isValid != tt.wantValid {
				t.Errorf("TokenClaims.Valid() = %v, want %v", isValid, tt.wantValid)
				if err != nil {
					t.Logf("Error: %v", err)
				}
			}
		})
	}
}

func TestTokenClaimsCustomValidation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		setup       func() *TokenClaims
		wantErr     bool
		description string
	}{
		{
			name: "Valid token",
			setup: func() *TokenClaims {
				return &TokenClaims{
					UserID: 1,
					StandardClaims: jwt.StandardClaims{
						ExpiresAt: now.Add(time.Hour).Unix(),
						IssuedAt:  now.Unix(),
					},
				}
			},
			wantErr:     false,
			description: "Token with all valid fields should pass validation",
		},
		{
			name: "Missing user ID",
			setup: func() *TokenClaims {
				return &TokenClaims{
					UserID: 0,
					StandardClaims: jwt.StandardClaims{
						ExpiresAt: now.Add(time.Hour).Unix(),
						IssuedAt:  now.Unix(),
					},
				}
			},
			wantErr:     true,
			description: "Token without user ID should fail validation",
		},
		{
			name: "Missing expiry",
			setup: func() *TokenClaims {
				return &TokenClaims{
					UserID: 1,
					StandardClaims: jwt.StandardClaims{
						IssuedAt: now.Unix(),
					},
				}
			},
			wantErr:     true,
			description: "Token without expiry should fail validation",
		},
		{
			name: "Missing issued at",
			setup: func() *TokenClaims {
				return &TokenClaims{
					UserID: 1,
					StandardClaims: jwt.StandardClaims{
						ExpiresAt: now.Add(time.Hour).Unix(),
					},
				}
			},
			wantErr:     true,
			description: "Token without issued at time should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := tt.setup()
			err := claims.Valid()
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenClaims.Valid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
