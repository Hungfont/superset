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
	AllowDML        bool   `json:"allow_dml"`
	ExposeInSQLLab  bool   `json:"expose_in_sqllab"`
	AllowRunAsync   bool   `json:"allow_run_async"`
	AllowFileUpload bool   `json:"allow_file_upload"`
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
