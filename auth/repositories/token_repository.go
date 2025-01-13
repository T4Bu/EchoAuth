package repositories

import (
	"auth/models"
	"time"

	"gorm.io/gorm"
)

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// CreateRefreshToken creates a new refresh token for a user
func (r *TokenRepository) CreateRefreshToken(userID uint, token string, expiresAt time.Time, deviceInfo, ip string) (*models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{
		UserID:     userID,
		Token:      token,
		ExpiresAt:  expiresAt,
		DeviceInfo: deviceInfo,
		IP:         ip,
	}

	if err := r.db.Create(refreshToken).Error; err != nil {
		return nil, err
	}
	return refreshToken, nil
}

// GetRefreshToken retrieves a refresh token by its token string
func (r *TokenRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken
	if err := r.db.Where("token = ?", token).First(&refreshToken).Error; err != nil {
		return nil, err
	}
	return &refreshToken, nil
}

// RotateRefreshToken marks the current token as used and creates a new one
func (r *TokenRepository) RotateRefreshToken(currentToken *models.RefreshToken, newToken string, expiresAt time.Time) (*models.RefreshToken, error) {
	var result *models.RefreshToken
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Mark current token as used
		currentToken.Used = true
		if err := tx.Save(currentToken).Error; err != nil {
			return err
		}

		// Create new token with reference to the previous one
		newRefreshToken := &models.RefreshToken{
			UserID:     currentToken.UserID,
			Token:      newToken,
			ExpiresAt:  expiresAt,
			PreviousID: &currentToken.ID,
			DeviceInfo: currentToken.DeviceInfo,
			IP:         currentToken.IP,
		}

		if err := tx.Create(newRefreshToken).Error; err != nil {
			return err
		}

		result = newRefreshToken
		return nil
	})

	return result, err
}

// RevokeRefreshToken marks a refresh token as revoked
func (r *TokenRepository) RevokeRefreshToken(token string) error {
	now := time.Now()
	return r.db.Model(&models.RefreshToken{}).
		Where("token = ?", token).
		Update("revoked_at", &now).Error
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *TokenRepository) RevokeAllUserTokens(userID uint) error {
	now := time.Now()
	return r.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &now).Error
}

// CleanupExpiredTokens removes expired and used tokens
func (r *TokenRepository) CleanupExpiredTokens() error {
	return r.db.Where("expires_at < ? OR used = ?", time.Now(), true).
		Delete(&models.RefreshToken{}).Error
}
