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
	var tabs []*query.TabState
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND active = ?", userID, true).
		Order("created_on ASC").
		Find(&tabs).Error
	if err != nil {
		return nil, fmt.Errorf("listing tabs by user: %w", err)
	}
	return tabs, nil
}

func (r *sqllabRepo) GetByID(ctx context.Context, id uint, userID uint) (*query.TabState, error) {
	var tab query.TabState
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&tab).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting tab by id: %w", err)
	}
	return &tab, nil
}

var _ query.SQLLabRepository = (*sqllabRepo)(nil)
