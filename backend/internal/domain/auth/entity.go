package auth

import "time"

// RegisterUser maps to ab_register_user — pending email verification.
type RegisterUser struct {
	ID               uint      `gorm:"primaryKey;autoIncrement"`
	FirstName        string    `gorm:"column:first_name;not null"`
	LastName         string    `gorm:"column:last_name;not null"`
	Username         string    `gorm:"column:username;uniqueIndex;not null"`
	Email            string    `gorm:"column:email;uniqueIndex;not null"`
	Password         string    `gorm:"column:password;not null"` // bcrypt hash
	RegistrationHash string    `gorm:"column:registration_hash;uniqueIndex;not null"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (RegisterUser) TableName() string { return "ab_register_user" }

// User maps to ab_user — activated accounts.
type User struct {
	ID         uint       `gorm:"primaryKey;autoIncrement"`
	FirstName  string     `gorm:"column:first_name;not null"`
	LastName   string     `gorm:"column:last_name;not null"`
	Username   string     `gorm:"column:username;uniqueIndex;not null"`
	Email      string     `gorm:"column:email;uniqueIndex;not null"`
	Password   string     `gorm:"column:password;not null"`
	Active     bool       `gorm:"column:active;default:true"`
	LoginCount int        `gorm:"column:login_count;default:0"`
	LastLogin  *time.Time `gorm:"column:last_login"`
	CreatedOn  time.Time  `gorm:"column:created_on;autoCreateTime"`
	ChangedOn  time.Time  `gorm:"column:changed_on;autoUpdateTime"`
}

func (User) TableName() string { return "ab_user" }

// RegisterRequest holds the raw input from the HTTP request.
type RegisterRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name"  binding:"required"`
	Username  string `json:"username"   binding:"required"`
	Email     string `json:"email"      binding:"required,email"`
	Password  string `json:"password"   binding:"required"`
}

// LoginRequest holds credentials from the login HTTP request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest holds the refresh token submitted by the client.
// The token is read from the HttpOnly cookie set at login.
type RefreshRequest struct {
	Token string // populated from cookie "refresh_token"
}

// LogoutRequest carries normalized data needed for logout and revocation.
type LogoutRequest struct {
	UserID               uint
	JTI                  string
	AccessTokenExpiresAt time.Time
	RefreshToken         string
	LogoutAll            bool
}

// UserContext is injected into Gin context by the JWT middleware.
type UserContext struct {
	ID       uint
	Username string
	Email    string
	Active   bool
}

// Role maps to ab_role.
type Role struct {
	ID   uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"column:name;uniqueIndex;not null" json:"name"`
}

func (Role) TableName() string { return "ab_role" }

// UpsertRoleRequest is used by create/update role endpoints.
type UpsertRoleRequest struct {
	Name string `json:"name" binding:"required,max=64"`
}

// UpsertRolePermissionsRequest is used by role-permission assignment endpoints.
type UpsertRolePermissionsRequest struct {
	PermissionViewIDs []uint `json:"permission_view_ids"`
}

// RolePermissionsPayload is returned for role-permission assignment queries/mutations.
type RolePermissionsPayload struct {
	RoleID            uint   `json:"role_id"`
	PermissionViewIDs []uint `json:"permission_view_ids"`
}

// RoleListItem is returned by GET /api/v1/admin/roles with aggregate counts.
type RoleListItem struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	UserCount       int64  `json:"user_count"`
	PermissionCount int64  `json:"permission_count"`
	BuiltIn         bool   `json:"built_in"`
}

// Permission maps to ab_permission.
type Permission struct {
	ID   uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"column:name;uniqueIndex;not null" json:"name"`
}

func (Permission) TableName() string { return "ab_permission" }

// ViewMenu maps to ab_view_menu.
type ViewMenu struct {
	ID   uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"column:name;uniqueIndex;not null" json:"name"`
}

func (ViewMenu) TableName() string { return "ab_view_menu" }

// PermissionView maps to ab_permission_view.
type PermissionView struct {
	ID           uint `gorm:"primaryKey;autoIncrement" json:"id"`
	PermissionID uint `gorm:"column:permission_id;not null;uniqueIndex:idx_perm_view,priority:1" json:"permission_id"`
	ViewMenuID   uint `gorm:"column:view_menu_id;not null;uniqueIndex:idx_perm_view,priority:2" json:"view_menu_id"`

	PermissionName string `gorm:"->;-:migration;column:permission_name" json:"permission_name,omitempty"`
	ViewMenuName   string `gorm:"->;-:migration;column:view_menu_name" json:"view_menu_name,omitempty"`
}

func (PermissionView) TableName() string { return "ab_permission_view" }

// UpsertPermissionRequest is used by create permission endpoint.
type UpsertPermissionRequest struct {
	Name string `json:"name" binding:"required,max=128"`
}

// UpsertViewMenuRequest is used by create view menu endpoint.
type UpsertViewMenuRequest struct {
	Name string `json:"name" binding:"required,max=128"`
}

// CreatePermissionViewRequest is used by create permission-view endpoint.
type CreatePermissionViewRequest struct {
	PermissionID uint `json:"permission_id" binding:"required"`
	ViewMenuID   uint `json:"view_menu_id" binding:"required"`
}

// PermissionViewSeed is used during startup to seed default permission views.
type PermissionViewSeed struct {
	PermissionName string
	ViewMenuName   string
}
