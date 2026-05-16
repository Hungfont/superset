package chart

import "time"

// Slice maps to slices (chart definitions).
type Slice struct {
	ID                   uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SliceName            string    `gorm:"column:slice_name;not null" json:"slice_name"`
	VizType              string    `gorm:"column:viz_type;not null" json:"viz_type"`
	DatasourceID         string    `gorm:"column:datasource_id;not null" json:"datasource_id"`
	DatasourceType       string    `gorm:"column:datasource_type;not null" json:"datasource_type"`
	DatasourceName       string    `gorm:"column:datasource_name;not null" json:"datasource_name"`
	Params               string    `gorm:"column:params;type:text" json:"params"`
	QueryContext          string    `gorm:"column:query_context;type:text" json:"query_context"`
	Description          string    `gorm:"column:description" json:"description"`
	CacheTimeout         int       `gorm:"column:cache_timeout;default:0" json:"cache_timeout"`
	Perm                 string    `gorm:"column:perm;not null" json:"perm"`
	SchemaPerm           string    `gorm:"column:schema_perm" json:"schema_perm"`
	CertifiedBy          string    `gorm:"column:certified_by" json:"certified_by"`
	CertificationDetails string    `gorm:"column:certification_details" json:"certification_details"`
	IsManagedExternally  bool      `gorm:"column:is_managed_externally;default:false" json:"is_managed_externally"`
	ExternalURL          string    `gorm:"column:external_url" json:"external_url"`
	LastSavedAt          time.Time `gorm:"column:last_saved_at" json:"last_saved_at"`
	LastSavedByFK        uint      `gorm:"column:last_saved_by_fk;index" json:"last_saved_by_fk"`
	CreatedByFK          uint      `gorm:"column:created_by_fk;index" json:"-"`
	ChangedByFK          uint      `gorm:"column:changed_by_fk;index" json:"-"`
	CreatedOn            time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn            time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (Slice) TableName() string { return "slices" }

// Dashboard maps to dashboards.
type Dashboard struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DashboardTitle      string    `gorm:"column:dashboard_title;not null" json:"dashboard_title"`
	PositionJSON        string    `gorm:"column:position_json;type:text" json:"position_json"`
	CSS                 string    `gorm:"column:css;type:text" json:"css"`
	Description         string    `gorm:"column:description" json:"description"`
	Slug                string    `gorm:"column:slug;uniqueIndex" json:"slug"`
	JSONMetadata        string    `gorm:"column:json_metadata;type:text" json:"json_metadata"`
	Published           bool      `gorm:"column:published;default:false" json:"published"`
	IsManagedExternally bool      `gorm:"column:is_managed_externally;default:false" json:"is_managed_externally"`
	ExternalURL         string    `gorm:"column:external_url" json:"external_url"`
	CertifiedBy         string    `gorm:"column:certified_by" json:"certified_by"`
	CertificationDetails string   `gorm:"column:certification_details" json:"certification_details"`
	CreatedByFK         uint      `gorm:"column:created_by_fk;index" json:"-"`
	ChangedByFK         uint      `gorm:"column:changed_by_fk;index" json:"-"`
	CreatedOn           time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn           time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (Dashboard) TableName() string { return "dashboards" }

// DashboardSlice maps to dashboard_slices.
type DashboardSlice struct {
	ID          uint `gorm:"primaryKey;autoIncrement" json:"id"`
	DashboardID uint `gorm:"column:dashboard_id;not null;index;uniqueIndex:idx_dash_slice,priority:1" json:"dashboard_id"`
	SliceID     uint `gorm:"column:slice_id;not null;index;uniqueIndex:idx_dash_slice,priority:2" json:"slice_id"`
}

func (DashboardSlice) TableName() string { return "dashboard_slices" }

// DashboardUser maps to dashboard_user.
type DashboardUser struct {
	ID          uint `gorm:"primaryKey;autoIncrement" json:"id"`
	DashboardID uint `gorm:"column:dashboard_id;not null;index" json:"dashboard_id"`
	UserID      uint `gorm:"column:user_id;not null;index" json:"user_id"`
}

func (DashboardUser) TableName() string { return "dashboard_user" }

// SliceUser maps to slice_user.
type SliceUser struct {
	ID      uint `gorm:"primaryKey;autoIncrement" json:"id"`
	SliceID uint `gorm:"column:slice_id;not null;index" json:"slice_id"`
	UserID  uint `gorm:"column:user_id;not null;index" json:"user_id"`
}

func (SliceUser) TableName() string { return "slice_user" }

// DashboardRole maps to dashboard_roles.
type DashboardRole struct {
	ID          uint `gorm:"primaryKey;autoIncrement" json:"id"`
	DashboardID uint `gorm:"column:dashboard_id;not null;index" json:"dashboard_id"`
	RoleID      uint `gorm:"column:role_id;not null;index" json:"role_id"`
}

func (DashboardRole) TableName() string { return "dashboard_roles" }
