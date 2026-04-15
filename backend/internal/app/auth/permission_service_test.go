package auth_test

import (
	"context"
	"errors"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
)

type fakePermissionRepo struct {
	permissions           []domain.Permission
	viewMenus             []domain.ViewMenu
	permissionViews       []domain.PermissionView
	createPermission      *domain.Permission
	createViewMenu        *domain.ViewMenu
	createPermissionView  *domain.PermissionView
	permissionViewUseByID map[uint]int64
	deletePermissionView  uint

	createPermissionErr     error
	createViewMenuErr       error
	createPermissionViewErr error
}

func (f *fakePermissionRepo) ListPermissions(_ context.Context) ([]domain.Permission, error) {
	return f.permissions, nil
}

func (f *fakePermissionRepo) CreatePermission(_ context.Context, permission *domain.Permission) error {
	f.createPermission = permission
	if f.createPermissionErr != nil {
		return f.createPermissionErr
	}
	permission.ID = 1
	return nil
}

func (f *fakePermissionRepo) ListViewMenus(_ context.Context) ([]domain.ViewMenu, error) {
	return f.viewMenus, nil
}

func (f *fakePermissionRepo) CreateViewMenu(_ context.Context, viewMenu *domain.ViewMenu) error {
	f.createViewMenu = viewMenu
	if f.createViewMenuErr != nil {
		return f.createViewMenuErr
	}
	viewMenu.ID = 1
	return nil
}

func (f *fakePermissionRepo) ListPermissionViews(_ context.Context) ([]domain.PermissionView, error) {
	return f.permissionViews, nil
}

func (f *fakePermissionRepo) CreatePermissionView(_ context.Context, permissionView *domain.PermissionView) error {
	f.createPermissionView = permissionView
	if f.createPermissionViewErr != nil {
		return f.createPermissionViewErr
	}
	permissionView.ID = 1
	return nil
}

func (f *fakePermissionRepo) CountRoleAssignmentsByPermissionView(_ context.Context, permissionViewID uint) (int64, error) {
	if f.permissionViewUseByID == nil {
		return 0, nil
	}
	return f.permissionViewUseByID[permissionViewID], nil
}

func (f *fakePermissionRepo) DeletePermissionView(_ context.Context, permissionViewID uint) error {
	f.deletePermissionView = permissionViewID
	return nil
}

func (f *fakePermissionRepo) SeedPermissionViews(_ context.Context, _ []domain.PermissionViewSeed) error {
	return nil
}

type fakePermissionCacheRepo struct {
	bustCount int
}

func (f *fakePermissionCacheRepo) BustRBAC(_ context.Context) error {
	f.bustCount++
	return nil
}

func TestPermissionService_CreatePermissionReturnsCreatedEntity(t *testing.T) {
	repo := &fakePermissionRepo{}
	cacheRepo := &fakePermissionCacheRepo{}
	svc := svcauth.NewPermissionService(repo, cacheRepo)

	permission, err := svc.CreatePermission(context.Background(), 1, domain.UpsertPermissionRequest{Name: "can_read"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if permission.ID == 0 || permission.Name != "can_read" {
		t.Fatalf("expected created permission, got %+v", permission)
	}
	if cacheRepo.bustCount != 1 {
		t.Fatalf("expected cache bust once, got %d", cacheRepo.bustCount)
	}
}

func TestPermissionService_ListPermissionViewsReturnsNamesForUI(t *testing.T) {
	repo := &fakePermissionRepo{permissionViews: []domain.PermissionView{{
		ID:             10,
		PermissionID:   1,
		ViewMenuID:     2,
		PermissionName: "can_read",
		ViewMenuName:   "Dashboard",
	}}}
	svc := svcauth.NewPermissionService(repo, &fakePermissionCacheRepo{})

	permissionViews, err := svc.ListPermissionViews(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(permissionViews) != 1 {
		t.Fatalf("expected 1 permission view, got %d", len(permissionViews))
	}

	if permissionViews[0].PermissionName != "can_read" || permissionViews[0].ViewMenuName != "Dashboard" {
		t.Fatalf("expected names for UI, got %+v", permissionViews[0])
	}
}

func TestPermissionService_CreatePermissionViewDuplicateReturnsConflict(t *testing.T) {
	repo := &fakePermissionRepo{createPermissionViewErr: domain.ErrPermissionViewDuplicate}
	svc := svcauth.NewPermissionService(repo, &fakePermissionCacheRepo{})

	_, err := svc.CreatePermissionView(context.Background(), 1, domain.CreatePermissionViewRequest{PermissionID: 1, ViewMenuID: 2})
	if !errors.Is(err, domain.ErrPermissionViewDuplicate) {
		t.Fatalf("expected ErrPermissionViewDuplicate, got %v", err)
	}
}

func TestPermissionService_DeleteAssignedPermissionViewReturnsConflict(t *testing.T) {
	repo := &fakePermissionRepo{permissionViewUseByID: map[uint]int64{10: 2}}
	svc := svcauth.NewPermissionService(repo, &fakePermissionCacheRepo{})

	err := svc.DeletePermissionView(context.Background(), 1, 10)
	if !errors.Is(err, domain.ErrPermissionViewInUse) {
		t.Fatalf("expected ErrPermissionViewInUse, got %v", err)
	}
}
