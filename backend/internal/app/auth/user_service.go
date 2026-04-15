package auth

import (
	"context"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/auth"
	"superset/auth-service/internal/pkg/validator"

	"golang.org/x/crypto/bcrypt"
)

// UserService handles admin user CRUD and role assignment integration.
type UserService struct {
	repo      domain.UserAdminRepository
	cacheRepo domain.RoleCacheRepository
}

func NewUserService(repo domain.UserAdminRepository, cacheRepo domain.RoleCacheRepository) *UserService {
	return &UserService{repo: repo, cacheRepo: cacheRepo}
}

func (s *UserService) ListUsers(ctx context.Context, actorUserID uint) ([]domain.UserListItem, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	return users, nil
}

func (s *UserService) GetUser(ctx context.Context, actorUserID, userID uint) (*domain.UserDetail, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return user, nil
}

func (s *UserService) CreateUser(ctx context.Context, actorUserID uint, req domain.CreateUserRequest) (*domain.UserDetail, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	normalizedReq, normalizedRoleIDs, err := s.normalizeCreateRequest(req)
	if err != nil {
		return nil, err
	}

	if err := s.validateRoleIDs(ctx, normalizedRoleIDs); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(normalizedReq.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing user password: %w", err)
	}
	normalizedReq.Password = string(hashedPassword)

	userID, err := s.repo.CreateUser(ctx, normalizedReq)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	if err := s.repo.ReplaceUserRoles(ctx, userID, normalizedRoleIDs); err != nil {
		return nil, fmt.Errorf("assigning user roles: %w", err)
	}

	if err := s.cacheRepo.BustRBACForUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("busting user rbac cache: %w", err)
	}

	createdUser, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting created user: %w", err)
	}
	return createdUser, nil
}

func (s *UserService) UpdateUser(ctx context.Context, actorUserID, userID uint, req domain.UpdateUserRequest) (*domain.UserDetail, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	normalizedReq, normalizedRoleIDs, err := s.normalizeUpdateRequest(req)
	if err != nil {
		return nil, err
	}

	if err := s.validateRoleIDs(ctx, normalizedRoleIDs); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateUser(ctx, userID, normalizedReq); err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	if err := s.repo.ReplaceUserRoles(ctx, userID, normalizedRoleIDs); err != nil {
		return nil, fmt.Errorf("assigning updated user roles: %w", err)
	}

	if err := s.cacheRepo.BustRBACForUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("busting user rbac cache: %w", err)
	}

	updatedUser, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting updated user: %w", err)
	}
	return updatedUser, nil
}

func (s *UserService) DeleteUser(ctx context.Context, actorUserID, userID uint) error {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return err
	}

	if err := s.repo.DeactivateUser(ctx, userID); err != nil {
		return fmt.Errorf("deactivating user: %w", err)
	}

	if err := s.cacheRepo.BustRBACForUser(ctx, userID); err != nil {
		return fmt.Errorf("busting user rbac cache: %w", err)
	}

	return nil
}

func (s *UserService) ensureAdmin(ctx context.Context, actorUserID uint) error {
	// isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	// if err != nil {
	// 	return fmt.Errorf("checking admin role: %w", err)
	// }
	// if !isAdmin {
	// 	return domain.ErrForbidden
	// }
	return nil
}

func (s *UserService) validateRoleIDs(ctx context.Context, roleIDs []uint) error {
	if len(roleIDs) == 0 {
		return domain.ErrUserMustHaveRole
	}

	count, err := s.repo.CountExistingRoles(ctx, roleIDs)
	if err != nil {
		return fmt.Errorf("validating role ids: %w", err)
	}
	if count != int64(len(roleIDs)) {
		return domain.ErrInvalidRole
	}

	return nil
}

func (s *UserService) normalizeCreateRequest(req domain.CreateUserRequest) (domain.CreateUserRequest, []uint, error) {
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)
	password := req.Password

	if firstName == "" || lastName == "" || username == "" || email == "" || password == "" {
		return domain.CreateUserRequest{}, nil, domain.ErrInvalidUser
	}

	if err := validator.ValidatePasswordComplexity(password); err != nil {
		return domain.CreateUserRequest{}, nil, fmt.Errorf("%w: %s", domain.ErrInvalidUser, err.Error())
	}

	normalizedRoleIDs, err := normalizeRoleIDs(req.RoleIDs)
	if err != nil {
		return domain.CreateUserRequest{}, nil, err
	}

	normalized := domain.CreateUserRequest{
		FirstName: firstName,
		LastName:  lastName,
		Username:  username,
		Email:     email,
		Password:  password,
		Active:    req.Active,
		RoleIDs:   normalizedRoleIDs,
	}

	return normalized, normalizedRoleIDs, nil
}

func (s *UserService) normalizeUpdateRequest(req domain.UpdateUserRequest) (domain.UpdateUserRequest, []uint, error) {
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)

	if firstName == "" || lastName == "" || username == "" || email == "" {
		return domain.UpdateUserRequest{}, nil, domain.ErrInvalidUser
	}

	normalizedRoleIDs, err := normalizeRoleIDs(req.RoleIDs)
	if err != nil {
		return domain.UpdateUserRequest{}, nil, err
	}

	normalized := domain.UpdateUserRequest{
		FirstName: firstName,
		LastName:  lastName,
		Username:  username,
		Email:     email,
		Active:    req.Active,
		RoleIDs:   normalizedRoleIDs,
	}

	return normalized, normalizedRoleIDs, nil
}
