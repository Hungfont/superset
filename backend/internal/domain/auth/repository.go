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

// UserRepository handles user lookups for the JWT middleware.
type UserRepository interface {
	// FindByID returns the user with the given ID, or nil if not found.
	FindByID(ctx context.Context, id uint) (*User, error)
}

// JWTRepository manages JWT blacklist and user cache in Redis.
type JWTRepository interface {
	// IsBlacklisted returns true if the given jti has been revoked.
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
	// BlacklistJTI stores the jti as revoked for the provided TTL.
	BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error
	// GetCachedUser returns the cached UserContext for the given user ID.
	// Returns nil, nil when the key is absent (cache miss).
	GetCachedUser(ctx context.Context, userID uint) (*UserContext, error)
	// SetCachedUser stores a UserContext in Redis with a 5-minute TTL.
	SetCachedUser(ctx context.Context, userID uint, u *UserContext) error
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
}

// RefreshRepository manages refresh token lifecycle in Redis.
// Each token is stored as "refresh:{token}" → userID.
// A secondary set "user_tokens:{userID}" tracks all active tokens per user,
// enabling full session revocation on reuse-attack detection.
type RefreshRepository interface {
	// Store persists a refresh token mapped to userID (7-day TTL) and
	// registers it in the per-user token set.
	Store(ctx context.Context, token string, userID uint) error
	// GetUserID returns the userID for the given token.
	// Returns found=false when the token is absent (expired or unknown).
	GetUserID(ctx context.Context, token string) (userID uint, found bool, err error)
	// Delete removes a single refresh token.
	// Returns deleted=true when the key existed and was removed.
	Delete(ctx context.Context, token string) (deleted bool, err error)
	// DeleteAllForUser revokes every active refresh token belonging to userID.
	// Used to terminate all sessions after a reuse-attack is detected.
	DeleteAllForUser(ctx context.Context, userID uint) error
}
