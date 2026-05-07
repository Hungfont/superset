package query

import (
	"time"
)

// Query represents a query execution record in the database
// Model aligned with docs/db/db_query_saveQuery.md
type Query struct {
	// Primary key (string UUID for backward compatibility with async infra)
	ID       string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	ClientID string `gorm:"type:varchar(36);index" json:"client_id"`

	// Foreign keys
	DatabaseID uint `gorm:"index" json:"database_id"`
	UserID     uint `gorm:"index" json:"user_id"`

	// Tab metadata
	Status       string `gorm:"type:varchar(20);index" json:"status"`
	TabName      string `gorm:"type:varchar(255)" json:"tab_name"`
	SqlEditorID  string `gorm:"type:varchar(36)" json:"sql_editor_id"`

	// Query context
	Schema  string `json:"schema"`
	Catalog string `gorm:"type:varchar(255)" json:"catalog"`

	// SQL content
	SQL         string `gorm:"type:text" json:"sql"`
	SelectSQL   string `gorm:"type:text" json:"select_sql"`
	ExecutedSQL string `gorm:"type:text" json:"executed_sql"`

	// Row limits
	Limit          int  `json:"limit"`
	LimitingFactor int  `json:"limiting_factor"`

	// CTA (Create Table As) support
	SelectAsCTA     bool `json:"select_as_cta"`
	SelectAsCTAUsed bool `json:"select_as_cta_used"`

	// Execution progress
	Progress string `gorm:"type:varchar(100)" json:"progress"`

	// Results
	Rows         int    `json:"rows"`
	ErrorMessage string `gorm:"type:text" json:"error_message"`
	ResultsKey   string `gorm:"type:varchar(255)" json:"results_key"`

	// Timing
	StartTime            *time.Time `json:"start_time"`
	StartRunningTime     *time.Time `json:"start_running_time"`
	EndTime              *time.Time `json:"end_time"`
	EndResultBackendTime int        `json:"end_result_backend_time"`

	// Temp table for CTA
	TmpTableName  string `gorm:"type:varchar(255)" json:"tmp_table_name"`
	TrackingURL   string `gorm:"type:varchar(500)" json:"tracking_url"`
	TmpSchemaName bool   `json:"tmp_schema_name"`

	// Cache / state
	CachedData string `gorm:"type:text" json:"cached_data"`
	IsSaved    bool   `json:"is_saved"`
	ExtraJSON  string `gorm:"type:text" json:"extra_json"`

	// Tenant isolation (extended field, not in base model)
	TenantID uint `json:"tenant_id"`

	// Timestamps
	ChangedOn *time.Time `json:"changed_on"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (Query) TableName() string {
	return "query"
}

// ExecuteRequest represents a request to execute a query (QE-001)
type ExecuteRequest struct {
	DatabaseID    uint   `json:"database_id" binding:"required"`
	SQL           string `json:"sql" binding:"required"`
	Limit         *int   `json:"limit"`
	Schema        string `json:"schema"`
	Catalog       string `json:"catalog"`
	TabName       string `json:"tab_name"`
	SqlEditorID   string `json:"sql_editor_id"`
	ClientID      string `json:"client_id"`
	ForceRefresh  bool   `json:"force_refresh"`
	SelectAsCTA   bool   `json:"select_as_cta"`
}

// ColumnInfo represents a column in query results
type ColumnInfo struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// ExecuteResponse represents the response after executing a query (QE-001)
type ExecuteResponse struct {
	Data              interface{}  `json:"data"`
	Columns           []ColumnInfo `json:"columns"`
	FromCache         bool         `json:"from_cache"`
	ResultsTruncated  bool         `json:"results_truncated"`
	Query             ExecuteMeta  `json:"query"`
}

// ExecuteMeta contains metadata about the executed query
type ExecuteMeta struct {
	ID               string     `json:"id"`
	ClientID         string     `json:"client_id"`
	SQL              string     `json:"sql"`
	ExecutedSQL      string     `json:"executed_sql"`
	RLSApplied       bool       `json:"rls_applied"`
	Rows             int        `json:"rows"`
	Limit            int        `json:"limit"`
	LimitingFactor   int        `json:"limiting_factor"`
	Progress         string     `json:"progress"`
	Status           string     `json:"status"`
	ResultsKey       string     `json:"results_key,omitempty"`
	StartTime        time.Time  `json:"start_time"`
	StartRunningTime *time.Time `json:"start_running_time,omitempty"`
	EndTime          time.Time  `json:"end_time"`
	SelectAsCTAUsed  bool       `json:"select_as_cta_used"`
}

// QueryTask represents a task to be processed by the async worker (QE-004)
type QueryTask struct {
	QueryID      string   `json:"query_id"`
	DatabaseID   uint     `json:"database_id"`
	SQL          string   `json:"sql"`
	Limit        *int     `json:"limit"`
	Schema       string   `json:"schema"`
	Catalog      string   `json:"catalog"`
	TabName      string   `json:"tab_name"`
	SqlEditorID  string   `json:"sql_editor_id"`
	ClientID     string   `json:"client_id"`
	ForceRefresh bool     `json:"force_refresh"`
	SelectAsCTA  bool     `json:"select_as_cta"`
	UserID       uint     `json:"user_id"`
	Username     string   `json:"username"`
	Roles        []string `json:"roles"`
}

// AsyncSubmitRequest represents a request to submit an async query (QE-004)
type AsyncSubmitRequest struct {
	DatabaseID    uint   `json:"database_id" binding:"required"`
	SQL           string `json:"sql" binding:"required"`
	Limit         *int   `json:"limit"`
	Schema        string `json:"schema"`
	Catalog       string `json:"catalog"`
	TabName       string `json:"tab_name"`
	SqlEditorID   string `json:"sql_editor_id"`
	ClientID      string `json:"client_id"`
	ForceRefresh  bool   `json:"force_refresh"`
	SelectAsCTA   bool   `json:"select_as_cta"`
}

// AsyncSubmitResponse represents the response after submitting an async query (QE-004)
type AsyncSubmitResponse struct {
	QueryID string `json:"query_id"`
	Status  string `json:"status"`
	Queue   string `json:"queue"`
}

// QueryStatusResponse represents the status of an async query (QE-004)
type QueryStatusResponse struct {
	QueryID    string    `json:"query_id"`
	Status     string    `json:"status"`
	Progress   string    `json:"progress,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
	Rows       int       `json:"rows"`
	ResultsKey string    `json:"results_key,omitempty"`
	Error      string    `json:"error,omitempty"`
	ElapsedMs  int64     `json:"elapsed_ms"`
	TimeoutAt  time.Time `json:"timeout_at,omitempty"`
}

// ListFilter defines filters for listing queries (QE-007)
type ListFilter struct {
	Status     string
	DatabaseID uint
	SQLLike    string
	UserID     uint
	Page       int
	PageSize   int
}
