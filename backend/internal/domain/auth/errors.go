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
	ErrForbidden        = errors.New("forbidden")
	ErrBuiltInRole      = errors.New("built-in role cannot be modified")
	ErrRoleHasUsers     = errors.New("role has assigned users")
	ErrRoleNotFound     = errors.New("role not found")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidUser      = errors.New("invalid user")
	ErrInvalidRole      = errors.New("invalid role")
	ErrUserMustHaveRole = errors.New("user must have at least one role")
)

// Sentinel errors for permission/view-menu management.
var (
	ErrInvalidPermission       = errors.New("invalid permission")
	ErrInvalidViewMenu         = errors.New("invalid view menu")
	ErrInvalidPermissionViewID = errors.New("invalid permission view id")
	ErrPermissionDuplicate     = errors.New("permission already exists")
	ErrViewMenuDuplicate       = errors.New("view menu already exists")
	ErrPermissionViewDuplicate = errors.New("permission view already exists")
	ErrPermissionViewInUse     = errors.New("permission view is in use")
	ErrPermissionViewNotFound  = errors.New("permission view not found")
)
