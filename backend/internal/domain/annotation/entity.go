package annotation

import "time"

// Layer maps to annotation_layer.
type Layer struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;not null" json:"name"`
	Descr       string    `gorm:"column:descr" json:"descr"`
	CreatedByFK uint      `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK uint      `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn   time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn   time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (Layer) TableName() string { return "annotation_layer" }

// Annotation maps to annotation.
type Annotation struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	LayerID      uint      `gorm:"column:layer_id;not null" json:"layer_id"`
	ShortDescr   string    `gorm:"column:short_descr;not null" json:"short_descr"`
	LongDescr    string    `gorm:"column:long_descr;type:text" json:"long_descr"`
	StartDttm    time.Time `gorm:"column:start_dttm;not null" json:"start_dttm"`
	EndDttm      time.Time `gorm:"column:end_dttm;not null" json:"end_dttm"`
	JSONMetadata string    `gorm:"column:json_metadata;type:text" json:"json_metadata"`
	CreatedByFK  uint      `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK  uint      `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn    time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn    time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (Annotation) TableName() string { return "annotation" }
