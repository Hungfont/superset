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

// UserAdminRepository manages admin user CRUD and role assignment integration.
type UserAdminRepository interface {
	// IsAdmin reports whether the given user has Admin role.
	IsAdmin(ctx context.Context, userID uint) (bool, error)
	// ListUsers returns users for admin user management.
	ListUsers(ctx context.Context) ([]UserListItem, error)
	// GetUserByID returns one user detail by id.
	GetUserByID(ctx context.Context, userID uint) (*UserDetail, error)
	// CreateUser inserts one user and returns the new id.
	CreateUser(ctx context.Context, req CreateUserRequest) (uint, error)
	// UpdateUser updates user profile fields.
	UpdateUser(ctx context.Context, userID uint, req UpdateUserRequest) error
	// DeactivateUser performs soft delete by setting active=false.
	DeactivateUser(ctx context.Context, userID uint) error

	// CountExistingRoles returns how many provided role ids exist.
	CountExistingRoles(ctx context.Context, roleIDs []uint) (int64, error)
	// ReplaceUserRoles atomically replaces all assigned role ids for a user.
	ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error
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

// RoleRepository manages role CRUD, guards, and aggregate queries.
type RoleRepository interface {
	// IsAdmin reports whether the given user has Admin role.
	IsAdmin(ctx context.Context, userID uint) (bool, error)
	// ListWithCounts returns roles with assigned user and permission counts.
	ListWithCounts(ctx context.Context) ([]RoleListItem, error)
	// Create inserts a new role.
	Create(ctx context.Context, role *Role) error
	// UpdateName updates role name and returns the updated role.
	UpdateName(ctx context.Context, roleID uint, name string) (*Role, error)
	// CountUsersByRole returns number of users assigned to role.
	CountUsersByRole(ctx context.Context, roleID uint) (int64, error)
	// IsBuiltInRole reports whether role is built-in and cannot be deleted.
	IsBuiltInRole(ctx context.Context, roleID uint) (bool, error)
	// Delete removes a role by id.
	Delete(ctx context.Context, roleID uint) error
	// RoleExists reports whether the role exists.
	RoleExists(ctx context.Context, roleID uint) (bool, error)

	// ListPermissionViewIDsByRole returns assigned permission_view ids for a role.
	ListPermissionViewIDsByRole(ctx context.Context, roleID uint) ([]uint, error)
	// CountExistingPermissionViews returns how many provided permission_view ids exist.
	CountExistingPermissionViews(ctx context.Context, permissionViewIDs []uint) (int64, error)
	// ReplacePermissionViews atomically replaces all assigned permission_view ids for a role.
	ReplacePermissionViews(ctx context.Context, roleID uint, permissionViewIDs []uint) error
	// AddPermissionViews assigns additional permission_view ids to a role.
	AddPermissionViews(ctx context.Context, roleID uint, permissionViewIDs []uint) error
	// RemovePermissionView revokes one permission_view id from a role.
	RemovePermissionView(ctx context.Context, roleID uint, permissionViewID uint) error
}

// UserRoleRepository manages user-role assignment operations.
type UserRoleRepository interface {
	// IsAdmin reports whether the given user has Admin role.
	IsAdmin(ctx context.Context, userID uint) (bool, error)
	// ListRoleIDsByUser returns assigned role ids for a user.
	ListRoleIDsByUser(ctx context.Context, userID uint) ([]uint, error)
	// CountExistingRoles returns how many provided role ids exist.
	CountExistingRoles(ctx context.Context, roleIDs []uint) (int64, error)
	// ReplaceUserRoles atomically replaces all assigned role ids for a user.
	ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error
}

// RoleCacheRepository manages role-related cache invalidation.
type RoleCacheRepository interface {
	BustRBAC(ctx context.Context) error
	BustRBACForUser(ctx context.Context, userID uint) error
}

// RBACPermissionRepository resolves effective permission tuples for a user.
type RBACPermissionRepository interface {
	ListPermissionTuplesByUser(ctx context.Context, userID uint) ([]PermissionTuple, error)
}

// RBACPermissionCacheRepository caches user permission tuple sets.
type RBACPermissionCacheRepository interface {
	GetPermissionSet(ctx context.Context, userID uint) ([]string, error)
	SetPermissionSet(ctx context.Context, userID uint, values []string) error
}

// PermissionRepository manages permission, view-menu, and permission-view CRUD.
type PermissionRepository interface {
	ListPermissions(ctx context.Context) ([]Permission, error)
	CreatePermission(ctx context.Context, permission *Permission) error

	ListViewMenus(ctx context.Context) ([]ViewMenu, error)
	CreateViewMenu(ctx context.Context, viewMenu *ViewMenu) error

	ListPermissionViews(ctx context.Context) ([]PermissionView, error)
	CreatePermissionView(ctx context.Context, permissionView *PermissionView) error
	CountRoleAssignmentsByPermissionView(ctx context.Context, permissionViewID uint) (int64, error)
	DeletePermissionView(ctx context.Context, permissionViewID uint) error

	SeedPermissionViews(ctx context.Context, seeds []PermissionViewSeed) error
}
