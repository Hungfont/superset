package auth

import (
	"context"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/auth"
)

var defaultPermissionViewSeeds = []domain.PermissionViewSeed{
	{PermissionName: "can_read", ViewMenuName: "Dashboard"},
	{PermissionName: "can_read", ViewMenuName: "Chart"},
	{PermissionName: "can_write", ViewMenuName: "Dashboard"},
	{PermissionName: "can_write", ViewMenuName: "Chart"},
}

// PermissionService handles AUTH-008 permission and view-menu management use cases.
type PermissionService struct {
	repo      domain.PermissionRepository
	cacheRepo domain.RoleCacheRepository
}

func NewPermissionService(repo domain.PermissionRepository, cacheRepo domain.RoleCacheRepository) *PermissionService {
	return &PermissionService{repo: repo, cacheRepo: cacheRepo}
}

func (s *PermissionService) SeedDefaults(ctx context.Context) error {
	if err := s.repo.SeedPermissionViews(ctx, defaultPermissionViewSeeds); err != nil {
		return fmt.Errorf("seeding permission views: %w", err)
	}
	return nil
}

func (s *PermissionService) ListPermissions(ctx context.Context, actorUserID uint) ([]domain.Permission, error) {
	permissions, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing permissions: %w", err)
	}
	return permissions, nil
}

func (s *PermissionService) CreatePermission(ctx context.Context, actorUserID uint, req domain.UpsertPermissionRequest) (*domain.Permission, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidPermission
	}

	permission := &domain.Permission{Name: name}
	if err := s.repo.CreatePermission(ctx, permission); err != nil {
		return nil, fmt.Errorf("creating permission: %w", err)
	}
	if err := s.cacheRepo.BustRBAC(ctx); err != nil {
		return nil, fmt.Errorf("busting rbac cache: %w", err)
	}

	return permission, nil
}

func (s *PermissionService) ListViewMenus(ctx context.Context, actorUserID uint) ([]domain.ViewMenu, error) {
	viewMenus, err := s.repo.ListViewMenus(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing view menus: %w", err)
	}
	return viewMenus, nil
}

func (s *PermissionService) CreateViewMenu(ctx context.Context, actorUserID uint, req domain.UpsertViewMenuRequest) (*domain.ViewMenu, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidViewMenu
	}

	viewMenu := &domain.ViewMenu{Name: name}
	if err := s.repo.CreateViewMenu(ctx, viewMenu); err != nil {
		return nil, fmt.Errorf("creating view menu: %w", err)
	}
	if err := s.cacheRepo.BustRBAC(ctx); err != nil {
		return nil, fmt.Errorf("busting rbac cache: %w", err)
	}

	return viewMenu, nil
}

func (s *PermissionService) ListPermissionViews(ctx context.Context, actorUserID uint) ([]domain.PermissionView, error) {
	permissionViews, err := s.repo.ListPermissionViews(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing permission views: %w", err)
	}
	return permissionViews, nil
}

func (s *PermissionService) CreatePermissionView(ctx context.Context, actorUserID uint, req domain.CreatePermissionViewRequest) (*domain.PermissionView, error) {
	if req.PermissionID == 0 || req.ViewMenuID == 0 {
		return nil, domain.ErrInvalidPermission
	}

	permissionView := &domain.PermissionView{
		PermissionID: req.PermissionID,
		ViewMenuID:   req.ViewMenuID,
	}
	if err := s.repo.CreatePermissionView(ctx, permissionView); err != nil {
		return nil, fmt.Errorf("creating permission view: %w", err)
	}
	if err := s.cacheRepo.BustRBAC(ctx); err != nil {
		return nil, fmt.Errorf("busting rbac cache: %w", err)
	}

	return permissionView, nil
}

func (s *PermissionService) DeletePermissionView(ctx context.Context, actorUserID, permissionViewID uint) error {
	assignedCount, err := s.repo.CountRoleAssignmentsByPermissionView(ctx, permissionViewID)
	if err != nil {
		return fmt.Errorf("counting permission view assignments: %w", err)
	}
	if assignedCount > 0 {
		return domain.ErrPermissionViewInUse
	}

	if err := s.repo.DeletePermissionView(ctx, permissionViewID); err != nil {
		return fmt.Errorf("deleting permission view: %w", err)
	}
	if err := s.cacheRepo.BustRBAC(ctx); err != nil {
		return fmt.Errorf("busting rbac cache: %w", err)
	}

	return nil
}
