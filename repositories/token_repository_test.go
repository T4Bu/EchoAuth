package repositories

import (
	"EchoAuth/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTokenTest() (*TokenRepository, func()) {
	// Clear the database before each test
	testDB.Exec("DELETE FROM refresh_tokens")

	repo := NewTokenRepository(testDB)

	return repo, func() {
		testDB.Exec("DELETE FROM refresh_tokens")
	}
}

func TestCreateRefreshToken(t *testing.T) {
	repo, cleanup := setupTokenTest()
	defer cleanup()

	token := "test-token"
	userID := uint(1)
	deviceInfo := "test-device"
	ip := "127.0.0.1"
	expiresAt := time.Now().Add(24 * time.Hour)

	refreshToken, err := repo.CreateRefreshToken(userID, token, expiresAt, deviceInfo, ip)
	assert.NoError(t, err)
	assert.NotNil(t, refreshToken)
	assert.Equal(t, token, refreshToken.Token)
	assert.Equal(t, userID, refreshToken.UserID)
	assert.Equal(t, deviceInfo, refreshToken.DeviceInfo)
	assert.Equal(t, ip, refreshToken.IP)
	assert.False(t, refreshToken.Used)
	assert.Nil(t, refreshToken.RevokedAt)
}

func TestGetRefreshToken(t *testing.T) {
	repo, cleanup := setupTokenTest()
	defer cleanup()

	// Create a test token
	token := "test-token"
	userID := uint(1)
	deviceInfo := "test-device"
	ip := "127.0.0.1"
	expiresAt := time.Now().Add(24 * time.Hour)

	created, err := repo.CreateRefreshToken(userID, token, expiresAt, deviceInfo, ip)
	assert.NoError(t, err)

	// Test getting the token
	found, err := repo.GetRefreshToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, token, found.Token)

	// Test getting non-existent token
	found, err = repo.GetRefreshToken("non-existent")
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestRotateRefreshToken(t *testing.T) {
	repo, cleanup := setupTokenTest()
	defer cleanup()

	// Create initial token
	token := "test-token"
	userID := uint(1)
	deviceInfo := "test-device"
	ip := "127.0.0.1"
	expiresAt := time.Now().Add(24 * time.Hour)

	currentToken, err := repo.CreateRefreshToken(userID, token, expiresAt, deviceInfo, ip)
	assert.NoError(t, err)

	// Rotate token
	newToken := "new-test-token"
	newExpiresAt := time.Now().Add(48 * time.Hour)

	rotated, err := repo.RotateRefreshToken(currentToken, newToken, newExpiresAt)
	assert.NoError(t, err)
	assert.NotNil(t, rotated)
	assert.Equal(t, newToken, rotated.Token)
	assert.Equal(t, currentToken.UserID, rotated.UserID)
	assert.Equal(t, currentToken.DeviceInfo, rotated.DeviceInfo)
	assert.Equal(t, currentToken.IP, rotated.IP)
	assert.Equal(t, currentToken.ID, *rotated.PreviousID)

	// Verify old token is marked as used
	oldToken, err := repo.GetRefreshToken(token)
	assert.NoError(t, err)
	assert.True(t, oldToken.Used)
}

func TestRevokeRefreshToken(t *testing.T) {
	repo, cleanup := setupTokenTest()
	defer cleanup()

	// Create a test token
	token := "test-token"
	userID := uint(1)
	deviceInfo := "test-device"
	ip := "127.0.0.1"
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := repo.CreateRefreshToken(userID, token, expiresAt, deviceInfo, ip)
	assert.NoError(t, err)

	// Revoke token
	err = repo.RevokeRefreshToken(token)
	assert.NoError(t, err)

	// Verify token is revoked
	found, err := repo.GetRefreshToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, found.RevokedAt)
}

func TestRevokeAllUserTokens(t *testing.T) {
	repo, cleanup := setupTokenTest()
	defer cleanup()

	userID := uint(1)
	deviceInfo := "test-device"
	ip := "127.0.0.1"
	expiresAt := time.Now().Add(24 * time.Hour)

	// Create multiple tokens for the same user
	tokens := []string{"token1", "token2", "token3"}
	for _, token := range tokens {
		_, err := repo.CreateRefreshToken(userID, token, expiresAt, deviceInfo, ip)
		assert.NoError(t, err)
	}

	// Revoke all tokens
	err := repo.RevokeAllUserTokens(userID)
	assert.NoError(t, err)

	// Verify all tokens are revoked
	for _, token := range tokens {
		found, err := repo.GetRefreshToken(token)
		assert.NoError(t, err)
		assert.NotNil(t, found.RevokedAt)
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	repo, cleanup := setupTokenTest()
	defer cleanup()

	userID := uint(1)
	deviceInfo := "test-device"
	ip := "127.0.0.1"

	// Create expired token
	expiredToken := "expired-token"
	expiredAt := time.Now().Add(-24 * time.Hour)
	_, err := repo.CreateRefreshToken(userID, expiredToken, expiredAt, deviceInfo, ip)
	assert.NoError(t, err)

	// Create valid token
	validToken := "valid-token"
	validExpiresAt := time.Now().Add(24 * time.Hour)
	_, err = repo.CreateRefreshToken(userID, validToken, validExpiresAt, deviceInfo, ip)
	assert.NoError(t, err)

	// Create used token
	usedToken := "used-token"
	_, err = repo.CreateRefreshToken(userID, usedToken, validExpiresAt, deviceInfo, ip)
	assert.NoError(t, err)
	err = testDB.Model(&models.RefreshToken{}).Where("token = ?", usedToken).Update("used", true).Error
	assert.NoError(t, err)

	// Cleanup expired and used tokens
	err = repo.CleanupExpiredTokens()
	assert.NoError(t, err)

	// Verify expired and used tokens are deleted
	_, err = repo.GetRefreshToken(expiredToken)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	_, err = repo.GetRefreshToken(usedToken)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Verify valid token still exists
	validFound, err := repo.GetRefreshToken(validToken)
	assert.NoError(t, err)
	assert.NotNil(t, validFound)
}
