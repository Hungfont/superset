package redis

import (
	"context"
	"fmt"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/redis/go-redis/v9"
)

type roleCacheRepo struct {
	client *redis.Client
}

func NewRoleCacheRepository(client *redis.Client) domain.RoleCacheRepository {
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
