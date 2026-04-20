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
	// UpdateDatasetMetadata updates dataset metadata.
	UpdateDatasetMetadata(ctx context.Context, id uint, req UpdateDatasetMetadataRequest) error
	// GetColumnByName retrieves a column by table_id and column_name.
	GetColumnByName(ctx context.Context, tableID uint, columnName string) (*Column, error)
	// GetColumnByID retrieves a column by ID.
	GetColumnByID(ctx context.Context, columnID uint) (*Column, error)
	// UpdateColumn updates a single column.
	UpdateColumn(ctx context.Context, columnID uint, req UpdateColumnRequest) error
	// BulkUpdateColumns updates multiple columns in a transaction.
	BulkUpdateColumns(ctx context.Context, columns []UpdateColumnRequest) error
	// GetMetricByID retrieves a metric by ID.
	GetMetricByID(ctx context.Context, metricID uint) (*SqlMetric, error)
	// GetMetricsByTableID retrieves all metrics for a dataset.
	GetMetricsByTableID(ctx context.Context, tableID uint) ([]SqlMetric, error)
	// CreateMetric inserts a new metric.
	CreateMetric(ctx context.Context, metric *SqlMetric) error
	// UpdateMetric updates a metric.
	UpdateMetric(ctx context.Context, metricID uint, req UpdateMetricRequest) error
	// DeleteMetric deletes a metric.
	DeleteMetric(ctx context.Context, metricID uint) error
	// BulkReplaceMetrics replaces all metrics for a dataset (transaction).
	BulkReplaceMetrics(ctx context.Context, tableID uint, metrics []SqlMetric) error
	// MetricNameExists checks if a metric name already exists for a table.
	MetricNameExists(ctx context.Context, tableID uint, metricName string, excludeID uint) (bool, error)
	// DeleteDataset deletes a dataset and its related data in a transaction.
	DeleteDataset(ctx context.Context, id uint) error
	// CountChartsByDatasetID counts charts using this dataset.
	CountChartsByDatasetID(ctx context.Context, datasetID uint) (int64, error)
}
