package controllers

import (
	"auth/services"
	"encoding/json"
	"net/http"
)

type PasswordResetController struct {
	resetService *services.PasswordResetService
}

func NewPasswordResetController(resetService *services.PasswordResetService) *PasswordResetController {
	return &PasswordResetController{
		resetService: resetService,
	}
}

type RequestResetRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// RequestReset handles requests to generate a password reset token
func (c *PasswordResetController) RequestReset(w http.ResponseWriter, r *http.Request) {
	var req RequestResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := c.resetService.GenerateResetToken(req.Email)
	if err != nil {
		// Don't reveal whether the email exists
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "If your email is registered, you will receive a reset link shortly",
		})
		return
	}

	// TODO: Send email with reset link
	// For now, just return the token in the response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token":   token,
		"message": "Reset token generated successfully",
	})
}

// ResetPassword handles password reset requests
func (c *PasswordResetController) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NewPassword == "" {
		http.Error(w, "New password is required", http.StatusBadRequest)
		return
	}

	err := c.resetService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password reset successfully",
	})
}
