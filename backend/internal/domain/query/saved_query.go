package query

import (
	"time"
)

// SavedQuery represents a persisted/saved query (saved_query table)
// Model aligned with docs/db/db_query_saveQuery.md
type SavedQuery struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	DbID        uint   `gorm:"index" json:"db_id"`
	UserID      uint   `gorm:"index" json:"user_id"`
	Label       string `gorm:"type:varchar(255)" json:"label"`
	Schema      string `json:"schema"`
	Catalog     string `gorm:"type:varchar(255)" json:"catalog"`
	SQL         string `gorm:"type:text" json:"sql"`
	Description string `gorm:"type:text" json:"description"`
	SQLTables   string `gorm:"type:text" json:"sql_tables"`
	ExtraJSON   string `gorm:"type:text" json:"extra_json"`
	Published   bool   `json:"published"`

	CreatedOn  time.Time  `json:"created_on"`
	ChangedOn  time.Time  `json:"changed_on"`
	CreatedByFK uint      `gorm:"index" json:"created_by_fk"`
	ChangedByFK uint      `gorm:"index" json:"changed_by_fk"`

	Tags string `gorm:"type:text" json:"tags"`
}

func (SavedQuery) TableName() string {
	return "saved_query"
}
