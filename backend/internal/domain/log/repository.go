package log

import (
	"context"
	"time"
)

// Repository defines the interface for log storage.
type Repository interface {
	Create(ctx context.Context, l *Log) error
	List(ctx context.Context, filter *ListFilter) ([]*Log, int64, error)
	DeleteOlderThan(ctx context.Context, olderThan time.Time) (int64, error)
}

// ListFilter defines filters for listing logs.
type ListFilter struct {
	UserID      uint
	DashboardID uint
	SliceID     uint
	Action      string
	Page        int
	PageSize    int
}
