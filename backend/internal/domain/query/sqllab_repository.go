package query

import "context"

// SQLLabRepository defines the interface for SQL Lab tab persistence.
type SQLLabRepository interface {
	Create(ctx context.Context, tab *TabState) error
	ListByUser(ctx context.Context, userID uint, includeClosed bool) ([]*TabState, error)
	GetByID(ctx context.Context, id uint, userID uint) (*TabState, error)
	Update(ctx context.Context, tab *TabState) error
	CloseTab(ctx context.Context, id uint, userID uint) error
	CloseAllTabs(ctx context.Context, userID uint, exceptID *uint) (int64, error)
	HardDelete(ctx context.Context, id uint, userID uint) error

	CreateSavedQuery(ctx context.Context, sq *SavedQuery) error
	LabelExists(ctx context.Context, userID uint, label string) (bool, error)
	ListSavedQueries(ctx context.Context, userID uint, params SavedQueryListParams) ([]*SavedQuery, int64, error)
	GetSavedQuery(ctx context.Context, id uint, userID uint) (*SavedQuery, error)
	UpdateSavedQuery(ctx context.Context, sq *SavedQuery) error
	DeleteSavedQuery(ctx context.Context, id uint, userID uint) error
	ForkSavedQuery(ctx context.Context, id uint, userID uint) (*SavedQuery, error)
}

// TabWithQueryStatus is a join result for tabs with their latest query status.
type TabWithQueryStatus struct {
	TabState
	LatestQueryStatus *string `gorm:"column:latest_query_status" json:"latest_query_status,omitempty"`
}
