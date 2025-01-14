package models

import (
	"EchoAuth/utils/validator"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                  uint       `json:"id"`
	Email               string     `json:"email"`
	Password            string     `json:"-"`
	FirstName           string     `json:"first_name"`
	LastName            string     `json:"last_name"`
	PasswordResetToken  string     `json:"-"`
	ResetTokenExpiresAt time.Time  `json:"-"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"-"`
}

// HashPassword hashes the provided password and stores it in the user model
func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword checks if the provided password matches the hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// Validate performs validation on the user model
func (u *User) Validate() error {
	if err := validator.ValidateEmail(u.Email); err != nil {
		return err
	}
	if u.Password == "" {
		return validator.ErrPasswordTooShort
	}
	return nil
}
