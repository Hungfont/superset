package redis

import (
	"context"
	"fmt"
	"time"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/redis/go-redis/v9"
)

const (
	loginRateTTL  = 60 * time.Second
	lockoutTTL    = 15 * time.Minute
	refreshTTL    = 7 * 24 * time.Hour
	lockoutMaxAge = 15 * time.Minute
)

// rateLimitRepo implements domain.RateLimitRepository using Redis.
type rateLimitRepo struct {
	client *redis.Client
}

// NewRateLimitRepository returns a RateLimitRepository backed by Redis.
func NewRateLimitRepository(client *redis.Client) domain.RateLimitRepository {
	return &rateLimitRepo{client: client}
}

func (r *rateLimitRepo) IncrLoginAttempt(ctx context.Context, ip string) (int64, error) {
	key := "rate:login:" + ip
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("incrementing rate limit: %w", err)
	}
	if count == 1 {
		r.client.Expire(ctx, key, loginRateTTL)
	}
	return count, nil
}

func (r *rateLimitRepo) IncrFailedLogin(ctx context.Context, username string) (int64, error) {
	key := "failed_login:" + username
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("incrementing failed login: %w", err)
	}
	if count == 1 {
		r.client.Expire(ctx, key, lockoutTTL)
	}
	return count, nil
}

func (r *rateLimitRepo) ResetFailedLogin(ctx context.Context, username string) error {
	if err := r.client.Del(ctx, "failed_login:"+username).Err(); err != nil {
		return fmt.Errorf("resetting failed login counter: %w", err)
	}
	return nil
}

func (r *rateLimitRepo) GetFailedLoginCount(ctx context.Context, username string) (int64, error) {
	count, err := r.client.Get(ctx, "failed_login:"+username).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("getting failed login count: %w", err)
	}
	return count, nil
}

func (r *rateLimitRepo) SetLockout(ctx context.Context, username string) (time.Time, error) {
	key := "lockout:" + username
	expiry := time.Now().Add(lockoutMaxAge)
	if err := r.client.Set(ctx, key, expiry.Unix(), lockoutTTL).Err(); err != nil {
		return time.Time{}, fmt.Errorf("setting lockout: %w", err)
	}
	return expiry, nil
}

func (r *rateLimitRepo) GetLockoutExpiry(ctx context.Context, username string) (time.Time, error) {
	key := "lockout:" + username
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return time.Time{}, fmt.Errorf("getting lockout TTL: %w", err)
	}
	if ttl <= 0 {
		return time.Time{}, nil
	}
	return time.Now().Add(ttl), nil
}

func (r *rateLimitRepo) StoreRefreshToken(ctx context.Context, token string, userID uint) error {
	key := "refresh:" + token
	if err := r.client.Set(ctx, key, userID, refreshTTL).Err(); err != nil {
		return fmt.Errorf("storing refresh token: %w", err)
	}
	return nil
}
