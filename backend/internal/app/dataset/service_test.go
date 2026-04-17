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
}

func (f *fakeDatasetRepository) ExistsPhysicalDataset(_ context.Context, _ uint, _ string, _ string) (bool, error) {
	if f.datasetExistsErr != nil {
		return false, f.datasetExistsErr
	}
	return f.datasetExists, nil
}

func (f *fakeDatasetRepository) CreatePhysicalDataset(_ context.Context, dataset *domain.Dataset) error {
	if f.createErr != nil {
		return f.createErr
	}
	copyValue := *dataset
	if copyValue.ID == 0 {
		copyValue.ID = 42
	}
	f.createdDataset = &copyValue
	dataset.ID = copyValue.ID
	return nil
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
