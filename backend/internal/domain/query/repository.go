package query

import (
	"context"
	"time"
)

// Repository defines the interface for query storage
type Repository interface {
	Create(ctx context.Context, query *Query) error
	GetByID(ctx context.Context, id string) (*Query, error)
	GetByClientID(ctx context.Context, clientID string) (*Query, error)
	Update(ctx context.Context, query *Query) error
	List(ctx context.Context, filter *ListFilter) ([]*Query, int64, error)
	// UpdateStatusConditional updates the query status only if the current status is in allowedFromStatuses.
	// Returns true if the update was applied, false if no row matched (race or already updated).
	UpdateStatusConditional(ctx context.Context, id string, status string, allowedFromStatuses []string, extra map[string]interface{}) (bool, error)
	// ListHistory returns paginated query history with database name via JOIN (QE-007).
	ListHistory(ctx context.Context, filter *ListFilter) ([]*HistoryResponseItem, int64, error)
	// DeleteOlderThan deletes query records where created_at is older than the given time (QE-007).
	DeleteOlderThan(ctx context.Context, olderThan time.Time) (int64, error)
}
