package validator

import (
	"errors"
	"strings"
	"unicode"
)

var (
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters long")
	ErrPasswordTooSimple = errors.New("password must contain at least one uppercase letter, one lowercase letter, one number, and one special character")
	ErrPasswordCommon    = errors.New("password is too common or easily guessable")
)

// Common passwords that should be rejected
var commonPasswords = map[string]bool{
	"password123": true,
	"12345678":    true,
	"qwerty123":   true,
	"admin123":    true,
	// Add more common passwords as needed
}

// ValidatePassword checks if a password meets security requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	// Check if password is in common password list first
	if commonPasswords[strings.ToLower(password)] {
		return ErrPasswordCommon
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return ErrPasswordTooSimple
	}

	return nil
}
