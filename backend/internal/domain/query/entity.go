package query

import (
	"time"
)

// Query represents a query record in the database
type Query struct {
	ID           string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	ClientID     string     `gorm:"type:varchar(36)" json:"client_id"`
	DatabaseID   uint       `json:"database_id"`
	UserID       uint       `json:"user_id"`
	TenantID     uint       `json:"tenant_id"`
	SQL          string     `gorm:"type:text" json:"sql"`
	ExecutedSQL  string     `gorm:"type:text" json:"executed_sql"`
	Status       string     `gorm:"type:varchar(20)" json:"status"`
	StartTime    *time.Time `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	Rows         int        `json:"rows"`
	ResultsKey   string     `gorm:"type:varchar(255)" json:"results_key"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Schema       string     `json:"schema"`
}

func (Query) TableName() string {
	return "query"
}

// ExecuteRequest represents a request to execute a query
type ExecuteRequest struct {
	DatabaseID   uint   `json:"database_id" binding:"required"`
	SQL          string `json:"sql" binding:"required"`
	Limit        *int   `json:"limit"`
	Schema       string `json:"schema"`
	ClientID     string `json:"client_id"`
	ForceRefresh bool   `json:"force_refresh"`
}

// ColumnInfo represents a column in query results
type ColumnInfo struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// ExecuteResponse represents the response after executing a query
type ExecuteResponse struct {
	Data              interface{} `json:"data"`
	Columns           []ColumnInfo   `json:"columns"`
	FromCache        bool       `json:"from_cache"`
	ResultsTruncated bool       `json:"results_truncated"`
	Query           ExecuteMeta `json:"query"`
}

// ExecuteMeta contains metadata about the executed query
type ExecuteMeta struct {
	ExecutedSQL       string    `json:"executed_sql"`
	RLSApplied       bool      `json:"rls_applied"`
	Rows             int       `json:"rows"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
}

// QueryTask represents a task to be processed by the async worker
type QueryTask struct {
	QueryID      string   `json:"query_id"`
	DatabaseID   uint     `json:"database_id"`
	SQL          string   `json:"sql"`
	Limit        *int     `json:"limit"`
	Schema       string   `json:"schema"`
	ClientID     string   `json:"client_id"`
	ForceRefresh bool     `json:"force_refresh"`
	UserID       uint     `json:"user_id"`
	Username     string   `json:"username"`
	Roles        []string `json:"roles"` // G-5: roles for queue routing (Admin→critical, Alpha→default, Gamma→low)
}

// AsyncSubmitRequest represents a request to submit an async query
type AsyncSubmitRequest struct {
	DatabaseID   uint   `json:"database_id" binding:"required"`
	SQL          string `json:"sql" binding:"required"`
	Limit        *int   `json:"limit"`
	Schema       string `json:"schema"`
	ClientID     string `json:"client_id"`
	ForceRefresh bool   `json:"force_refresh"`
}

// AsyncSubmitResponse represents the response after submitting an async query
type AsyncSubmitResponse struct {
	QueryID string `json:"query_id"`
	Status  string `json:"status"`
	Queue   string `json:"queue"`
}

// QueryStatusResponse represents the status of an async query
type QueryStatusResponse struct {
	QueryID     string    `json:"query_id"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Rows        int       `json:"rows"`
	ResultsKey  string    `json:"results_key,omitempty"`
	Error       string    `json:"error,omitempty"`
	ElapsedMs   int64     `json:"elapsed_ms"`
	TimeoutAt   time.Time `json:"timeout_at,omitempty"` // Unix timestamp when query will timeout (30s from start_time)
}

// ListFilter defines filters for listing queries
type ListFilter struct {
	Status     string
	DatabaseID uint
	SQLLike    string
	UserID     uint
	Page       int
	PageSize   int
}
