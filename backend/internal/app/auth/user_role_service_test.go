package auth_test

import (
	"context"
	"errors"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
)

type fakeUserRoleRepo struct {
	isAdmin bool

	roleIDsByUser map[uint][]uint
	validRoleIDs  map[uint]bool

	replaceUserID  uint
	replaceRoleIDs []uint
}

func (f *fakeUserRoleRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	return f.isAdmin, nil
}

func (f *fakeUserRoleRepo) ListRoleIDsByUser(_ context.Context, userID uint) ([]uint, error) {
	if f.roleIDsByUser == nil {
		return []uint{}, nil
	}
	ids := f.roleIDsByUser[userID]
	cloned := make([]uint, len(ids))
	copy(cloned, ids)
	return cloned, nil
}

func (f *fakeUserRoleRepo) CountExistingRoles(_ context.Context, roleIDs []uint) (int64, error) {
	if f.validRoleIDs == nil {
		return int64(len(roleIDs)), nil
	}
	count := int64(0)
	for _, roleID := range roleIDs {
		if f.validRoleIDs[roleID] {
			count++
		}
	}
	return count, nil
}

func (f *fakeUserRoleRepo) ReplaceUserRoles(_ context.Context, userID uint, roleIDs []uint) error {
	f.replaceUserID = userID
	f.replaceRoleIDs = append([]uint{}, roleIDs...)
	if f.roleIDsByUser == nil {
		f.roleIDsByUser = map[uint][]uint{}
	}
	f.roleIDsByUser[userID] = append([]uint{}, roleIDs...)
	return nil
}

type fakeUserRoleCacheRepo struct {
	bustedUserID uint
	bustCount    int
}

func (f *fakeUserRoleCacheRepo) BustRBAC(_ context.Context) error {
	return nil
}

func (f *fakeUserRoleCacheRepo) BustRBACForUser(_ context.Context, userID uint) error {
	f.bustedUserID = userID
	f.bustCount++
	return nil
}

func TestUserRoleService_SetUserRoles_ReplacesAssignmentsAndBustsCache(t *testing.T) {
	repo := &fakeUserRoleRepo{
		isAdmin: true,
		validRoleIDs: map[uint]bool{
			1: true,
			3: true,
		},
	}
	cache := &fakeUserRoleCacheRepo{}
	svc := svcauth.NewUserRoleService(repo, cache)

	updatedRoleIDs, err := svc.SetUserRoles(context.Background(), 10, 22, []uint{1, 3})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(updatedRoleIDs) != 2 || updatedRoleIDs[0] != 1 || updatedRoleIDs[1] != 3 {
		t.Fatalf("expected role ids [1 3], got %v", updatedRoleIDs)
	}
	if repo.replaceUserID != 22 {
		t.Fatalf("expected replace user id 22, got %d", repo.replaceUserID)
	}
	if cache.bustedUserID != 22 || cache.bustCount != 1 {
		t.Fatalf("expected cache bust for user 22 once, got user=%d count=%d", cache.bustedUserID, cache.bustCount)
	}
}

func TestUserRoleService_SetUserRoles_EmptyRolesReturns422DomainError(t *testing.T) {
	repo := &fakeUserRoleRepo{isAdmin: true}
	svc := svcauth.NewUserRoleService(repo, &fakeUserRoleCacheRepo{})

	_, err := svc.SetUserRoles(context.Background(), 10, 22, []uint{})
	if !errors.Is(err, domain.ErrUserMustHaveRole) {
		t.Fatalf("expected ErrUserMustHaveRole, got %v", err)
	}
}

func TestUserRoleService_SetUserRoles_InvalidRoleIDReturns422DomainError(t *testing.T) {
	repo := &fakeUserRoleRepo{
		isAdmin: true,
		validRoleIDs: map[uint]bool{
			1: true,
		},
	}
	svc := svcauth.NewUserRoleService(repo, &fakeUserRoleCacheRepo{})

	_, err := svc.SetUserRoles(context.Background(), 10, 22, []uint{1, 999})
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestUserRoleService_SetUserRoles_NonAdminReturnsForbidden(t *testing.T) {
	repo := &fakeUserRoleRepo{isAdmin: false}
	svc := svcauth.NewUserRoleService(repo, &fakeUserRoleCacheRepo{})

	_, err := svc.SetUserRoles(context.Background(), 10, 22, []uint{1})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
