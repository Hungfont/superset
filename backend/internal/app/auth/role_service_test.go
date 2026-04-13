package auth_test

import (
	"context"
	"errors"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
)

type fakeRoleRepo struct {
	isAdmin         bool
	roles           []domain.RoleListItem
	createRole      *domain.Role
	updateRole      *domain.Role
	userCountByRole map[uint]int64
	builtInByRole   map[uint]bool
	deleteRoleID    uint
}

func (f *fakeRoleRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	return f.isAdmin, nil
}

func (f *fakeRoleRepo) ListWithCounts(_ context.Context) ([]domain.RoleListItem, error) {
	return f.roles, nil
}

func (f *fakeRoleRepo) Create(_ context.Context, role *domain.Role) error {
	f.createRole = role
	role.ID = 11
	return nil
}

func (f *fakeRoleRepo) UpdateName(_ context.Context, roleID uint, name string) (*domain.Role, error) {
	f.updateRole = &domain.Role{ID: roleID, Name: name}
	return f.updateRole, nil
}

func (f *fakeRoleRepo) CountUsersByRole(_ context.Context, roleID uint) (int64, error) {
	if f.userCountByRole == nil {
		return 0, nil
	}
	return f.userCountByRole[roleID], nil
}

func (f *fakeRoleRepo) IsBuiltInRole(_ context.Context, roleID uint) (bool, error) {
	if f.builtInByRole == nil {
		return false, nil
	}
	return f.builtInByRole[roleID], nil
}

func (f *fakeRoleRepo) Delete(_ context.Context, roleID uint) error {
	f.deleteRoleID = roleID
	return nil
}

type fakeRoleCacheRepo struct {
	bustCount int
}

func (f *fakeRoleCacheRepo) BustRBAC(_ context.Context) error {
	f.bustCount++
	return nil
}

func TestRoleService_ListRoles_NonAdminForbidden(t *testing.T) {
	repo := &fakeRoleRepo{isAdmin: false}
	svc := svcauth.NewRoleService(repo, &fakeRoleCacheRepo{})

	_, err := svc.ListRoles(context.Background(), 1)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRoleService_ListRoles_ReturnsCounts(t *testing.T) {
	repo := &fakeRoleRepo{
		isAdmin: true,
		roles: []domain.RoleListItem{{
			ID:              2,
			Name:            "Editor",
			UserCount:       3,
			PermissionCount: 7,
			BuiltIn:         false,
		}},
	}
	svc := svcauth.NewRoleService(repo, &fakeRoleCacheRepo{})

	roles, err := svc.ListRoles(context.Background(), 9)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(roles) != 1 || roles[0].PermissionCount != 7 {
		t.Fatalf("expected one role with permission_count=7, got %+v", roles)
	}
}

func TestRoleService_CreateRole_BustsCache(t *testing.T) {
	repo := &fakeRoleRepo{isAdmin: true}
	cache := &fakeRoleCacheRepo{}
	svc := svcauth.NewRoleService(repo, cache)

	created, err := svc.CreateRole(context.Background(), 1, domain.UpsertRoleRequest{Name: "Analyst"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if created.Name != "Analyst" {
		t.Fatalf("expected role name Analyst, got %s", created.Name)
	}
	if cache.bustCount != 1 {
		t.Fatalf("expected cache bust once, got %d", cache.bustCount)
	}
}

func TestRoleService_DeleteRole_WithUsersConflict(t *testing.T) {
	repo := &fakeRoleRepo{
		isAdmin:         true,
		userCountByRole: map[uint]int64{5: 2},
		builtInByRole:   map[uint]bool{5: false},
	}
	svc := svcauth.NewRoleService(repo, &fakeRoleCacheRepo{})

	err := svc.DeleteRole(context.Background(), 1, 5)
	if !errors.Is(err, domain.ErrRoleHasUsers) {
		t.Fatalf("expected ErrRoleHasUsers, got %v", err)
	}
}

func TestRoleService_DeleteRole_BuiltInForbidden(t *testing.T) {
	repo := &fakeRoleRepo{
		isAdmin:         true,
		userCountByRole: map[uint]int64{3: 0},
		builtInByRole:   map[uint]bool{3: true},
	}
	svc := svcauth.NewRoleService(repo, &fakeRoleCacheRepo{})

	err := svc.DeleteRole(context.Background(), 1, 3)
	if !errors.Is(err, domain.ErrBuiltInRole) {
		t.Fatalf("expected ErrBuiltInRole, got %v", err)
	}
}

func TestRoleService_UpdateRole_BuiltInForbidden(t *testing.T) {
	repo := &fakeRoleRepo{
		isAdmin:       true,
		builtInByRole: map[uint]bool{1: true},
	}
	svc := svcauth.NewRoleService(repo, &fakeRoleCacheRepo{})

	_, err := svc.UpdateRole(context.Background(), 1, 1, domain.UpsertRoleRequest{Name: "Renamed"})
	if !errors.Is(err, domain.ErrBuiltInRole) {
		t.Fatalf("expected ErrBuiltInRole, got %v", err)
	}
}
