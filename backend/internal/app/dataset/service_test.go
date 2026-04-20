package dataset_test

import (
	"context"
	"errors"
	"testing"

	datasetsvc "superset/auth-service/internal/app/dataset"
	domain "superset/auth-service/internal/domain/dataset"
	dbdomain "superset/auth-service/internal/domain/db"
)

type fakeDatasetRepository struct {
	datasetExists    bool
	datasetExistsErr error
	createErr        error
	createdDataset   *domain.Dataset
	datasets         map[uint]*domain.Dataset
	metrics          map[uint][]domain.SqlMetric
	metricIDCounter  uint
	metricNameExists bool
	metricNameExistsErr error
	createMetricErr error
	updateMetricErr error
	deleteMetricErr error
}

func (f *fakeDatasetRepository) init() {
	if f.datasets == nil {
		f.datasets = make(map[uint]*domain.Dataset)
	}
	if f.metrics == nil {
		f.metrics = make(map[uint][]domain.SqlMetric)
	}
}

func (f *fakeDatasetRepository) ExistsPhysicalDataset(_ context.Context, _ uint, _ string, _ string) (bool, error) {
	if f.datasetExistsErr != nil {
		return false, f.datasetExistsErr
	}
	return f.datasetExists, nil
}

func (f *fakeDatasetRepository) CreatePhysicalDataset(_ context.Context, dataset *domain.Dataset) error {
	f.init()
	if f.createErr != nil {
		return f.createErr
	}
	copyValue := *dataset
	if copyValue.ID == 0 {
		copyValue.ID = 42
	}
	f.createdDataset = &copyValue
	f.datasets[copyValue.ID] = &copyValue
	dataset.ID = copyValue.ID
	return nil
}

func (f *fakeDatasetRepository) ExistsVirtualDataset(_ context.Context, _ uint, _ string) (bool, error) {
	return false, nil
}

func (f *fakeDatasetRepository) CreateVirtualDataset(_ context.Context, dataset *domain.Dataset) error {
	f.init()
	copyValue := *dataset
	if copyValue.ID == 0 {
		copyValue.ID = 42
	}
	f.datasets[copyValue.ID] = &copyValue
	dataset.ID = copyValue.ID
	return nil
}

func (f *fakeDatasetRepository) GetDatasetByID(_ context.Context, id uint) (*domain.Dataset, error) {
	f.init()
	if ds, ok := f.datasets[id]; ok {
		return ds, nil
	}
	return nil, nil
}

func (f *fakeDatasetRepository) CreateColumns(_ context.Context, _ []domain.Column) error {
	return nil
}

func (f *fakeDatasetRepository) ListDatasets(_ context.Context, _ uint, _ domain.DatasetListFilters) (*domain.DatasetListResult, error) {
	return &domain.DatasetListResult{Items: []domain.DatasetWithCounts{}}, nil
}

func (f *fakeDatasetRepository) GetDatasetDetail(_ context.Context, id uint) (*domain.DatasetDetail, error) {
	f.init()
	if ds, ok := f.datasets[id]; ok {
		return &domain.DatasetDetail{
			DatasetWithCounts: domain.DatasetWithCounts{
				ID:          ds.ID,
				TableName:   ds.Name,
				DatabaseID:  ds.DatabaseID,
				CreatedByFK: ds.CreatedByFK,
			},
			SqlMetrics: f.metrics[id],
		}, nil
	}
	return nil, domain.ErrDatasetNotFound
}

func (f *fakeDatasetRepository) UpdateDatasetMetadata(_ context.Context, _ uint, _ domain.UpdateDatasetMetadataRequest) error {
	return nil
}

func (f *fakeDatasetRepository) GetColumnByName(_ context.Context, _ uint, _ string) (*domain.Column, error) {
	return nil, nil
}

func (f *fakeDatasetRepository) GetColumnByID(_ context.Context, _ uint) (*domain.Column, error) {
	return nil, nil
}

func (f *fakeDatasetRepository) UpdateColumn(_ context.Context, _ uint, _ domain.UpdateColumnRequest) error {
	return nil
}

func (f *fakeDatasetRepository) BulkUpdateColumns(_ context.Context, _ []domain.UpdateColumnRequest) error {
	return nil
}

func (f *fakeDatasetRepository) GetMetricByID(_ context.Context, metricID uint) (*domain.SqlMetric, error) {
	f.init()
	for _, metrics := range f.metrics {
		for _, m := range metrics {
			if m.ID == metricID {
				return &m, nil
			}
		}
	}
	return nil, nil
}

func (f *fakeDatasetRepository) GetMetricsByTableID(_ context.Context, tableID uint) ([]domain.SqlMetric, error) {
	f.init()
	return f.metrics[tableID], nil
}

func (f *fakeDatasetRepository) CreateMetric(_ context.Context, metric *domain.SqlMetric) error {
	f.init()
	if f.createMetricErr != nil {
		return f.createMetricErr
	}
	f.metricIDCounter++
	metric.ID = f.metricIDCounter
	metric.CreatedOn = domain.SqlMetric{}.CreatedOn
	f.metrics[metric.TableID] = append(f.metrics[metric.TableID], *metric)
	return nil
}

func (f *fakeDatasetRepository) UpdateMetric(_ context.Context, metricID uint, _ domain.UpdateMetricRequest) error {
	if f.updateMetricErr != nil {
		return f.updateMetricErr
	}
	f.init()
	for tableID, tableMetrics := range f.metrics {
		for i, m := range tableMetrics {
			if m.ID == metricID {
				f.metrics[tableID][i].MetricName = "updated"
				return nil
			}
		}
	}
	return nil
}

func (f *fakeDatasetRepository) DeleteMetric(_ context.Context, metricID uint) error {
	if f.deleteMetricErr != nil {
		return f.deleteMetricErr
	}
	f.init()
	for tableID, tableMetrics := range f.metrics {
		for i, m := range tableMetrics {
			if m.ID == metricID {
				f.metrics[tableID] = append(tableMetrics[:i], tableMetrics[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (f *fakeDatasetRepository) BulkReplaceMetrics(_ context.Context, tableID uint, metrics []domain.SqlMetric) error {
	f.init()
	f.metrics[tableID] = metrics
	return nil
}

func (f *fakeDatasetRepository) MetricNameExists(_ context.Context, _ uint, _ string, _ uint) (bool, error) {
	if f.metricNameExistsErr != nil {
		return false, f.metricNameExistsErr
	}
	return f.metricNameExists, nil
}

type fakeDatabaseLookupRepository struct {
	roleNames   []string
	database    *dbdomain.Database
	databaseErr error
}

func (f *fakeDatabaseLookupRepository) GetRoleNamesByUser(_ context.Context, _ uint) ([]string, error) {
	return append([]string(nil), f.roleNames...), nil
}

func (f *fakeDatabaseLookupRepository) GetDatabaseByID(_ context.Context, _ uint) (*dbdomain.Database, error) {
	if f.databaseErr != nil {
		return nil, f.databaseErr
	}
	if f.database == nil {
		return nil, dbdomain.ErrDatabaseNotFound
	}
	copyValue := *f.database
	return &copyValue, nil
}

type fakeSyncQueue struct {
	enqueueErr error
	called     int
	datasetID  uint
}

func (f *fakeSyncQueue) EnqueueSyncColumns(_ context.Context, datasetID uint) (string, error) {
	f.called++
	f.datasetID = datasetID
	if f.enqueueErr != nil {
		return "", f.enqueueErr
	}
	return "job-1", nil
}

func TestDatasetService_CreatePhysicalDatasetSuccess(t *testing.T) {
	repo := &fakeDatasetRepository{}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
		database:  &dbdomain.Database{ID: 7, DatabaseName: "analytics"},
	}
	queue := &fakeSyncQueue{}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, queue)
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	created, err := svc.CreatePhysicalDataset(context.Background(), 11, domain.CreatePhysicalDatasetRequest{
		DatabaseID: 7,
		Schema:     "public",
		TableName:  "orders",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if created.ID != 42 {
		t.Fatalf("expected id 42, got %d", created.ID)
	}
	if created.TableName != "orders" {
		t.Fatalf("expected table_name orders, got %s", created.TableName)
	}
	if !created.BackgroundSync {
		t.Fatal("expected background_sync=true")
	}
	if queue.called != 1 {
		t.Fatalf("expected enqueue called once, got %d", queue.called)
	}
	if queue.datasetID != 42 {
		t.Fatalf("expected enqueue dataset id 42, got %d", queue.datasetID)
	}
	if repo.createdDataset == nil {
		t.Fatal("expected dataset persisted")
	}
	if repo.createdDataset.Perm != "[can_read].[analytics].[orders]" {
		t.Fatalf("expected perm computed, got %s", repo.createdDataset.Perm)
	}
}

func TestDatasetService_NewServiceReturnsErrorWhenQueueIsNil(t *testing.T) {
	repo := &fakeDatasetRepository{}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
		database:  &dbdomain.Database{ID: 7, DatabaseName: "analytics"},
	}

	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, nil)
	if !errors.Is(err, datasetsvc.ErrSyncQueueRequired) {
		t.Fatalf("expected ErrSyncQueueRequired, got %v", err)
	}
	if svc != nil {
		t.Fatal("expected nil service when sync queue is nil")
	}
}

func TestDatasetService_CreatePhysicalDatasetGammaForbidden(t *testing.T) {
	repo := &fakeDatasetRepository{}
	databaseLookupRepo := &fakeDatabaseLookupRepository{roleNames: []string{"Gamma"}}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	_, err = svc.CreatePhysicalDataset(context.Background(), 22, domain.CreatePhysicalDatasetRequest{
		DatabaseID: 9,
		Schema:     "public",
		TableName:  "events",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDatasetService_CreatePhysicalDatasetDuplicateReturnsConflict(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasetExists: true,
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Alpha"},
		database:  &dbdomain.Database{ID: 7, DatabaseName: "analytics"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	_, err = svc.CreatePhysicalDataset(context.Background(), 11, domain.CreatePhysicalDatasetRequest{
		DatabaseID: 7,
		Schema:     "public",
		TableName:  "orders",
	})
	if !errors.Is(err, domain.ErrDatasetDuplicate) {
		t.Fatalf("expected ErrDatasetDuplicate, got %v", err)
	}
}

func TestDatasetService_CreatePhysicalDatasetInvalidDatabaseIDReturns422(t *testing.T) {
	repo := &fakeDatasetRepository{}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames:   []string{"Admin"},
		databaseErr: dbdomain.ErrDatabaseNotFound,
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	_, err = svc.CreatePhysicalDataset(context.Background(), 11, domain.CreatePhysicalDatasetRequest{
		DatabaseID: 999,
		Schema:     "public",
		TableName:  "orders",
	})
	if !errors.Is(err, domain.ErrInvalidDatabase) {
		t.Fatalf("expected ErrInvalidDatabase, got %v", err)
	}
}

func TestDatasetService_CreateMetricSuccess(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
		database:  &dbdomain.Database{ID: 1, DatabaseName: "analytics"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	created, err := svc.CreateMetric(context.Background(), 10, 1, domain.CreateMetricRequest{
		MetricName: "total_count",
		MetricType: "SUM",
		Expression: "COUNT(*)",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if created.ID == 0 {
		t.Fatal("expected metric ID to be set")
	}
}

func TestDatasetService_CreateMetricNoAggregateReturnsError(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	_, err = svc.CreateMetric(context.Background(), 10, 1, domain.CreateMetricRequest{
		MetricName: "invalid_metric",
		MetricType: "SUM",
		Expression: "column_name",
	})
	if !errors.Is(err, domain.ErrNoAggregateFunction) {
		t.Fatalf("expected ErrNoAggregateFunction, got %v", err)
	}
}

func TestDatasetService_CreateMetricDuplicateNameReturnsConflict(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
		metricNameExists: true,
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	_, err = svc.CreateMetric(context.Background(), 10, 1, domain.CreateMetricRequest{
		MetricName: "total_count",
		MetricType: "SUM",
		Expression: "COUNT(*)",
	})
	if !errors.Is(err, domain.ErrMetricDuplicate) {
		t.Fatalf("expected ErrMetricDuplicate, got %v", err)
	}
}

func TestDatasetService_CreateMetricInvalidNameReturnsError(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	_, err = svc.CreateMetric(context.Background(), 10, 1, domain.CreateMetricRequest{
		MetricName: "ab",
		MetricType: "SUM",
		Expression: "COUNT(*)",
	})
	if !errors.Is(err, domain.ErrInvalidDataset) {
		t.Fatalf("expected ErrInvalidDataset for short name, got %v", err)
	}
}

func TestDatasetService_CreateMetricDatasetNotFound(t *testing.T) {
	repo := &fakeDatasetRepository{}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	_, err = svc.CreateMetric(context.Background(), 10, 999, domain.CreateMetricRequest{
		MetricName: "total_count",
		MetricType: "SUM",
		Expression: "COUNT(*)",
	})
	if !errors.Is(err, domain.ErrDatasetNotFound) {
		t.Fatalf("expected ErrDatasetNotFound, got %v", err)
	}
}

func TestDatasetService_UpdateMetricSuccess(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
		metrics: map[uint][]domain.SqlMetric{
			1: {{ID: 1, TableID: 1, MetricName: "total_count", MetricType: "SUM", Expression: "COUNT(*)"}},
		},
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	updated, err := svc.UpdateMetric(context.Background(), 10, 1, 1, domain.UpdateMetricRequest{
		MetricName: "new_count",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if updated.ID != 1 {
		t.Fatalf("expected metric ID 1, got %d", updated.ID)
	}
}

func TestDatasetService_UpdateMetricNotFound(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
		metrics: map[uint][]domain.SqlMetric{
			1: {{ID: 1, TableID: 1, MetricName: "total_count", MetricType: "SUM", Expression: "COUNT(*)"}},
		},
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	_, err = svc.UpdateMetric(context.Background(), 10, 1, 999, domain.UpdateMetricRequest{
		MetricName: "new_count",
	})
	if !errors.Is(err, domain.ErrMetricNotFound) {
		t.Fatalf("expected ErrMetricNotFound, got %v", err)
	}
}

func TestDatasetService_DeleteMetricSuccess(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
		metrics: map[uint][]domain.SqlMetric{
			1: {{ID: 1, TableID: 1, MetricName: "total_count", MetricType: "SUM", Expression: "COUNT(*)"}},
		},
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	result, err := svc.DeleteMetric(context.Background(), 10, 1, 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
}

func TestDatasetService_GetMetricsSuccess(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
		metrics: map[uint][]domain.SqlMetric{
			1: {
				{ID: 1, TableID: 1, MetricName: "total_count", MetricType: "SUM", Expression: "COUNT(*)"},
				{ID: 2, TableID: 1, MetricName: "avg_sales", MetricType: "AVG", Expression: "AVG(sales)"},
			},
		},
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	metrics, err := svc.GetMetrics(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(metrics))
	}
}

func TestDatasetService_BulkUpdateMetricsSuccess(t *testing.T) {
	repo := &fakeDatasetRepository{
		datasets: map[uint]*domain.Dataset{
			1: {ID: 1, Name: "orders", DatabaseID: 1, CreatedByFK: 10},
		},
		metrics: map[uint][]domain.SqlMetric{
			1: {{ID: 1, TableID: 1, MetricName: "total_count", MetricType: "SUM", Expression: "COUNT(*)"}},
		},
	}
	databaseLookupRepo := &fakeDatabaseLookupRepository{
		roleNames: []string{"Admin"},
	}
	svc, err := datasetsvc.NewService(repo, databaseLookupRepo, &fakeSyncQueue{})
	if err != nil {
		t.Fatalf("expected nil error creating service, got %v", err)
	}

	result, err := svc.BulkUpdateMetrics(context.Background(), 10, 1, domain.BulkUpdateMetricsRequest{
		Metrics: []domain.MetricUpsertRequest{
			{MetricName: "new_metric", MetricType: "SUM", Expression: "SUM(amount)"},
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result.UpdatedCount != 1 {
		t.Fatalf("expected 1 updated count, got %d", result.UpdatedCount)
	}
}
