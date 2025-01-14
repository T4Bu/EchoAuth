package repositories

import (
	"EchoAuth/database"
	"EchoAuth/models"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type TokenRepositoryInterface interface {
	CreateRefreshToken(userID uint, token string, expiresAt time.Time, deviceInfo, ip string) (*models.RefreshToken, error)
	GetRefreshToken(token string) (*models.RefreshToken, error)
	RotateRefreshToken(currentToken *models.RefreshToken, newToken string, expiresAt time.Time) (*models.RefreshToken, error)
	RevokeRefreshToken(token string) error
	RevokeAllUserTokens(userID uint) error
	CleanupExpiredTokens() error
}

type TokenRepository struct {
	db *database.DB
}

func NewTokenRepository(db *database.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// CreateRefreshToken creates a new refresh token for a user
func (r *TokenRepository) CreateRefreshToken(userID uint, token string, expiresAt time.Time, deviceInfo, ip string) (*models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{
		ID:         uuid.New(),
		UserID:     userID,
		Token:      token,
		ExpiresAt:  expiresAt,
		DeviceInfo: deviceInfo,
		IP:         ip,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, device_info, ip, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Exec(query,
		refreshToken.ID, refreshToken.UserID, refreshToken.Token,
		refreshToken.ExpiresAt, refreshToken.DeviceInfo, refreshToken.IP,
		refreshToken.CreatedAt, refreshToken.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return refreshToken, nil
}

// GetRefreshToken retrieves a refresh token by its token string
func (r *TokenRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{}
	query := `
		SELECT id, user_id, token, used, revoked_at, expires_at, created_at, updated_at,
			previous_id, device_info, ip
		FROM refresh_tokens
		WHERE token = $1`

	err := r.db.QueryRow(query, token).Scan(
		&refreshToken.ID, &refreshToken.UserID, &refreshToken.Token,
		&refreshToken.Used, &refreshToken.RevokedAt, &refreshToken.ExpiresAt,
		&refreshToken.CreatedAt, &refreshToken.UpdatedAt,
		&refreshToken.PreviousID, &refreshToken.DeviceInfo, &refreshToken.IP)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return refreshToken, nil
}

// RotateRefreshToken marks the current token as used and creates a new one
func (r *TokenRepository) RotateRefreshToken(currentToken *models.RefreshToken, newToken string, expiresAt time.Time) (*models.RefreshToken, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Mark current token as used
	updateQuery := `
		UPDATE refresh_tokens
		SET used = true, updated_at = $1
		WHERE id = $2`

	now := time.Now()
	_, err = tx.Exec(updateQuery, now, currentToken.ID)
	if err != nil {
		return nil, err
	}

	// Create new token with reference to the previous one
	newRefreshToken := &models.RefreshToken{
		ID:         uuid.New(),
		UserID:     currentToken.UserID,
		Token:      newToken,
		ExpiresAt:  expiresAt,
		PreviousID: &currentToken.ID,
		DeviceInfo: currentToken.DeviceInfo,
		IP:         currentToken.IP,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	insertQuery := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, previous_id, device_info, ip, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = tx.Exec(insertQuery,
		newRefreshToken.ID, newRefreshToken.UserID, newRefreshToken.Token,
		newRefreshToken.ExpiresAt, newRefreshToken.PreviousID,
		newRefreshToken.DeviceInfo, newRefreshToken.IP,
		newRefreshToken.CreatedAt, newRefreshToken.UpdatedAt)

	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return newRefreshToken, nil
}

// RevokeRefreshToken marks a refresh token as revoked
func (r *TokenRepository) RevokeRefreshToken(token string) error {
	now := time.Now()
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1, updated_at = $2
		WHERE token = $3`

	result, err := r.db.Exec(query, now, now, token)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *TokenRepository) RevokeAllUserTokens(userID uint) error {
	now := time.Now()
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1, updated_at = $2
		WHERE user_id = $3 AND revoked_at IS NULL`

	_, err := r.db.Exec(query, now, now, userID)
	return err
}

// CleanupExpiredTokens removes expired and used tokens
func (r *TokenRepository) CleanupExpiredTokens() error {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1 OR used = true`

	_, err := r.db.Exec(query, time.Now())
	return err
}
