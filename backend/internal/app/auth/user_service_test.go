package auth_test

import (
	"context"
	"errors"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
)

type fakeUserAdminRepo struct {
	isAdmin bool

	nextUsers    []domain.UserListItem
	nextUser     *domain.UserDetail
	validRoleIDs map[uint]bool

	createdInput *domain.CreateUserRequest
	updatedInput *domain.UpdateUserRequest

	deactivateUserID uint
	replaceUserID    uint
	replaceRoleIDs   []uint

	createdID uint
	updatedID uint

	notFoundOnGet        bool
	notFoundOnUpdate     bool
	notFoundOnDeactivate bool
}

func (f *fakeUserAdminRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	return f.isAdmin, nil
}

func (f *fakeUserAdminRepo) ListUsers(_ context.Context) ([]domain.UserListItem, error) {
	return f.nextUsers, nil
}

func (f *fakeUserAdminRepo) GetUserByID(_ context.Context, userID uint) (*domain.UserDetail, error) {
	if f.notFoundOnGet {
		return nil, domain.ErrUserNotFound
	}
	if f.nextUser == nil {
		return &domain.UserDetail{ID: userID, Username: "demo", Email: "demo@example.com", Active: true, RoleIDs: []uint{1}}, nil
	}
	return f.nextUser, nil
}

func (f *fakeUserAdminRepo) CreateUser(_ context.Context, req domain.CreateUserRequest) (uint, error) {
	f.createdInput = &req
	if f.createdID != 0 {
		return f.createdID, nil
	}
	return 101, nil
}

func (f *fakeUserAdminRepo) UpdateUser(_ context.Context, userID uint, req domain.UpdateUserRequest) error {
	if f.notFoundOnUpdate {
		return domain.ErrUserNotFound
	}
	f.updatedID = userID
	f.updatedInput = &req
	return nil
}

func (f *fakeUserAdminRepo) DeactivateUser(_ context.Context, userID uint) error {
	if f.notFoundOnDeactivate {
		return domain.ErrUserNotFound
	}
	f.deactivateUserID = userID
	return nil
}

func (f *fakeUserAdminRepo) CountExistingRoles(_ context.Context, roleIDs []uint) (int64, error) {
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

func (f *fakeUserAdminRepo) ReplaceUserRoles(_ context.Context, userID uint, roleIDs []uint) error {
	f.replaceUserID = userID
	f.replaceRoleIDs = append([]uint{}, roleIDs...)
	return nil
}

type fakeUserAdminCacheRepo struct {
	bustForUserID uint
	bustCount     int
}

func (f *fakeUserAdminCacheRepo) BustRBAC(_ context.Context) error {
	return nil
}

func (f *fakeUserAdminCacheRepo) BustRBACForUser(_ context.Context, userID uint) error {
	f.bustForUserID = userID
	f.bustCount++
	return nil
}

func TestUserService_ListUsers_ReturnsData(t *testing.T) {
	repo := &fakeUserAdminRepo{
		isAdmin: true,
		nextUsers: []domain.UserListItem{{
			ID: 1, Username: "admin", Email: "admin@example.com", Active: true, RoleIDs: []uint{1},
		}},
	}
	svc := svcauth.NewUserService(repo, &fakeUserAdminCacheRepo{})

	users, err := svc.ListUsers(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(users) != 1 || users[0].Username != "admin" {
		t.Fatalf("unexpected users response: %+v", users)
	}
}

func TestUserService_CreateUser_CreatesAndAssignsRoles(t *testing.T) {
	repo := &fakeUserAdminRepo{
		isAdmin:      true,
		validRoleIDs: map[uint]bool{1: true, 3: true},
		nextUser: &domain.UserDetail{
			ID:        101,
			Username:  "newuser",
			Email:     "new@example.com",
			Active:    true,
			RoleIDs:   []uint{1, 3},
			FirstName: "New",
			LastName:  "User",
		},
	}
	cache := &fakeUserAdminCacheRepo{}
	svc := svcauth.NewUserService(repo, cache)

	created, err := svc.CreateUser(context.Background(), 1, domain.CreateUserRequest{
		FirstName: "New",
		LastName:  "User",
		Username:  "newuser",
		Email:     "new@example.com",
		Password:  "StrongPass@123",
		RoleIDs:   []uint{1, 3},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if created.ID != 101 {
		t.Fatalf("expected id=101, got %d", created.ID)
	}
	if repo.replaceUserID != 101 || len(repo.replaceRoleIDs) != 2 {
		t.Fatalf("expected role replacement for new user, got user=%d roles=%v", repo.replaceUserID, repo.replaceRoleIDs)
	}
	if cache.bustCount != 1 || cache.bustForUserID != 101 {
		t.Fatalf("expected cache bust for new user, got user=%d count=%d", cache.bustForUserID, cache.bustCount)
	}
}

func TestUserService_CreateUser_EmptyRolesReturnsError(t *testing.T) {
	repo := &fakeUserAdminRepo{isAdmin: true}
	svc := svcauth.NewUserService(repo, &fakeUserAdminCacheRepo{})

	_, err := svc.CreateUser(context.Background(), 1, domain.CreateUserRequest{
		FirstName: "New",
		LastName:  "User",
		Username:  "newuser",
		Email:     "new@example.com",
		Password:  "StrongPass@123",
		RoleIDs:   []uint{},
	})
	if !errors.Is(err, domain.ErrUserMustHaveRole) {
		t.Fatalf("expected ErrUserMustHaveRole, got %v", err)
	}
}

func TestUserService_UpdateUser_NonAdminForbidden(t *testing.T) {
	repo := &fakeUserAdminRepo{isAdmin: false}
	svc := svcauth.NewUserService(repo, &fakeUserAdminCacheRepo{})

	_, err := svc.UpdateUser(context.Background(), 2, 7, domain.UpdateUserRequest{RoleIDs: []uint{1}})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUserService_UpdateUser_InvalidRolesReturnsError(t *testing.T) {
	repo := &fakeUserAdminRepo{isAdmin: true, validRoleIDs: map[uint]bool{1: true}}
	svc := svcauth.NewUserService(repo, &fakeUserAdminCacheRepo{})

	_, err := svc.UpdateUser(context.Background(), 1, 7, domain.UpdateUserRequest{
		FirstName: "Updated",
		LastName:  "User",
		Username:  "updated",
		Email:     "updated@example.com",
		Active:    true,
		RoleIDs:   []uint{1, 9},
	})
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestUserService_DeactivateUser_ReturnsNotFound(t *testing.T) {
	repo := &fakeUserAdminRepo{isAdmin: true, notFoundOnDeactivate: true}
	svc := svcauth.NewUserService(repo, &fakeUserAdminCacheRepo{})

	err := svc.DeleteUser(context.Background(), 1, 77)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
