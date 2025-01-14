package repositories

import (
	"EchoAuth/database"
	"EchoAuth/models"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupTokenTestDB(t *testing.T) (*database.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	db := &database.DB{DB: mockDB}
	return db, mock
}

func TestTokenRepository_CreateRefreshToken(t *testing.T) {
	db, mock := setupTokenTestDB(t)
	defer db.Close()

	repo := NewTokenRepository(db)
	userID := uint(1)
	token := "test-token"
	expiresAt := time.Now().Add(24 * time.Hour)
	deviceInfo := "test-device"
	ip := "127.0.0.1"

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(sqlmock.AnyArg(), userID, token, expiresAt, deviceInfo, ip,
			sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	refreshToken, err := repo.CreateRefreshToken(userID, token, expiresAt, deviceInfo, ip)
	assert.NoError(t, err)
	assert.NotNil(t, refreshToken)
	assert.Equal(t, userID, refreshToken.UserID)
	assert.Equal(t, token, refreshToken.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenRepository_GetRefreshToken(t *testing.T) {
	db, mock := setupTokenTestDB(t)
	defer db.Close()

	repo := NewTokenRepository(db)
	token := "test-token"
	tokenID := uuid.New()
	userID := uint(1)
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	mock.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE token = \$1`).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "token", "used", "revoked_at", "expires_at",
			"created_at", "updated_at", "previous_id", "device_info", "ip",
		}).AddRow(
			tokenID, userID, token, false, nil, expiresAt,
			now, now, nil, "test-device", "127.0.0.1",
		))

	refreshToken, err := repo.GetRefreshToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, refreshToken)
	assert.Equal(t, token, refreshToken.Token)
	assert.Equal(t, userID, refreshToken.UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenRepository_GetRefreshToken_NotFound(t *testing.T) {
	db, mock := setupTokenTestDB(t)
	defer db.Close()

	repo := NewTokenRepository(db)
	token := "nonexistent-token"

	mock.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE token = \$1`).
		WithArgs(token).
		WillReturnError(sql.ErrNoRows)

	refreshToken, err := repo.GetRefreshToken(token)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, refreshToken)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenRepository_RotateRefreshToken(t *testing.T) {
	db, mock := setupTokenTestDB(t)
	defer db.Close()

	repo := NewTokenRepository(db)
	currentToken := &models.RefreshToken{
		ID:         uuid.New(),
		UserID:     1,
		Token:      "old-token",
		DeviceInfo: "test-device",
		IP:         "127.0.0.1",
	}
	newTokenStr := "new-token"
	expiresAt := time.Now().Add(24 * time.Hour)

	// Expect transaction
	mock.ExpectBegin()

	// Expect update of current token
	mock.ExpectExec(`UPDATE refresh_tokens SET used = true, updated_at = \$1 WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), currentToken.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect insert of new token
	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(sqlmock.AnyArg(), currentToken.UserID, newTokenStr, expiresAt,
			currentToken.ID, currentToken.DeviceInfo, currentToken.IP,
			sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	newToken, err := repo.RotateRefreshToken(currentToken, newTokenStr, expiresAt)
	assert.NoError(t, err)
	assert.NotNil(t, newToken)
	assert.Equal(t, newTokenStr, newToken.Token)
	assert.Equal(t, currentToken.UserID, newToken.UserID)
	assert.Equal(t, &currentToken.ID, newToken.PreviousID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenRepository_RevokeRefreshToken(t *testing.T) {
	db, mock := setupTokenTestDB(t)
	defer db.Close()

	repo := NewTokenRepository(db)
	token := "test-token"

	mock.ExpectExec(`UPDATE refresh_tokens SET revoked_at = \$1, updated_at = \$2 WHERE token = \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), token).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RevokeRefreshToken(token)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenRepository_RevokeAllUserTokens(t *testing.T) {
	db, mock := setupTokenTestDB(t)
	defer db.Close()

	repo := NewTokenRepository(db)
	userID := uint(1)

	mock.ExpectExec(`UPDATE refresh_tokens SET revoked_at = \$1, updated_at = \$2 WHERE user_id = \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := repo.RevokeAllUserTokens(userID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenRepository_CleanupExpiredTokens(t *testing.T) {
	db, mock := setupTokenTestDB(t)
	defer db.Close()

	repo := NewTokenRepository(db)

	mock.ExpectExec(`DELETE FROM refresh_tokens WHERE expires_at < \$1 OR used = true`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 5))

	err := repo.CleanupExpiredTokens()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
