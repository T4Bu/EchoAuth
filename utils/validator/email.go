package validator

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrEmailEmpty    = errors.New("email cannot be empty")
	ErrEmailInvalid  = errors.New("email format is invalid")
	ErrEmailTooLong  = errors.New("email is too long")
	ErrDomainInvalid = errors.New("email domain is invalid")
)

// Regular expression for basic email validation
// This regex checks for:
// - At least one character before @
// - @ symbol
// - At least one character for domain
// - At least one dot in domain
// - At least 2 characters after last dot
// - Only allows letters, numbers, dots, hyphens, and underscores
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]\.[a-zA-Z]{2,}$`)

// Maximum length for email address (254 is the practical limit)
const maxEmailLength = 254

// ValidateEmail checks if an email address is valid
func ValidateEmail(email string) error {
	// Check if email is empty
	if email = strings.TrimSpace(email); email == "" {
		return ErrEmailEmpty
	}

	// Check email length
	if len(email) > maxEmailLength {
		return ErrEmailTooLong
	}

	// Split email into local and domain parts
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ErrEmailInvalid
	}

	// Additional domain validation
	domain := parts[1]
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return ErrDomainInvalid
	}

	// Check for consecutive dots in domain
	if strings.Contains(domain, "..") {
		return ErrDomainInvalid
	}

	// Check email format using regex
	if !emailRegex.MatchString(email) {
		return ErrEmailInvalid
	}

	return nil
}
