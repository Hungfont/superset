package log

import "time"

// Log maps to logs.
type Log struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Action      string    `gorm:"column:action;not null" json:"action"`
	UserID      uint      `gorm:"column:user_id;index" json:"user_id"`
	DashboardID uint      `gorm:"column:dashboard_id;index" json:"dashboard_id"`
	SliceID     uint      `gorm:"column:slice_id;index" json:"slice_id"`
	JSON        string    `gorm:"column:json;type:text" json:"json"`
	DurationMS  int       `gorm:"column:duration_ms" json:"duration_ms"`
	Referrer    string    `gorm:"column:referrer" json:"referrer"`
	Dttm        time.Time `gorm:"column:dtm;not null;index" json:"dttm"`
}

func (Log) TableName() string { return "logs" }
