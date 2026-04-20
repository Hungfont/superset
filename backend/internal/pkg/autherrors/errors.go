package autherrors

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

// Sentinel errors for database connection management.
var (
	ErrInvalidDatabase              = errors.New("invalid database payload")
	ErrInvalidDatabaseURI           = errors.New("invalid sqlalchemy uri")
	ErrDatabaseNotFound             = errors.New("database not found")
	ErrDatabaseNameExists           = errors.New("database name already exists")
	ErrDatabaseInUse                = errors.New("database is in use")
	ErrDatabaseUnreachable          = errors.New("database unreachable")
	ErrDatabaseTimeout              = errors.New("database introspection timeout")
	ErrDatabaseConnectionTestFailed = errors.New("database connection test failed")
	ErrDatabaseCredentialEncryption = errors.New("database credential encryption failed")
	ErrUnknownDatabaseDriver        = errors.New("unknown database driver")
)

// Sentinel errors for dataset management.
var (
	ErrInvalidDataset     = errors.New("invalid dataset payload")
	ErrDatasetDuplicate   = errors.New("dataset already exists")
	ErrDatasetSyncEnqueue = errors.New("dataset sync enqueue failed")
	ErrInvalidMainDttmCol = errors.New("invalid main_dttm_col: column not found or not a datetime column")
)

// Sentinel errors for virtual dataset SQL validation.
var (
	ErrInvalidSQL       = errors.New("invalid SQL query")
	ErrSQLNotSelect     = errors.New("SQL must be a SELECT statement")
	ErrSQLSemicolon     = errors.New("SQL should not contain semicolons")
	ErrSQLSemanticError = errors.New("SQL semantic error")
)

// Sentinel errors for columns dataset validation.
var (
	ErrColumnNotFound    = errors.New("column not found")
	ErrInvalidExpression = errors.New("invalid expression")
	ErrInvalidDateFormat = errors.New("invalid python date format")
)
