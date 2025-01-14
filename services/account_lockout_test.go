package services

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupLockoutTest(t *testing.T) (*AccountLockoutService, context.Context) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use a different DB for tests
	})

	ctx := context.Background()

	// Clear test database
	err := redisClient.FlushDB(ctx).Err()
	assert.NoError(t, err)

	return NewAccountLockoutService(redisClient), ctx
}

func TestRecordFailedAttempt(t *testing.T) {
	svc, ctx := setupLockoutTest(t)
	email := "test@example.com"

	// First attempt should succeed
	err := svc.RecordFailedAttempt(ctx, email)
	assert.NoError(t, err)

	// Check remaining attempts
	remaining, err := svc.GetRemainingAttempts(ctx, email)
	assert.NoError(t, err)
	assert.Equal(t, 4, remaining)

	// Record more attempts until locked
	for i := 0; i < 4; i++ {
		err = svc.RecordFailedAttempt(ctx, email)
		assert.NoError(t, err)
	}

	// Next attempt should return ErrAccountLocked
	err = svc.RecordFailedAttempt(ctx, email)
	assert.ErrorIs(t, err, ErrAccountLocked)

	// Verify account is locked
	locked, err := svc.IsLocked(ctx, email)
	assert.NoError(t, err)
	assert.True(t, locked)
}

func TestIsLocked(t *testing.T) {
	svc, ctx := setupLockoutTest(t)
	email := "test@example.com"

	// Account should not be locked initially
	locked, err := svc.IsLocked(ctx, email)
	assert.NoError(t, err)
	assert.False(t, locked)

	// Lock account by exceeding attempts
	for i := 0; i < 5; i++ {
		err = svc.RecordFailedAttempt(ctx, email)
		assert.NoError(t, err)
	}

	// Account should now be locked
	locked, err = svc.IsLocked(ctx, email)
	assert.NoError(t, err)
	assert.True(t, locked)
}

func TestResetAttempts(t *testing.T) {
	svc, ctx := setupLockoutTest(t)
	email := "test@example.com"

	// Record some failed attempts
	for i := 0; i < 3; i++ {
		err := svc.RecordFailedAttempt(ctx, email)
		assert.NoError(t, err)
	}

	// Reset attempts
	err := svc.ResetAttempts(ctx, email)
	assert.NoError(t, err)

	// Check remaining attempts is back to max
	remaining, err := svc.GetRemainingAttempts(ctx, email)
	assert.NoError(t, err)
	assert.Equal(t, svc.maxAttempts, remaining)

	// Verify account is not locked
	locked, err := svc.IsLocked(ctx, email)
	assert.NoError(t, err)
	assert.False(t, locked)
}

func TestGetRemainingAttempts(t *testing.T) {
	svc, ctx := setupLockoutTest(t)
	email := "test@example.com"

	// Should start with max attempts
	remaining, err := svc.GetRemainingAttempts(ctx, email)
	assert.NoError(t, err)
	assert.Equal(t, svc.maxAttempts, remaining)

	// Record some failed attempts
	for i := 0; i < 3; i++ {
		err := svc.RecordFailedAttempt(ctx, email)
		assert.NoError(t, err)
	}

	// Should have 2 attempts remaining
	remaining, err = svc.GetRemainingAttempts(ctx, email)
	assert.NoError(t, err)
	assert.Equal(t, 2, remaining)
}
