package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"EchoAuth/models"
	"EchoAuth/utils/response"

	"github.com/go-playground/validator/v10"
)

type AuthService interface {
	Register(email, password, firstName, lastName string) error
	LoginWithRefresh(email, password, deviceInfo, ip string) (string, string, error)
	Logout(token string) error
	ValidateToken(token string) (*models.TokenClaims, error)
	RefreshToken(refreshToken, deviceInfo, ip string) (string, string, error)
	GetJWTExpiry() time.Duration
	GetUserByEmail(email string) (*models.User, error)
}

type AuthController struct {
	authService AuthService
}

func NewAuthController(authService AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	TokenResponse
	User *models.User `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

var validate = validator.New()

func (ac *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		response.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := ac.authService.Register(req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		if err.Error() == "user already exists" {
			response.JSONError(w, err.Error(), http.StatusConflict)
			return
		}
		response.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	response.JSONResponse(w, map[string]string{"message": "User registered successfully"}, http.StatusCreated)
}

func (ac *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		response.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	deviceInfo := r.Header.Get("User-Agent")
	ip := r.RemoteAddr

	accessToken, refreshToken, err := ac.authService.LoginWithRefresh(req.Email, req.Password, deviceInfo, ip)
	if err != nil {
		response.JSONError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	user, err := ac.authService.GetUserByEmail(req.Email)
	if err != nil {
		response.JSONError(w, "User not found", http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{
		TokenResponse: TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int(ac.authService.GetJWTExpiry().Seconds()),
		},
		User: user,
	}

	response.JSONResponse(w, resp, http.StatusOK)
}

func (ac *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := ac.authService.Logout(req.RefreshToken); err != nil {
		response.JSONError(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	response.JSONResponse(w, map[string]string{"message": "Successfully logged out"}, http.StatusOK)
}

func (ac *AuthController) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		response.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	deviceInfo := r.Header.Get("User-Agent")
	ip := r.RemoteAddr

	accessToken, refreshToken, err := ac.authService.RefreshToken(req.RefreshToken, deviceInfo, ip)
	if err != nil {
		response.JSONError(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	resp := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(ac.authService.GetJWTExpiry().Seconds()),
	}

	response.JSONResponse(w, resp, http.StatusOK)
}
