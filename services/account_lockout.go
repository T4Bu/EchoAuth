package services

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrAccountLocked = errors.New("account is locked due to too many failed attempts")
)

type AccountLockoutService struct {
	redis         *redis.Client
	maxAttempts   int
	lockDuration  time.Duration
	attemptExpiry time.Duration
}

func NewAccountLockoutService(redis *redis.Client) *AccountLockoutService {
	return &AccountLockoutService{
		redis:         redis,
		maxAttempts:   5,                // Lock after 5 failed attempts
		lockDuration:  15 * time.Minute, // Lock for 15 minutes
		attemptExpiry: 1 * time.Hour,    // Reset attempts after 1 hour
	}
}

// RecordFailedAttempt increments the failed attempt counter for an email
func (s *AccountLockoutService) RecordFailedAttempt(ctx context.Context, email string) error {
	// Check if account is locked
	locked, err := s.IsLocked(ctx, email)
	if err != nil {
		return err
	}
	if locked {
		return ErrAccountLocked
	}

	// Increment failed attempts
	attemptsKey := "failed_attempts:" + email
	pipe := s.redis.Pipeline()
	pipe.Incr(ctx, attemptsKey)
	pipe.Expire(ctx, attemptsKey, s.attemptExpiry)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}

	// Check if account should be locked
	attempts, err := s.redis.Get(ctx, attemptsKey).Int()
	if err != nil {
		return err
	}

	if attempts >= s.maxAttempts {
		lockKey := "account_locked:" + email
		err = s.redis.Set(ctx, lockKey, true, s.lockDuration).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

// IsLocked checks if an account is currently locked
func (s *AccountLockoutService) IsLocked(ctx context.Context, email string) (bool, error) {
	lockKey := "account_locked:" + email
	exists, err := s.redis.Exists(ctx, lockKey).Result()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

// ResetAttempts resets the failed attempt counter for an email (e.g., after successful login)
func (s *AccountLockoutService) ResetAttempts(ctx context.Context, email string) error {
	attemptsKey := "failed_attempts:" + email
	lockKey := "account_locked:" + email

	pipe := s.redis.Pipeline()
	pipe.Del(ctx, attemptsKey)
	pipe.Del(ctx, lockKey)
	_, err := pipe.Exec(ctx)
	return err
}

// GetRemainingAttempts returns the number of attempts remaining before account lockout
func (s *AccountLockoutService) GetRemainingAttempts(ctx context.Context, email string) (int, error) {
	attemptsKey := "failed_attempts:" + email
	attempts, err := s.redis.Get(ctx, attemptsKey).Int()
	if err == redis.Nil {
		return s.maxAttempts, nil
	}
	if err != nil {
		return 0, err
	}
	remaining := s.maxAttempts - attempts
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}
