package auth_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/db"
	domain "superset/auth-service/internal/domain/db"
)

type fakeDatabaseRepo struct {
	isAdmin           bool
	databaseNameTaken bool
	createErr         error
	updateErr         error
	deleteErr         error
	getByIDResult     *domain.Database
	getByIDErr        error
	datasetCount      int64
	roleNames         []string
	listResult        domain.DatabaseListResult
	listErr           error
	visibleByIDResult *domain.DatabaseWithDatasetCount
	visibleByIDErr    error

	created *domain.Database
	updated *domain.Database
	deleted uint
}

func (f *fakeDatabaseRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	return f.isAdmin, nil
}

func (f *fakeDatabaseRepo) DatabaseNameExists(_ context.Context, _ string) (bool, error) {
	return f.databaseNameTaken, nil
}

func (f *fakeDatabaseRepo) CreateDatabase(_ context.Context, db *domain.Database) error {
	if db.ID == 0 {
		db.ID = 401
	}
	copyValue := *db
	f.created = &copyValue
	return f.createErr
}

func (f *fakeDatabaseRepo) UpdateDatabase(_ context.Context, db *domain.Database) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	copyValue := *db
	f.updated = &copyValue
	if f.getByIDResult != nil && f.getByIDResult.ID == db.ID {
		current := copyValue
		f.getByIDResult = &current
	}
	return nil
}

func (f *fakeDatabaseRepo) DeleteDatabase(_ context.Context, databaseID uint) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = databaseID
	return nil
}

func (f *fakeDatabaseRepo) CountDatasetsByDatabaseID(_ context.Context, _ uint) (int64, error) {
	return f.datasetCount, nil
}

func (f *fakeDatabaseRepo) GetDatabaseByID(_ context.Context, _ uint) (*domain.Database, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	if f.getByIDResult == nil {
		return nil, domain.ErrDatabaseNotFound
	}
	copyValue := *f.getByIDResult
	return &copyValue, nil
}

func (f *fakeDatabaseRepo) GetRoleNamesByUser(_ context.Context, _ uint) ([]string, error) {
	return append([]string(nil), f.roleNames...), nil
}

func (f *fakeDatabaseRepo) ListDatabases(_ context.Context, _ domain.DatabaseListFilters) (domain.DatabaseListResult, error) {
	if f.listErr != nil {
		return domain.DatabaseListResult{}, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeDatabaseRepo) GetVisibleDatabaseByID(_ context.Context, _ uint, _ domain.DatabaseVisibilityScope, _ uint) (*domain.DatabaseWithDatasetCount, error) {
	if f.visibleByIDErr != nil {
		return nil, f.visibleByIDErr
	}
	if f.visibleByIDResult == nil {
		return nil, domain.ErrDatabaseNotFound
	}
	copyValue := *f.visibleByIDResult
	return &copyValue, nil
}

type fakeDatabaseTester struct {
	err     error
	called  int
	lastURI string
}

func (f *fakeDatabaseTester) TestConnection(_ context.Context, sqlalchemyURI string) error {
	f.called++
	f.lastURI = sqlalchemyURI
	return f.err
}

type fakeDatabaseAuditLogger struct {
	called int
	lastID uint
}

type fakeConnectionProbe struct {
	result  domain.TestConnectionResult
	err     error
	called  int
	lastURI string
}

func (f *fakeConnectionProbe) Probe(_ context.Context, sqlalchemyURI string) (domain.TestConnectionResult, error) {
	f.called++
	f.lastURI = sqlalchemyURI
	return f.result, f.err
}

type fakeTestRateLimiter struct {
	allow bool
	err   error

	called  int
	lastKey string
	lastCap int
	lastTTL time.Duration
}

func (f *fakeTestRateLimiter) Allow(_ context.Context, key string, cap int, ttl time.Duration) (bool, error) {
	f.called++
	f.lastKey = key
	f.lastCap = cap
	f.lastTTL = ttl
	if f.err != nil {
		return false, f.err
	}
	return f.allow, nil
}

type fakeConnectionPool struct {
	closeErr      error
	shutdownErr   error
	closeCalled   int
	shutdownCalls int
	lastClosedID  uint
}

type fakeSchemaInspector struct {
	schemas      []string
	tables       []domain.DatabaseTable
	tablesTotal  int64
	columns      []domain.DatabaseColumn
	schemasErr   error
	tablesErr    error
	columnsErr   error
	schemasCalls int
	tablesCalls  int
	columnsCalls int
}

func (f *fakeSchemaInspector) ListSchemas(_ context.Context, _ svcauth.SQLConnection) ([]string, error) {
	f.schemasCalls++
	if f.schemasErr != nil {
		return nil, f.schemasErr
	}
	return append([]string(nil), f.schemas...), nil
}

func (f *fakeSchemaInspector) ListTables(_ context.Context, _ svcauth.SQLConnection, _ string, _ int, _ int) ([]domain.DatabaseTable, int64, error) {
	f.tablesCalls++
	if f.tablesErr != nil {
		return nil, 0, f.tablesErr
	}
	return append([]domain.DatabaseTable(nil), f.tables...), f.tablesTotal, nil
}

func (f *fakeSchemaInspector) ListColumns(_ context.Context, _ svcauth.SQLConnection, _ string, _ string) ([]domain.DatabaseColumn, error) {
	f.columnsCalls++
	if f.columnsErr != nil {
		return nil, f.columnsErr
	}
	return append([]domain.DatabaseColumn(nil), f.columns...), nil
}

type fakeSchemaCache struct {
	store map[string]string
}

func (f *fakeSchemaCache) Get(_ context.Context, key string) (string, bool, error) {
	if f.store == nil {
		return "", false, nil
	}
	value, ok := f.store[key]
	if !ok {
		return "", false, nil
	}
	return value, true, nil
}

func (f *fakeSchemaCache) Set(_ context.Context, key string, value string, _ time.Duration) error {
	if f.store == nil {
		f.store = map[string]string{}
	}
	f.store[key] = value
	return nil
}

func (f *fakeConnectionPool) Get(_ context.Context, _ uint, _ string) (svcauth.SQLConnection, error) {
	return nil, nil
}

func (f *fakeConnectionPool) Close(_ context.Context, databaseID uint) error {
	f.closeCalled++
	f.lastClosedID = databaseID
	return f.closeErr
}

func (f *fakeConnectionPool) Shutdown(_ context.Context) error {
	f.shutdownCalls++
	return f.shutdownErr
}

func (f *fakeDatabaseAuditLogger) LogDatabaseCreated(_ context.Context, databaseID uint) {
	f.called++
	f.lastID = databaseID
}

func TestDatabaseService_CreateDatabaseEncryptsAndMasksURI(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true}
	tester := &fakeDatabaseTester{}
	audit := &fakeDatabaseAuditLogger{}
	svc, err := svcauth.NewDatabaseService(repo, tester, audit, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	created, err := svc.CreateDatabase(context.Background(), 1, domain.CreateDatabaseRequest{
		DatabaseName:   "analytics",
		SQLAlchemyURI:  "postgresql://superset:secret-pass@localhost:5432/analytics",
		AllowDML:       true,
		ExposeInSQLLab: true,
		AllowRunAsync:  true,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if tester.called != 1 {
		t.Fatalf("expected tester to be called once, got %d", tester.called)
	}
	if !strings.Contains(created.SQLAlchemyURI, "***") {
		t.Fatalf("expected masked sqlalchemy uri in response, got %s", created.SQLAlchemyURI)
	}
	if strings.Contains(created.SQLAlchemyURI, "secret-pass") {
		t.Fatalf("response leaked plaintext password: %s", created.SQLAlchemyURI)
	}
	if repo.created == nil {
		t.Fatal("expected repository CreateDatabase call")
	}
	if strings.Contains(repo.created.SQLAlchemyURI, "secret-pass") {
		t.Fatalf("stored sqlalchemy uri leaked plaintext password: %s", repo.created.SQLAlchemyURI)
	}
	if created.ID == 0 {
		t.Fatalf("expected created id to be assigned, got %d", created.ID)
	}
	if created.DatabaseName != "analytics" {
		t.Fatalf("expected database_name analytics, got %s", created.DatabaseName)
	}
}

func TestDatabaseService_CreateDatabaseDuplicateNameReturnsConflict(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true, databaseNameTaken: true}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	_, createErr := svc.CreateDatabase(context.Background(), 1, domain.CreateDatabaseRequest{
		DatabaseName:  "analytics",
		SQLAlchemyURI: "postgresql://superset:secret-pass@localhost:5432/analytics",
	})
	if !errors.Is(createErr, domain.ErrDatabaseNameExists) {
		t.Fatalf("expected ErrDatabaseNameExists, got %v", createErr)
	}
}

func TestDatabaseService_CreateDatabaseStrictTestFailureReturns422Error(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true}
	tester := &fakeDatabaseTester{err: errors.New("dial tcp timeout")}
	svc, err := svcauth.NewDatabaseService(repo, tester, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	_, createErr := svc.CreateDatabase(context.Background(), 1, domain.CreateDatabaseRequest{
		DatabaseName:  "analytics",
		SQLAlchemyURI: "postgresql://superset:secret-pass@localhost:5432/analytics",
		StrictTest:    boolPtr(true),
	})
	if !errors.Is(createErr, domain.ErrDatabaseConnectionTestFailed) {
		t.Fatalf("expected ErrDatabaseConnectionTestFailed, got %v", createErr)
	}
}

func TestDatabaseService_CreateDatabaseSkipsTestWhenStrictTestFalse(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true}
	tester := &fakeDatabaseTester{err: errors.New("dial tcp timeout")}
	svc, err := svcauth.NewDatabaseService(repo, tester, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	_, createErr := svc.CreateDatabase(context.Background(), 1, domain.CreateDatabaseRequest{
		DatabaseName:  "analytics",
		SQLAlchemyURI: "postgresql://superset:secret-pass@localhost:5432/analytics",
		StrictTest:    boolPtr(false),
	})
	if createErr != nil {
		t.Fatalf("expected nil error when strict_test=false, got %v", createErr)
	}
}

func TestNewDatabaseService_RejectsInvalidEncryptionKey(t *testing.T) {
	_, err := svcauth.NewDatabaseService(&fakeDatabaseRepo{isAdmin: true}, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "short-key")
	if !errors.Is(err, domain.ErrDatabaseCredentialEncryption) {
		t.Fatalf("expected ErrDatabaseCredentialEncryption, got %v", err)
	}
}

func TestDatabaseService_CreateDatabaseNonAdminReturnsForbidden(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: false}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	_, createErr := svc.CreateDatabase(context.Background(), 77, domain.CreateDatabaseRequest{
		DatabaseName:  "analytics",
		SQLAlchemyURI: "postgresql://superset:secret-pass@localhost:5432/analytics",
	})
	if !errors.Is(createErr, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", createErr)
	}
}

func TestDatabaseService_TestConnectionReturnsSuccessResult(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	probe := &fakeConnectionProbe{result: domain.TestConnectionResult{Success: true, LatencyMS: 42, DBVersion: "PostgreSQL 15.4", Driver: "pgx"}}
	limiter := &fakeTestRateLimiter{allow: true}
	svc.SetConnectionProber(probe)
	svc.SetTestRateLimiter(limiter)

	result, testErr := svc.TestConnection(context.Background(), 1, domain.TestDatabaseConnectionRequest{
		SQLAlchemyURI: "postgresql://alice:secret@localhost:5432/analytics",
	}, "user:1:ip:127.0.0.1")
	if testErr != nil {
		t.Fatalf("expected nil error, got %v", testErr)
	}
	if !result.Success {
		t.Fatalf("expected success=true, got %+v", result)
	}
	if result.LatencyMS != 42 {
		t.Fatalf("expected latency 42, got %d", result.LatencyMS)
	}
	if probe.called != 1 {
		t.Fatalf("expected prober call once, got %d", probe.called)
	}
}

func TestDatabaseService_TestConnectionBadCredentialsReturnsSuccessFalse(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	probe := &fakeConnectionProbe{result: domain.TestConnectionResult{Success: false, Driver: "pgx", Error: "password authentication failed"}}
	limiter := &fakeTestRateLimiter{allow: true}
	svc.SetConnectionProber(probe)
	svc.SetTestRateLimiter(limiter)

	result, testErr := svc.TestConnection(context.Background(), 1, domain.TestDatabaseConnectionRequest{
		SQLAlchemyURI: "postgresql://alice:secret@localhost:5432/analytics",
	}, "user:1:ip:127.0.0.1")
	if testErr != nil {
		t.Fatalf("expected nil error, got %v", testErr)
	}
	if result.Success {
		t.Fatalf("expected success=false, got %+v", result)
	}
	if result.Error == "" {
		t.Fatal("expected error message in result")
	}
}

func TestDatabaseService_TestConnectionUnknownDriverReturns422Error(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	probe := &fakeConnectionProbe{err: domain.ErrUnknownDatabaseDriver}
	limiter := &fakeTestRateLimiter{allow: true}
	svc.SetConnectionProber(probe)
	svc.SetTestRateLimiter(limiter)

	_, testErr := svc.TestConnection(context.Background(), 1, domain.TestDatabaseConnectionRequest{
		SQLAlchemyURI: "snowflake://account/warehouse",
	}, "user:1:ip:127.0.0.1")
	if !errors.Is(testErr, domain.ErrUnknownDatabaseDriver) {
		t.Fatalf("expected ErrUnknownDatabaseDriver, got %v", testErr)
	}
}

func TestDatabaseService_TestConnectionRateLimitedReturns429Error(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	probe := &fakeConnectionProbe{result: domain.TestConnectionResult{Success: true, Driver: "pgx"}}
	limiter := &fakeTestRateLimiter{allow: false}
	svc.SetConnectionProber(probe)
	svc.SetTestRateLimiter(limiter)

	_, testErr := svc.TestConnection(context.Background(), 1, domain.TestDatabaseConnectionRequest{
		SQLAlchemyURI: "postgresql://alice:secret@localhost:5432/analytics",
	}, "user:1:ip:127.0.0.1")
	if !errors.Is(testErr, domain.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", testErr)
	}
}

func TestDatabaseService_TestConnectionByIDDecryptsAndProbes(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}
	repo.getByIDResult = &domain.Database{ID: 7, SQLAlchemyURI: encryptedURI}

	probe := &fakeConnectionProbe{result: domain.TestConnectionResult{Success: true, LatencyMS: 17, DBVersion: "PostgreSQL 15.4", Driver: "pgx"}}
	limiter := &fakeTestRateLimiter{allow: true}
	svc.SetConnectionProber(probe)
	svc.SetTestRateLimiter(limiter)

	result, testErr := svc.TestConnectionByID(context.Background(), 1, 7, "user:1:ip:127.0.0.1")
	if testErr != nil {
		t.Fatalf("expected nil error, got %v", testErr)
	}
	if !result.Success {
		t.Fatalf("expected success=true, got %+v", result)
	}
	if !strings.Contains(probe.lastURI, "secret") {
		t.Fatalf("expected decrypted password in probe URI, got %s", probe.lastURI)
	}
}

func TestDatabaseService_ListDatabasesAppliesGammaVisibilityAndMasksURI(t *testing.T) {
	repo := &fakeDatabaseRepo{
		roleNames: []string{"Gamma"},
		listResult: domain.DatabaseListResult{
			Items: []domain.DatabaseWithDatasetCount{{
				Database: domain.Database{
					ID:             10,
					DatabaseName:   "analytics",
					SQLAlchemyURI:  "postgresql://alice:secret@localhost:5432/analytics",
					ExposeInSQLLab: true,
				},
				DatasetCount: 3,
			}},
			Total: 1,
		},
	}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	result, listErr := svc.ListDatabases(context.Background(), 9, domain.DatabaseListQuery{Page: 1, PageSize: 20})
	if listErr != nil {
		t.Fatalf("expected nil error, got %v", listErr)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one database, got %d", len(result.Items))
	}
	if !strings.Contains(result.Items[0].SQLAlchemyURI, "***") {
		t.Fatalf("expected masked URI, got %s", result.Items[0].SQLAlchemyURI)
	}
	if result.Items[0].Backend != "postgresql" {
		t.Fatalf("expected backend postgresql, got %s", result.Items[0].Backend)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

func TestDatabaseService_GetDatabaseReturnsDatasetCountAndMaskedURI(t *testing.T) {
	repo := &fakeDatabaseRepo{
		roleNames: []string{"Admin"},
		visibleByIDResult: &domain.DatabaseWithDatasetCount{
			Database: domain.Database{
				ID:             22,
				DatabaseName:   "warehouse",
				SQLAlchemyURI:  "postgresql://superset:secret@localhost:5432/warehouse",
				ExposeInSQLLab: true,
			},
			DatasetCount: 12,
		},
	}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	detail, getErr := svc.GetDatabase(context.Background(), 1, 22)
	if getErr != nil {
		t.Fatalf("expected nil error, got %v", getErr)
	}
	if detail.DatasetCount != 12 {
		t.Fatalf("expected dataset_count 12, got %d", detail.DatasetCount)
	}
	if !strings.Contains(detail.SQLAlchemyURI, "***") {
		t.Fatalf("expected masked URI, got %s", detail.SQLAlchemyURI)
	}
}

func TestDatabaseService_UpdateDatabaseMergesMaskedPasswordAndReturnsMaskedURI(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 7, DatabaseName: "analytics", SQLAlchemyURI: encryptedURI}}
	tester := &fakeDatabaseTester{}
	pool := &fakeConnectionPool{}
	svc, err := svcauth.NewDatabaseService(repo, tester, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}
	svc.SetConnectionPool(pool)

	name := "analytics-updated"
	uri := "postgresql://alice:***@localhost:5432/analytics"
	updated, updateErr := svc.UpdateDatabase(context.Background(), 1, 7, domain.UpdateDatabaseRequest{
		DatabaseName:  &name,
		SQLAlchemyURI: &uri,
	})
	if updateErr != nil {
		t.Fatalf("expected nil error, got %v", updateErr)
	}
	if tester.called != 1 {
		t.Fatalf("expected tester to be called once, got %d", tester.called)
	}
	if repo.updated == nil {
		t.Fatal("expected updated database in repository")
	}
	if !strings.Contains(updated.SQLAlchemyURI, "***") {
		t.Fatalf("expected masked uri in response, got %s", updated.SQLAlchemyURI)
	}
	if repo.updated.DatabaseName != "analytics-updated" {
		t.Fatalf("expected updated name, got %s", repo.updated.DatabaseName)
	}
	if pool.closeCalled != 1 {
		t.Fatalf("expected close called once, got %d", pool.closeCalled)
	}
	if pool.lastClosedID != 7 {
		t.Fatalf("expected close id 7, got %d", pool.lastClosedID)
	}
}

func TestDatabaseService_DeleteDatabaseReturnsInUseWhenDatasetsExist(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: "postgresql://alice:enc@localhost:5432/analytics"}, datasetCount: 2}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	deleteErr := svc.DeleteDatabase(context.Background(), 1, 11)
	if !errors.Is(deleteErr, domain.ErrDatabaseInUse) {
		t.Fatalf("expected ErrDatabaseInUse, got %v", deleteErr)
	}
}

func TestDatabaseService_DeleteDatabaseDeletesWhenUnused(t *testing.T) {
	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: "postgresql://alice:enc@localhost:5432/analytics"}, datasetCount: 0}
	pool := &fakeConnectionPool{}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}
	svc.SetConnectionPool(pool)

	deleteErr := svc.DeleteDatabase(context.Background(), 1, 11)
	if deleteErr != nil {
		t.Fatalf("expected nil error, got %v", deleteErr)
	}
	if repo.deleted != 11 {
		t.Fatalf("expected deleted id 11, got %d", repo.deleted)
	}
	if pool.closeCalled != 1 {
		t.Fatalf("expected close called once, got %d", pool.closeCalled)
	}
	if pool.lastClosedID != 11 {
		t.Fatalf("expected close id 11, got %d", pool.lastClosedID)
	}
}

func TestDatabaseService_ShutdownConnectionPoolsDelegatesToManager(t *testing.T) {
	pool := &fakeConnectionPool{}
	svc, err := svcauth.NewDatabaseService(&fakeDatabaseRepo{isAdmin: true}, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}
	svc.SetConnectionPool(pool)

	err = svc.ShutdownConnectionPools(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pool.shutdownCalls != 1 {
		t.Fatalf("expected shutdown calls 1, got %d", pool.shutdownCalls)
	}
}

func TestDatabaseService_ListSchemasUsesCacheOnSecondRequest(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 9, SQLAlchemyURI: encryptedURI}}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	inspector := &fakeSchemaInspector{schemas: []string{"public", "analytics"}}
	svc.SetSchemaInspector(inspector)
	svc.SetSchemaCache(&fakeSchemaCache{store: map[string]string{}})
	svc.SetConnectionPool(&fakeConnectionPool{})

	first, firstErr := svc.ListSchemas(context.Background(), 1, 9, false, "")
	if firstErr != nil {
		t.Fatalf("expected nil error, got %v", firstErr)
	}

	second, secondErr := svc.ListSchemas(context.Background(), 1, 9, false, "")
	if secondErr != nil {
		t.Fatalf("expected nil error, got %v", secondErr)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected same schemas from cache, got first=%v second=%v", first, second)
	}
	if inspector.schemasCalls != 1 {
		t.Fatalf("expected inspector called once due cache hit, got %d", inspector.schemasCalls)
	}
}

func TestDatabaseService_ListTablesForceRefreshBypassesCache(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 10, SQLAlchemyURI: encryptedURI}}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	inspector := &fakeSchemaInspector{
		tables:      []domain.DatabaseTable{{Name: "orders"}},
		tablesTotal: 1,
	}
	limiter := &fakeTestRateLimiter{allow: true}

	svc.SetSchemaInspector(inspector)
	svc.SetSchemaCache(&fakeSchemaCache{store: map[string]string{}})
	svc.SetConnectionPool(&fakeConnectionPool{})
	svc.SetTestRateLimiter(limiter)

	first, firstErr := svc.ListTables(context.Background(), 1, 10, domain.ListDatabaseTablesRequest{Schema: "public", Page: 1, PageSize: 10}, false, "")
	if firstErr != nil {
		t.Fatalf("expected nil error, got %v", firstErr)
	}
	if len(first.Items) != 1 || first.Items[0].Name != "orders" {
		t.Fatalf("unexpected first table result: %+v", first.Items)
	}

	inspector.tables = []domain.DatabaseTable{{Name: "customers"}}

	cached, cachedErr := svc.ListTables(context.Background(), 1, 10, domain.ListDatabaseTablesRequest{Schema: "public", Page: 1, PageSize: 10}, false, "")
	if cachedErr != nil {
		t.Fatalf("expected nil error, got %v", cachedErr)
	}
	if len(cached.Items) != 1 || cached.Items[0].Name != "orders" {
		t.Fatalf("expected cached table result, got %+v", cached.Items)
	}

	refreshed, refreshErr := svc.ListTables(context.Background(), 1, 10, domain.ListDatabaseTablesRequest{Schema: "public", Page: 1, PageSize: 10}, true, "schema-refresh")
	if refreshErr != nil {
		t.Fatalf("expected nil error, got %v", refreshErr)
	}
	if len(refreshed.Items) != 1 || refreshed.Items[0].Name != "customers" {
		t.Fatalf("expected force refresh to bypass cache, got %+v", refreshed.Items)
	}
	if inspector.tablesCalls != 2 {
		t.Fatalf("expected inspector called twice, got %d", inspector.tablesCalls)
	}
	if limiter.called != 1 {
		t.Fatalf("expected limiter called once for force refresh, got %d", limiter.called)
	}
}

func TestDatabaseService_ListColumnsForceRefreshRateLimited(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: encryptedURI}}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	inspector := &fakeSchemaInspector{columns: []domain.DatabaseColumn{{Name: "created_at", DataType: "timestamp", IsDttm: true}}}
	limiter := &fakeTestRateLimiter{allow: false}

	svc.SetSchemaInspector(inspector)
	svc.SetConnectionPool(&fakeConnectionPool{})
	svc.SetTestRateLimiter(limiter)

	_, listErr := svc.ListColumns(context.Background(), 1, 11, domain.ListDatabaseColumnsRequest{Schema: "public", Table: "orders"}, true, "schema-refresh")
	if !errors.Is(listErr, domain.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", listErr)
	}
	if inspector.columnsCalls != 0 {
		t.Fatalf("expected inspector not called when rate-limited, got %d", inspector.columnsCalls)
	}
}

func TestDatabaseService_ListColumnsMapsTimeoutToGatewayTimeout(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 12, SQLAlchemyURI: encryptedURI}}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	inspector := &fakeSchemaInspector{columnsErr: context.DeadlineExceeded}
	svc.SetSchemaInspector(inspector)
	svc.SetConnectionPool(&fakeConnectionPool{})

	_, listErr := svc.ListColumns(context.Background(), 1, 12, domain.ListDatabaseColumnsRequest{Schema: "public", Table: "orders"}, false, "")
	if !errors.Is(listErr, domain.ErrDatabaseTimeout) {
		t.Fatalf("expected ErrDatabaseTimeout, got %v", listErr)
	}
}

func TestDatabaseService_ListColumnsMapsConnectionErrorsToBadGateway(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 13, SQLAlchemyURI: encryptedURI}}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	inspector := &fakeSchemaInspector{columnsErr: errors.New("connection reset by peer")}
	svc.SetSchemaInspector(inspector)
	svc.SetConnectionPool(&fakeConnectionPool{})

	_, listErr := svc.ListColumns(context.Background(), 1, 13, domain.ListDatabaseColumnsRequest{Schema: "public", Table: "orders"}, false, "")
	if !errors.Is(listErr, domain.ErrDatabaseUnreachable) {
		t.Fatalf("expected ErrDatabaseUnreachable, got %v", listErr)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
