package alert

import "time"

// ReportSchedule maps to report_schedule.
type ReportSchedule struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Type                string    `gorm:"column:type;not null" json:"type"`
	Name                string    `gorm:"column:name;not null" json:"name"`
	Description         string    `gorm:"column:description" json:"description"`
	Active              bool      `gorm:"column:active;default:true" json:"active"`
	Crontab             string    `gorm:"column:crontab;not null" json:"crontab"`
	Timezone            string    `gorm:"column:timezone;default:UTC" json:"timezone"`
	SQL                 string    `gorm:"column:sql;type:text" json:"sql"`
	ChartID             *uint     `gorm:"column:chart_id" json:"chart_id"`
	DashboardID         *uint     `gorm:"column:dashboard_id" json:"dashboard_id"`
	DatabaseID          *uint     `gorm:"column:database_id" json:"database_id"`
	LastEvalDttm        string    `gorm:"column:last_eval_dttm" json:"last_eval_dttm"`
	LastState           string    `gorm:"column:last_state" json:"last_state"`
	ValidatorType       string    `gorm:"column:validator_type" json:"validator_type"`
	ValidatorConfigJSON string    `gorm:"column:validator_config_json;type:text" json:"validator_config_json"`
	LogRetention        int       `gorm:"column:log_retention" json:"log_retention"`
	GracePeriod         int       `gorm:"column:grace_period" json:"grace_period"`
	WorkingTimeout      int       `gorm:"column:working_timeout" json:"working_timeout"`
	CreationMethod      string    `gorm:"column:creation_method" json:"creation_method"`
	ForceScreenshot     bool      `gorm:"column:force_screenshot;default:false" json:"force_screenshot"`
	ReportFormat        string    `gorm:"column:report_format" json:"report_format"`
	Extra               string    `gorm:"column:extra;type:text" json:"extra"`
	CreatedByFK         uint      `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK         uint      `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn           time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn           time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (ReportSchedule) TableName() string { return "report_schedule" }

// ReportRecipient maps to report_recipient.
type ReportRecipient struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportScheduleID    uint      `gorm:"column:report_schedule_id;not null" json:"report_schedule_id"`
	Type                string    `gorm:"column:type;not null" json:"type"`
	RecipientConfigJSON string    `gorm:"column:recipient_config_json;type:text" json:"recipient_config_json"`
	CreatedByFK         uint      `gorm:"column:created_by_fk" json:"-"`
	ChangedByFK         uint      `gorm:"column:changed_by_fk" json:"-"`
	CreatedOn           time.Time `gorm:"column:created_on;autoCreateTime" json:"created_on"`
	ChangedOn           time.Time `gorm:"column:changed_on;autoUpdateTime" json:"changed_on"`
}

func (ReportRecipient) TableName() string { return "report_recipient" }

// ReportExecutionLog maps to report_execution_log.
type ReportExecutionLog struct {
	ID                uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportScheduleID  uint       `gorm:"column:report_schedule_id;not null" json:"report_schedule_id"`
	State             string     `gorm:"column:state;not null" json:"state"`
	Value             string     `gorm:"column:value" json:"value"`
	ValueRowJSON      string     `gorm:"column:value_row_json;type:text" json:"value_row_json"`
	ErrorMessage      string     `gorm:"column:error_message;type:text" json:"error_message"`
	UUID              string     `gorm:"column:uuid" json:"uuid"`
	StartDttm         *time.Time `gorm:"column:start_dttm" json:"start_dttm"`
	EndDttm           *time.Time `gorm:"column:end_dttm" json:"end_dttm"`
	ScheduledDttm     *time.Time `gorm:"column:scheduled_dttm" json:"scheduled_dttm"`
}

func (ReportExecutionLog) TableName() string { return "report_execution_log" }

// ReportScheduleUser maps to report_schedule_user.
type ReportScheduleUser struct {
	ID               uint `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportScheduleID uint `gorm:"column:report_schedule_id;not null" json:"report_schedule_id"`
	UserID           uint `gorm:"column:user_id;not null" json:"user_id"`
}

func (ReportScheduleUser) TableName() string { return "report_schedule_user" }
