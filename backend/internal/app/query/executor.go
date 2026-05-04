package query

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	dbpool "superset/auth-service/internal/app/db"
	authdomain "superset/auth-service/internal/domain/auth"
	"superset/auth-service/internal/domain/dataset"
	domdb "superset/auth-service/internal/domain/db"
	"superset/auth-service/internal/domain/query"

	"github.com/redis/go-redis/v9"
)

const (
	MaxCacheSize    = 10 * 1024 * 1024 // 10MB
	DefaultCacheTTL = 24 * time.Hour

	// Role-based row limits (QE-001 #3)
	RowLimitGamma = 10000
	RowLimitAlpha = 100000
	RowLimitAdmin = 10000000

	// Query timeout (QE-001 #4)
	QueryTimeout = 30 * time.Second
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

// getRowLimit returns the row limit based on user role
func getRowLimit(roles []string) int {
	for _, role := range roles {
		if role == "Admin" {
			return RowLimitAdmin
		}
		if role == "Alpha" {
			return RowLimitAlpha
		}
	}
	return RowLimitGamma
}

type QueryExecutor struct {
	rlsInjector      *RLSInjector
	rlsRepo          authdomain.RLSFilterRepository
	datasetRepo      dataset.Repository
	databaseRepo     domdb.DatabaseRepository // For DB permission check (QE-001 #5)
	queryRepo       query.Repository        // For query recording (QE-001)
	rdb             *redis.Client
	connectionPool  dbpool.DatabaseConnectionPool // Connection pool for query execution
}

func NewQueryExecutor(rlsInjector *RLSInjector, rlsRepo authdomain.RLSFilterRepository, datasetRepo dataset.Repository, databaseRepo domdb.DatabaseRepository, queryRepo query.Repository, rdb *redis.Client, connectionPool dbpool.DatabaseConnectionPool) *QueryExecutor {
	executor := &QueryExecutor{
		rlsInjector:      rlsInjector,
		rlsRepo:          rlsRepo,
		datasetRepo:      datasetRepo,
		databaseRepo:     databaseRepo,
		queryRepo:       queryRepo,
		rdb:             rdb,
		connectionPool:  connectionPool,
	}

	// Connect RLS injector to real repository with Redis cache
	if rlsInjector != nil && rdb != nil {
		repoAdapter := &rlsRepoAdapter{
			rlsRepo: rlsRepo,
			rdb:     rdb,
		}
		executor.rlsInjector = NewRLSInjectorWithRedis(repoAdapter, rdb)
	}

	return executor
}

type rlsRepoAdapter struct {
	rlsRepo authdomain.RLSFilterRepository
	rdb     *redis.Client
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

// cachedResult is the internal type for caching - uses domain ExecuteResponse.Query
type cachedResult struct {
	Data      interface{}        `json:"data"`
	Columns   []query.ColumnInfo `json:"columns"`
	FromCache bool             `json:"from_cache"`
	Query    query.ExecuteMeta `json:"query"`
}

var (
	singleLineCommentRE = regexp.MustCompile(`--[^\n]*`)
	multiLineCommentRE  = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	whitespaceRE        = regexp.MustCompile(`\s+`)
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

func (e *QueryExecutor) Execute(ctx context.Context, req ExecuteRequest, userCtx authdomain.UserContext) (*ExecuteResponse, error) {
	startTime := time.Now()

	// Get user roles
	roleNames, err := e.rlsRepo.GetRoleNamesByUser(ctx, userCtx.ID)
	if err != nil {
		return nil, fmt.Errorf("getting user roles: %w", err)
	}

	// QE-001 #5: Check database permission
	db, err := e.databaseRepo.GetDatabaseByID(ctx, uint(req.DatabaseID))
	if err != nil {
		return nil, fmt.Errorf("database not found: %w", err)
	}
	if db == nil {
		return nil, fmt.Errorf("database not found")
	}
	// Check: user is creator OR database exposes in sqllab
	isCreator := db.CreatedByFK == userCtx.ID
	if !isCreator && !db.ExposeInSQLLab {
		return nil, fmt.Errorf("access denied to database")
	}

	// QE-001 #8: Client ID deduplication (check for existing running query)
	if req.ClientID != "" && e.rdb != nil {
		existingKey := "query:dedup:" + req.ClientID
		existingStatus, err := e.rdb.Get(ctx, existingKey).Result()
		if err == nil && existingStatus != "" {
			// Return cached result if available
			resultKey := "query:result:" + req.ClientID
			resultData, err := e.rdb.Get(ctx, resultKey).Bytes()
			if err == nil {
				var result cachedResult
				if err := json.Unmarshal(resultData, &result); err == nil {
					return &ExecuteResponse{
						Data:      result.Data,
						Columns:   result.Columns,
						FromCache: true,
						Query: query.ExecuteMeta{
							ExecutedSQL: req.SQL,
							RLSApplied:  false,
							StartTime:   startTime,
							EndTime:     time.Now(),
						},
					}, nil
				}
			}
		}
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
	ds, err := e.datasetRepo.GetDatasetByID(ctx, uint(req.DatabaseID))
	if err == nil && ds != nil {
		cacheTimeout = ds.CacheTimeout
	}

	if cacheTimeout == -1 || req.ForceRefresh {
		// QE-001 #2: Create query record with status="running"
		var queryID string
		if e.queryRepo != nil && req.ClientID != "" {
			q := &query.Query{
				ID:          uuid.New().String(),
				ClientID:    req.ClientID,
				DatabaseID:  req.DatabaseID,
				UserID:      userCtx.ID,
				SQL:         req.SQL,
				ExecutedSQL: executedSQL,
				Status:      "running",
				StartTime:   &startTime,
				Schema:      schema,
			}
			if err := e.queryRepo.Create(ctx, q); err != nil {
				fmt.Printf("failed to create query record: %v\n", err)
			} else {
				queryID = q.ID
			}
		}
		return e.executeAndRespond(ctx, req, userCtx, executedSQL, rlsApplied, startTime, false, roleNames, queryID, rlsHash)
	}

	cachedData, cacheHit, err := e.CheckCache(ctx, normSQL, schema, int(req.DatabaseID), rlsHash)
	if err != nil {
		fmt.Printf("cache check error: %v\n", err)
	} else if cacheHit {
		var result cachedResult
		if err := json.Unmarshal(cachedData, &result); err == nil {
			rowCount := 0
			if dataSlice, ok := result.Data.([]interface{}); ok {
				rowCount = len(dataSlice)
			}
			return &ExecuteResponse{
				Data:              result.Data,
				Columns:           result.Columns,
				FromCache:        true,
				ResultsTruncated: false,
				Query: query.ExecuteMeta{
					ExecutedSQL: executedSQL,
					RLSApplied:  rlsApplied,
					Rows:       rowCount,
					StartTime:   startTime,
					EndTime:     time.Now(),
				},
			}, nil
		}
	}

	// QE-001 #2: Create query record with status="running"
	var queryID string
	if e.queryRepo != nil && req.ClientID != "" {
		q := &query.Query{
			ID:          uuid.New().String(),
			ClientID:    req.ClientID,
			DatabaseID:  req.DatabaseID,
			UserID:      userCtx.ID,
			SQL:         req.SQL,
			ExecutedSQL: executedSQL,
			Status:      "running",
			StartTime:   &startTime,
			Schema:      schema,
		}
		if err := e.queryRepo.Create(ctx, q); err != nil {
			fmt.Printf("failed to create query record: %v\n", err)
		} else {
			queryID = q.ID
		}
	}

	return e.executeAndRespond(ctx, req, userCtx, executedSQL, rlsApplied, startTime, false, roleNames, queryID, rlsHash)
}

func (e *QueryExecutor) executeSQL(ctx context.Context, databaseID uint, querySQL string) ([]interface{}, []string, int, error) {
	dbInfo, err := e.databaseRepo.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("getting database: %w", err)
	}

	dbConn, err := sql.Open("postgres", dbInfo.SQLAlchemyURI)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("opening database connection: %w", err)
	}
	defer dbConn.Close()

	rows, err := dbConn.QueryContext(ctx, querySQL)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("getting columns: %w", err)
	}

	var result []interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		result = append(result, values)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, 0, fmt.Errorf("iterating rows: %w", err)
	}

	return result, columns, len(result), nil
}

func (e *QueryExecutor) executeAndRespond(ctx context.Context, req ExecuteRequest, userCtx authdomain.UserContext, executedSQL string, rlsApplied bool, startTime time.Time, fromCache bool, roleNames []string, queryID string, rlsHash string) (*ExecuteResponse, error) {
	// QE-001 #3: Apply role-based row limit
	rowLimit := getRowLimit(roleNames)
	effectiveLimit := rowLimit
	if req.Limit != nil && *req.Limit < rowLimit {
		effectiveLimit = *req.Limit
	}

	// Check if results will be truncated (user asked for more than role allows)
	resultsTruncated := req.Limit != nil && *req.Limit > rowLimit

	// Apply LIMIT to SQL
	if effectiveLimit > 0 {
		executedSQL = fmt.Sprintf("SELECT * FROM (%s) AS _sub LIMIT %d", executedSQL, effectiveLimit)
	}

	// QE-001 #4: Apply 30s timeout with proper context handling
	execCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var resultData []interface{}
	var columns []string
	var columnInfos []query.ColumnInfo
	var rowCount int
	var execErr error

	// Actually execute the query using connection pool
	if e.connectionPool != nil {
		dbInfo, err := e.databaseRepo.GetDatabaseByID(ctx, uint(req.DatabaseID))
		if err != nil {
			return e.buildErrorResponse(execCtx, err, queryID, "getting database", 500)
		}

		conn, err := e.connectionPool.Get(execCtx, uint(req.DatabaseID), dbInfo.SQLAlchemyURI)
		if err != nil {
			return e.buildErrorResponse(execCtx, err, queryID, "getting connection", 500)
		}

		rows, err := conn.QueryContext(execCtx, executedSQL)
		if err != nil {
			// QE-001 #6: Handle SQL errors as 400 Bad Request
			return e.buildErrorResponse(execCtx, err, queryID, "executing query", 400)
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return e.buildErrorResponse(execCtx, err, queryID, "getting columns", 500)
		}
		columns = cols

		// Convert column names to ColumnInfo type
		columnInfos := make([]query.ColumnInfo, len(cols))
		for i, col := range cols {
			columnInfos[i] = query.ColumnInfo{Name: col}
		}

		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return e.buildErrorResponse(execCtx, err, queryID, "scanning row", 500)
			}
			resultData = append(resultData, values)
		}

		if err := rows.Err(); err != nil {
			return e.buildErrorResponse(execCtx, err, queryID, "iterating rows", 500)
		}
		rowCount = len(resultData)
	} else {
		// Fallback: use direct SQL connection if no pool
		resultData, columns, rowCount, execErr = e.executeSQL(execCtx, req.DatabaseID, executedSQL)
		if execErr != nil {
			// Check if it's a timeout error (QE-001 #4)
			if execCtx.Err() == context.DeadlineExceeded {
				e.updateQueryStatus(ctx, queryID, "timed_out", 0)
				return nil, &QueryError{Code: 408, Message: "Query exceeded 30s timeout"}
			}
			// Check for SQL syntax errors (QE-001 #6)
			return e.buildErrorResponse(execCtx, execErr, queryID, "executing query", 400)
		}

		// Convert column names to ColumnInfo type
		columnInfos = make([]query.ColumnInfo, len(columns))
		for i, col := range columns {
			columnInfos[i] = query.ColumnInfo{Name: col}
		}
	}

	endTime := time.Now()

	result := cachedResult{
		Data:      resultData,
		Columns:   columnInfos,
		FromCache: fromCache,
		Query: query.ExecuteMeta{
			ExecutedSQL: executedSQL,
			RLSApplied: rlsApplied,
			Rows:      rowCount,
			StartTime: startTime,
			EndTime:   endTime,
		},
	}

	// QE-003: Cache result if not from cache and result is small enough
	if e.rdb != nil && !fromCache {
		resultBytes, err := json.Marshal(result)
		if err == nil && len(resultBytes) <= MaxCacheSize {
			normSQL := normalizeSQL(req.SQL)
			schema := req.Schema
			if schema == "" {
				schema = "public"
			}

			cacheTimeout := 0
			if e.datasetRepo != nil {
				ds, _ := e.datasetRepo.GetDatasetByID(ctx, uint(req.DatabaseID))
				if ds != nil {
					cacheTimeout = ds.CacheTimeout
				}
			}

			if cacheTimeout != -1 {
				e.SetCache(ctx, normSQL, schema, int(req.DatabaseID), rlsHash, resultBytes, cacheTimeout)
			}
		}
	}

	// QE-001 #4: Update query record with status, rows, end_time
	e.updateQueryStatus(ctx, queryID, "success", rowCount)

	return &ExecuteResponse{
		Data:              result.Data,
		Columns:           result.Columns,
		FromCache:        fromCache,
		ResultsTruncated: resultsTruncated,
		Query:           result.Query,
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

// QueryError represents a query execution error with HTTP status code
type QueryError struct {
	Code    int
	Message string
}

func (e *QueryError) Error() string {
	return e.Message
}

// buildErrorResponse creates an error response and updates query status
func (e *QueryExecutor) buildErrorResponse(ctx context.Context, err error, queryID string, operation string, statusCode int) (*ExecuteResponse, error) {
	errMsg := err.Error()

	// Update query status to failed
	if queryID != "" {
		e.updateQueryStatus(ctx, queryID, "failed", 0)
	}

	// Map common PostgreSQL errors to 400
	if strings.Contains(errMsg, "syntax error at") ||
		strings.Contains(errMsg, "42601") || // syntax_error
		strings.Contains(errMsg, "42P01") || // undefined_table
		strings.Contains(errMsg, "42703") || // undefined_column
		strings.Contains(errMsg, "22P02") { // invalid_text_representation
		return nil, &QueryError{
			Code:    400,
			Message: fmt.Sprintf("invalid_sql: %s", errMsg),
		}
	}

	return nil, &QueryError{
		Code:    statusCode,
		Message: fmt.Sprintf("%s: %s", operation, errMsg),
	}
}

// updateQueryStatus updates the query record in the database
func (e *QueryExecutor) updateQueryStatus(ctx context.Context, queryID string, status string, rowCount int) {
	if e.queryRepo == nil || queryID == "" {
		return
	}

	q, err := e.queryRepo.GetByID(ctx, queryID)
	if err != nil || q == nil {
		return
	}

	q.Status = status
	if status == "success" {
		q.Rows = rowCount
	}
	now := time.Now()
	q.EndTime = &now
	if e.rdb != nil && rowCount > 0 {
		q.ResultsKey = fmt.Sprintf("query:result:%s", queryID)
	}

	if err := e.queryRepo.Update(ctx, q); err != nil {
		fmt.Printf("failed to update query status: %v\n", err)
	}
}
