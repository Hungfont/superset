package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	domain "superset/auth-service/internal/domain/db"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	databaseTestConnectionTimeout = 5 * time.Second
	databaseTestRateLimitCap      = 10
	databaseTestRateLimitWindow   = time.Minute
	databaseListDefaultPage       = 1
	databaseListDefaultPageSize   = 10
	databaseListMaxPageSize       = 100
)

// DatabaseConnectionTester validates a database connection before persistence.
type DatabaseConnectionTester interface {
	TestConnection(ctx context.Context, sqlalchemyURI string) error
}

// DatabaseConnectionProber executes a live connection test and returns test metadata.
type DatabaseConnectionProber interface {
	Probe(ctx context.Context, sqlalchemyURI string) (domain.TestConnectionResult, error)
}

// DatabaseTestRateLimiter checks whether a caller can run another test.
type DatabaseTestRateLimiter interface {
	Allow(ctx context.Context, key string, cap int, ttl time.Duration) (bool, error)
}

// DatabaseAuditLogger emits asynchronous audit events.
type DatabaseAuditLogger interface {
	LogDatabaseCreated(ctx context.Context, databaseID uint)
}

type defaultDatabaseConnectionTester struct{}

type defaultDatabaseConnectionProber struct{}

type defaultDatabaseTestRateLimiter struct {
	mu      sync.Mutex
	entries map[string]databaseRateLimitState
}

type databaseRateLimitState struct {
	count   int
	resetAt time.Time
}

type noopDatabaseAuditLogger struct{}

func (defaultDatabaseConnectionTester) TestConnection(_ context.Context, sqlalchemyURI string) error {
	if _, err := parseSQLAlchemyURI(sqlalchemyURI); err != nil {
		return err
	}
	return nil
}

func (noopDatabaseAuditLogger) LogDatabaseCreated(_ context.Context, _ uint) {}

func newDefaultDatabaseTestRateLimiter() *defaultDatabaseTestRateLimiter {
	return &defaultDatabaseTestRateLimiter{entries: map[string]databaseRateLimitState{}}
}

func (l *defaultDatabaseTestRateLimiter) Allow(_ context.Context, key string, cap int, ttl time.Duration) (bool, error) {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.entries[key]
	if !ok || now.After(state.resetAt) {
		l.entries[key] = databaseRateLimitState{count: 1, resetAt: now.Add(ttl)}
		return true, nil
	}

	if state.count >= cap {
		return false, nil
	}

	state.count++
	l.entries[key] = state
	return true, nil
}

func (defaultDatabaseConnectionProber) Probe(ctx context.Context, sqlalchemyURI string) (domain.TestConnectionResult, error) {
	parsedURI, err := parseSQLAlchemyURI(sqlalchemyURI)
	if err != nil {
		return domain.TestConnectionResult{}, err
	}

	driverName, driverLabel, err := resolveSQLDriver(parsedURI.Scheme)
	if err != nil {
		return domain.TestConnectionResult{}, err
	}

	startedAt := time.Now()
	db, err := sql.Open(driverName, sqlalchemyURI)
	if err != nil {
		return domain.TestConnectionResult{
			Success: false,
			Driver:  driverLabel,
			Error:   sanitizeError(err, sqlalchemyURI),
		}, nil
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return domain.TestConnectionResult{
			Success:   false,
			LatencyMS: time.Since(startedAt).Milliseconds(),
			Driver:    driverLabel,
			Error:     sanitizeError(err, sqlalchemyURI),
		}, nil
	}

	var dbVersion string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&dbVersion); err != nil {
		return domain.TestConnectionResult{
			Success:   false,
			LatencyMS: time.Since(startedAt).Milliseconds(),
			Driver:    driverLabel,
			Error:     sanitizeError(err, sqlalchemyURI),
		}, nil
	}

	return domain.TestConnectionResult{
		Success:   true,
		LatencyMS: time.Since(startedAt).Milliseconds(),
		DBVersion: dbVersion,
		Driver:    driverLabel,
	}, nil
}

func resolveSQLDriver(scheme string) (string, string, error) {
	value := strings.ToLower(strings.TrimSpace(scheme))
	switch value {
	case "postgres", "postgresql":
		return "pgx", "postgresql", nil
	default:
		return "", "", domain.ErrUnknownDatabaseDriver
	}
}

// DatabaseService handles admin database connection management.
type DatabaseService struct {
	repo          domain.DatabaseRepository
	tester        DatabaseConnectionTester
	prober        DatabaseConnectionProber
	testRateLimit DatabaseTestRateLimiter
	poolManager   DatabaseConnectionPool
	auditLogger   DatabaseAuditLogger
	encryptionKey []byte
}

func NewDatabaseService(repo domain.DatabaseRepository, tester DatabaseConnectionTester, auditLogger DatabaseAuditLogger, encryptionKey string) (*DatabaseService, error) {
	parsedKey, err := parseDatabaseEncryptionKey(encryptionKey)
	if err != nil {
		return nil, err
	}

	resolvedTester := tester
	if resolvedTester == nil {
		resolvedTester = defaultDatabaseConnectionTester{}
	}

	resolvedAuditLogger := auditLogger
	if resolvedAuditLogger == nil {
		resolvedAuditLogger = noopDatabaseAuditLogger{}
	}

	resolvedProber := DatabaseConnectionProber(defaultDatabaseConnectionProber{})
	resolvedRateLimiter := DatabaseTestRateLimiter(newDefaultDatabaseTestRateLimiter())
	resolvedPoolManager := DatabaseConnectionPool(NewConnectionPoolManager(nil, ConnectionPoolManagerConfig{}))

	return &DatabaseService{
		repo:          repo,
		tester:        resolvedTester,
		prober:        resolvedProber,
		testRateLimit: resolvedRateLimiter,
		auditLogger:   resolvedAuditLogger,
		encryptionKey: parsedKey,
		poolManager:   resolvedPoolManager,
	}, nil
}

// SetConnectionProber replaces the default probe implementation, mainly for tests.
func (s *DatabaseService) SetConnectionProber(prober DatabaseConnectionProber) {
	if prober == nil {
		return
	}
	s.prober = prober
}

// SetConnectionPool replaces the default pool manager, mainly for tests.
func (s *DatabaseService) SetConnectionPool(pool DatabaseConnectionPool) {
	if pool == nil {
		return
	}
	s.poolManager = pool
}

// ShutdownConnectionPools closes all managed pools and stops the health monitor.
func (s *DatabaseService) ShutdownConnectionPools(ctx context.Context) error {
	if s.poolManager == nil {
		return nil
	}
	return s.poolManager.Shutdown(ctx)
}

// SetTestRateLimiter replaces the default in-memory limiter, mainly for tests.
func (s *DatabaseService) SetTestRateLimiter(limiter DatabaseTestRateLimiter) {
	if limiter == nil {
		return
	}
	s.testRateLimit = limiter
}

func (s *DatabaseService) CreateDatabase(ctx context.Context, actorUserID uint, req domain.CreateDatabaseRequest) (*domain.DatabaseDetail, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	normalizedReq, strictTest, err := normalizeCreateDatabaseRequest(req)
	if err != nil {
		return nil, err
	}

	exists, err := s.repo.DatabaseNameExists(ctx, normalizedReq.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("checking duplicate database name: %w", err)
	}
	if exists {
		return nil, domain.ErrDatabaseNameExists
	}

	encryptedURI, err := encryptSQLAlchemyURIPassword(normalizedReq.SQLAlchemyURI, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	if strictTest {
		if err := s.tester.TestConnection(ctx, normalizedReq.SQLAlchemyURI); err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrDatabaseConnectionTestFailed, err)
		}
	}

	database := domain.Database{
		DatabaseName:    normalizedReq.DatabaseName,
		SQLAlchemyURI:   encryptedURI,
		AllowDML:        normalizedReq.AllowDML,
		ExposeInSQLLab:  normalizedReq.ExposeInSQLLab,
		AllowRunAsync:   normalizedReq.AllowRunAsync,
		AllowFileUpload: normalizedReq.AllowFileUpload,
		CreatedByFK:     actorUserID,
	}

	if err := s.repo.CreateDatabase(ctx, &database); err != nil {
		if errors.Is(err, domain.ErrDatabaseNameExists) {
			return nil, domain.ErrDatabaseNameExists
		}
		return nil, fmt.Errorf("creating database: %w", err)
	}

	maskedURI, err := maskSQLAlchemyURI(normalizedReq.SQLAlchemyURI)
	if err != nil {
		return nil, err
	}

	go s.auditLogger.LogDatabaseCreated(context.Background(), database.ID)

	return &domain.DatabaseDetail{
		ID:              database.ID,
		DatabaseName:    database.DatabaseName,
		SQLAlchemyURI:   maskedURI,
		Backend:         extractBackend(normalizedReq.SQLAlchemyURI),
		AllowDML:        database.AllowDML,
		ExposeInSQLLab:  database.ExposeInSQLLab,
		AllowRunAsync:   database.AllowRunAsync,
		AllowFileUpload: database.AllowFileUpload,
	}, nil
}

func (s *DatabaseService) ListDatabases(ctx context.Context, actorUserID uint, query domain.DatabaseListQuery) (*domain.DatabaseListResponse, error) {
	normalized := normalizeListQuery(query)
	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	result, err := s.repo.ListDatabases(ctx, domain.DatabaseListFilters{
		SearchQ:         normalized.SearchQ,
		Backend:         normalized.Backend,
		Offset:          (normalized.Page - 1) * normalized.PageSize,
		Limit:           normalized.PageSize,
		VisibilityScope: visibilityScope,
		ActorUserID:     actorUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing databases: %w", err)
	}

	items := make([]domain.DatabaseListItem, 0, len(result.Items))
	for _, record := range result.Items {
		maskedURI, maskErr := maskSQLAlchemyURI(record.SQLAlchemyURI)
		if maskErr != nil {
			return nil, maskErr
		}

		items = append(items, domain.DatabaseListItem{
			ID:              record.ID,
			DatabaseName:    record.DatabaseName,
			Backend:         extractBackend(record.SQLAlchemyURI),
			SQLAlchemyURI:   maskedURI,
			AllowDML:        record.AllowDML,
			ExposeInSQLLab:  record.ExposeInSQLLab,
			AllowRunAsync:   record.AllowRunAsync,
			AllowFileUpload: record.AllowFileUpload,
			DatasetCount:    record.DatasetCount,
		})
	}

	return &domain.DatabaseListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}, nil
}

func (s *DatabaseService) GetDatabase(ctx context.Context, actorUserID uint, databaseID uint) (*domain.DatabaseDetail, error) {
	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	record, err := s.repo.GetVisibleDatabaseByID(ctx, databaseID, visibilityScope, actorUserID)
	if err != nil {
		return nil, err
	}

	maskedURI, err := maskSQLAlchemyURI(record.SQLAlchemyURI)
	if err != nil {
		return nil, err
	}

	return &domain.DatabaseDetail{
		ID:              record.ID,
		DatabaseName:    record.DatabaseName,
		SQLAlchemyURI:   maskedURI,
		Backend:         extractBackend(record.SQLAlchemyURI),
		AllowDML:        record.AllowDML,
		ExposeInSQLLab:  record.ExposeInSQLLab,
		AllowRunAsync:   record.AllowRunAsync,
		AllowFileUpload: record.AllowFileUpload,
		DatasetCount:    record.DatasetCount,
	}, nil
}

func (s *DatabaseService) UpdateDatabase(ctx context.Context, actorUserID uint, databaseID uint, req domain.UpdateDatabaseRequest) (*domain.DatabaseDetail, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	if databaseID == 0 {
		return nil, domain.ErrInvalidDatabase
	}

	existing, err := s.repo.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	normalizedReq, strictTest, err := normalizeUpdateDatabaseRequest(req)
	if err != nil {
		return nil, err
	}

	updated := *existing
	if normalizedReq.DatabaseName != nil {
		updated.DatabaseName = *normalizedReq.DatabaseName
	}
	if normalizedReq.AllowDML != nil {
		updated.AllowDML = *normalizedReq.AllowDML
	}
	if normalizedReq.ExposeInSQLLab != nil {
		updated.ExposeInSQLLab = *normalizedReq.ExposeInSQLLab
	}
	if normalizedReq.AllowRunAsync != nil {
		updated.AllowRunAsync = *normalizedReq.AllowRunAsync
	}
	if normalizedReq.AllowFileUpload != nil {
		updated.AllowFileUpload = *normalizedReq.AllowFileUpload
	}

	if normalizedReq.SQLAlchemyURI != nil {
		decryptedExistingURI, decryptErr := decryptSQLAlchemyURIPassword(existing.SQLAlchemyURI, s.encryptionKey)
		if decryptErr != nil {
			return nil, decryptErr
		}

		mergedURI, mergeErr := mergeSQLAlchemyURIWithMaskedPassword(*normalizedReq.SQLAlchemyURI, decryptedExistingURI)
		if mergeErr != nil {
			return nil, mergeErr
		}

		if strictTest {
			if err := s.tester.TestConnection(ctx, mergedURI); err != nil {
				return nil, fmt.Errorf("%w: %v", domain.ErrDatabaseConnectionTestFailed, err)
			}
		}

		encryptedURI, encryptErr := encryptSQLAlchemyURIPassword(mergedURI, s.encryptionKey)
		if encryptErr != nil {
			return nil, encryptErr
		}
		updated.SQLAlchemyURI = encryptedURI
	}

	if err := s.repo.UpdateDatabase(ctx, &updated); err != nil {
		if errors.Is(err, domain.ErrDatabaseNameExists) {
			return nil, domain.ErrDatabaseNameExists
		}
		return nil, err
	}

	if s.poolManager != nil {
		if err := s.poolManager.Close(ctx, databaseID); err != nil {
			return nil, fmt.Errorf("closing database pool: %w", err)
		}
	}

	maskedURI, err := maskSQLAlchemyURI(updated.SQLAlchemyURI)
	if err != nil {
		return nil, err
	}

	return &domain.DatabaseDetail{
		ID:              updated.ID,
		DatabaseName:    updated.DatabaseName,
		SQLAlchemyURI:   maskedURI,
		Backend:         extractBackend(updated.SQLAlchemyURI),
		AllowDML:        updated.AllowDML,
		ExposeInSQLLab:  updated.ExposeInSQLLab,
		AllowRunAsync:   updated.AllowRunAsync,
		AllowFileUpload: updated.AllowFileUpload,
	}, nil
}

func (s *DatabaseService) DeleteDatabase(ctx context.Context, actorUserID uint, databaseID uint) error {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return err
	}

	if databaseID == 0 {
		return domain.ErrInvalidDatabase
	}

	if _, err := s.repo.GetDatabaseByID(ctx, databaseID); err != nil {
		return err
	}

	datasetCount, err := s.repo.CountDatasetsByDatabaseID(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("checking database dependencies: %w", err)
	}

	if datasetCount > 0 {
		return domain.ErrDatabaseInUse
	}

	if s.poolManager != nil {
		if err := s.poolManager.Close(ctx, databaseID); err != nil {
			return fmt.Errorf("closing database pool: %w", err)
		}
	}

	if err := s.repo.DeleteDatabase(ctx, databaseID); err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) TestConnection(ctx context.Context, actorUserID uint, req domain.TestDatabaseConnectionRequest, rateLimitKey string) (domain.TestConnectionResult, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return domain.TestConnectionResult{}, err
	}

	sqlalchemyURI := strings.TrimSpace(req.SQLAlchemyURI)
	if sqlalchemyURI == "" {
		return domain.TestConnectionResult{}, domain.ErrInvalidDatabaseURI
	}

	if _, err := parseSQLAlchemyURI(sqlalchemyURI); err != nil {
		return domain.TestConnectionResult{}, err
	}

	return s.runProbeWithRateLimit(ctx, sqlalchemyURI, rateLimitKey)
}

func (s *DatabaseService) TestConnectionByID(ctx context.Context, actorUserID uint, databaseID uint, rateLimitKey string) (domain.TestConnectionResult, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return domain.TestConnectionResult{}, err
	}

	database, err := s.repo.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return domain.TestConnectionResult{}, err
	}

	decryptedURI, err := decryptSQLAlchemyURIPassword(database.SQLAlchemyURI, s.encryptionKey)
	if err != nil {
		return domain.TestConnectionResult{}, err
	}

	return s.runProbeWithRateLimit(ctx, decryptedURI, rateLimitKey)
}

func (s *DatabaseService) runProbeWithRateLimit(ctx context.Context, sqlalchemyURI string, rateLimitKey string) (domain.TestConnectionResult, error) {
	key := strings.TrimSpace(rateLimitKey)
	if key == "" {
		key = "database-test:global"
	}

	allowed, err := s.testRateLimit.Allow(ctx, key, databaseTestRateLimitCap, databaseTestRateLimitWindow)
	if err != nil {
		return domain.TestConnectionResult{}, fmt.Errorf("checking test connection rate limit: %w", err)
	}
	if !allowed {
		return domain.TestConnectionResult{}, domain.ErrRateLimited
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, databaseTestConnectionTimeout)
	defer cancel()

	result, err := s.prober.Probe(timeoutCtx, sqlalchemyURI)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownDatabaseDriver) || errors.Is(err, domain.ErrInvalidDatabaseURI) {
			return domain.TestConnectionResult{}, err
		}
		return domain.TestConnectionResult{Success: false, Error: sanitizeError(err, sqlalchemyURI)}, nil
	}

	if !result.Success && result.Error == "" {
		result.Error = domain.ErrDatabaseConnectionTestFailed.Error()
	}

	return result, nil
}

func (s *DatabaseService) ensureAdmin(ctx context.Context, actorUserID uint) error {
	isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	if err != nil {
		return fmt.Errorf("checking admin role: %w", err)
	}
	if !isAdmin {
		return domain.ErrForbidden
	}
	return nil
}

func (s *DatabaseService) resolveVisibilityScope(ctx context.Context, actorUserID uint) (domain.DatabaseVisibilityScope, error) {
	roleNames, err := s.repo.GetRoleNamesByUser(ctx, actorUserID)
	if err != nil {
		return "", fmt.Errorf("loading actor role names: %w", err)
	}

	for _, roleName := range roleNames {
		value := strings.ToLower(strings.TrimSpace(roleName))
		if value == "admin" {
			return domain.DatabaseVisibilityAdmin, nil
		}
	}

	for _, roleName := range roleNames {
		value := strings.ToLower(strings.TrimSpace(roleName))
		if value == "alpha" {
			return domain.DatabaseVisibilityAlpha, nil
		}
	}

	return domain.DatabaseVisibilityGamma, nil
}

func normalizeListQuery(query domain.DatabaseListQuery) domain.DatabaseListQuery {
	page := query.Page
	if page < 1 {
		page = databaseListDefaultPage
	}

	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = databaseListDefaultPageSize
	}
	if pageSize > databaseListMaxPageSize {
		pageSize = databaseListMaxPageSize
	}

	return domain.DatabaseListQuery{
		SearchQ:  strings.TrimSpace(query.SearchQ),
		Backend:  strings.ToLower(strings.TrimSpace(query.Backend)),
		Page:     page,
		PageSize: pageSize,
	}
}

func extractBackend(sqlalchemyURI string) string {
	parsedURI, err := url.Parse(sqlalchemyURI)
	if err != nil {
		return "unknown"
	}
	if parsedURI.Scheme == "" {
		return "unknown"
	}
	return strings.ToLower(strings.TrimSpace(parsedURI.Scheme))
}

func normalizeCreateDatabaseRequest(req domain.CreateDatabaseRequest) (domain.CreateDatabaseRequest, bool, error) {
	databaseName := strings.TrimSpace(req.DatabaseName)
	sqlalchemyURI := strings.TrimSpace(req.SQLAlchemyURI)
	if databaseName == "" || sqlalchemyURI == "" {
		return domain.CreateDatabaseRequest{}, false, domain.ErrInvalidDatabase
	}

	if _, err := parseSQLAlchemyURI(sqlalchemyURI); err != nil {
		return domain.CreateDatabaseRequest{}, false, err
	}

	strictTest := true
	if req.StrictTest != nil {
		strictTest = *req.StrictTest
	}

	return domain.CreateDatabaseRequest{
		DatabaseName:    databaseName,
		SQLAlchemyURI:   sqlalchemyURI,
		AllowDML:        req.AllowDML,
		ExposeInSQLLab:  req.ExposeInSQLLab,
		AllowRunAsync:   req.AllowRunAsync,
		AllowFileUpload: req.AllowFileUpload,
		StrictTest:      req.StrictTest,
	}, strictTest, nil
}

func normalizeUpdateDatabaseRequest(req domain.UpdateDatabaseRequest) (domain.UpdateDatabaseRequest, bool, error) {
	strictTest := true
	if req.StrictTest != nil {
		strictTest = *req.StrictTest
	}

	normalized := req
	if req.DatabaseName != nil {
		databaseName := strings.TrimSpace(*req.DatabaseName)
		if databaseName == "" {
			return domain.UpdateDatabaseRequest{}, false, domain.ErrInvalidDatabase
		}
		normalized.DatabaseName = &databaseName
	}

	if req.SQLAlchemyURI != nil {
		sqlalchemyURI := strings.TrimSpace(*req.SQLAlchemyURI)
		if sqlalchemyURI == "" {
			return domain.UpdateDatabaseRequest{}, false, domain.ErrInvalidDatabaseURI
		}
		if _, err := parseSQLAlchemyURI(sqlalchemyURI); err != nil {
			return domain.UpdateDatabaseRequest{}, false, err
		}
		normalized.SQLAlchemyURI = &sqlalchemyURI
	}

	return normalized, strictTest, nil
}

func mergeSQLAlchemyURIWithMaskedPassword(nextURI string, existingURI string) (string, error) {
	nextParsedURI, err := parseSQLAlchemyURI(nextURI)
	if err != nil {
		return "", err
	}

	if nextParsedURI.User == nil {
		return nextParsedURI.String(), nil
	}

	password, hasPassword := nextParsedURI.User.Password()
	if !hasPassword || password != "***" {
		return nextParsedURI.String(), nil
	}

	existingParsedURI, err := parseSQLAlchemyURI(existingURI)
	if err != nil {
		return "", err
	}

	if existingParsedURI.User == nil {
		return "", domain.ErrInvalidDatabaseURI
	}

	existingPassword, hasExistingPassword := existingParsedURI.User.Password()
	if !hasExistingPassword || existingPassword == "" {
		return "", domain.ErrInvalidDatabaseURI
	}

	nextParsedURI.User = url.UserPassword(nextParsedURI.User.Username(), existingPassword)
	return nextParsedURI.String(), nil
}

func parseDatabaseEncryptionKey(rawKey string) ([]byte, error) {
	trimmed := strings.TrimSpace(rawKey)
	if trimmed == "" {
		return nil, domain.ErrDatabaseCredentialEncryption
	}

	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	if len(trimmed) == 32 {
		return []byte(trimmed), nil
	}

	return nil, domain.ErrDatabaseCredentialEncryption
}

func decryptField(encryptedText string, encryptionKey []byte) (string, error) {
	combined, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize {
		return "", errors.New("invalid ciphertext")
	}

	nonce := combined[:nonceSize]
	cipherText := combined[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

func parseSQLAlchemyURI(sqlalchemyURI string) (*url.URL, error) {
	parsedURI, err := url.Parse(sqlalchemyURI)
	if err != nil {
		return nil, domain.ErrInvalidDatabaseURI
	}
	if parsedURI.Scheme == "" || parsedURI.Host == "" {
		return nil, domain.ErrInvalidDatabaseURI
	}
	return parsedURI, nil
}

func decryptSQLAlchemyURIPassword(sqlalchemyURI string, encryptionKey []byte) (string, error) {
	parsedURI, err := parseSQLAlchemyURI(sqlalchemyURI)
	if err != nil {
		return "", err
	}

	if parsedURI.User == nil {
		return parsedURI.String(), nil
	}

	username := parsedURI.User.Username()
	encryptedPassword, hasPassword := parsedURI.User.Password()
	if !hasPassword || encryptedPassword == "" {
		return parsedURI.String(), nil
	}

	plainPassword, err := decryptField(encryptedPassword, encryptionKey)
	if err != nil {
		return "", domain.ErrDatabaseCredentialEncryption
	}

	parsedURI.User = url.UserPassword(username, plainPassword)
	return parsedURI.String(), nil
}

func encryptSQLAlchemyURIPassword(sqlalchemyURI string, encryptionKey []byte) (string, error) {
	parsedURI, err := parseSQLAlchemyURI(sqlalchemyURI)
	if err != nil {
		return "", err
	}

	if parsedURI.User == nil {
		return parsedURI.String(), nil
	}

	username := parsedURI.User.Username()
	password, hasPassword := parsedURI.User.Password()
	if !hasPassword || password == "" {
		return parsedURI.String(), nil
	}

	encryptedPassword, err := encryptField(password, encryptionKey)
	if err != nil {
		return "", domain.ErrDatabaseCredentialEncryption
	}

	parsedURI.User = url.UserPassword(username, encryptedPassword)
	return parsedURI.String(), nil
}

func maskSQLAlchemyURI(sqlalchemyURI string) (string, error) {
	parsedURI, err := parseSQLAlchemyURI(sqlalchemyURI)
	if err != nil {
		return "", err
	}

	if parsedURI.User == nil {
		return parsedURI.String(), nil
	}

	username := parsedURI.User.Username()
	_, hasPassword := parsedURI.User.Password()
	if !hasPassword {
		return parsedURI.String(), nil
	}

	parsedURI.User = url.UserPassword(username, "***")
	maskedURI := parsedURI.String()
	return strings.Replace(maskedURI, "%2A%2A%2A", "***", 1), nil
}

func sanitizeError(err error, sqlalchemyURI string) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	parsedURI, parseErr := url.Parse(sqlalchemyURI)
	if parseErr == nil && parsedURI != nil && parsedURI.User != nil {
		username := parsedURI.User.Username()
		password, hasPassword := parsedURI.User.Password()
		if hasPassword && password != "" {
			message = strings.ReplaceAll(message, username+":"+password+"@", username+":***@")
		}

		maskedURI, maskErr := maskSQLAlchemyURI(sqlalchemyURI)
		if maskErr == nil {
			message = strings.ReplaceAll(message, sqlalchemyURI, maskedURI)
		}
	}

	return message
}

// EncryptSQLAlchemyURIPasswordForTest exposes URI password encryption for black-box tests.
func EncryptSQLAlchemyURIPasswordForTest(sqlalchemyURI string, rawKey string) (string, error) {
	key, err := parseDatabaseEncryptionKey(rawKey)
	if err != nil {
		return "", err
	}
	return encryptSQLAlchemyURIPassword(sqlalchemyURI, key)
}

func encryptField(plainText string, encryptionKey []byte) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nil, nonce, []byte(plainText), nil)
	combined := append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(combined), nil
}
