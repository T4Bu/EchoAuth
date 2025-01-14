package middlewares

import (
	"EchoAuth/services"
	"EchoAuth/utils/response"
	"context"
	"net/http"
	"strings"
)

type AuthMiddleware struct {
	authService services.AuthServiceInterface
}

func NewAuthMiddleware(authService services.AuthServiceInterface) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.JSONError(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			response.JSONError(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		claims, err := m.authService.ValidateToken(tokenParts[1])
		if err != nil {
			response.JSONError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
