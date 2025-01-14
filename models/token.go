package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uint       `json:"user_id"`
	Token      string     `json:"token" gorm:"unique;not null"`
	Used       bool       `json:"used" gorm:"default:false"`
	RevokedAt  *time.Time `json:"revoked_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	PreviousID *uuid.UUID `json:"previous_id" gorm:"type:uuid"`
	DeviceInfo string     `json:"device_info"`
	IP         string     `json:"ip"`
}

// IsValid checks if the refresh token is still valid
func (rt *RefreshToken) IsValid() bool {
	return !rt.Used && rt.RevokedAt == nil && rt.ExpiresAt.After(time.Now())
}
