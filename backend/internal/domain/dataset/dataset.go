package dataset

import "time"

// Dataset maps to tables.
type Dataset struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string    `gorm:"column:table_name;not null" json:"table_name"`
	Schema              string    `gorm:"column:schema" json:"schema,omitempty"`
	DatabaseID          uint      `gorm:"column:database_id;not null" json:"database_id"`
	SQL                 string    `gorm:"column:sql" json:"sql,omitempty"`
	Perm                string    `gorm:"column:perm;not null" json:"perm"`
	FilterSelectEnabled bool      `gorm:"column:filter_select_enabled;default:false" json:"filter_select_enabled"`
	NormalizeColumns    bool      `gorm:"column:normalize_columns;default:false" json:"normalize_columns"`
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
