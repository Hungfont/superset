package redis

import (
	"context"
	"fmt"
	"strconv"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/redis/go-redis/v9"
)

const refreshTokenTTL = refreshTTL // 7 days, defined in rate_repo.go

type refreshRepo struct {
	client *redis.Client
}

// NewRefreshRepository returns a RefreshRepository backed by Redis.
func NewRefreshRepository(client *redis.Client) domain.RefreshRepository {
	return &refreshRepo{client: client}
}

// tokenKey returns the primary key for a refresh token.
func tokenKey(token string) string { return "refresh:" + token }

// userSetKey returns the key for the per-user set of active tokens.
func userSetKey(userID uint) string { return fmt.Sprintf("user_tokens:%d", userID) }

// Store persists the refresh token and registers it in the per-user token set.
func (r *refreshRepo) Store(ctx context.Context, token string, userID uint) error {
	pipe := r.client.TxPipeline()
	pipe.Set(ctx, tokenKey(token), userID, refreshTokenTTL)
	pipe.SAdd(ctx, userSetKey(userID), token)
	pipe.Expire(ctx, userSetKey(userID), refreshTokenTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("storing refresh token: %w", err)
	}
	return nil
}

// GetUserID fetches the userID associated with the given token.
// Returns found=false when the token is absent (expired or unknown).
func (r *refreshRepo) GetUserID(ctx context.Context, token string) (uint, bool, error) {
	val, err := r.client.Get(ctx, tokenKey(token)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("getting refresh token: %w", err)
	}
	uid, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parsing user id from refresh token: %w", err)
	}
	return uint(uid), true, nil
}

// Delete removes a single refresh token from the primary store.
// Returns deleted=true when the key existed.
func (r *refreshRepo) Delete(ctx context.Context, token string) (bool, error) {
	n, err := r.client.Del(ctx, tokenKey(token)).Result()
	if err != nil {
		return false, fmt.Errorf("deleting refresh token: %w", err)
	}
	return n > 0, nil
}

// DeleteAllForUser revokes every active refresh token for the given user.
// It fetches all token values from the per-user set, deletes each primary
// key, then drops the set itself — atomically via a pipeline.
func (r *refreshRepo) DeleteAllForUser(ctx context.Context, userID uint) error {
	setKey := userSetKey(userID)
	tokens, err := r.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return fmt.Errorf("fetching user token set: %w", err)
	}
	if len(tokens) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, t := range tokens {
		pipe.Del(ctx, tokenKey(t))
	}
	pipe.Del(ctx, setKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("deleting all user tokens: %w", err)
	}
	return nil
}
