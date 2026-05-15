package postgres

import (
	"context"
	"fmt"
	"time"

	"superset/auth-service/internal/domain/query"

	"gorm.io/gorm"
)

type queryRepo struct {
	db *gorm.DB
}

func NewQueryRepository(db *gorm.DB) query.Repository {
	return &queryRepo{db: db}
}

func (r *queryRepo) Create(ctx context.Context, q *query.Query) error {
	return r.db.WithContext(ctx).Create(q).Error
}

func (r *queryRepo) GetByID(ctx context.Context, id string) (*query.Query, error) {
	var q query.Query
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&q).Error; err != nil {
		return nil, fmt.Errorf("getting query by id: %w", err)
	}
	return &q, nil
}

func (r *queryRepo) GetByClientID(ctx context.Context, clientID string) (*query.Query, error) {
	var q query.Query
	if err := r.db.WithContext(ctx).Where("client_id = ?", clientID).First(&q).Error; err != nil {
		return nil, fmt.Errorf("getting query by client_id: %w", err)
	}
	return &q, nil
}

func (r *queryRepo) Update(ctx context.Context, q *query.Query) error {
	return r.db.WithContext(ctx).Save(q).Error
}

func (r *queryRepo) UpdateStatusConditional(ctx context.Context, id string, status string, allowedFromStatuses []string, extra map[string]interface{}) (bool, error) {
	updates := map[string]interface{}{"status": status}
	for k, v := range extra {
		updates[k] = v
	}
	result := r.db.WithContext(ctx).Model(&query.Query{}).
		Where("id = ? AND status IN ?", id, allowedFromStatuses).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *queryRepo) List(ctx context.Context, filter *query.ListFilter) ([]*query.Query, int64, error) {
	db := r.db.WithContext(ctx).Model(&query.Query{})

	if filter.UserID > 0 {
		db = db.Where("user_id = ?", filter.UserID)
	}
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if filter.DatabaseID > 0 {
		db = db.Where("database_id = ?", filter.DatabaseID)
	}
	if filter.SQLLike != "" {
		db = db.Where("sql ILIKE ?", "%"+filter.SQLLike+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting queries: %w", err)
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	offset := (filter.Page - 1) * filter.PageSize

	var queries []*query.Query
	if err := db.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&queries).Error; err != nil {
		return nil, 0, fmt.Errorf("listing queries: %w", err)
	}

	return queries, total, nil
}

func (r *queryRepo) ListHistory(ctx context.Context, filter *query.ListFilter) ([]*query.HistoryResponseItem, int64, error) {
	db := r.db.WithContext(ctx).
		Table("query").
		Select(`query.id, query.client_id, query.status, query.sql, query.database_id,
				COALESCE(dbs.database_name, '') AS database_name, query.rows, query.start_time, query.end_time,
				COALESCE((EXTRACT(EPOCH FROM (query.end_time - query.start_time)) * 1000)::BIGINT, 0) AS duration_ms,
				query.error_message, query.results_key, query.user_id`).
		Joins("LEFT JOIN dbs ON dbs.id = query.database_id")

	if filter.UserID > 0 {
		db = db.Where("query.user_id = ?", filter.UserID)
	}
	if filter.Status != "" {
		db = db.Where("query.status = ?", filter.Status)
	}
	if filter.DatabaseID > 0 {
		db = db.Where("query.database_id = ?", filter.DatabaseID)
	}
	if filter.SQLLike != "" {
		db = db.Where("query.sql ILIKE ?", "%"+filter.SQLLike+"%")
	}

	var total int64
	countDb := db
	if err := countDb.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting history: %w", err)
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	offset := (filter.Page - 1) * filter.PageSize

	var items []*query.HistoryResponseItem
	if err := db.Order("query.start_time DESC NULLS LAST").
		Offset(offset).Limit(filter.PageSize).
		Scan(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("listing history: %w", err)
	}

	return items, total, nil
}

func (r *queryRepo) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("start_time < ?", olderThan).
		Delete(&query.Query{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting old queries: %w", result.Error)
	}
	return result.RowsAffected, nil
}