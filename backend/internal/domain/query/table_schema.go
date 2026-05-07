package query

import (
	"time"
)

// TableSchema represents an expanded schema in a SQL Lab tab (table_schema table)
// Model aligned with docs/db/db_query_saveQuery.md
type TableSchema struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	TabStateID uint   `gorm:"index" json:"tab_state_id"`
	DbID       uint   `gorm:"index" json:"db_id"`
	Schema     string `json:"schema"`
	Catalog    string `gorm:"type:varchar(255)" json:"catalog"`
	Table      string `gorm:"type:varchar(255)" json:"table"`
	Description string `gorm:"type:text" json:"description"`
	Expanded   bool   `json:"expanded"`

	CreatedOn time.Time `json:"created_on"`
	ChangedOn time.Time `json:"changed_on"`
}

func (TableSchema) TableName() string {
	return "table_schema"
}
