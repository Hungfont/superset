package postgres

import (
	"context"

	"superset/auth-service/internal/domain/auth"
	"gorm.io/gorm"
)

type registerUserRepo struct {
	db *gorm.DB
}

// NewRegisterUserRepository returns a GORM-backed RegisterUserRepository.
func NewRegisterUserRepository(db *gorm.DB) auth.RegisterUserRepository {
	return &registerUserRepo{db: db}
}

func (r *registerUserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&auth.User{}).
		Where("email = ?", email).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	err = r.db.WithContext(ctx).
		Model(&auth.RegisterUser{}).
		Where("email = ?", email).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *registerUserRepo) UsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&auth.User{}).
		Where("username = ?", username).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	err = r.db.WithContext(ctx).
		Model(&auth.RegisterUser{}).
		Where("username = ?", username).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *registerUserRepo) Create(ctx context.Context, reg *auth.RegisterUser) error {
	return r.db.WithContext(ctx).Create(reg).Error
}
