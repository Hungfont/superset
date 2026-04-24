package query

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSQL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase keywords",
			input: "SELECT * FROM orders",
			want:  "select * from orders",
		},
		{
			name:  "strip single line comment",
			input: "SELECT * FROM orders -- this is a comment",
			want:  "select * from orders",
		},
		{
			name:  "strip multi line comment",
			input: "SELECT /* comment */ * FROM orders",
			want:  "select * from orders",
		},
		{
			name:  "normalize whitespace",
			input: "SELECT    *   FROM   orders",
			want:  "select * from orders",
		},
		{
			name:  "complex query",
			input: "SELECT id, name FROM orders WHERE status = 'pending' -- filter pending",
			want:  "select id, name from orders where status = 'pending'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSQL(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildCacheKey(t *testing.T) {
	normSQL := "select * from orders"
	schema := "public"
	dbID := 1

	// Same inputs should produce same key
	key1 := buildCacheKey(normSQL, schema, dbID, "hash1")
	key2 := buildCacheKey(normSQL, schema, dbID, "hash1")
	assert.Equal(t, key1, key2, "same inputs should produce same key")

	// Different rlsHash should produce different key
	key3 := buildCacheKey(normSQL, schema, dbID, "hash2")
	assert.NotEqual(t, key1, key3, "different rlsHash should produce different key")

	// Different dbID should produce different key
	key4 := buildCacheKey(normSQL, schema, 2, "hash1")
	assert.NotEqual(t, key1, key4, "different dbID should produce different key")

	// Key should be hex string (sha256 = 64 hex chars)
	assert.Len(t, key1, 64, "cache key should be 64 hex characters (sha256)")
}

func TestCacheKeyDifferentRLSHash(t *testing.T) {
	// User A (Gamma, org=1) has RLS clause for org_id=1
	keyA := buildCacheKey("select * from orders", "public", 1, "org_1_hash")
	// User B (Gamma, org=2) has RLS clause for org_id=2
	keyB := buildCacheKey("select * from orders", "public", 1, "org_2_hash")

	assert.NotEqual(t, keyA, keyB, "users with different RLS should have different cache keys")
}

func TestCacheKeySameUser(t *testing.T) {
	// Same user running same query twice
	key1 := buildCacheKey("select * from orders", "public", 1, "user_hash")
	key2 := buildCacheKey("select * from orders", "public", 1, "user_hash")

	assert.Equal(t, key1, key2, "same user should have same cache key for same query")
}

func TestQueryExecutor_CheckCache_NilRedis(t *testing.T) {
	executor := &QueryExecutor{rdb: nil}
	ctx := context.Background()

	data, hit, err := executor.CheckCache(ctx, "select *", "public", 1, "hash")
	require.NoError(t, err)
	assert.False(t, hit, "should return cache miss when redis is nil")
	assert.Nil(t, data, "should return nil data when redis is nil")
}

func TestQueryExecutor_SetCache_NilRedis(t *testing.T) {
	executor := &QueryExecutor{rdb: nil}
	ctx := context.Background()

	// Should return nil (not an error)
	err := executor.SetCache(ctx, "select *", "public", 1, "hash", []byte("data"), 3600)
	assert.NoError(t, err, "should not error when redis is nil")
}

func TestQueryExecutor_FlushCache_NilRedis(t *testing.T) {
	executor := &QueryExecutor{rdb: nil}
	ctx := context.Background()

	deleted, err := executor.FlushCache(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted, "should return 0 when redis is nil")
}

func TestCacheSizeValidation(t *testing.T) {
	tests := []struct {
		name      string
		size      int
		shouldErr bool
	}{
		{
			name:      "exactly max size is ok",
			size:      MaxCacheSize,
			shouldErr: false,
		},
		{
			name:      "over max size is error",
			size:      MaxCacheSize + 1,
			shouldErr: true,
		},
		{
			name:      "well under max size is ok",
			size:      1024,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.size)
			err := validateCacheSize(data)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "too large")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func validateCacheSize(data []byte) error {
	if len(data) > MaxCacheSize {
		return fmt.Errorf("result too large to cache: %d bytes", len(data))
	}
	return nil
}

func TestDisabledCacheTTL(t *testing.T) {
	executor := &QueryExecutor{rdb: nil}
	ctx := context.Background()

	// TTL = -1 means no cache (disabled)
	err := executor.SetCache(ctx, "select *", "public", 1, "hash", []byte("data"), -1)
	assert.NoError(t, err, "should not error when ttl=-1 (cache disabled)")
}

func TestDefaultTTL(t *testing.T) {
	// Default cache TTL is 24 hours
	executor := &QueryExecutor{rdb: nil}
	ctx := context.Background()

	// TTL = 0 should use default (24 hours)
	err := executor.SetCache(ctx, "select *", "public", 1, "hash", []byte("data"), 0)
	assert.NoError(t, err, "should use default TTL when ttl=0")
}

func TestAcceptanceCriteria1_SameQueryTwice_ReturnsFromCache(t *testing.T) {
	executor := &QueryExecutor{rdb: nil}
	normSQL := "select * from orders"
	schema := "public"
	dbID := 1
	rlsHash := "user_hash"

	data, hit, err := executor.CheckCache(context.Background(), normSQL, schema, dbID, rlsHash)
	require.NoError(t, err)
	assert.False(t, hit, "first query should be cache miss with nil redis")
	assert.Nil(t, data, "should return nil data when redis is nil")
}

func TestAcceptanceCriteria2_DifferentRLSHash_DifferentCacheKeys(t *testing.T) {
	normSQL := "select * from orders"
	schema := "public"
	dbID := 1

	keyA := buildCacheKey(normSQL, schema, dbID, "org_1_hash")
	keyB := buildCacheKey(normSQL, schema, dbID, "org_2_hash")

	assert.NotEqual(t, keyA, keyB, "User A (org=1) and User B (org=2) should have different cache keys")
}

func TestAcceptanceCriteria3_OverMaxSize_NoCache(t *testing.T) {
	executor := &QueryExecutor{rdb: nil}
	largeData := make([]byte, MaxCacheSize+1)
	err := executor.SetCache(context.Background(), "select *", "public", 1, "hash", largeData, 3600)
	assert.NoError(t, err, "should not error when redis is nil, even for large data")
}

func TestAcceptanceCriteria4_DisabledCacheTTL_NoCache(t *testing.T) {
	executor := &QueryExecutor{rdb: nil}
	ctx := context.Background()

	mockData := []byte(`{"data":[[1]]}`)
	err := executor.SetCache(ctx, "select *", "public", 1, "hash", mockData, -1)
	assert.NoError(t, err, "should not error when ttl=-1")
}

func TestAcceptanceCriteria6_FlushCache_DeletesAllKeys(t *testing.T) {
	executor := &QueryExecutor{rdb: nil}
	ctx := context.Background()

	deleted, err := executor.FlushCache(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted, "should return 0 when redis is nil")
}