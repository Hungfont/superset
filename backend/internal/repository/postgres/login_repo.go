package postgres

import (
	"context"
	"fmt"
	"time"

	domain "superset/auth-service/internal/domain/auth"

	"gorm.io/gorm"
)

// loginRepo implements domain.LoginRepository using PostgreSQL via GORM.
type loginRepo struct {
	db *gorm.DB
}

// NewLoginRepository returns a LoginRepository backed by PostgreSQL.
func NewLoginRepository(db *gorm.DB) domain.LoginRepository {
	return &loginRepo{db: db}
}

func (r *loginRepo) FindByUsernameOrEmail(ctx context.Context, identifier string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("username = ? OR email = ?", identifier, identifier).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("finding user: %w", err)
	}
	return &user, nil
}

func (r *loginRepo) UpdateLastLogin(ctx context.Context, userID uint, loginCount int, lastLogin time.Time) error {
	err := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"login_count": loginCount,
			"last_login":  lastLogin,
		}).Error
	if err != nil {
		return fmt.Errorf("updating last login: %w", err)
	}
	return nil
}
