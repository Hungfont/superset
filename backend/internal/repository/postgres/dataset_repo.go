package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/dataset"

	"gorm.io/gorm"
)

const (
	defaultListPage     = 1
	defaultListPageSize = 10
	maxListPageSize     = 100
)

type datasetRepo struct {
	db *gorm.DB
}

func NewDatasetRepository(db *gorm.DB) domain.Repository {
	return &datasetRepo{db: db}
}

func (r *datasetRepo) ExistsPhysicalDataset(ctx context.Context, databaseID uint, schema string, tableName string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("tables").
		Where("database_id = ?", databaseID).
		Where("COALESCE(schema, '') = ?", schema).
		Where("LOWER(table_name) = LOWER(?)", tableName).
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
	var count int64
	err := r.db.WithContext(ctx).
		Table("tables").
		Where("database_id = ?", databaseID).
		Where("LOWER(table_name) = LOWER(?)", tableName).
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

func (r *datasetRepo) ListDatasets(ctx context.Context, actorUserID uint, filters domain.DatasetListFilters) (*domain.DatasetListResult, error) {
	// Visibility scope → filter chain → GORM.Paginate
	query := r.db.WithContext(ctx).Table("tables").
		Select(`tables.id, tables.table_name, tables.schema, tables.database_id, 
			dbs.database_name, 
			CASE WHEN tables.sql IS NULL OR tables.sql = '' THEN 'physical' ELSE 'virtual' END as type,
			tables.perm, tables.created_by_fk, 
			ab_user.username as owner_name,
			(SELECT COUNT(*) FROM table_columns WHERE table_id = tables.id AND is_active = true) as column_count,
			(SELECT COUNT(*) FROM sql_metrics WHERE table_id = tables.id) as metric_count,
			tables.changed_on`).
		Joins("LEFT JOIN dbs ON dbs.id = tables.database_id").
		Joins("LEFT JOIN ab_user ON ab_user.id = tables.created_by_fk")

	// 1. Visibility scope filter
	switch filters.VisibilityScope {
	case domain.VisibilityScopeAdmin, domain.VisibilityScopeAlpha:
		// Admin/Alpha can see all datasets - no additional filter
	case domain.VisibilityScopeGamma:
		// Gamma can only see their own datasets
		query = query.Where("tables.created_by_fk = ?", filters.ActorUserID)
	default:
		// Default: can only see own datasets
		query = query.Where("tables.created_by_fk = ?", filters.ActorUserID)
	}

	// 2. Filter chain
	if filters.SearchQ != "" {
		searchPattern := "%" + strings.TrimSpace(filters.SearchQ) + "%"
		query = query.Where("tables.table_name ILIKE ? OR dbs.database_name ILIKE ?", searchPattern, searchPattern)
	}

	if filters.DatabaseID > 0 {
		query = query.Where("tables.database_id = ?", filters.DatabaseID)
	}

	if filters.Schema != "" {
		query = query.Where("tables.schema = ?", filters.Schema)
	}

	if filters.Type != "" {
		if filters.Type == "physical" {
			query = query.Where("(tables.sql IS NULL OR tables.sql = '')")
		} else if filters.Type == "virtual" {
			query = query.Where("(tables.sql IS NOT NULL AND tables.sql != '')")
		}
	}

	if filters.Owner > 0 {
		query = query.Where("tables.created_by_fk = ?", filters.Owner)
	}

	// Order by then pagination
	orderBy := "tables.changed_on DESC"
	if filters.OrderBy != "" {
		orderBy = filters.OrderBy
	}
	query = query.Order(orderBy)

	offset := (filters.Page - 1) * filters.Limit
	query = query.Offset(offset).Limit(filters.Limit)

	// 3. GORM.Paginate - execute query
	var items []domain.DatasetWithCounts
	if err := query.Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("listing datasets: %w", err)
	}

	if items == nil {
		items = []domain.DatasetWithCounts{}
	}

	// Count total for pagination
	var total int64
	countQuery := r.db.WithContext(ctx).Table("tables").
		Joins("LEFT JOIN dbs ON dbs.id = tables.database_id")

	switch filters.VisibilityScope {
	case domain.VisibilityScopeAdmin, domain.VisibilityScopeAlpha:
	case domain.VisibilityScopeGamma:
		countQuery = countQuery.Where("tables.created_by_fk = ?", filters.ActorUserID)
	default:
		countQuery = countQuery.Where("tables.created_by_fk = ?", filters.ActorUserID)
	}

	if filters.SearchQ != "" {
		searchPattern := "%" + strings.TrimSpace(filters.SearchQ) + "%"
		countQuery = countQuery.Where("tables.table_name ILIKE ? OR dbs.database_name ILIKE ?", searchPattern, searchPattern)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("counting datasets: %w", err)
	}

	return &domain.DatasetListResult{
		Items:    items,
		Total:    total,
		Page:     filters.Page,
		PageSize: filters.Limit,
	}, nil
}

func (r *datasetRepo) GetDatasetDetail(ctx context.Context, id uint) (*domain.DatasetDetail, error) {
	var dataset domain.DatasetWithCounts
	err := r.db.WithContext(ctx).Table("tables").
		Select(`tables.id, tables.table_name, tables.schema, tables.database_id, dbs.database_name, 
			CASE WHEN tables.sql = '' OR tables.sql IS NULL THEN 'physical' ELSE 'virtual' END as type,
			tables.perm, tables.created_by_fk, owners.username as owner_name,
			(SELECT COUNT(*) FROM table_columns WHERE table_id = tables.id AND is_active = true) as column_count,
			(SELECT COUNT(*) FROM sql_metrics WHERE table_id = tables.id) as metric_count,
			tables.changed_on`).
		Joins("LEFT JOIN dbs ON dbs.id = tables.database_id").
		Joins("LEFT JOIN ab_user as owners ON owners.id = tables.created_by_fk").
		Where("tables.id = ?", id).
		Scan(&dataset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDatasetNotFound
		}
		return nil, fmt.Errorf("getting dataset detail: %w", err)
	}

	if dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	var columns []domain.Column
	if err := r.db.WithContext(ctx).Table("table_columns").
		Where("table_id = ? AND is_active = true", id).
		Order("column_name").
		Scan(&columns).Error; err != nil {
		return nil, fmt.Errorf("getting dataset columns: %w", err)
	}

	var metrics []domain.SqlMetric
	if err := r.db.WithContext(ctx).Table("sql_metrics").
		Where("table_id = ?", id).
		Order("metric_name").
		Scan(&metrics).Error; err != nil {
		return nil, fmt.Errorf("getting dataset metrics: %w", err)
	}

	return &domain.DatasetDetail{
		DatasetWithCounts: dataset,
		TableColumns:    columns,
		SqlMetrics:     metrics,
	}, nil
}

var _ domain.Repository = (*datasetRepo)(nil)