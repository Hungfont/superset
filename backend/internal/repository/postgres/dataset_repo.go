package postgres

import (
	"context"
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

var _ domain.Repository = (*datasetRepo)(nil)
