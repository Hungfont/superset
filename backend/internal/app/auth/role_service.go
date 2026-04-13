package auth

import (
	"context"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/auth"
)

// RoleService handles role management use cases.
type RoleService struct {
	repo      domain.RoleRepository
	cacheRepo domain.RoleCacheRepository
}

func NewRoleService(repo domain.RoleRepository, cacheRepo domain.RoleCacheRepository) *RoleService {
	return &RoleService{repo: repo, cacheRepo: cacheRepo}
}

func (s *RoleService) ListRoles(ctx context.Context, actorUserID uint) ([]domain.RoleListItem, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}
	roles, err := s.repo.ListWithCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}
	return roles, nil
}

func (s *RoleService) CreateRole(ctx context.Context, actorUserID uint, req domain.UpsertRoleRequest) (*domain.Role, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidRole
	}

	role := &domain.Role{Name: name}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("creating role: %w", err)
	}
	if err := s.cacheRepo.BustRBAC(ctx); err != nil {
		return nil, fmt.Errorf("busting rbac cache: %w", err)
	}
	return role, nil
}

func (s *RoleService) UpdateRole(ctx context.Context, actorUserID, roleID uint, req domain.UpsertRoleRequest) (*domain.Role, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	builtIn, err := s.repo.IsBuiltInRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("checking built-in role: %w", err)
	}
	if builtIn {
		return nil, domain.ErrBuiltInRole
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidRole
	}

	updatedRole, err := s.repo.UpdateName(ctx, roleID, name)
	if err != nil {
		return nil, fmt.Errorf("updating role: %w", err)
	}
	if err := s.cacheRepo.BustRBAC(ctx); err != nil {
		return nil, fmt.Errorf("busting rbac cache: %w", err)
	}
	return updatedRole, nil
}

func (s *RoleService) DeleteRole(ctx context.Context, actorUserID, roleID uint) error {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return err
	}

	builtIn, err := s.repo.IsBuiltInRole(ctx, roleID)
	if err != nil {
		return fmt.Errorf("checking built-in role: %w", err)
	}
	if builtIn {
		return domain.ErrBuiltInRole
	}

	userCount, err := s.repo.CountUsersByRole(ctx, roleID)
	if err != nil {
		return fmt.Errorf("counting role users: %w", err)
	}
	if userCount > 0 {
		return domain.ErrRoleHasUsers
	}

	if err := s.repo.Delete(ctx, roleID); err != nil {
		return fmt.Errorf("deleting role: %w", err)
	}
	if err := s.cacheRepo.BustRBAC(ctx); err != nil {
		return fmt.Errorf("busting rbac cache: %w", err)
	}
	return nil
}

func (s *RoleService) ensureAdmin(ctx context.Context, actorUserID uint) error {
	isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	if err != nil {
		return fmt.Errorf("checking admin role: %w", err)
	}
	if !isAdmin {
		return domain.ErrForbidden
	}
	return nil
}
