package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "superset/auth-service/internal/domain/db"

	"github.com/redis/go-redis/v9"
)

type databaseSchemaCacheRepo struct {
	client *redis.Client
}

func NewDatabaseSchemaCacheRepository(client *redis.Client) domain.SchemaCacheRepository {
	return &databaseSchemaCacheRepo{client: client}
}

func (r *databaseSchemaCacheRepo) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("getting schema cache value: %w", err)
	}
	return value, true, nil
}

func (r *databaseSchemaCacheRepo) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("setting schema cache value: %w", err)
	}
	return nil
}
