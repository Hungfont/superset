package dataset

import (
	"errors"
	"time"
)

// Dataset maps to tables.
type Dataset struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string    `gorm:"column:table_name;not null" json:"table_name"`
	Schema              string    `gorm:"column:schema" json:"schema,omitempty"`
	DatabaseID          uint      `gorm:"column:database_id;not null" json:"database_id"`
	SQL                 string    `gorm:"column:sql" json:"sql,omitempty"`
	Perm                string    `gorm:"column:perm;not null" json:"perm"`
	Description         string    `gorm:"column:description" json:"description,omitempty"`
	MainDttmCol        string    `gorm:"column:main_dttm_col" json:"main_dttm_col,omitempty"`
	CacheTimeout        int       `gorm:"column:cache_timeout;default:0" json:"cache_timeout"`
	FilterSelectEnabled bool      `gorm:"column:filter_select_enabled;default:false" json:"filter_select_enabled"`
	NormalizeColumns    bool      `gorm:"column:normalize_columns;default:false" json:"normalize_columns"`
	IsFeatured          bool      `gorm:"column:is_featured;default:false" json:"is_featured"`
	CreatedByFK         uint      `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK         uint      `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn           time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn           time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (Dataset) TableName() string { return "tables" }

// DatabaseRef is a minimal database projection used by dataset creation.
type DatabaseRef struct {
	ID           uint
	DatabaseName string
}

// CreatePhysicalDatasetRequest is used by POST /api/v1/datasets.
type CreatePhysicalDatasetRequest struct {
	DatabaseID uint   `json:"database_id" binding:"required"`
	Schema     string `json:"schema"`
	TableName  string `json:"table_name" binding:"required,max=255"`
}

// CreatePhysicalDatasetResponse is returned by POST /api/v1/datasets.
type CreatePhysicalDatasetResponse struct {
	ID             uint   `json:"id"`
	TableName      string `json:"table_name"`
	BackgroundSync bool   `json:"background_sync"`
}

// CreateVirtualDatasetRequest is used by POST /api/v1/datasets (virtual).
type CreateVirtualDatasetRequest struct {
	DatabaseID  uint   `json:"database_id" binding:"required"`
	TableName   string `json:"table_name" binding:"required,max=255"`
	SQL         string `json:"sql" binding:"required"`
	ValidateSQL bool   `json:"validate_sql"`
}

// CreateVirtualDatasetResponse is returned by POST /api/v1/datasets (virtual).
type CreateVirtualDatasetResponse struct {
	ID               uint     `json:"id"`
	TableName        string   `json:"table_name"`
	BackgroundSync   bool     `json:"background_sync"`
	Columns          []Column `json:"columns,omitempty"`
}

// Column represents a dataset column.
type Column struct {
	ID                uint   `json:"id"`
	TableID           uint   `gorm:"column:table_id" json:"-"`
	ColumnName        string `json:"column_name"`
	Type              string `json:"type"`
	IsDateTime        bool   `gorm:"column:is_dttm" json:"is_dttm"`
	IsActive          bool   `gorm:"column:is_active" json:"is_active"`
	VerboseName       string `gorm:"column:verbose_name" json:"verbose_name,omitempty"`
	Description       string `gorm:"column:description" json:"description,omitempty"`
	Filterable        bool   `gorm:"column:filterable" json:"filterable"`
	GroupBy           bool   `gorm:"column:groupby" json:"groupby"`
	PythonDateFormat  string `gorm:"column:python_date_format" json:"python_date_format,omitempty"`
	Expression        string `gorm:"column:expression" json:"expression,omitempty"`
	ColumnType        string `gorm:"column:type" json:"column_type,omitempty"`
	Exported          bool   `gorm:"column:exported" json:"exported"`
}

func (Column) TableName() string { return "table_columns" }

// DatasetWithCounts includes aggregate counts for list view.
type DatasetWithCounts struct {
	ID                  uint      `json:"id"`
	TableName           string    `json:"table_name"`
	Schema              string    `json:"schema,omitempty"`
	DatabaseID          uint      `json:"database_id"`
	DatabaseName        string    `json:"database_name,omitempty"`
	Type                string    `json:"type"`
	Perm                string    `json:"perm"`
	Description         string    `json:"description,omitempty"`
	MainDttmCol         string    `json:"main_dttm_col,omitempty"`
	CacheTimeout        int       `json:"cache_timeout"`
	FilterSelectEnabled bool      `json:"filter_select_enabled"`
	NormalizeColumns    bool      `json:"normalize_columns"`
	IsFeatured          bool      `json:"is_featured"`
	CreatedByFK         uint      `json:"created_by_fk"`
	OwnerName           string    `json:"owner_name,omitempty"`
	ColumnCount         int       `json:"column_count"`
	MetricCount         int       `json:"metric_count"`
	ChangedOn           time.Time `json:"changed_on"`
}

// DatasetDetail includes full columns/metrics for detail view.
type DatasetDetail struct {
	DatasetWithCounts
	TableColumns []Column   `json:"table_columns"`
	SqlMetrics  []SqlMetric `json:"sql_metrics"`
}

// SqlMetric maps to sql_metrics table.
type SqlMetric struct {
	ID            uint      `json:"id"`
	MetricName   string    `json:"metric_name"`
	MetricType   string    `json:"metric_type"`
	Expression   string    `json:"expression"`
	CreatedOn    time.Time `json:"created_on"`
}

// DatasetListQuery is used by GET /api/v1/datasets.
type DatasetListQuery struct {
	Q          string `form:"q"`
	DatabaseID uint   `form:"database_id"`
	Schema     string `form:"schema"`
	Type       string `form:"type"`
	Owner      uint   `form:"owner"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	OrderBy    string `form:"order_by"`
}

// DatasetListFilters is used by repository and service layer.
type DatasetListFilters struct {
	SearchQ          string
	DatabaseID       uint
	Schema          string
	Type           string
	Owner          uint
	Page           int
	PageSize       int
	Offset         int
	Limit          int
	OrderBy        string
	VisibilityScope DatasetVisibilityScope
	ActorUserID      uint
}

// DatasetVisibilityScope determines which datasets a user can see.
type DatasetVisibilityScope string

const (
	VisibilityScopeAdmin  DatasetVisibilityScope = "admin"
	VisibilityScopeAlpha DatasetVisibilityScope = "alpha"
	VisibilityScopeGamma DatasetVisibilityScope = "gamma"
)

// DatasetListResult is returned by list endpoints.
type DatasetListResult struct {
	Items    []DatasetWithCounts `json:"items"`
	Total   int64              `json:"total"`
	Page    int                `json:"page"`
	PageSize int               `json:"page_size"`
}

// ErrDatasetNotFound is returned when dataset doesn't exist.
var ErrDatasetNotFound = errors.New("dataset not found")

// UpdateDatasetMetadataRequest is used by PUT /api/v1/datasets/:id.
type UpdateDatasetMetadataRequest struct {
	TableName            string `json:"table_name"`
	Description          string `json:"description"`
	MainDttmCol          string `json:"main_dttm_col"`
	CacheTimeout         int    `json:"cache_timeout"`
	NormalizeColumns     bool   `json:"normalize_columns"`
	FilterSelectEnabled  bool   `json:"filter_select_enabled"`
	IsFeatured           bool   `json:"is_featured"`
	SQL                  string `json:"sql"`
	ValidateSQL          bool   `json:"validate_sql"`
}

// UpdateDatasetMetadataResponse is returned by PUT /api/v1/datasets/:id.
type UpdateDatasetMetadataResponse struct {
	ID             uint   `json:"id"`
	TableName     string `json:"table_name"`
	BackgroundSync bool  `json:"background_sync,omitempty"`
}

// UpdateColumnRequest is used by PUT /api/v1/datasets/:id/columns/:col_id and bulk updates.
type UpdateColumnRequest struct {
	ID               uint   `json:"id"`
	VerboseName      string `json:"verbose_name"`
	Description     string `json:"description"`
	Filterable       *bool  `json:"filterable"`
	GroupBy          *bool  `json:"groupby"`
	IsDateTime       *bool  `json:"is_dttm"`
	PythonDateFormat string `json:"python_date_format"`
	Expression       string `json:"expression"`
	ColumnType       string `json:"column_type"`
	Exported         *bool  `json:"exported"`
}

// UpdateColumnResponse is returned by PUT /api/v1/datasets/:id/columns/:col_id.
type UpdateColumnResponse struct {
	ID   uint `json:"id"`
}

// BulkUpdateColumnRequest is used by PUT /api/v1/datasets/:id/columns.
type BulkUpdateColumnRequest struct {
	Columns []UpdateColumnRequest `json:"columns" binding:"required"`
}

// BulkUpdateColumnResponse is returned by PUT /api/v1/datasets/:id/columns.
type BulkUpdateColumnResponse struct {
	UpdatedCount int `json:"updated_count"`
}


