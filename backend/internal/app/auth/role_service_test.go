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
	roleExistsByID  map[uint]bool

	rolePermissionIDs        map[uint][]uint
	validPermissionViewIDs   map[uint]bool
	replacePermissionRoleID  uint
	replacePermissionViewIDs []uint
	addPermissionRoleID      uint
	addPermissionViewIDs     []uint
	removePermissionRoleID   uint
	removePermissionViewID   uint
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

func (f *fakeRoleRepo) RoleExists(_ context.Context, roleID uint) (bool, error) {
	if f.roleExistsByID == nil {
		return true, nil
	}
	exists, ok := f.roleExistsByID[roleID]
	if !ok {
		return false, nil
	}
	return exists, nil
}

func (f *fakeRoleRepo) ListPermissionViewIDsByRole(_ context.Context, roleID uint) ([]uint, error) {
	if f.rolePermissionIDs == nil {
		return []uint{}, nil
	}
	ids := f.rolePermissionIDs[roleID]
	cloned := make([]uint, len(ids))
	copy(cloned, ids)
	return cloned, nil
}

func (f *fakeRoleRepo) CountExistingPermissionViews(_ context.Context, permissionViewIDs []uint) (int64, error) {
	if f.validPermissionViewIDs == nil {
		return int64(len(permissionViewIDs)), nil
	}
	count := int64(0)
	for _, id := range permissionViewIDs {
		if f.validPermissionViewIDs[id] {
			count++
		}
	}
	return count, nil
}

func (f *fakeRoleRepo) ReplacePermissionViews(_ context.Context, roleID uint, permissionViewIDs []uint) error {
	f.replacePermissionRoleID = roleID
	f.replacePermissionViewIDs = append([]uint{}, permissionViewIDs...)
	if f.rolePermissionIDs == nil {
		f.rolePermissionIDs = map[uint][]uint{}
	}
	f.rolePermissionIDs[roleID] = append([]uint{}, permissionViewIDs...)
	return nil
}

func (f *fakeRoleRepo) AddPermissionViews(_ context.Context, roleID uint, permissionViewIDs []uint) error {
	f.addPermissionRoleID = roleID
	f.addPermissionViewIDs = append([]uint{}, permissionViewIDs...)
	if f.rolePermissionIDs == nil {
		f.rolePermissionIDs = map[uint][]uint{}
	}
	existing := f.rolePermissionIDs[roleID]
	for _, candidate := range permissionViewIDs {
		seen := false
		for _, current := range existing {
			if current == candidate {
				seen = true
				break
			}
		}
		if !seen {
			existing = append(existing, candidate)
		}
	}
	f.rolePermissionIDs[roleID] = existing
	return nil
}

func (f *fakeRoleRepo) RemovePermissionView(_ context.Context, roleID uint, permissionViewID uint) error {
	f.removePermissionRoleID = roleID
	f.removePermissionViewID = permissionViewID
	if f.rolePermissionIDs == nil {
		return nil
	}
	existing := f.rolePermissionIDs[roleID]
	filtered := make([]uint, 0, len(existing))
	for _, id := range existing {
		if id != permissionViewID {
			filtered = append(filtered, id)
		}
	}
	f.rolePermissionIDs[roleID] = filtered
	return nil
}

type fakeRoleCacheRepo struct {
	bustCount int
}

func (f *fakeRoleCacheRepo) BustRBAC(_ context.Context) error {
	f.bustCount++
	return nil
}

func (f *fakeRoleCacheRepo) BustRBACForUser(_ context.Context, _ uint) error {
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

func TestRoleService_SetRolePermissions_InvalidPermissionViewID(t *testing.T) {
	repo := &fakeRoleRepo{
		isAdmin:                true,
		roleExistsByID:         map[uint]bool{9: true},
		validPermissionViewIDs: map[uint]bool{1: true, 2: true},
	}
	svc := svcauth.NewRoleService(repo, &fakeRoleCacheRepo{})

	_, err := svc.SetRolePermissions(context.Background(), 1, 9, []uint{1, 999})
	if !errors.Is(err, domain.ErrInvalidPermissionViewID) {
		t.Fatalf("expected ErrInvalidPermissionViewID, got %v", err)
	}
}

func TestRoleService_SetRolePermissions_BustsCache(t *testing.T) {
	repo := &fakeRoleRepo{
		isAdmin:                true,
		roleExistsByID:         map[uint]bool{9: true},
		validPermissionViewIDs: map[uint]bool{1: true, 2: true, 3: true},
	}
	cache := &fakeRoleCacheRepo{}
	svc := svcauth.NewRoleService(repo, cache)

	assigned, err := svc.SetRolePermissions(context.Background(), 1, 9, []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(assigned) != 3 {
		t.Fatalf("expected 3 assigned permission views, got %v", assigned)
	}
	if cache.bustCount != 1 {
		t.Fatalf("expected cache bust once, got %d", cache.bustCount)
	}
}

func TestRoleService_AddAndRemoveRolePermissions(t *testing.T) {
	repo := &fakeRoleRepo{
		isAdmin:                true,
		roleExistsByID:         map[uint]bool{5: true},
		rolePermissionIDs:      map[uint][]uint{5: []uint{1, 2}},
		validPermissionViewIDs: map[uint]bool{1: true, 2: true, 3: true},
	}
	cache := &fakeRoleCacheRepo{}
	svc := svcauth.NewRoleService(repo, cache)

	assignedAfterAdd, err := svc.AddRolePermissions(context.Background(), 1, 5, []uint{3})
	if err != nil {
		t.Fatalf("expected nil error on add, got %v", err)
	}
	if len(assignedAfterAdd) != 3 {
		t.Fatalf("expected 3 permission views after add, got %v", assignedAfterAdd)
	}

	assignedAfterRemove, err := svc.RemoveRolePermission(context.Background(), 1, 5, 2)
	if err != nil {
		t.Fatalf("expected nil error on remove, got %v", err)
	}
	if len(assignedAfterRemove) != 2 {
		t.Fatalf("expected 2 permission views after remove, got %v", assignedAfterRemove)
	}
	if cache.bustCount != 2 {
		t.Fatalf("expected cache bust count 2, got %d", cache.bustCount)
	}
}
