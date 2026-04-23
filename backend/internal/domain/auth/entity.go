package auth

import (
	"time"

	datasetdomain "superset/auth-service/internal/domain/dataset"
)

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

// UpsertUserRolesRequest is used by user-role assignment endpoints.
type UpsertUserRolesRequest struct {
	RoleIDs []uint `json:"role_ids"`
}

// UserRolesPayload is returned for user-role assignment queries/mutations.
type UserRolesPayload struct {
	UserID  uint   `json:"user_id"`
	RoleIDs []uint `json:"role_ids"`
}

// CreateUserRequest is used by admin user create endpoint.
type CreateUserRequest struct {
	FirstName string `json:"first_name" binding:"required,max=128"`
	LastName  string `json:"last_name" binding:"required,max=128"`
	Username  string `json:"username" binding:"required,max=64"`
	Email     string `json:"email" binding:"required,email,max=256"`
	Password  string `json:"password" binding:"required"`
	Active    *bool  `json:"active,omitempty"`
	RoleIDs   []uint `json:"role_ids" binding:"required"`
}

// UpdateUserRequest is used by admin user update endpoint.
type UpdateUserRequest struct {
	FirstName string `json:"first_name" binding:"required,max=128"`
	LastName  string `json:"last_name" binding:"required,max=128"`
	Username  string `json:"username" binding:"required,max=64"`
	Email     string `json:"email" binding:"required,email,max=256"`
	Active    bool   `json:"active"`
	RoleIDs   []uint `json:"role_ids" binding:"required"`
}

// UserListItem is returned by GET /api/v1/admin/users.
type UserListItem struct {
	ID         uint       `json:"id"`
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	Username   string     `json:"username"`
	Email      string     `json:"email"`
	Active     bool       `json:"active"`
	LoginCount int        `json:"login_count"`
	LastLogin  *time.Time `json:"last_login,omitempty"`
	RoleIDs    []uint     `json:"role_ids"`
}

// UserDetail is returned by user detail and mutation endpoints.
type UserDetail struct {
	ID         uint       `json:"id"`
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	Username   string     `json:"username"`
	Email      string     `json:"email"`
	Active     bool       `json:"active"`
	LoginCount int        `json:"login_count"`
	LastLogin  *time.Time `json:"last_login,omitempty"`
	CreatedOn  time.Time  `json:"created_on"`
	ChangedOn  time.Time  `json:"changed_on"`
	RoleIDs    []uint     `json:"role_ids"`
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

// PermissionTuple represents one RBAC relation pair: action on resource.
type PermissionTuple struct {
	Action   string
	Resource string
}

type RLSFilterType string

const (
	RLSFilterTypeRegular RLSFilterType = "Regular"
	RLSFilterTypeBase   RLSFilterType = "Base"
)

type RLSFilter struct {
	ID           uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string       `gorm:"column:name;uniqueIndex;not null" json:"name"`
	FilterType RLSFilterType `gorm:"column:filter_type;not null" json:"filter_type"`
	Clause     string       `gorm:"column:clause;not null" json:"clause"`
	GroupKey   string       `gorm:"column:group_key;default:''" json:"group_key"`
	Description string    `gorm:"column:description" json:"description"`
	CreatedByFK uint        `gorm:"column:created_by_fk" json:"created_by_fk"`
	ChangedByFK uint        `gorm:"column:changed_by_fk" json:"changed_by_fk"`
	CreatedOn  time.Time   `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn  time.Time   `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
	Roles      []Role     `gorm:"many2many:rls_filter_roles" json:"roles"`
	Tables     []RLSFilterTableJunction `gorm:"foreignKey:RLSID;references:ID" json:"tables"`
}

func (RLSFilter) TableName() string { return "row_level_security_filters" }

type RLSFilterRoleJunction struct {
	ID     uint `gorm:"primaryKey;autoIncrement" json:"id"`
	RLSID  uint `gorm:"column:rls_id;not null" json:"rls_id"`
	RoleID uint `gorm:"column:role_id;not null" json:"role_id"`
}

func (RLSFilterRoleJunction) TableName() string { return "rls_filter_roles" }

type RLSFilterTableJunction struct {
	ID              uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	RLSID           uint   `gorm:"column:rls_id;not null" json:"rls_id"`
	DatasourceID   uint   `gorm:"column:datasource_id;not null" json:"datasource_id"`
	DatasourceType string `gorm:"column:datasource_type;not null" json:"datasource_type"`
	Table           string `gorm:"column:table_name;not null" json:"table_name"`
	DbName         string `gorm:"column:database_name;not null" json:"database_name"`
}

func (RLSFilterTableJunction) TableName() string { return "rls_filter_tables" }

func FromDataset(ds *datasetdomain.Dataset, rlsID uint) RLSFilterTableJunction {
	return RLSFilterTableJunction{
		RLSID:           rlsID,
		DatasourceID:   ds.ID,
		DatasourceType: "table",
		Table:        ds.Name,
		DbName:       ds.Schema,
	}
}

type RLSAuditEventType string

const (
	RLSAuditEventFilterCreated RLSAuditEventType = "filter_created"
	RLSAuditEventFilterUpdated RLSAuditEventType = "filter_updated"
	RLSAuditEventFilterDeleted RLSAuditEventType = "filter_deleted"
)

type RLSAuditLog struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	FilterID   uint           `gorm:"column:rls_id;not null" json:"rls_id"`
	FilterName string         `gorm:"column:rls_name;not null" json:"rls_name"`
	EventType RLSAuditEventType `gorm:"column:event_type;not null" json:"event_type"`
	OldValue   string         `gorm:"column:old_value" json:"old_value"`
	NewValue   string         `gorm:"column:new_value" json:"new_value"`
	ChangedBy  uint           `gorm:"column:changed_by;not null" json:"changed_by"`
	IPAddress  string         `gorm:"column:ip_address" json:"ip_address"`
	CreatedOn  time.Time      `gorm:"column:created_on;autoCreateTime" json:"created_on"`
}

func (RLSAuditLog) TableName() string { return "rls_audit_log" }

type CreateRLSFilterRequest struct {
	Name        string `json:"name" binding:"required,max=255"`
	FilterType string `json:"filter_type" binding:"required,oneof=Regular Base"`
	Clause     string `json:"clause" binding:"required,max=5000"`
	GroupKey   string `json:"group_key"`
	Description string `json:"description"`
	RoleIDs    []uint `json:"role_ids" binding:"required,min=1"`
	TableIDs   []uint `json:"table_ids" binding:"required,min=1"`
}

type UpdateRLSFilterRequest struct {
	Name        string `json:"name" binding:"max=255"`
	FilterType string `json:"filter_type" binding:"oneof=Regular Base"`
	Clause     string `json:"clause" binding:"max=5000"`
	GroupKey   string `json:"group_key"`
	Description string `json:"description"`
	RoleIDs    []uint `json:"role_ids"`
	TableIDs  []uint `json:"table_ids"`
}

type RLSFilterTableInfo struct {
	DatasourceID   uint   `json:"datasource_id"`
	DatasourceType string `json:"datasource_type"`
	TableName     string `json:"table_name"`
	DatabaseName string `json:"database_name"`
}

type RLSFilterResponse struct {
	ID           uint               `json:"id"`
	Name        string             `json:"name"`
	FilterType string             `json:"filter_type"`
	Clause     string             `json:"clause"`
	GroupKey   string             `json:"group_key"`
	Description string           `json:"description"`
	Roles      []Role             `json:"roles"`
	Tables     []RLSFilterTableInfo `json:"tables"`
	CreatedBy  uint               `json:"created_by"`
	CreatedOn  time.Time         `json:"created_on"`
	ChangedOn  time.Time         `json:"changed_on"`
}

type RLSFilterListParams struct {
	Page        int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Q          string `form:"q"`
	FilterType string `form:"filter_type"`
	RoleID    uint   `form:"role_id"`
	DatasourceID uint `form:"datasource_id"`
}

type RLSFilterListResult struct {
	Total  int64               `json:"total"`
	Page   int                `json:"page"`
	Pages int                `json:"pages"`
	Data  []RLSFilterResponse `json:"data"`
}
