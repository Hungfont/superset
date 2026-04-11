package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"superset/auth-service/internal/domain/auth"
	"gorm.io/gorm"
)

type verifyRepo struct {
	db *gorm.DB
}

// NewVerifyRepository returns a GORM-backed VerifyRepository.
func NewVerifyRepository(db *gorm.DB) auth.VerifyRepository {
	return &verifyRepo{db: db}
}

func (r *verifyRepo) FindByHash(ctx context.Context, hash string) (*auth.RegisterUser, error) {
	var reg auth.RegisterUser
	err := r.db.WithContext(ctx).
		Where("registration_hash = ?", hash).
		First(&reg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding registration by hash: %w", err)
	}
	return &reg, nil
}

// isUniqueViolation returns true for PostgreSQL unique-constraint errors (code 23505).
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505") ||
		strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "duplicate key")
}

func (r *verifyRepo) Activate(ctx context.Context, reg *auth.RegisterUser) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := &auth.User{
			FirstName: reg.FirstName,
			LastName:  reg.LastName,
			Username:  reg.Username,
			Email:     reg.Email,
			Password:  reg.Password,
			Active:    true,
		}
		if err := tx.Create(user).Error; err != nil {
			// Unique constraint violation means another request already activated this hash.
			if isUniqueViolation(err) {
				return auth.ErrAlreadyActivated
			}
			return fmt.Errorf("creating user: %w", err)
		}
		if err := tx.Delete(reg).Error; err != nil {
			return fmt.Errorf("deleting pending registration: %w", err)
		}
		return nil
	})
}
