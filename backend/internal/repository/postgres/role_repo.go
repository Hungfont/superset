package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/auth"

	"gorm.io/gorm"
)

type roleRepo struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) domain.RoleRepository {
	return &roleRepo{db: db}
}

func (r *roleRepo) IsAdmin(ctx context.Context, userID uint) (bool, error) {
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

func (r *roleRepo) ListWithCounts(ctx context.Context) ([]domain.RoleListItem, error) {
	var roles []domain.RoleListItem
	err := r.db.WithContext(ctx).
		Table("ab_role r").
		Select(`
			r.id,
			r.name,
			COUNT(DISTINCT ur.user_id) AS user_count,
			COUNT(DISTINCT pvr.permission_view_id) AS permission_count
		`).
		Joins("LEFT JOIN ab_user_role ur ON ur.role_id = r.id").
		Joins("LEFT JOIN ab_permission_view_role pvr ON pvr.role_id = r.id").
		Group("r.id, r.name").
		Order("r.id ASC").
		Scan(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("listing roles with counts: %w", err)
	}

	for i := range roles {
		roles[i].BuiltIn = isBuiltInRoleName(roles[i].Name)
	}

	return roles, nil
}

func (r *roleRepo) Create(ctx context.Context, role *domain.Role) error {
	if err := r.db.WithContext(ctx).Create(role).Error; err != nil {
		return fmt.Errorf("creating role: %w", err)
	}
	return nil
}

func (r *roleRepo) UpdateName(ctx context.Context, roleID uint, name string) (*domain.Role, error) {
	res := r.db.WithContext(ctx).
		Model(&domain.Role{}).
		Where("id = ?", roleID).
		Update("name", name)
	if res.Error != nil {
		return nil, fmt.Errorf("updating role name: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, domain.ErrRoleNotFound
	}

	var role domain.Role
	if err := r.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRoleNotFound
		}
		return nil, fmt.Errorf("loading updated role: %w", err)
	}
	return &role, nil
}

func (r *roleRepo) CountUsersByRole(ctx context.Context, roleID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("ab_user_role").
		Where("role_id = ?", roleID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting role users: %w", err)
	}
	return count, nil
}

func (r *roleRepo) IsBuiltInRole(ctx context.Context, roleID uint) (bool, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).First(&role, roleID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, domain.ErrRoleNotFound
		}
		return false, fmt.Errorf("loading role: %w", err)
	}
	return isBuiltInRoleName(role.Name), nil
}

func (r *roleRepo) Delete(ctx context.Context, roleID uint) error {
	res := r.db.WithContext(ctx).Delete(&domain.Role{}, roleID)
	if res.Error != nil {
		return fmt.Errorf("deleting role: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrRoleNotFound
	}
	return nil
}

func isBuiltInRoleName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "admin", "gamma", "public":
		return true
	default:
		return false
	}
}
