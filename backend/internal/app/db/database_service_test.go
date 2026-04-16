package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	svcauth "superset/auth-service/internal/app/db"
	domain "superset/auth-service/internal/domain/db"
)

type fakeDatabaseRepo struct {
	isAdmin           bool
	databaseNameTaken bool
	createErr         error

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

func boolPtr(value bool) *bool {
	return &value
}
