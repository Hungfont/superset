package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/redis/go-redis/v9"
)

type roleCacheRepo struct {
	client *redis.Client
}

const rbacPermissionTTL = 5 * time.Minute

func NewRoleCacheRepository(client *redis.Client) domain.RoleCacheRepository {
	return &roleCacheRepo{client: client}
}

func NewRBACPermissionCacheRepository(client *redis.Client) domain.RBACPermissionCacheRepository {
	return &roleCacheRepo{client: client}
}

func (r *roleCacheRepo) BustRBAC(ctx context.Context) error {
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, "rbac:*", 200).Result()
		if err != nil {
			return fmt.Errorf("scanning rbac cache keys: %w", err)
		}
		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("deleting rbac cache keys: %w", err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (r *roleCacheRepo) BustRBACForUser(ctx context.Context, userID uint) error {
	key := fmt.Sprintf("rbac:%d", userID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("deleting user rbac cache key: %w", err)
	}
	return nil
}

func (r *roleCacheRepo) GetPermissionSet(ctx context.Context, userID uint) ([]string, error) {
	key := fmt.Sprintf("rbac:%d", userID)
	raw, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting user permission set: %w", err)
	}

	values := make([]string, 0)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("unmarshalling user permission set: %w", err)
	}

	return values, nil
}

func (r *roleCacheRepo) SetPermissionSet(ctx context.Context, userID uint, values []string) error {
	key := fmt.Sprintf("rbac:%d", userID)
	raw, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshalling user permission set: %w", err)
	}

	if err := r.client.Set(ctx, key, raw, rbacPermissionTTL).Err(); err != nil {
		return fmt.Errorf("setting user permission set: %w", err)
	}

	return nil
}

var _ domain.RBACPermissionCacheRepository = (*roleCacheRepo)(nil)
