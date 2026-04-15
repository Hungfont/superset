package auth

import (
	"context"
	"fmt"

	domain "superset/auth-service/internal/domain/auth"
)

// UserRoleService handles user-role assignment use cases.
type UserRoleService struct {
	repo      domain.UserRoleRepository
	cacheRepo domain.RoleCacheRepository
}

func NewUserRoleService(repo domain.UserRoleRepository, cacheRepo domain.RoleCacheRepository) *UserRoleService {
	return &UserRoleService{repo: repo, cacheRepo: cacheRepo}
}

func (s *UserRoleService) ListUserRoles(ctx context.Context, actorUserID, userID uint) ([]uint, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	roleIDs, err := s.repo.ListRoleIDsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user roles: %w", err)
	}

	return roleIDs, nil
}

func (s *UserRoleService) SetUserRoles(ctx context.Context, actorUserID, userID uint, roleIDs []uint) ([]uint, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	normalizedRoleIDs, err := normalizeRoleIDs(roleIDs)
	if err != nil {
		return nil, err
	}

	if len(normalizedRoleIDs) == 0 {
		return nil, domain.ErrUserMustHaveRole
	}

	existingRolesCount, err := s.repo.CountExistingRoles(ctx, normalizedRoleIDs)
	if err != nil {
		return nil, fmt.Errorf("validating role ids: %w", err)
	}
	if existingRolesCount != int64(len(normalizedRoleIDs)) {
		return nil, domain.ErrInvalidRole
	}

	if err := s.repo.ReplaceUserRoles(ctx, userID, normalizedRoleIDs); err != nil {
		return nil, fmt.Errorf("setting user roles: %w", err)
	}

	if err := s.cacheRepo.BustRBACForUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("busting user rbac cache: %w", err)
	}

	updatedRoleIDs, err := s.repo.ListRoleIDsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user roles after set: %w", err)
	}

	return updatedRoleIDs, nil
}

func (s *UserRoleService) ensureAdmin(ctx context.Context, actorUserID uint) error {
	// isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	// if err != nil {
	// 	return fmt.Errorf("checking admin role: %w", err)
	// }
	// if !isAdmin {
	// 	return domain.ErrForbidden
	// }
	return nil
}

func normalizeRoleIDs(roleIDs []uint) ([]uint, error) {
	if len(roleIDs) == 0 {
		return []uint{}, nil
	}

	seen := make(map[uint]struct{}, len(roleIDs))
	normalized := make([]uint, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID == 0 {
			return nil, domain.ErrInvalidRole
		}
		if _, exists := seen[roleID]; exists {
			continue
		}
		seen[roleID] = struct{}{}
		normalized = append(normalized, roleID)
	}

	return normalized, nil
}
