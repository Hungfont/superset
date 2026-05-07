package query

import (
	"time"
)

// TabState represents the state of a SQL Lab tab (tab_state table)
// Model aligned with docs/db/db_query_saveQuery.md
type TabState struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint   `gorm:"index" json:"user_id"`
	DbID         uint   `gorm:"index" json:"db_id"`
	Schema       string `json:"schema"`
	Catalog      string `gorm:"type:varchar(255)" json:"catalog"`
	Label        string `gorm:"type:varchar(255)" json:"label"`
	Active       bool   `json:"active"`
	SQL          string `gorm:"type:text" json:"sql"`
	QueryLimit   string `gorm:"type:varchar(20)" json:"query_limit"`
	LatestQueryID string `gorm:"type:varchar(36)" json:"latest_query_id"`
	HideLeftBar  bool   `json:"hide_left_bar"`
	SavedQueryID *uint  `json:"saved_query_id"`

	CreatedOn  time.Time `json:"created_on"`
	ChangedOn  time.Time `json:"changed_on"`
	CreatedByFK uint     `gorm:"index" json:"created_by_fk"`
	ChangedByFK uint     `gorm:"index" json:"changed_by_fk"`

	ExtraJSON string `gorm:"type:text" json:"extra_json"`
}

func (TabState) TableName() string {
	return "tab_state"
}
