package embedded

import "time"

// CssTemplate maps to css_templates.
type CssTemplate struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TemplateName string    `gorm:"column:template_name;not null" json:"template_name"`
	CSS          string    `gorm:"column:css;type:text" json:"css"`
	CreatedByFK  uint      `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK  uint      `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn    time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn    time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (CssTemplate) TableName() string { return "css_templates" }

// KeyValue maps to key_value.
type KeyValue struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Resource    string     `gorm:"column:resource;not null" json:"resource"`
	UUID        string     `gorm:"column:uuid;uniqueIndex" json:"uuid"`
	Value       string     `gorm:"column:value;type:text" json:"value"`
	CreatedByFK uint       `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK uint       `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn   time.Time  `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn   time.Time  `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
	ExpiresOn   *time.Time `gorm:"column:expires_on" json:"expires_on"`
}

func (KeyValue) TableName() string { return "key_value" }

// EmbeddedDashboard maps to embedded_dashboards.
type EmbeddedDashboard struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID           string    `gorm:"column:uuid;uniqueIndex;not null" json:"uuid"`
	DashboardID    uint      `gorm:"column:dashboard_id;not null" json:"dashboard_id"`
	AllowedDomains string    `gorm:"column:allowed_domains;type:text" json:"allowed_domains"`
	CreatedByFK    uint      `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK    uint      `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn      time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn      time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (EmbeddedDashboard) TableName() string { return "embedded_dashboards" }

// UserAttribute maps to user_attribute.
type UserAttribute struct {
	ID                 uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID             uint   `gorm:"column:user_id;uniqueIndex;not null" json:"user_id"`
	WelcomeDashboardID string `gorm:"column:welcome_dashboard_id" json:"welcome_dashboard_id"`
	AvatarURL          string `gorm:"column:avatar_url" json:"avatar_url"`
}

func (UserAttribute) TableName() string { return "user_attribute" }
