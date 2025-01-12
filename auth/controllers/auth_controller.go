package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"auth/models"
)

type AuthService interface {
	Register(email, password, firstName, lastName string) error
	Login(ctx context.Context, email, password string) (string, error)
	Logout(token string) error
	ValidateToken(token string) (*models.TokenClaims, error)
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
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (ac *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	err := ac.authService.Register(req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (ac *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := ac.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	response := LoginResponse{Token: token}
	json.NewEncoder(w).Encode(response)
}

func (ac *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	// Since we're using JWT, we don't need server-side logout
	// The client should simply remove the token
	w.WriteHeader(http.StatusOK)
}
