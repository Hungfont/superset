package auth_test

import (
	"context"
	"errors"
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
	getByIDResult     *domain.Database
	getByIDErr        error

	created *domain.Database
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

func boolPtr(value bool) *bool {
	return &value
}
