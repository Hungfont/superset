package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/dataset"

	"gorm.io/gorm"
)

type datasetRepo struct {
	db *gorm.DB
}

func NewDatasetRepository(db *gorm.DB) domain.Repository {
	return &datasetRepo{db: db}
}

func (r *datasetRepo) ExistsPhysicalDataset(ctx context.Context, databaseID uint, schema string, tableName string) (bool, error) {
	normalizedSchema := strings.TrimSpace(schema)
	normalizedTable := strings.TrimSpace(tableName)
	if normalizedTable == "" {
		return false, domain.ErrInvalidDataset
	}

	var count int64
	err := r.db.WithContext(ctx).
		Table("tables").
		Where("database_id = ?", databaseID).
		Where("COALESCE(schema, '') = ?", normalizedSchema).
		Where("LOWER(table_name) = LOWER(?)", normalizedTable).
		Where("COALESCE(NULLIF(TRIM(sql), ''), '') = ''").
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking physical dataset duplicate: %w", err)
	}

	return count > 0, nil
}

func (r *datasetRepo) CreatePhysicalDataset(ctx context.Context, dataset *domain.Dataset) error {
	if err := r.db.WithContext(ctx).Create(dataset).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDatasetDuplicate
		}
		return fmt.Errorf("creating physical dataset: %w", err)
	}

	return nil
}

func (r *datasetRepo) ExistsVirtualDataset(ctx context.Context, databaseID uint, tableName string) (bool, error) {
	normalizedTable := strings.TrimSpace(tableName)
	if normalizedTable == "" {
		return false, domain.ErrInvalidDataset
	}

	var count int64
	err := r.db.WithContext(ctx).
		Table("tables").
		Where("database_id = ?", databaseID).
		Where("LOWER(table_name) = LOWER(?)", normalizedTable).
		Where("COALESCE(NULLIF(TRIM(sql), ''), '') != ''").
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking virtual dataset duplicate: %w", err)
	}

	return count > 0, nil
}

func (r *datasetRepo) CreateVirtualDataset(ctx context.Context, dataset *domain.Dataset) error {
	if err := r.db.WithContext(ctx).Create(dataset).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDatasetDuplicate
		}
		return fmt.Errorf("creating virtual dataset: %w", err)
	}

	return nil
}

func (r *datasetRepo) GetDatasetByID(ctx context.Context, id uint) (*domain.Dataset, error) {
	var dataset domain.Dataset
	err := r.db.WithContext(ctx).
		Table("tables").
		Where("id = ?", id).
		First(&dataset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting dataset by id: %w", err)
	}

	return &dataset, nil
}

func (r *datasetRepo) CreateColumns(ctx context.Context, columns []domain.Column) error {
	if len(columns) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).Table("table_columns").Create(&columns).Error; err != nil {
		return fmt.Errorf("creating columns: %w", err)
	}

	return nil
}

var _ domain.Repository = (*datasetRepo)(nil)
