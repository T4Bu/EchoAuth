package models

import "github.com/golang-jwt/jwt"

type TokenClaims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}
