package chart

import "context"

// Repository defines the interface for chart and dashboard storage.
type Repository interface {
	CreateSlice(ctx context.Context, slice *Slice) error
	CreateSliceUser(ctx context.Context, su *SliceUser) error
	GetSliceByID(ctx context.Context, id uint) (*Slice, error)
	UpdateSlice(ctx context.Context, slice *Slice) error
	DeleteSlice(ctx context.Context, id uint) error
	ListSlices(ctx context.Context, filter *SliceListFilter) ([]*Slice, int64, error)

	CreateDashboard(ctx context.Context, dashboard *Dashboard) error
	GetDashboardByID(ctx context.Context, id uint) (*Dashboard, error)
	GetDashboardBySlug(ctx context.Context, slug string) (*Dashboard, error)
	UpdateDashboard(ctx context.Context, dashboard *Dashboard) error
	DeleteDashboard(ctx context.Context, id uint) error
	ListDashboards(ctx context.Context, filter *DashboardListFilter) ([]*Dashboard, int64, error)

	AddSliceToDashboard(ctx context.Context, dashboardID, sliceID uint) error
	RemoveSliceFromDashboard(ctx context.Context, dashboardID, sliceID uint) error
	ListDashboardSlices(ctx context.Context, dashboardID uint) ([]*DashboardSlice, error)
}

// SliceListFilter defines filters for listing slices.
type SliceListFilter struct {
	DatasourceID   uint
	DatasourceType string
	VizType        string
	Page           int
	PageSize       int
}

// DashboardListFilter defines filters for listing dashboards.
type DashboardListFilter struct {
	Q        string
	Page     int
	PageSize int
}
