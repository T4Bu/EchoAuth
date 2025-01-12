package validator

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
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
			name:    "Valid email with numbers",
			email:   "test123@example.com",
			wantErr: nil,
		},
		{
			name:    "Valid email with dots",
			email:   "test.name@example.com",
			wantErr: nil,
		},
		{
			name:    "Valid email with plus",
			email:   "test+tag@example.com",
			wantErr: nil,
		},
		{
			name:    "Valid email with subdomain",
			email:   "test@sub.example.com",
			wantErr: nil,
		},
		{
			name:    "Empty email",
			email:   "",
			wantErr: ErrEmailEmpty,
		},
		{
			name:    "Whitespace only",
			email:   "   ",
			wantErr: ErrEmailEmpty,
		},
		{
			name:    "Missing @",
			email:   "testexample.com",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "Multiple @",
			email:   "test@example@com",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "Missing domain",
			email:   "test@",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "Missing local part",
			email:   "@example.com",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "Invalid characters",
			email:   "test*@example.com",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "Domain starts with dot",
			email:   "test@.example.com",
			wantErr: ErrDomainInvalid,
		},
		{
			name:    "Domain ends with dot",
			email:   "test@example.com.",
			wantErr: ErrDomainInvalid,
		},
		{
			name:    "Consecutive dots in domain",
			email:   "test@example..com",
			wantErr: ErrDomainInvalid,
		},
		{
			name:    "Too long email",
			email:   "test@" + strings.Repeat("a", maxEmailLength) + ".com",
			wantErr: ErrEmailTooLong,
		},
		{
			name:    "Missing top-level domain",
			email:   "test@example",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "Single character top-level domain",
			email:   "test@example.a",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "Valid email with hyphens",
			email:   "test@my-example.com",
			wantErr: nil,
		},
		{
			name:    "Valid email with underscores",
			email:   "test_name@example.com",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if err != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateEmailEdgeCases tests specific edge cases for email validation
func TestValidateEmailEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "Email with IP address domain",
			email:   "test@[123.123.123.123]",
			wantErr: ErrEmailInvalid, // We don't support IP addresses in domains
		},
		{
			name:    "Unicode in local part",
			email:   "тест@example.com",
			wantErr: ErrEmailInvalid, // We don't support Unicode in local part
		},
		{
			name:    "Unicode in domain",
			email:   "test@пример.com",
			wantErr: ErrEmailInvalid, // We don't support Unicode in domain
		},
		{
			name:    "Quoted local part",
			email:   `"test.name"@example.com`,
			wantErr: ErrEmailInvalid, // We don't support quoted local parts
		},
		{
			name:    "Comments in email",
			email:   "test(comment)@example.com",
			wantErr: ErrEmailInvalid, // We don't support comments
		},
		{
			name:    "Local part exactly at limit",
			email:   strings.Repeat("a", 64) + "@example.com",
			wantErr: nil,
		},
		{
			name:    "Domain part exactly at limit",
			email:   "test@" + strings.Repeat("a", 63) + ".com",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if err != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
