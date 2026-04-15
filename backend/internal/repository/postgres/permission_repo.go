package postgres

import (
	"context"
	"fmt"

	domain "superset/auth-service/internal/domain/auth"

	"gorm.io/gorm"
)

type permissionRepo struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) domain.PermissionRepository {
	return &permissionRepo{db: db}
}

func (r *permissionRepo) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	var permissions []domain.Permission
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("listing permissions: %w", err)
	}
	return permissions, nil
}

func (r *permissionRepo) CreatePermission(ctx context.Context, permission *domain.Permission) error {
	if err := r.db.WithContext(ctx).Create(permission).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrPermissionDuplicate
		}
		return fmt.Errorf("creating permission: %w", err)
	}
	return nil
}

func (r *permissionRepo) ListViewMenus(ctx context.Context) ([]domain.ViewMenu, error) {
	var viewMenus []domain.ViewMenu
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&viewMenus).Error; err != nil {
		return nil, fmt.Errorf("listing view menus: %w", err)
	}
	return viewMenus, nil
}

func (r *permissionRepo) CreateViewMenu(ctx context.Context, viewMenu *domain.ViewMenu) error {
	if err := r.db.WithContext(ctx).Create(viewMenu).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrViewMenuDuplicate
		}
		return fmt.Errorf("creating view menu: %w", err)
	}
	return nil
}

func (r *permissionRepo) ListPermissionViews(ctx context.Context) ([]domain.PermissionView, error) {
	var permissionViews []domain.PermissionView
	if err := r.db.WithContext(ctx).
		Table("ab_permission_view pv").
		Select("pv.id, pv.permission_id, pv.view_menu_id, p.name AS permission_name, vm.name AS view_menu_name").
		Joins("JOIN ab_permission p ON p.id = pv.permission_id").
		Joins("JOIN ab_view_menu vm ON vm.id = pv.view_menu_id").
		Order("pv.id ASC").
		Scan(&permissionViews).Error; err != nil {
		return nil, fmt.Errorf("listing permission views: %w", err)
	}

	return permissionViews, nil
}

func (r *permissionRepo) CreatePermissionView(ctx context.Context, permissionView *domain.PermissionView) error {
	if err := r.db.WithContext(ctx).Create(permissionView).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrPermissionViewDuplicate
		}
		return fmt.Errorf("creating permission view: %w", err)
	}
	return nil
}

func (r *permissionRepo) CountRoleAssignmentsByPermissionView(ctx context.Context, permissionViewID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("ab_permission_view_role").
		Where("permission_view_id = ?", permissionViewID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting permission view role assignments: %w", err)
	}
	return count, nil
}

func (r *permissionRepo) DeletePermissionView(ctx context.Context, permissionViewID uint) error {
	result := r.db.WithContext(ctx).Delete(&domain.PermissionView{}, permissionViewID)
	if result.Error != nil {
		return fmt.Errorf("deleting permission view: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrPermissionViewNotFound
	}
	return nil
}

func (r *permissionRepo) SeedPermissionViews(ctx context.Context, seeds []domain.PermissionViewSeed) error {
	for _, seed := range seeds {
		seedItem := seed
		if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			permission := domain.Permission{Name: seedItem.PermissionName}
			if err := tx.Where("name = ?", seedItem.PermissionName).FirstOrCreate(&permission).Error; err != nil {
				return fmt.Errorf("first-or-create permission %s: %w", seedItem.PermissionName, err)
			}

			viewMenu := domain.ViewMenu{Name: seedItem.ViewMenuName}
			if err := tx.Where("name = ?", seedItem.ViewMenuName).FirstOrCreate(&viewMenu).Error; err != nil {
				return fmt.Errorf("first-or-create view menu %s: %w", seedItem.ViewMenuName, err)
			}

			permissionView := domain.PermissionView{PermissionID: permission.ID, ViewMenuID: viewMenu.ID}
			if err := tx.Where("permission_id = ? AND view_menu_id = ?", permission.ID, viewMenu.ID).FirstOrCreate(&permissionView).Error; err != nil {
				if isUniqueViolation(err) {
					return nil
				}
				return fmt.Errorf("first-or-create permission view (%d,%d): %w", permission.ID, viewMenu.ID, err)
			}

			return nil
		}); err != nil {
			return fmt.Errorf("seeding permission view %s:%s: %w", seedItem.PermissionName, seedItem.ViewMenuName, err)
		}
	}

	return nil
}

var _ domain.PermissionRepository = (*permissionRepo)(nil)
