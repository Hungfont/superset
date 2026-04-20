package worker

import (
	"context"

	"superset/auth-service/internal/domain/dataset"
	dbdomain "superset/auth-service/internal/domain/db"
)

type datasetRepoWithDB interface {
	GetDatasetByID(ctx context.Context, id uint) (*dataset.Dataset, error)
	RefreshDatasetColumns(ctx context.Context, datasetID uint, columns []dataset.Column) error
	GetDatabaseByID(ctx context.Context, id uint) (*dbdomain.Database, error)
}

type DatasetRepoWrapper struct {
	datasetRepo dataset.Repository
	dbRepo     dbdomain.DatabaseRepository
}

func NewDatasetRepoWrapper(datasetRepo dataset.Repository, dbRepo dbdomain.DatabaseRepository) *DatasetRepoWrapper {
	return &DatasetRepoWrapper{
		datasetRepo: datasetRepo,
		dbRepo:     dbRepo,
	}
}

func (w *DatasetRepoWrapper) GetDatasetByID(ctx context.Context, id uint) (*dataset.Dataset, error) {
	return w.datasetRepo.GetDatasetByID(ctx, id)
}

func (w *DatasetRepoWrapper) RefreshDatasetColumns(ctx context.Context, datasetID uint, columns []dataset.Column) error {
	return w.datasetRepo.RefreshDatasetColumns(ctx, datasetID, columns)
}

func (w *DatasetRepoWrapper) GetDatabaseByID(ctx context.Context, id uint) (*dbdomain.Database, error) {
	return w.dbRepo.GetDatabaseByID(ctx, id)
}