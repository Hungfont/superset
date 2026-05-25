package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xwb1989/sqlparser"
	"gorm.io/gorm"
	query "superset/auth-service/internal/domain/query"
)

type sqllabRepo struct {
	db *gorm.DB
}

func NewSQLLabRepository(db *gorm.DB) query.SQLLabRepository {
	return &sqllabRepo{db: db}
}

func extractSQLTables(sqlText string) string {
	stmt, err := sqlparser.Parse(sqlText)
	if err != nil {
		return ""
	}
	tables := make(map[string]bool)
	_ = sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		if tn, ok := node.(*sqlparser.TableName); ok {
			name := tn.Name.String()
			if q := tn.Qualifier.String(); q != "" {
				name = q + "." + name
			}
			tables[strings.ToLower(name)] = true
		}
		return true, nil
	}, stmt)
	names := make([]string, 0, len(tables))
	for t := range tables {
		names = append(names, t)
	}
	return strings.Join(names, ",")
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

func (r *sqllabRepo) CreateSavedQuery(ctx context.Context, sq *query.SavedQuery) error {
	sq.SQLTables = extractSQLTables(sq.SQL)
	return r.db.WithContext(ctx).Create(sq).Error
}

func (r *sqllabRepo) LabelExists(ctx context.Context, userID uint, label string) (bool, error) {
	var existing query.SavedQuery
	err := r.db.WithContext(ctx).
		Where("LOWER(label) = LOWER(?) AND created_by_fk = ?", label, userID).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check label exists: %w", err)
	}
	return true, nil
}

func (r *sqllabRepo) ListSavedQueries(ctx context.Context, userID uint, params query.SavedQueryListParams) ([]*query.SavedQuery, int64, error) {
	q := r.db.WithContext(ctx).Model(&query.SavedQuery{}).Where("created_by_fk = ?", userID)

	if params.Search != "" {
		like := "%" + strings.ToLower(params.Search) + "%"
		q = q.Where("LOWER(label) LIKE ?", like)
	}

	if params.Published != nil {
		q = q.Where("published = ?", *params.Published)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count saved queries: %w", err)
	}

	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	offset := (params.Page - 1) * params.Limit

	var rows []*query.SavedQuery
	if err := q.Order("changed_on DESC").Limit(params.Limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list saved queries: %w", err)
	}
	return rows, total, nil
}

func (r *sqllabRepo) GetSavedQuery(ctx context.Context, id uint, userID uint) (*query.SavedQuery, error) {
	var sq query.SavedQuery
	err := r.db.WithContext(ctx).Where("id = ? AND created_by_fk = ?", id, userID).First(&sq).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get saved query: %w", err)
	}
	return &sq, nil
}

func (r *sqllabRepo) UpdateSavedQuery(ctx context.Context, sq *query.SavedQuery) error {
	if sq.SQL != "" {
		sq.SQLTables = extractSQLTables(sq.SQL)
	}
	return r.db.WithContext(ctx).Save(sq).Error
}

func (r *sqllabRepo) DeleteSavedQuery(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sq query.SavedQuery
		if err := tx.Where("id = ? AND created_by_fk = ?", id, userID).First(&sq).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("not found")
			}
			return fmt.Errorf("delete saved query: find: %w", err)
		}
		if err := tx.Model(&query.TabState{}).Where("saved_query_id = ?", id).Update("saved_query_id", gorm.Expr("NULL")).Error; err != nil {
			return fmt.Errorf("delete saved query: null tab refs: %w", err)
		}
		if err := tx.Delete(&sq).Error; err != nil {
			return fmt.Errorf("delete saved query: %w", err)
		}
		return nil
	})
}

func (r *sqllabRepo) ForkSavedQuery(ctx context.Context, id uint, userID uint) (*query.SavedQuery, error) {
	var original query.SavedQuery
	err := r.db.WithContext(ctx).Where("id = ? AND created_by_fk = ?", id, userID).First(&original).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("not found")
		}
		return nil, fmt.Errorf("fork saved query: find: %w", err)
	}

	now := time.Now()
	fork := original
	fork.ID = 0
	fork.Label = "Copy of " + original.Label
	fork.CreatedOn = now
	fork.ChangedOn = now
	fork.SQLTables = original.SQLTables

	if err := r.db.WithContext(ctx).Create(&fork).Error; err != nil {
		return nil, fmt.Errorf("fork saved query: create: %w", err)
	}
	return &fork, nil
}

var _ query.SQLLabRepository = (*sqllabRepo)(nil)
