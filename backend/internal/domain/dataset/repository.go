package dataset

import "context"

// Repository manages dataset create dependencies and persistence.
type Repository interface {
	// ExistsPhysicalDataset reports whether a physical dataset already exists for key tuple.
	ExistsPhysicalDataset(ctx context.Context, databaseID uint, schema string, tableName string) (bool, error)
	// CreatePhysicalDataset inserts one physical dataset row.
	CreatePhysicalDataset(ctx context.Context, dataset *Dataset) error
}
