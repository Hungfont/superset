package postgres

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	query "superset/auth-service/internal/domain/query"
)

type sqllabRepo struct {
	db *gorm.DB
}

func NewSQLLabRepository(db *gorm.DB) query.SQLLabRepository {
	return &sqllabRepo{db: db}
}

func (r *sqllabRepo) Create(ctx context.Context, tab *query.TabState) error {
	return r.db.WithContext(ctx).Create(tab).Error
}

func (r *sqllabRepo) ListByUser(ctx context.Context, userID uint) ([]*query.TabState, error) {
	var rows []*query.TabWithQueryStatus
	err := r.db.WithContext(ctx).
		Table("tab_state").
		Select("tab_state.*, query.status AS latest_query_status").
		Joins("LEFT JOIN query ON query.id = tab_state.latest_query_id").
		Where("tab_state.user_id = ? AND tab_state.active = ?", userID, true).
		Order("tab_state.created_on ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("listing tabs by user: %w", err)
	}
	tabs := make([]*query.TabState, 0, len(rows))
	for _, r := range rows {
		t := r.TabState
		if r.LatestQueryStatus != nil {
			t.ExtraJSON = `{"latest_query_status":"` + *r.LatestQueryStatus + `"}`
		}
		tabs = append(tabs, &t)
	}
	return tabs, nil
}

func (r *sqllabRepo) GetByID(ctx context.Context, id uint, userID uint) (*query.TabState, error) {
	var row query.TabWithQueryStatus
	err := r.db.WithContext(ctx).
		Table("tab_state").
		Select("tab_state.*, query.status AS latest_query_status").
		Joins("LEFT JOIN query ON query.id = tab_state.latest_query_id").
		Where("tab_state.id = ? AND tab_state.user_id = ?", id, userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting tab by id: %w", err)
	}
	t := row.TabState
	if row.LatestQueryStatus != nil {
		t.ExtraJSON = `{"latest_query_status":"` + *row.LatestQueryStatus + `"}`
	}
	return &t, nil
}

func (r *sqllabRepo) Update(ctx context.Context, tab *query.TabState) error {
	return r.db.WithContext(ctx).Save(tab).Error
}

var _ query.SQLLabRepository = (*sqllabRepo)(nil)
