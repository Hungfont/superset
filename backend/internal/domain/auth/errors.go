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
	ErrTokenMissing = errors.New("token missing")
	ErrTokenInvalid = errors.New("token invalid")
	ErrTokenRevoked = errors.New("token revoked")
	ErrTokenReused  = errors.New("refresh token reuse detected")
)

// Sentinel errors for role management.
var (
	ErrForbidden    = errors.New("forbidden")
	ErrBuiltInRole  = errors.New("built-in role cannot be modified")
	ErrRoleHasUsers = errors.New("role has assigned users")
	ErrRoleNotFound = errors.New("role not found")
	ErrInvalidRole  = errors.New("invalid role")
)
