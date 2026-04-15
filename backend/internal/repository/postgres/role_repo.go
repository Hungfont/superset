package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/auth"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type roleRepo struct {
	db *gorm.DB
}

type permissionViewRoleRow struct {
	RoleID           uint `gorm:"column:role_id"`
	PermissionViewID uint `gorm:"column:permission_view_id"`
}

type userRoleRow struct {
	UserID uint `gorm:"column:user_id"`
	RoleID uint `gorm:"column:role_id"`
}

func (permissionViewRoleRow) TableName() string { return "ab_permission_view_role" }
func (userRoleRow) TableName() string           { return "ab_user_role" }

func NewRoleRepository(db *gorm.DB) domain.RoleRepository {
	return &roleRepo{db: db}
}

func NewUserRoleRepository(db *gorm.DB) domain.UserRoleRepository {
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

func (r *roleRepo) RoleExists(ctx context.Context, roleID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("ab_role").
		Where("id = ?", roleID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking role exists: %w", err)
	}
	return count > 0, nil
}

func (r *roleRepo) ListPermissionViewIDsByRole(ctx context.Context, roleID uint) ([]uint, error) {
	ids := make([]uint, 0)
	err := r.db.WithContext(ctx).
		Table("ab_permission_view_role").
		Where("role_id = ?", roleID).
		Order("permission_view_id ASC").
		Pluck("permission_view_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("listing role permission views: %w", err)
	}
	return ids, nil
}

func (r *roleRepo) CountExistingPermissionViews(ctx context.Context, permissionViewIDs []uint) (int64, error) {
	uniqueIDs := uniqueUintIDs(permissionViewIDs)
	if len(uniqueIDs) == 0 {
		return 0, nil
	}

	var count int64
	err := r.db.WithContext(ctx).
		Table("ab_permission_view").
		Where("id IN ?", uniqueIDs).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting permission views: %w", err)
	}
	return count, nil
}

func (r *roleRepo) ReplacePermissionViews(ctx context.Context, roleID uint, permissionViewIDs []uint) error {
	rows := buildPermissionViewRoleRows(roleID, permissionViewIDs)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ab_permission_view_role").Where("role_id = ?", roleID).Delete(&permissionViewRoleRow{}).Error; err != nil {
			return fmt.Errorf("deleting existing role permission views: %w", err)
		}

		if len(rows) == 0 {
			return nil
		}

		if err := tx.Table("ab_permission_view_role").CreateInBatches(rows, 100).Error; err != nil {
			return fmt.Errorf("creating role permission views: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("replacing role permission views: %w", err)
	}

	return nil
}

func (r *roleRepo) AddPermissionViews(ctx context.Context, roleID uint, permissionViewIDs []uint) error {
	rows := buildPermissionViewRoleRows(roleID, permissionViewIDs)
	if len(rows) == 0 {
		return nil
	}

	err := r.db.WithContext(ctx).
		Table("ab_permission_view_role").
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(rows, 100).Error
	if err != nil {
		return fmt.Errorf("adding role permission views: %w", err)
	}

	return nil
}

func (r *roleRepo) RemovePermissionView(ctx context.Context, roleID uint, permissionViewID uint) error {
	err := r.db.WithContext(ctx).
		Table("ab_permission_view_role").
		Where("role_id = ? AND permission_view_id = ?", roleID, permissionViewID).
		Delete(&permissionViewRoleRow{}).Error
	if err != nil {
		return fmt.Errorf("removing role permission view: %w", err)
	}
	return nil
}

func (r *roleRepo) ListRoleIDsByUser(ctx context.Context, userID uint) ([]uint, error) {
	roleIDs := make([]uint, 0)
	err := r.db.WithContext(ctx).
		Table("ab_user_role").
		Where("user_id = ?", userID).
		Order("role_id ASC").
		Pluck("role_id", &roleIDs).Error
	if err != nil {
		return nil, fmt.Errorf("listing user roles: %w", err)
	}
	return roleIDs, nil
}

func (r *roleRepo) CountExistingRoles(ctx context.Context, roleIDs []uint) (int64, error) {
	uniqueIDs := uniqueUintIDs(roleIDs)
	if len(uniqueIDs) == 0 {
		return 0, nil
	}

	var count int64
	err := r.db.WithContext(ctx).
		Table("ab_role").
		Where("id IN ?", uniqueIDs).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting roles: %w", err)
	}

	return count, nil
}

func (r *roleRepo) ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	uniqueIDs := uniqueUintIDs(roleIDs)
	rows := make([]userRoleRow, 0, len(uniqueIDs))
	for _, roleID := range uniqueIDs {
		rows = append(rows, userRoleRow{UserID: userID, RoleID: roleID})
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ab_user_role").Where("user_id = ?", userID).Delete(&userRoleRow{}).Error; err != nil {
			return fmt.Errorf("deleting existing user roles: %w", err)
		}

		if err := tx.Table("ab_user_role").CreateInBatches(rows, 100).Error; err != nil {
			return fmt.Errorf("creating user roles: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("replacing user roles: %w", err)
	}

	return nil
}

func buildPermissionViewRoleRows(roleID uint, permissionViewIDs []uint) []permissionViewRoleRow {
	uniqueIDs := uniqueUintIDs(permissionViewIDs)
	rows := make([]permissionViewRoleRow, 0, len(uniqueIDs))
	for _, permissionViewID := range uniqueIDs {
		rows = append(rows, permissionViewRoleRow{RoleID: roleID, PermissionViewID: permissionViewID})
	}
	return rows
}

func uniqueUintIDs(values []uint) []uint {
	if len(values) == 0 {
		return []uint{}
	}

	seen := make(map[uint]struct{}, len(values))
	unique := make([]uint, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func isBuiltInRoleName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "admin", "gamma", "public":
		return true
	default:
		return false
	}
}
