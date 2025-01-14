package models

import (
	"errors"

	"github.com/golang-jwt/jwt"
)

type TokenClaims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

// Valid implements the jwt.Claims interface and adds custom validation
func (c *TokenClaims) Valid() error {
	if err := c.StandardClaims.Valid(); err != nil {
		return err
	}

	if c.UserID == 0 {
		return errors.New("missing or invalid user ID")
	}

	if c.ExpiresAt == 0 {
		return errors.New("missing expiry time")
	}

	if c.IssuedAt == 0 {
		return errors.New("missing issued at time")
	}

	return nil
}
