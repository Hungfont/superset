package auth

import (
	"context"
	"sync"
	"time"

	domain "superset/auth-service/internal/domain/db"
)

type inMemorySchemaCache struct {
	mu      sync.RWMutex
	entries map[string]schemaCacheEntry
}

type schemaCacheEntry struct {
	value     string
	expiresAt time.Time
}

func newInMemorySchemaCache() domain.SchemaCacheRepository {
	return &inMemorySchemaCache{entries: map[string]schemaCacheEntry{}}
}

func (c *inMemorySchemaCache) Get(_ context.Context, key string) (string, bool, error) {
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return "", false, nil
	}

	if now.After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return "", false, nil
	}

	return entry.value, true, nil
}

func (c *inMemorySchemaCache) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)
	if ttl <= 0 {
		expiresAt = time.Now()
	}

	c.mu.Lock()
	c.entries[key] = schemaCacheEntry{value: value, expiresAt: expiresAt}
	c.mu.Unlock()
	return nil
}
