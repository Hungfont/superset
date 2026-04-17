package db

import "time"

// Database maps to dbs.
type Database struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DatabaseName    string    `gorm:"column:database_name;uniqueIndex;not null" json:"database_name"`
	SQLAlchemyURI   string    `gorm:"column:sqlalchemy_uri;not null" json:"sqlalchemy_uri"`
	AllowDML        bool      `gorm:"column:allow_dml;default:false" json:"allow_dml"`
	ExposeInSQLLab  bool      `gorm:"column:expose_in_sqllab;default:false" json:"expose_in_sqllab"`
	AllowRunAsync   bool      `gorm:"column:allow_run_async;default:false" json:"allow_run_async"`
	AllowFileUpload bool      `gorm:"column:allow_file_upload;default:false" json:"allow_file_upload"`
	CreatedByFK     uint      `gorm:"column:created_by_fk" json:"-"`
	CreatedOn       time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn       time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (Database) TableName() string { return "dbs" }

// CreateDatabaseRequest is used by POST /api/v1/admin/databases.
type CreateDatabaseRequest struct {
	DatabaseName    string `json:"database_name" binding:"required,max=128"`
	SQLAlchemyURI   string `json:"sqlalchemy_uri" binding:"required"`
	AllowDML        bool   `json:"allow_dml"`
	ExposeInSQLLab  bool   `json:"expose_in_sqllab"`
	AllowRunAsync   bool   `json:"allow_run_async"`
	AllowFileUpload bool   `json:"allow_file_upload"`
	StrictTest      *bool  `json:"strict_test,omitempty"`
}

// DatabaseDetail is returned by create endpoint.
type DatabaseDetail struct {
	ID              uint   `json:"id"`
	DatabaseName    string `json:"database_name"`
	SQLAlchemyURI   string `json:"sqlalchemy_uri"`
	Backend         string `json:"backend"`
	AllowDML        bool   `json:"allow_dml"`
	ExposeInSQLLab  bool   `json:"expose_in_sqllab"`
	AllowRunAsync   bool   `json:"allow_run_async"`
	AllowFileUpload bool   `json:"allow_file_upload"`
	DatasetCount    int64  `json:"dataset_count,omitempty"`
}

// DatabaseVisibilityScope controls list/get filtering by actor role.
type DatabaseVisibilityScope string

const (
	DatabaseVisibilityAdmin DatabaseVisibilityScope = "admin"
	DatabaseVisibilityAlpha DatabaseVisibilityScope = "alpha"
	DatabaseVisibilityGamma DatabaseVisibilityScope = "gamma"
)

// DatabaseWithDatasetCount stores one database row plus derived dataset usage.
type DatabaseWithDatasetCount struct {
	Database
	DatasetCount int64 `gorm:"column:dataset_count"`
}

// DatabaseListQuery represents list endpoint query params.
type DatabaseListQuery struct {
	SearchQ  string
	Backend  string
	Page     int
	PageSize int
}

// DatabaseListFilters stores normalized filters passed into repository layer.
type DatabaseListFilters struct {
	SearchQ         string
	Backend         string
	Offset          int
	Limit           int
	VisibilityScope DatabaseVisibilityScope
	ActorUserID     uint
}

// DatabaseListResult is repository output with rows and total records.
type DatabaseListResult struct {
	Items []DatabaseWithDatasetCount
	Total int64
}

// DatabaseListItem is one item in list response payload.
type DatabaseListItem struct {
	ID              uint   `json:"id"`
	DatabaseName    string `json:"database_name"`
	Backend         string `json:"backend"`
	SQLAlchemyURI   string `json:"sqlalchemy_uri"`
	AllowDML        bool   `json:"allow_dml"`
	ExposeInSQLLab  bool   `json:"expose_in_sqllab"`
	AllowRunAsync   bool   `json:"allow_run_async"`
	AllowFileUpload bool   `json:"allow_file_upload"`
	DatasetCount    int64  `json:"dataset_count"`
}

// DatabaseListResponse is returned by GET /api/v1/admin/databases.
type DatabaseListResponse struct {
	Items    []DatabaseListItem `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// TestDatabaseConnectionRequest is used by POST /api/v1/admin/databases/test.
type TestDatabaseConnectionRequest struct {
	SQLAlchemyURI string `json:"sqlalchemy_uri" binding:"required"`
}

// TestConnectionResult is returned by test endpoints.
type TestConnectionResult struct {
	Success   bool   `json:"success"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
	DBVersion string `json:"db_version,omitempty"`
	Driver    string `json:"driver,omitempty"`
	Error     string `json:"error,omitempty"`
}

// UpdateDatabaseRequest is used by PUT /api/v1/admin/databases/:id.
type UpdateDatabaseRequest struct {
	DatabaseName    *string `json:"database_name,omitempty"`
	SQLAlchemyURI   *string `json:"sqlalchemy_uri,omitempty"`
	AllowDML        *bool   `json:"allow_dml,omitempty"`
	ExposeInSQLLab  *bool   `json:"expose_in_sqllab,omitempty"`
	AllowRunAsync   *bool   `json:"allow_run_async,omitempty"`
	AllowFileUpload *bool   `json:"allow_file_upload,omitempty"`
	StrictTest      *bool   `json:"strict_test,omitempty"`
}

// ListDatabaseTablesRequest is used by GET /api/v1/admin/databases/:id/tables.
type ListDatabaseTablesRequest struct {
	Schema   string
	Page     int
	PageSize int
}

// ListDatabaseColumnsRequest is used by GET /api/v1/admin/databases/:id/columns.
type ListDatabaseColumnsRequest struct {
	Schema string
	Table  string
}

// DatabaseTable is one table item discovered from database metadata.
type DatabaseTable struct {
	Name string `json:"name"`
}

// DatabaseTableListResponse represents paginated introspection table output.
type DatabaseTableListResponse struct {
	Items    []DatabaseTable `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// DatabaseColumn is one column metadata item discovered from database metadata.
type DatabaseColumn struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value,omitempty"`
	IsDttm       bool   `json:"is_dttm"`
}
