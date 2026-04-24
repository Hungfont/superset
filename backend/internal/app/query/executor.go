package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"superset/auth-service/internal/domain/auth"
	"superset/auth-service/internal/domain/dataset"

	"github.com/redis/go-redis/v9"
)

const (
	MaxCacheSize     = 10 * 1024 * 1024 // 10MB
	DefaultCacheTTL = 24 * time.Hour
)

type RLSFilterClause struct {
	Clause     string
	FilterType string
	GroupKey   string
}

type RLSInjectorOption func(*injectorOptions)

type injectorOptions struct {
	userID   int
	username string
}

func WithUserID(id int) RLSInjectorOption {
	return func(o *injectorOptions) {
		o.userID = id
	}
}

func WithUsername(name string) RLSInjectorOption {
	return func(o *injectorOptions) {
		o.username = name
	}
}

type RLSInjectorRepo interface {
	GetFiltersByDatasourceAndRoles(ctx context.Context, datasourceID int, roleNames []string) ([]RLSFilterClause, error)
	CacheGet(ctx context.Context, key string) ([]RLSFilterClause, error)
	CacheSet(ctx context.Context, key string, clauses []RLSFilterClause) error
}

type RLSInjector struct {
	repo RLSInjectorRepo
	rdb  *redis.Client
}

func NewRLSInjector(repo RLSInjectorRepo) *RLSInjector {
	return &RLSInjector{repo: repo}
}

func NewRLSInjectorWithRedis(repo RLSInjectorRepo, rdb *redis.Client) *RLSInjector {
	return &RLSInjector{repo: repo, rdb: rdb}
}

func (s *RLSInjector) InjectRLS(ctx context.Context, sql string, datasourceID int, roles []string, opts ...RLSInjectorOption) (string, error) {
	options := &injectorOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if isAdmin(roles) {
		return sql, nil
	}

	filteredRoles := filterNonAdminRoles(roles)
	if len(filteredRoles) == 0 {
		return sql, nil
	}

	cacheKey := s.buildCacheKey(filteredRoles, datasourceID)

	var clauses []RLSFilterClause
	var cacheErr error

	if s.rdb != nil {
		clauses, cacheErr = s.cacheGet(ctx, cacheKey)
		if cacheErr == nil && clauses != nil {
			return s.applyRLSClauses(sql, clauses, options)
		}
	} else if s.repo != nil {
		clauses, cacheErr = s.repo.CacheGet(ctx, cacheKey)
		if cacheErr == nil && clauses != nil {
			return s.applyRLSClauses(sql, clauses, options)
		}
	}

	repoClauses, err := s.repo.GetFiltersByDatasourceAndRoles(ctx, datasourceID, filteredRoles)
	if err != nil {
		return "", fmt.Errorf("getting RLS filters: %w", err)
	}

	if len(repoClauses) == 0 {
		return sql, nil
	}

	if s.rdb != nil {
		s.cacheSet(ctx, cacheKey, repoClauses)
	} else if s.repo != nil {
		s.repo.CacheSet(ctx, cacheKey, repoClauses)
	}

	return s.applyRLSClauses(sql, repoClauses, options)
}

func (s *RLSInjector) InjectRLSWithClauses(ctx context.Context, sql string, datasourceID int, roles []string, opts ...RLSInjectorOption) (string, []RLSFilterClause, error) {
	options := &injectorOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if isAdmin(roles) {
		return sql, nil, nil
	}

	filteredRoles := filterNonAdminRoles(roles)
	if len(filteredRoles) == 0 {
		return sql, nil, nil
	}

	cacheKey := s.buildCacheKey(filteredRoles, datasourceID)

	var clauses []RLSFilterClause
	var cacheErr error

	if s.rdb != nil {
		clauses, cacheErr = s.cacheGet(ctx, cacheKey)
		if cacheErr == nil && clauses != nil {
			executed, err := s.applyRLSClauses(sql, clauses, options)
			return executed, clauses, err
		}
	} else if s.repo != nil {
		clauses, cacheErr = s.repo.CacheGet(ctx, cacheKey)
		if cacheErr == nil && clauses != nil {
			executed, err := s.applyRLSClauses(sql, clauses, options)
			return executed, clauses, err
		}
	}

	repoClauses, err := s.repo.GetFiltersByDatasourceAndRoles(ctx, datasourceID, filteredRoles)
	if err != nil {
		return "", nil, fmt.Errorf("getting RLS filters: %w", err)
	}

	if len(repoClauses) == 0 {
		return sql, nil, nil
	}

	if s.rdb != nil {
		s.cacheSet(ctx, cacheKey, repoClauses)
	} else if s.repo != nil {
		s.repo.CacheSet(ctx, cacheKey, repoClauses)
	}

	executed, err := s.applyRLSClauses(sql, repoClauses, options)
	return executed, repoClauses, err
}

func (s *RLSInjector) buildCacheKey(roles []string, datasourceID int) string {
	sortedRoles := make([]string, len(roles))
	copy(sortedRoles, roles)
	sort.Strings(sortedRoles)
	roleHash := sha256.Sum256([]byte(strings.Join(sortedRoles, ",")))
	return fmt.Sprintf("rls:%s:%d", hex.EncodeToString(roleHash[:4]), datasourceID)
}

func (s *RLSInjector) cacheGet(ctx context.Context, key string) ([]RLSFilterClause, error) {
	data, err := s.rdb.Get(ctx, "rls:"+key).Bytes()
	if err != nil {
		return nil, err
	}
	var clauses []RLSFilterClause
	if err := json.Unmarshal(data, &clauses); err != nil {
		return nil, err
	}
	return clauses, nil
}

func (s *RLSInjector) cacheSet(ctx context.Context, key string, clauses []RLSFilterClause) error {
	data, err := json.Marshal(clauses)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, "rls:"+key, data, 5*time.Minute).Err()
}

func (s *RLSInjector) applyRLSClauses(sql string, clauses []RLSFilterClause, opts *injectorOptions) (string, error) {
	var regularClauses []string
	var baseClause *string

	for _, fc := range clauses {
		rendered := s.renderTemplate(fc.Clause, opts.userID, opts.username)
		if fc.FilterType == "Base" {
			baseClause = &rendered
		} else {
			regularClauses = append(regularClauses, rendered)
		}
	}

	if len(regularClauses) == 0 && baseClause == nil {
		return sql, nil
	}

	if isUnionQuery(sql) {
		return s.applyRLSToUnion(sql, regularClauses, baseClause)
	}

	if baseClause != nil {
		return s.replaceWhereClause(sql, *baseClause)
	}

	return s.addWhereClause(sql, regularClauses)
}

func isUnionQuery(sql string) bool {
	upper := strings.ToUpper(sql)
	return strings.Contains(upper, "UNION") || strings.Contains(upper, "INTERSECT") || strings.Contains(upper, "EXCEPT")
}

func (s *RLSInjector) applyRLSToUnion(sql string, regularClauses []string, baseClause *string) (string, error) {
	unionPattern := regexp.MustCompile(`(?i)(UNION|INTERSECT|EXCEPT)`)
	parts := unionPattern.Split(sql, -1)

	matches := unionPattern.FindAllStringIndex(sql, -1)

	var result strings.Builder

	for i, part := range parts {
		part = strings.TrimSpace(part)

		if i > 0 && i-1 < len(matches) {
			keyword := sql[matches[i-1][0]:matches[i-1][1]]
			result.WriteString(keyword + " ")
		}

		if len(part) > 0 {
			var injectedPart string
			var err error

			if baseClause != nil {
				injectedPart, err = s.replaceWhereClause(part, *baseClause)
			} else if len(regularClauses) > 0 {
				injectedPart, err = s.addWhereClause(part, regularClauses)
			} else {
				injectedPart = part
			}

			if err != nil {
				return sql, err
			}
			result.WriteString(injectedPart)
		}
	}

	return result.String(), nil
}

func (s *RLSInjector) renderTemplate(clause string, userID int, username string) string {
	result := strings.ReplaceAll(clause, "{{current_user_id}}", fmt.Sprintf("%d", userID))
	result = strings.ReplaceAll(result, "{{current_username}}", fmt.Sprintf("'%s'", escapeSQLString(username)))
	return result
}

func (s *RLSInjector) addWhereClause(sql string, clauses []string) (string, error) {
	upperSQL := strings.ToUpper(sql)

	if strings.Contains(upperSQL, "WHERE") {
		whereIndex := strings.Index(upperSQL, "WHERE")
		afterWhere := sql[whereIndex+5:]

		closeParen := findClosingParen(afterWhere)
		if closeParen > 0 {
			prefix := sql[:whereIndex+5+closeParen]
			rest := sql[whereIndex+5+closeParen:]
			newClause := strings.Join(clauses, " AND ")
			return prefix + " AND (" + newClause + ")" + rest, nil
		}

		newClause := strings.Join(clauses, " AND ")
		return sql + " AND (" + newClause + ")", nil
	}

	if strings.Contains(upperSQL, "GROUP BY") {
		groupIndex := strings.Index(upperSQL, "GROUP BY")
		prefix := sql[:groupIndex]
		rest := sql[groupIndex:]
		newClause := strings.Join(clauses, " AND ")
		return prefix + "WHERE (" + newClause + ") " + rest, nil
	}

	if strings.Contains(upperSQL, "ORDER BY") {
		orderIndex := strings.Index(upperSQL, "ORDER BY")
		prefix := sql[:orderIndex]
		rest := sql[orderIndex:]
		newClause := strings.Join(clauses, " AND ")
		return prefix + "WHERE (" + newClause + ") " + rest, nil
	}

	newClause := strings.Join(clauses, " AND ")
	return sql + " WHERE (" + newClause + ")", nil
}

func (s *RLSInjector) replaceWhereClause(sql string, replacement string) (string, error) {
	upperSQL := strings.ToUpper(sql)

	if strings.Contains(upperSQL, "WHERE") {
		whereIndex := strings.Index(upperSQL, "WHERE")
		afterWhere := sql[whereIndex+5:]

		closeParen := findClosingParen(afterWhere)
		prefix := sql[:whereIndex]
		if closeParen > 0 {
			rest := sql[whereIndex+5+closeParen:]
			return prefix + replacement + rest, nil
		}

		if idx := findNextKeyword(afterWhere); idx >= 0 {
			rest := sql[whereIndex+5+idx:]
			return prefix + replacement + rest, nil
		}

		return prefix + replacement, nil
	}

	return sql + " WHERE " + replacement, nil
}

func findClosingParen(s string) int {
	depth := 0
	for i, c := range s {
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

func findNextKeyword(s string) int {
	upper := strings.ToUpper(s)
	keywords := []string{"GROUP BY", "ORDER BY", "LIMIT", "HAVING", "UNION", "INTERSECT", "EXCEPT"}

	minIdx := -1
	for _, kw := range keywords {
		idx := strings.Index(upper, kw)
		if idx >= 0 && (minIdx < 0 || idx < minIdx) {
			minIdx = idx
		}
	}

	return minIdx
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func isAdmin(roles []string) bool {
	for _, role := range roles {
		if role == "Admin" {
			return true
		}
	}
	return false
}

func filterNonAdminRoles(roles []string) []string {
	var result []string
	for _, role := range roles {
		if role != "Admin" {
			result = append(result, role)
		}
	}
	return result
}

type QueryExecutor struct {
	rlsInjector *RLSInjector
	rlsRepo     auth.RLSFilterRepository
	datasetRepo dataset.Repository
	rdb         *redis.Client
}

func NewQueryExecutor(rlsInjector *RLSInjector, rlsRepo auth.RLSFilterRepository, datasetRepo dataset.Repository, rdb *redis.Client) *QueryExecutor {
	executor := &QueryExecutor{
		rlsInjector: rlsInjector,
		rlsRepo:     rlsRepo,
		datasetRepo: datasetRepo,
		rdb:         rdb,
	}

	// Connect RLS injector to real repository with Redis cache
	if rlsInjector != nil && rdb != nil {
		repoAdapter := &rlsRepoAdapter{
			rlsRepo: rlsRepo,
			rdb:    rdb,
		}
		executor.rlsInjector = NewRLSInjectorWithRedis(repoAdapter, rdb)
	}

	return executor
}

type rlsRepoAdapter struct {
	rlsRepo auth.RLSFilterRepository
	rdb    *redis.Client
}

func (a *rlsRepoAdapter) GetFiltersByDatasourceAndRoles(ctx context.Context, datasourceID int, roleNames []string) ([]RLSFilterClause, error) {
	roleIDMap := map[string]uint{
		"Admin": 1,
		"Alpha": 2,
		"Gamma": 3,
	}

	var roleIDs []uint
	for _, name := range roleNames {
		if id, ok := roleIDMap[name]; ok {
			roleIDs = append(roleIDs, id)
		}
	}

	// Call real repository to fetch RLS filters from database
	filters, err := a.rlsRepo.GetFiltersByDatasourceAndRoles(ctx, uint(datasourceID), roleIDs)
	if err != nil {
		return nil, err
	}

	if len(filters) == 0 {
		return nil, nil
	}

	clauses := make([]RLSFilterClause, len(filters))
	for i, f := range filters {
		clauses[i] = RLSFilterClause{
			Clause:     f.Clause,
			FilterType: string(f.FilterType),
			GroupKey:   f.GroupKey,
		}
	}

	return clauses, nil
}

func (a *rlsRepoAdapter) CacheGet(ctx context.Context, key string) ([]RLSFilterClause, error) {
	if a.rdb == nil {
		return nil, nil
	}

	data, err := a.rdb.Get(ctx, "rls:"+key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var clauses []RLSFilterClause
	if err := json.Unmarshal(data, &clauses); err != nil {
		return nil, err
	}
	return clauses, nil
}

func (a *rlsRepoAdapter) CacheSet(ctx context.Context, key string, clauses []RLSFilterClause) error {
	if a.rdb == nil {
		return nil
	}

	data, err := json.Marshal(clauses)
	if err != nil {
		return err
	}
	// Cache RLS clauses for 5 minutes
	return a.rdb.Set(ctx, "rls:"+key, data, 5*time.Minute).Err()
}

type ExecuteRequest struct {
	DatabaseID   uint   `json:"database_id" binding:"required"`
	SQL          string `json:"sql" binding:"required"`
	Limit        *int   `json:"limit"`
	Schema       string `json:"schema"`
	ForceRefresh bool   `json:"force_refresh"`
}

type QueryResult struct {
	Data       interface{} `json:"data"`
	Columns    []string    `json:"columns"`
	FromCache  bool        `json:"from_cache"`
	Query      QueryMeta   `json:"query"`
}

type QueryMeta struct {
	ExecutedSQL string    `json:"executed_sql"`
	RLSApplied  bool      `json:"rls_applied"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

type ExecuteResponse struct {
	Data      interface{} `json:"data"`
	Columns   []string    `json:"columns"`
	FromCache bool        `json:"from_cache"`
	Query     struct {
		ExecutedSQL string    `json:"executed_sql"`
		RLSApplied  bool      `json:"rls_applied"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
	} `json:"query"`
}

var (
	singleLineCommentRE = regexp.MustCompile(`--[^\n]*`)
	multiLineCommentRE  = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	whitespaceRE      = regexp.MustCompile(`\s+`)
)

func normalizeSQL(sql string) string {
	sql = singleLineCommentRE.ReplaceAllString(sql, "")
	sql = multiLineCommentRE.ReplaceAllString(sql, "")
	sql = strings.ToLower(sql)
	sql = whitespaceRE.ReplaceAllString(sql, " ")
	return strings.TrimSpace(sql)
}

func buildCacheKey(normSQL, schema string, dbID int, rlsHash string) string {
	data := normSQL + "|" + fmt.Sprintf("%d", dbID) + "|" + schema + "|" + rlsHash
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (e *QueryExecutor) CheckCache(ctx context.Context, normSQL, schema string, dbID int, rlsHash string) ([]byte, bool, error) {
	if e.rdb == nil {
		return nil, false, nil
	}

	cacheKey := buildCacheKey(normSQL, schema, dbID, rlsHash)
	data, err := e.rdb.Get(ctx, "qcache:"+cacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	return data, true, nil
}

func (e *QueryExecutor) SetCache(ctx context.Context, normSQL, schema string, dbID int, rlsHash string, data []byte, ttlSeconds int) error {
	if e.rdb == nil {
		return nil
	}

	if len(data) > MaxCacheSize {
		return fmt.Errorf("result too large to cache: %d bytes", len(data))
	}

	ttl := DefaultCacheTTL
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	} else if ttlSeconds == -1 {
		return nil
	}

	cacheKey := buildCacheKey(normSQL, schema, dbID, rlsHash)
	return e.rdb.Set(ctx, "qcache:"+cacheKey, data, ttl).Err()
}

func (e *QueryExecutor) FlushCache(ctx context.Context, dbID uint) (int64, error) {
	if e.rdb == nil {
		return 0, nil
	}

	pattern := fmt.Sprintf("qcache:*:%d", dbID)
	iter := e.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	var deleted int64

	for iter.Next(ctx) {
		if err := e.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return deleted, err
		}
		deleted++
	}

	if err := iter.Err(); err != nil {
		return deleted, err
	}

	return deleted, nil
}

func (e *QueryExecutor) Execute(ctx context.Context, req ExecuteRequest, userCtx auth.UserContext) (*ExecuteResponse, error) {
	startTime := time.Now()
	roleNames, err := e.rlsRepo.GetRoleNamesByUser(ctx, userCtx.ID)
	if err != nil {
		return nil, fmt.Errorf("getting user roles: %w", err)
	}

	executedSQL, rlsClauses, err := e.rlsInjector.InjectRLSWithClauses(ctx, req.SQL, int(req.DatabaseID), roleNames, WithUserID(int(userCtx.ID)), WithUsername(userCtx.Username))
	if err != nil {
		return nil, fmt.Errorf("injecting RLS: %w", err)
	}

	rlsApplied := executedSQL != req.SQL
	rlsHash := computeRLSHash(rlsClauses)
	normSQL := normalizeSQL(req.SQL)
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}

	cacheTimeout := 0
	if e.datasetRepo != nil {
		ds, err := e.datasetRepo.GetDatasetByID(ctx, uint(req.DatabaseID))
		if err == nil && ds != nil {
			cacheTimeout = ds.CacheTimeout
		}
	}

	if cacheTimeout == -1 || req.ForceRefresh {
		return e.executeAndRespond(ctx, req, userCtx, executedSQL, rlsApplied, startTime, false)
	}

	if e.rdb != nil {
		cachedData, cacheHit, err := e.CheckCache(ctx, normSQL, schema, int(req.DatabaseID), rlsHash)
		if err != nil {
			fmt.Printf("cache check error: %v\n", err)
		} else if cacheHit {
			var result QueryResult
			if err := json.Unmarshal(cachedData, &result); err == nil {
				result.Query.StartTime = startTime
				result.Query.EndTime = time.Now()
				return &ExecuteResponse{
					Data:      result.Data,
					Columns:   result.Columns,
					FromCache: true,
					Query: struct {
						ExecutedSQL string    `json:"executed_sql"`
						RLSApplied  bool      `json:"rls_applied"`
						StartTime   time.Time `json:"start_time"`
						EndTime     time.Time `json:"end_time"`
					}{
						ExecutedSQL: executedSQL,
						RLSApplied:  rlsApplied,
						StartTime:   startTime,
						EndTime:     time.Now(),
					},
				}, nil
			}
		}
	}

	return e.executeAndRespond(ctx, req, userCtx, executedSQL, rlsApplied, startTime, false)
}

func (e *QueryExecutor) executeAndRespond(ctx context.Context, req ExecuteRequest, userCtx auth.UserContext, executedSQL string, rlsApplied bool, startTime time.Time, fromCache bool) (*ExecuteResponse, error) {
	resultData := []interface{}{}
	columns := []string{}

	result := QueryResult{
		Data:      resultData,
		Columns:   columns,
		FromCache: fromCache,
		Query: QueryMeta{
			ExecutedSQL: executedSQL,
			RLSApplied:  rlsApplied,
			StartTime:   startTime,
			EndTime:     time.Now(),
		},
	}

	if e.rdb != nil && !fromCache {
		resultBytes, err := json.Marshal(result)
		if err == nil && len(resultBytes) <= MaxCacheSize {
			roleNames, _ := e.rlsRepo.GetRoleNamesByUser(ctx, userCtx.ID)
			_, rlsClauses, _ := e.rlsInjector.InjectRLSWithClauses(ctx, req.SQL, int(req.DatabaseID), roleNames, WithUserID(int(userCtx.ID)), WithUsername(userCtx.Username))
			rlsHash := computeRLSHash(rlsClauses)
			normSQL := normalizeSQL(req.SQL)
			schema := req.Schema
			if schema == "" {
				schema = "public"
			}

			cacheTimeout := 0
			if e.datasetRepo != nil {
				ds, err := e.datasetRepo.GetDatasetByID(ctx, uint(req.DatabaseID))
				if err == nil && ds != nil {
					cacheTimeout = ds.CacheTimeout
				}
			}

			if cacheTimeout != -1 {
				e.SetCache(ctx, normSQL, schema, int(req.DatabaseID), rlsHash, resultBytes, cacheTimeout)
			}
		}
	}

	result.Query.StartTime = startTime
	result.Query.EndTime = time.Now()

	return &ExecuteResponse{
		Data:      result.Data,
		Columns:   result.Columns,
		FromCache: fromCache,
		Query: struct {
			ExecutedSQL string    `json:"executed_sql"`
			RLSApplied  bool      `json:"rls_applied"`
			StartTime   time.Time `json:"start_time"`
			EndTime     time.Time `json:"end_time"`
		}{
			ExecutedSQL: executedSQL,
			RLSApplied:  rlsApplied,
			StartTime:   startTime,
			EndTime:     result.Query.EndTime,
		},
	}, nil
}

func computeRLSHash(clauses []RLSFilterClause) string {
	if len(clauses) == 0 {
		return "no_rls"
	}
	sortedClauses := make([]string, len(clauses))
	for i, c := range clauses {
		sortedClauses[i] = c.Clause
	}
	sort.Strings(sortedClauses)
	hash := sha256.Sum256([]byte(strings.Join(sortedClauses, "|")))
	return hex.EncodeToString(hash[:])
}
