package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/db"

	"gorm.io/gorm"
)

type databaseRepo struct {
	db *gorm.DB
}

func NewDatabaseRepository(db *gorm.DB) domain.DatabaseRepository {
	return &databaseRepo{db: db}
}

func (r *databaseRepo) IsAdmin(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("ab_user_role ur").
		Joins("JOIN ab_role ro ON ro.id = ur.role_id").
		Where("ur.user_id = ? AND LOWER(ro.name) = ?", userID, "admin").
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking admin role: %w", err)
	}
	return count > 0, nil
}

func (r *databaseRepo) DatabaseNameExists(ctx context.Context, databaseName string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("dbs").
		Where("LOWER(database_name) = LOWER(?)", databaseName).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking database name exists: %w", err)
	}
	return count > 0, nil
}

func (r *databaseRepo) GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error) {
	roleNames := make([]string, 0)
	err := r.db.WithContext(ctx).
		Table("ab_user_role ur").
		Select("LOWER(ro.name)").
		Joins("JOIN ab_role ro ON ro.id = ur.role_id").
		Where("ur.user_id = ?", userID).
		Pluck("LOWER(ro.name)", &roleNames).Error
	if err != nil {
		return nil, fmt.Errorf("listing role names by user: %w", err)
	}

	return roleNames, nil
}

func (r *databaseRepo) ListDatabases(ctx context.Context, filters domain.DatabaseListFilters) (domain.DatabaseListResult, error) {
	base := r.db.WithContext(ctx).
		Table("dbs").
		Joins("LEFT JOIN tables ON tables.database_id = dbs.id")

	base = applyVisibilityScope(base, filters.VisibilityScope, filters.ActorUserID)

	if filters.SearchQ != "" {
		base = base.Where("LOWER(dbs.database_name) LIKE LOWER(?)", "%"+filters.SearchQ+"%")
	}

	if filters.Backend != "" {
		base = base.Where("LOWER(split_part(dbs.sqlalchemy_uri, '://', 1)) = ?", strings.ToLower(filters.Backend))
	}

	var total int64
	if err := base.
		Distinct("dbs.id").
		Count(&total).Error; err != nil {
		return domain.DatabaseListResult{}, fmt.Errorf("counting databases: %w", err)
	}

	items := make([]domain.DatabaseWithDatasetCount, 0)
	err := base.
		Select(`
			dbs.id,
			dbs.database_name,
			dbs.sqlalchemy_uri,
			dbs.allow_dml,
			dbs.expose_in_sqllab,
			dbs.allow_run_async,
			dbs.allow_file_upload,
			dbs.created_by_fk,
			COALESCE(COUNT(DISTINCT tables.id), 0) AS dataset_count
		`).
		Group("dbs.id").
		Order("dbs.id DESC").
		Offset(filters.Offset).
		Limit(filters.Limit).
		Scan(&items).Error
	if err != nil {
		return domain.DatabaseListResult{}, fmt.Errorf("listing databases: %w", err)
	}

	return domain.DatabaseListResult{Items: items, Total: total}, nil
}

func (r *databaseRepo) GetVisibleDatabaseByID(ctx context.Context, databaseID uint, scope domain.DatabaseVisibilityScope, actorUserID uint) (*domain.DatabaseWithDatasetCount, error) {
	query := r.db.WithContext(ctx).
		Table("dbs").
		Joins("LEFT JOIN tables ON tables.database_id = dbs.id")

	query = applyVisibilityScope(query, scope, actorUserID)

	var result domain.DatabaseWithDatasetCount
	err := query.
		Select(`
			dbs.id,
			dbs.database_name,
			dbs.sqlalchemy_uri,
			dbs.allow_dml,
			dbs.expose_in_sqllab,
			dbs.allow_run_async,
			dbs.allow_file_upload,
			dbs.created_by_fk,
			COALESCE(COUNT(DISTINCT tables.id), 0) AS dataset_count
		`).
		Where("dbs.id = ?", databaseID).
		Group("dbs.id").
		Take(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDatabaseNotFound
		}
		return nil, fmt.Errorf("loading visible database by id: %w", err)
	}

	return &result, nil
}

func (r *databaseRepo) GetDatabaseByID(ctx context.Context, databaseID uint) (*domain.Database, error) {
	var database domain.Database
	err := r.db.WithContext(ctx).Table("dbs").Where("id = ?", databaseID).First(&database).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDatabaseNotFound
		}
		return nil, fmt.Errorf("loading database by id: %w", err)
	}
	return &database, nil
}

func (r *databaseRepo) CreateDatabase(ctx context.Context, database *domain.Database) error {
	if err := r.db.WithContext(ctx).Create(database).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDatabaseNameExists
		}
		return fmt.Errorf("creating database: %w", err)
	}
	return nil
}

func (r *databaseRepo) UpdateDatabase(ctx context.Context, database *domain.Database) error {
	if database == nil || database.ID == 0 {
		return domain.ErrInvalidDatabase
	}

	result := r.db.WithContext(ctx).
		Table("dbs").
		Where("id = ?", database.ID).
		Updates(map[string]any{
			"database_name":     database.DatabaseName,
			"sqlalchemy_uri":    database.SQLAlchemyURI,
			"allow_dml":         database.AllowDML,
			"expose_in_sqllab":  database.ExposeInSQLLab,
			"allow_run_async":   database.AllowRunAsync,
			"allow_file_upload": database.AllowFileUpload,
		})
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return domain.ErrDatabaseNameExists
		}
		return fmt.Errorf("updating database: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrDatabaseNotFound
	}

	return nil
}

func (r *databaseRepo) DeleteDatabase(ctx context.Context, databaseID uint) error {
	result := r.db.WithContext(ctx).Table("dbs").Where("id = ?", databaseID).Delete(&domain.Database{})
	if result.Error != nil {
		return fmt.Errorf("deleting database: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrDatabaseNotFound
	}
	return nil
}

func (r *databaseRepo) CountDatasetsByDatabaseID(ctx context.Context, databaseID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("tables").Where("database_id = ?", databaseID).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting datasets by database id: %w", err)
	}
	return count, nil
}

func applyVisibilityScope(query *gorm.DB, scope domain.DatabaseVisibilityScope, actorUserID uint) *gorm.DB {
	switch scope {
	case domain.DatabaseVisibilityAdmin:
		return query
	case domain.DatabaseVisibilityAlpha:
		return query.Where("dbs.created_by_fk = ? OR dbs.expose_in_sqllab = ?", actorUserID, true)
	default:
		return query.Where("dbs.expose_in_sqllab = ?", true)
	}
}

var _ domain.DatabaseRepository = (*databaseRepo)(nil)
