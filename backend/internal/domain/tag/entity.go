package tag

import "time"

// Tag maps to tag.
type Tag struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;not null" json:"name"`
	Type        string    `gorm:"column:type;not null" json:"type"`
	Description string    `gorm:"column:description" json:"description"`
	CreatedByFK uint      `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK uint      `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn   time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn   time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (Tag) TableName() string { return "tag" }

// TaggedObject maps to tagged_object.
type TaggedObject struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TagID      uint      `gorm:"column:tag_id;not null;uniqueIndex:idx_tag_obj,priority:1" json:"tag_id"`
	ObjectID   uint      `gorm:"column:object_id;not null;uniqueIndex:idx_tag_obj,priority:2" json:"object_id"`
	ObjectType string    `gorm:"column:object_type;not null;uniqueIndex:idx_tag_obj,priority:3" json:"object_type"`
	CreatedOn  time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn  time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (TaggedObject) TableName() string { return "tagged_object" }
