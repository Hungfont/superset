package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *sqllabRepo) ListByUser(ctx context.Context, userID uint, includeClosed bool) ([]*query.TabState, error) {
	q := r.db.WithContext(ctx).
		Table("tab_state").
		Select("tab_state.*, query.status AS latest_query_status").
		Joins("LEFT JOIN query ON query.id = tab_state.latest_query_id").
		Where("tab_state.user_id = ?", userID)
	if !includeClosed {
		q = q.Where("tab_state.active = ?", true)
	}
	var rows []*query.TabWithQueryStatus
	err := q.Order("tab_state.created_on ASC").Find(&rows).Error
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

func (r *sqllabRepo) CloseTab(ctx context.Context, id uint, userID uint) error {
	result := r.db.WithContext(ctx).
		Model(&query.TabState{}).
		Where("id = ? AND user_id = ? AND active = ?", id, userID, true).
		Update("active", false)
	if result.Error != nil {
		return fmt.Errorf("close tab: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *sqllabRepo) CloseAllTabs(ctx context.Context, userID uint, exceptID *uint) (int64, error) {
	q := r.db.WithContext(ctx).
		Model(&query.TabState{}).
		Where("user_id = ? AND active = ?", userID, true)
	if exceptID != nil {
		q = q.Where("id != ?", *exceptID)
	}
	result := q.Update("active", false)
	if result.Error != nil {
		return 0, fmt.Errorf("close all tabs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *sqllabRepo) ReopenTab(ctx context.Context, id uint, userID uint) error {
	result := r.db.WithContext(ctx).
		Model(&query.TabState{}).
		Where("id = ? AND user_id = ? AND active = ?", id, userID, false).
		Updates(map[string]interface{}{
			"active":     true,
			"changed_on": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("reopen tab: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *sqllabRepo) HardDelete(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tab query.TabState
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&tab).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("not found")
			}
			return fmt.Errorf("hard delete: find tab: %w", err)
		}
		if err := tx.Where("tab_state_id = ?", id).Delete(&query.TableSchema{}).Error; err != nil {
			return fmt.Errorf("hard delete: table_schema: %w", err)
		}
		if err := tx.Delete(&tab).Error; err != nil {
			return fmt.Errorf("hard delete: tab: %w", err)
		}
		return nil
	})
}

var _ query.SQLLabRepository = (*sqllabRepo)(nil)
