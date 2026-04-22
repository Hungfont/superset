package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/db"
)

func (s *DatabaseService) ListSchemas(ctx context.Context, actorUserID uint, databaseID uint, forceRefresh bool, rateLimitKey string) ([]string, error) {
	if databaseID == 0 {
		return nil, domain.ErrInvalidDatabase
	}
	if err := s.enforceSchemaRefreshRateLimit(ctx, forceRefresh, rateLimitKey); err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("schema:%d:schemas", databaseID)
	if !forceRefresh {
		cachedValue := make([]string, 0)
		if hit := s.readSchemaCache(ctx, cacheKey, &cachedValue); hit {
			return cachedValue, nil
		}
	}

	connection, err := s.loadDatabaseConnectionForIntrospection(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, databaseSchemaIntrospectionTimeout)
	defer cancel()

	schemas, err := s.schemaInspector.ListSchemas(timeoutCtx, connection)
	if err != nil {
		return nil, mapSchemaIntrospectionError(err)
	}

	s.writeSchemaCache(ctx, cacheKey, schemas)
	return schemas, nil
}

func (s *DatabaseService) ListTables(ctx context.Context, actorUserID uint, databaseID uint, req domain.ListDatabaseTablesRequest, forceRefresh bool, rateLimitKey string) (*domain.DatabaseTableListResponse, error) {
	if databaseID == 0 {
		return nil, domain.ErrInvalidDatabase
	}
	if err := s.enforceSchemaRefreshRateLimit(ctx, forceRefresh, rateLimitKey); err != nil {
		return nil, err
	}

	normalized := normalizeListTablesRequest(req)
	if normalized.Schema == "" {
		return nil, domain.ErrInvalidDatabase
	}

	cacheKey := fmt.Sprintf("schema:%d:%s:tables:%d:%d", databaseID, normalized.Schema, normalized.Page, normalized.PageSize)
	if !forceRefresh {
		cachedValue := domain.DatabaseTableListResponse{}
		if hit := s.readSchemaCache(ctx, cacheKey, &cachedValue); hit {
			return &cachedValue, nil
		}
	}

	connection, err := s.loadDatabaseConnectionForIntrospection(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, databaseSchemaIntrospectionTimeout)
	defer cancel()

	tables, total, err := s.schemaInspector.ListTables(timeoutCtx, connection, normalized.Schema, normalized.Page, normalized.PageSize)
	if err != nil {
		return nil, mapSchemaIntrospectionError(err)
	}

	result := &domain.DatabaseTableListResponse{
		Items:    tables,
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}

	s.writeSchemaCache(ctx, cacheKey, result)
	return result, nil
}

func (s *DatabaseService) ListColumns(ctx context.Context, actorUserID uint, databaseID uint, req domain.ListDatabaseColumnsRequest, forceRefresh bool, rateLimitKey string) ([]domain.DatabaseColumn, error) {
	if databaseID == 0 {
		return nil, domain.ErrInvalidDatabase
	}
	if err := s.enforceSchemaRefreshRateLimit(ctx, forceRefresh, rateLimitKey); err != nil {
		return nil, err
	}

	normalized := normalizeListColumnsRequest(req)
	if normalized.Schema == "" || normalized.Table == "" {
		return nil, domain.ErrInvalidDatabase
	}

	cacheKey := fmt.Sprintf("schema:%d:%s:%s:columns", databaseID, normalized.Schema, normalized.Table)
	if !forceRefresh {
		cachedValue := make([]domain.DatabaseColumn, 0)
		if hit := s.readSchemaCache(ctx, cacheKey, &cachedValue); hit {
			return cachedValue, nil
		}
	}

	connection, err := s.loadDatabaseConnectionForIntrospection(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, databaseSchemaIntrospectionTimeout)
	defer cancel()

	columns, err := s.schemaInspector.ListColumns(timeoutCtx, connection, normalized.Schema, normalized.Table)
	if err != nil {
		return nil, mapSchemaIntrospectionError(err)
	}

	s.writeSchemaCache(ctx, cacheKey, columns)
	return columns, nil
}

func (s *DatabaseService) enforceSchemaRefreshRateLimit(ctx context.Context, forceRefresh bool, rateLimitKey string) error {
	if !forceRefresh {
		return nil
	}

	key := strings.TrimSpace(rateLimitKey)
	if key == "" {
		key = "database-schema-refresh:global"
	}

	allowed, err := s.testRateLimit.Allow(ctx, key, databaseSchemaRefreshRateLimitCap, databaseSchemaRefreshRateLimitWindow)
	if err != nil {
		return fmt.Errorf("checking schema refresh rate limit: %w", err)
	}
	if !allowed {
		return domain.ErrRateLimited
	}

	return nil
}

func (s *DatabaseService) loadDatabaseConnectionForIntrospection(ctx context.Context, databaseID uint) (SQLConnection, error) {
	database, err := s.repo.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	if s.poolManager == nil {
		return nil, domain.ErrDatabaseUnreachable
	}

	connection, err := s.poolManager.Get(ctx, databaseID, database.SQLAlchemyURI)
	if err != nil {
		return nil, mapSchemaIntrospectionError(err)
	}

	return connection, nil
}

func (s *DatabaseService) readSchemaCache(ctx context.Context, key string, target any) bool {
	if s.schemaCache == nil {
		return false
	}

	raw, found, err := s.schemaCache.Get(ctx, key)
	if err != nil || !found {
		return false
	}

	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return false
	}

	return true
}

func (s *DatabaseService) writeSchemaCache(ctx context.Context, key string, value any) {
	if s.schemaCache == nil {
		return
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return
	}

	_ = s.schemaCache.Set(ctx, key, string(raw), databaseSchemaCacheTTL)
}

func normalizeListTablesRequest(req domain.ListDatabaseTablesRequest) domain.ListDatabaseTablesRequest {
	page, pageSize := normalizeTablesPagination(req.Page, req.PageSize)
	return domain.ListDatabaseTablesRequest{
		Schema:   strings.TrimSpace(req.Schema),
		Page:     page,
		PageSize: pageSize,
	}
}

func normalizeListColumnsRequest(req domain.ListDatabaseColumnsRequest) domain.ListDatabaseColumnsRequest {
	return domain.ListDatabaseColumnsRequest{
		Schema: strings.TrimSpace(req.Schema),
		Table:  strings.TrimSpace(req.Table),
	}
}

func mapSchemaIntrospectionError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrForbidden),
		errors.Is(err, domain.ErrRateLimited),
		errors.Is(err, domain.ErrInvalidDatabase),
		errors.Is(err, domain.ErrDatabaseNotFound):
		return err
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled), errors.Is(err, domain.ErrDatabaseTimeout):
		return domain.ErrDatabaseTimeout
	case errors.Is(err, domain.ErrDatabaseUnreachable):
		return err
	default:
		return fmt.Errorf("%w: %v", domain.ErrDatabaseUnreachable, err)
	}
}
