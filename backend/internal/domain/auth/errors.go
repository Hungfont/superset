package auth

import "errors"

// Sentinel errors for the login flow.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account locked")
	ErrAccountInactive    = errors.New("account inactive")
	ErrRateLimited        = errors.New("rate limit exceeded")
)
