package auth

import pkgerrors "superset/auth-service/internal/pkg/autherrors"

// Sentinel errors for the login flow.
var (
	ErrInvalidCredentials = pkgerrors.ErrInvalidCredentials
	ErrAccountLocked      = pkgerrors.ErrAccountLocked
	ErrAccountInactive    = pkgerrors.ErrAccountInactive
	ErrRateLimited        = pkgerrors.ErrRateLimited
)

// Sentinel errors for JWT middleware.
var (
	ErrTokenMissing = pkgerrors.ErrTokenMissing
	ErrTokenInvalid = pkgerrors.ErrTokenInvalid
	ErrTokenRevoked = pkgerrors.ErrTokenRevoked
	ErrTokenReused  = pkgerrors.ErrTokenReused
)

// Sentinel errors for role management.
var (
	ErrForbidden        = pkgerrors.ErrForbidden
	ErrBuiltInRole      = pkgerrors.ErrBuiltInRole
	ErrRoleHasUsers     = pkgerrors.ErrRoleHasUsers
	ErrRoleNotFound     = pkgerrors.ErrRoleNotFound
	ErrUserNotFound     = pkgerrors.ErrUserNotFound
	ErrInvalidUser      = pkgerrors.ErrInvalidUser
	ErrInvalidRole      = pkgerrors.ErrInvalidRole
	ErrUserMustHaveRole = pkgerrors.ErrUserMustHaveRole
)

// Sentinel errors for permission/view-menu management.
var (
	ErrInvalidPermission       = pkgerrors.ErrInvalidPermission
	ErrInvalidViewMenu         = pkgerrors.ErrInvalidViewMenu
	ErrInvalidPermissionViewID = pkgerrors.ErrInvalidPermissionViewID
	ErrPermissionDuplicate     = pkgerrors.ErrPermissionDuplicate
	ErrViewMenuDuplicate       = pkgerrors.ErrViewMenuDuplicate
	ErrPermissionViewDuplicate = pkgerrors.ErrPermissionViewDuplicate
	ErrPermissionViewInUse     = pkgerrors.ErrPermissionViewInUse
	ErrPermissionViewNotFound  = pkgerrors.ErrPermissionViewNotFound
)

// Sentinel errors for database connection management.
var (
	ErrInvalidDatabase              = pkgerrors.ErrInvalidDatabase
	ErrInvalidDatabaseURI           = pkgerrors.ErrInvalidDatabaseURI
	ErrDatabaseNameExists           = pkgerrors.ErrDatabaseNameExists
	ErrDatabaseConnectionTestFailed = pkgerrors.ErrDatabaseConnectionTestFailed
	ErrDatabaseCredentialEncryption = pkgerrors.ErrDatabaseCredentialEncryption
)
