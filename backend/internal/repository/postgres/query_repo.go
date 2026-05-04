package postgres

import (
	"context"
	"fmt"

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

func (r *queryRepo) Update(ctx context.Context, q *query.Query) error {
	return r.db.WithContext(ctx).Save(q).Error
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