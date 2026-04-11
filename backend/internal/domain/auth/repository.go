package auth

import (
	"context"
	"fmt"
	"time"
)

// ErrAlreadyActivated is returned when the registration was already activated
// (concurrent duplicate request or race condition).
var ErrAlreadyActivated = fmt.Errorf("account already activated")

// RegisterUserRepository handles persistence of pending registrations.
type RegisterUserRepository interface {
	// EmailExists returns true if the email is taken in ab_user or ab_register_user.
	EmailExists(ctx context.Context, email string) (bool, error)
	// UsernameExists returns true if the username is taken in ab_user or ab_register_user.
	UsernameExists(ctx context.Context, username string) (bool, error)
	// Create persists a new pending registration.
	Create(ctx context.Context, r *RegisterUser) error
}

// VerifyRepository handles email verification and account activation.
type VerifyRepository interface {
	// FindByHash returns the pending registration for the given hash.
	// Returns nil, nil when no record is found.
	FindByHash(ctx context.Context, hash string) (*RegisterUser, error)
	// Activate atomically creates the ab_user row and deletes the ab_register_user row.
	Activate(ctx context.Context, reg *RegisterUser) error
}

// LoginRepository handles user lookup and last-login updates.
type LoginRepository interface {
	// FindByUsernameOrEmail returns the active user record matching the identifier.
	// Returns nil, nil when no record is found.
	FindByUsernameOrEmail(ctx context.Context, identifier string) (*User, error)
	// UpdateLastLogin increments login_count and sets last_login to now.
	UpdateLastLogin(ctx context.Context, userID uint, loginCount int, lastLogin time.Time) error
}

// RateLimitRepository manages rate limiting and account lockout state in Redis.
type RateLimitRepository interface {
	// IncrLoginAttempt increments the per-IP rate limit counter and returns the new count.
	// The TTL is set to 60s on the first increment.
	IncrLoginAttempt(ctx context.Context, ip string) (int64, error)
	// IncrFailedLogin increments the failed-login counter for a username.
	// Returns the new count. TTL is set to 15 minutes on the first increment.
	IncrFailedLogin(ctx context.Context, username string) (int64, error)
	// ResetFailedLogin deletes the failed-login counter for a username.
	ResetFailedLogin(ctx context.Context, username string) error
	// GetFailedLoginCount returns the current failed-login count (0 if key absent).
	GetFailedLoginCount(ctx context.Context, username string) (int64, error)
	// SetLockout creates a lockout key with a 15-minute TTL and returns the expiry time.
	SetLockout(ctx context.Context, username string) (time.Time, error)
	// GetLockoutExpiry returns the lockout expiry time, or zero time if not locked.
	GetLockoutExpiry(ctx context.Context, username string) (time.Time, error)
	// StoreRefreshToken persists a refresh token mapped to userID with a 7-day TTL.
	StoreRefreshToken(ctx context.Context, token string, userID uint) error
}
