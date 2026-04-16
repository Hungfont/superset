package postgres

import (
	"context"
	"errors"
	"fmt"

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

var _ domain.DatabaseRepository = (*databaseRepo)(nil)
