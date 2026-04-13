package auth

import "errors"

// Sentinel errors for the login flow.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account locked")
	ErrAccountInactive    = errors.New("account inactive")
	ErrRateLimited        = errors.New("rate limit exceeded")
)

// Sentinel errors for JWT middleware.
var (
	ErrTokenMissing  = errors.New("token missing")
	ErrTokenInvalid  = errors.New("token invalid")
	ErrTokenRevoked  = errors.New("token revoked")
	ErrTokenReused   = errors.New("refresh token reuse detected")
)
