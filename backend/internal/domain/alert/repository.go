package alert

import "context"

// Repository defines the interface for alert and report storage.
type Repository interface {
	CreateReportSchedule(ctx context.Context, rs *ReportSchedule) error
	GetReportScheduleByID(ctx context.Context, id uint) (*ReportSchedule, error)
	UpdateReportSchedule(ctx context.Context, rs *ReportSchedule) error
	DeleteReportSchedule(ctx context.Context, id uint) error
	ListReportSchedules(ctx context.Context, filter *ReportScheduleListFilter) ([]*ReportSchedule, int64, error)

	AddRecipient(ctx context.Context, recipient *ReportRecipient) error
	RemoveRecipient(ctx context.Context, id uint) error
	ListRecipients(ctx context.Context, reportScheduleID uint) ([]*ReportRecipient, error)

	CreateExecutionLog(ctx context.Context, log *ReportExecutionLog) error
	ListExecutionLogs(ctx context.Context, reportScheduleID uint, page, pageSize int) ([]*ReportExecutionLog, int64, error)
}

// ReportScheduleListFilter defines filters for listing report schedules.
type ReportScheduleListFilter struct {
	Type   string
	Active *bool
	Page   int
	PageSize int
}
