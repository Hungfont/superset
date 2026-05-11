package query

import (
	"context"
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
}
