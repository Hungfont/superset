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
			tables.perm, tables.description, tables.main_dttm_col, tables.cache_timeout,
			tables.filter_select_enabled, tables.normalize_columns, tables.is_featured,
			tables.created_by_fk, owners.username as owner_name,
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

func (r *datasetRepo) UpdateDatasetMetadata(ctx context.Context, id uint, req domain.UpdateDatasetMetadataRequest) error {
	updates := make(map[string]interface{})

	if req.TableName != "" {
		updates["table_name"] = req.TableName
	}
	if req.Description != "" || req.Description == "" {
		updates["description"] = req.Description
	}
	if req.MainDttmCol != "" {
		updates["main_dttm_col"] = req.MainDttmCol
	}
	updates["cache_timeout"] = req.CacheTimeout
	updates["normalize_columns"] = req.NormalizeColumns
	updates["filter_select_enabled"] = req.FilterSelectEnabled
	updates["is_featured"] = req.IsFeatured
	if req.SQL != "" {
		updates["sql"] = req.SQL
	}

	if len(updates) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).Table("tables").Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("updating dataset metadata: %w", err)
	}

	return nil
}

func (r *datasetRepo) GetColumnByName(ctx context.Context, tableID uint, columnName string) (*domain.Column, error) {
	var column domain.Column
	err := r.db.WithContext(ctx).Table("table_columns").
		Where("table_id = ? AND column_name = ? AND is_active = true", tableID, columnName).
		First(&column).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting column by name: %w", err)
	}

	return &column, nil
}

func (r *datasetRepo) GetColumnByID(ctx context.Context, columnID uint) (*domain.Column, error) {
	var column domain.Column
	err := r.db.WithContext(ctx).Table("table_columns").
		Where("id = ? AND is_active = true", columnID).
		First(&column).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting column by id: %w", err)
	}

	return &column, nil
}

func (r *datasetRepo) UpdateColumn(ctx context.Context, columnID uint, req domain.UpdateColumnRequest) error {
	updates := make(map[string]interface{})

	if req.VerboseName != "" || req.VerboseName == "" {
		updates["verbose_name"] = req.VerboseName
	}
	if req.Description != "" || req.Description == "" {
		updates["description"] = req.Description
	}
	if req.Filterable != nil {
		updates["filterable"] = *req.Filterable
	}
	if req.GroupBy != nil {
		updates["groupby"] = *req.GroupBy
	}
	if req.IsDateTime != nil {
		updates["is_dttm"] = *req.IsDateTime
	}
	if req.PythonDateFormat != "" {
		updates["python_date_format"] = req.PythonDateFormat
	}
	if req.Expression != "" {
		updates["expression"] = req.Expression
	}
	if req.ColumnType != "" {
		updates["type"] = req.ColumnType
	}
	if req.Exported != nil {
		updates["exported"] = *req.Exported
	}

	if len(updates) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).Table("table_columns").Where("id = ?", columnID).Updates(updates).Error; err != nil {
		return fmt.Errorf("updating column: %w", err)
	}

	return nil
}

func (r *datasetRepo) BulkUpdateColumns(ctx context.Context, columns []domain.UpdateColumnRequest) error {
	if len(columns) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, col := range columns {
			updates := make(map[string]interface{})

			if col.VerboseName != "" || col.VerboseName == "" {
				updates["verbose_name"] = col.VerboseName
			}
			if col.Description != "" || col.Description == "" {
				updates["description"] = col.Description
			}
			if col.Filterable != nil {
				updates["filterable"] = *col.Filterable
			}
			if col.GroupBy != nil {
				updates["groupby"] = *col.GroupBy
			}
			if col.IsDateTime != nil {
				updates["is_dttm"] = *col.IsDateTime
			}
			if col.PythonDateFormat != "" {
				updates["python_date_format"] = col.PythonDateFormat
			}
			if col.Expression != "" {
				updates["expression"] = col.Expression
			}
			if col.ColumnType != "" {
				updates["type"] = col.ColumnType
			}
			if col.Exported != nil {
				updates["exported"] = *col.Exported
			}

			if len(updates) == 0 {
				continue
			}

			if err := tx.Table("table_columns").Where("id = ?", col.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("updating column %d: %w", col.ID, err)
			}
		}
		return nil
	})
}

func (r *datasetRepo) GetMetricByID(ctx context.Context, metricID uint) (*domain.SqlMetric, error) {
	var metric domain.SqlMetric
	err := r.db.WithContext(ctx).Table("sql_metrics").
		Where("id = ?", metricID).
		First(&metric).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting metric by id: %w", err)
	}

	return &metric, nil
}

func (r *datasetRepo) GetMetricsByTableID(ctx context.Context, tableID uint) ([]domain.SqlMetric, error) {
	var metrics []domain.SqlMetric
	err := r.db.WithContext(ctx).Table("sql_metrics").
		Where("table_id = ?", tableID).
		Order("metric_name").
		Scan(&metrics).Error
	if err != nil {
		return nil, fmt.Errorf("getting metrics by table id: %w", err)
	}

	if metrics == nil {
		metrics = []domain.SqlMetric{}
	}

	return metrics, nil
}

func (r *datasetRepo) CreateMetric(ctx context.Context, metric *domain.SqlMetric) error {
	if err := r.db.WithContext(ctx).Table("sql_metrics").Create(metric).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrMetricDuplicate
		}
		return fmt.Errorf("creating metric: %w", err)
	}

	return nil
}

func (r *datasetRepo) UpdateMetric(ctx context.Context, metricID uint, req domain.UpdateMetricRequest) error {
	updates := make(map[string]interface{})

	if req.MetricName != "" {
		updates["metric_name"] = req.MetricName
	}
	if req.VerboseName != "" || req.VerboseName == "" {
		updates["verbose_name"] = req.VerboseName
	}
	if req.MetricType != "" {
		updates["metric_type"] = req.MetricType
	}
	if req.Expression != "" {
		updates["expression"] = req.Expression
	}
	if req.D3Format != "" || req.D3Format == "" {
		updates["d3_format"] = req.D3Format
	}
	if req.WarningText != "" || req.WarningText == "" {
		updates["warning_text"] = req.WarningText
	}
	if req.IsRestricted != nil {
		updates["is_restricted"] = *req.IsRestricted
	}
	if req.CertifiedBy != "" || req.CertifiedBy == "" {
		updates["certified_by"] = req.CertifiedBy
	}
	if req.CertificationDetails != "" || req.CertificationDetails == "" {
		updates["certification_details"] = req.CertificationDetails
	}

	if len(updates) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).Table("sql_metrics").Where("id = ?", metricID).Updates(updates).Error; err != nil {
		return fmt.Errorf("updating metric: %w", err)
	}

	return nil
}

func (r *datasetRepo) DeleteMetric(ctx context.Context, metricID uint) error {
	if err := r.db.WithContext(ctx).Table("sql_metrics").Where("id = ?", metricID).Delete(nil).Error; err != nil {
		return fmt.Errorf("deleting metric: %w", err)
	}

	return nil
}

func (r *datasetRepo) BulkReplaceMetrics(ctx context.Context, tableID uint, metrics []domain.SqlMetric) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("sql_metrics").Where("table_id = ?", tableID).Delete(nil).Error; err != nil {
			return fmt.Errorf("deleting existing metrics: %w", err)
		}

		if len(metrics) == 0 {
			return nil
		}

		if err := tx.Table("sql_metrics").Create(&metrics).Error; err != nil {
			return fmt.Errorf("creating metrics: %w", err)
		}

		return nil
	})
}

func (r *datasetRepo) MetricNameExists(ctx context.Context, tableID uint, metricName string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("sql_metrics").
		Where("table_id = ?", tableID).
		Where("LOWER(metric_name) = LOWER(?)", metricName)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking metric name exists: %w", err)
	}

	return count > 0, nil
}

func (r *datasetRepo) DeleteDataset(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("table_columns").Where("table_id = ?", id).Delete(nil).Error; err != nil {
			return fmt.Errorf("deleting columns: %w", err)
		}

		if err := tx.Table("sql_metrics").Where("table_id = ?", id).Delete(nil).Error; err != nil {
			return fmt.Errorf("deleting metrics: %w", err)
		}

		if err := tx.Table("tables").Where("id = ?", id).Delete(nil).Error; err != nil {
			return fmt.Errorf("deleting dataset: %w", err)
		}

		return nil
	})
}

func (r *datasetRepo) CountChartsByDatasetID(ctx context.Context, datasetID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("slices").
		Where("datasource_id = ?", datasetID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting charts: %w", err)
	}

	return count, nil
}

var _ domain.Repository = (*datasetRepo)(nil)