package query

import "context"

// SQLLabRepository defines the interface for SQL Lab tab persistence.
type SQLLabRepository interface {
	Create(ctx context.Context, tab *TabState) error
	ListByUser(ctx context.Context, userID uint) ([]*TabState, error)
	GetByID(ctx context.Context, id uint, userID uint) (*TabState, error)
}
