package dataset

import "context"

// Repository manages dataset create dependencies and persistence.
type Repository interface {
	// ExistsPhysicalDataset reports whether a physical dataset already exists for key tuple.
	ExistsPhysicalDataset(ctx context.Context, databaseID uint, schema string, tableName string) (bool, error)
	// CreatePhysicalDataset inserts one physical dataset row.
	CreatePhysicalDataset(ctx context.Context, dataset *Dataset) error
	// ExistsVirtualDataset reports whether a virtual dataset already exists for key tuple.
	ExistsVirtualDataset(ctx context.Context, databaseID uint, tableName string) (bool, error)
	// CreateVirtualDataset inserts one virtual dataset row.
	CreateVirtualDataset(ctx context.Context, dataset *Dataset) error
	// GetDatasetByID retrieves a dataset by ID.
	GetDatasetByID(ctx context.Context, id uint) (*Dataset, error)
	// CreateColumns inserts column rows for a dataset.
	CreateColumns(ctx context.Context, columns []Column) error
	// ListDatasets retrieves a paginated list of datasets with visibility filtering.
	ListDatasets(ctx context.Context, actorUserID uint, filters DatasetListFilters) (*DatasetListResult, error)
	// GetDatasetDetail retrieves a dataset with full columns and metrics.
	GetDatasetDetail(ctx context.Context, id uint) (*DatasetDetail, error)
}
