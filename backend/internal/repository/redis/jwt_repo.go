package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/redis/go-redis/v9"
)

const userCacheTTL = 5 * time.Minute

type jwtRepo struct {
	client *redis.Client
}

// NewJWTRepository returns a JWTRepository backed by Redis.
func NewJWTRepository(client *redis.Client) domain.JWTRepository {
	return &jwtRepo{client: client}
}

func (r *jwtRepo) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	exists, err := r.client.Exists(ctx, "jwt:blacklist:"+jti).Result()
	if err != nil {
		return false, fmt.Errorf("checking jwt blacklist: %w", err)
	}
	return exists > 0, nil
}

func (r *jwtRepo) BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	if err := r.client.Set(ctx, "jwt:blacklist:"+jti, "1", ttl).Err(); err != nil {
		return fmt.Errorf("blacklisting jwt jti: %w", err)
	}
	return nil
}

func (r *jwtRepo) GetCachedUser(ctx context.Context, userID uint) (*domain.UserContext, error) {
	key := fmt.Sprintf("user:%d", userID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting cached user: %w", err)
	}
	var u domain.UserContext
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, fmt.Errorf("unmarshalling cached user: %w", err)
	}
	return &u, nil
}

func (r *jwtRepo) SetCachedUser(ctx context.Context, userID uint, u *domain.UserContext) error {
	data, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("marshalling user for cache: %w", err)
	}
	key := fmt.Sprintf("user:%d", userID)
	if err := r.client.Set(ctx, key, data, userCacheTTL).Err(); err != nil {
		return fmt.Errorf("setting cached user: %w", err)
	}
	return nil
}
