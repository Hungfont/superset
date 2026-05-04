package query

import (
	"context"
)

// Repository defines the interface for query storage
type Repository interface {
	Create(ctx context.Context, query *Query) error
	GetByID(ctx context.Context, id string) (*Query, error)
	Update(ctx context.Context, query *Query) error
	List(ctx context.Context, filter *ListFilter) ([]*Query, int64, error)
}
