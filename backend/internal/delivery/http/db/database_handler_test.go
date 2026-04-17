package auth_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/db"
	httpauth "superset/auth-service/internal/delivery/http/db"
	"superset/auth-service/internal/delivery/http/middleware"
	domainauth "superset/auth-service/internal/domain/auth"
	domain "superset/auth-service/internal/domain/db"

	"github.com/gin-gonic/gin"
)

type handlerDatabaseRepo struct {
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
	updated           *domain.Database
	deletedID         uint
}

func (h *handlerDatabaseRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	return h.isAdmin, nil
}

func (h *handlerDatabaseRepo) DatabaseNameExists(_ context.Context, _ string) (bool, error) {
	return h.databaseNameTaken, nil
}

func (h *handlerDatabaseRepo) CreateDatabase(_ context.Context, db *domain.Database) error {
	if h.createErr != nil {
		return h.createErr
	}
	if db.ID == 0 {
		db.ID = 301
	}
	return nil
}

func (h *handlerDatabaseRepo) UpdateDatabase(_ context.Context, db *domain.Database) error {
	if h.updateErr != nil {
		return h.updateErr
	}
	copyValue := *db
	h.updated = &copyValue
	if h.getByIDResult != nil && h.getByIDResult.ID == db.ID {
		updated := copyValue
		h.getByIDResult = &updated
	}
	return nil
}

func (h *handlerDatabaseRepo) DeleteDatabase(_ context.Context, databaseID uint) error {
	if h.deleteErr != nil {
		return h.deleteErr
	}
	h.deletedID = databaseID
	return nil
}

func (h *handlerDatabaseRepo) CountDatasetsByDatabaseID(_ context.Context, _ uint) (int64, error) {
	return h.datasetCount, nil
}

func (h *handlerDatabaseRepo) GetDatabaseByID(_ context.Context, _ uint) (*domain.Database, error) {
	if h.getByIDErr != nil {
		return nil, h.getByIDErr
	}
	if h.getByIDResult == nil {
		return nil, domain.ErrDatabaseNotFound
	}
	copyValue := *h.getByIDResult
	return &copyValue, nil
}

func (h *handlerDatabaseRepo) GetRoleNamesByUser(_ context.Context, _ uint) ([]string, error) {
	return append([]string(nil), h.roleNames...), nil
}

func (h *handlerDatabaseRepo) ListDatabases(_ context.Context, _ domain.DatabaseListFilters) (domain.DatabaseListResult, error) {
	if h.listErr != nil {
		return domain.DatabaseListResult{}, h.listErr
	}
	return h.listResult, nil
}

func (h *handlerDatabaseRepo) GetVisibleDatabaseByID(_ context.Context, _ uint, _ domain.DatabaseVisibilityScope, _ uint) (*domain.DatabaseWithDatasetCount, error) {
	if h.visibleByIDErr != nil {
		return nil, h.visibleByIDErr
	}
	if h.visibleByIDResult == nil {
		return nil, domain.ErrDatabaseNotFound
	}
	copyValue := *h.visibleByIDResult
	return &copyValue, nil
}

type handlerDatabaseTester struct {
	err        error
	probeErr   error
	probeValue domain.TestConnectionResult
	allowRate  bool

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

func (h *handlerDatabaseTester) TestConnection(_ context.Context, _ string) error {
	return h.err
}

func (h *handlerDatabaseTester) Probe(_ context.Context, _ string) (domain.TestConnectionResult, error) {
	if h.probeErr != nil {
		return domain.TestConnectionResult{}, h.probeErr
	}
	return h.probeValue, nil
}

func (h *handlerDatabaseTester) Allow(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
	return h.allowRate, nil
}

func (h *handlerDatabaseTester) ListSchemas(_ context.Context, _ svcauth.SQLConnection) ([]string, error) {
	h.schemasCalls++
	if h.schemasErr != nil {
		return nil, h.schemasErr
	}
	return append([]string(nil), h.schemas...), nil
}

func (h *handlerDatabaseTester) ListTables(_ context.Context, _ svcauth.SQLConnection, _ string, _ int, _ int) ([]domain.DatabaseTable, int64, error) {
	h.tablesCalls++
	if h.tablesErr != nil {
		return nil, 0, h.tablesErr
	}
	return append([]domain.DatabaseTable(nil), h.tables...), h.tablesTotal, nil
}

func (h *handlerDatabaseTester) ListColumns(_ context.Context, _ svcauth.SQLConnection, _ string, _ string) ([]domain.DatabaseColumn, error) {
	h.columnsCalls++
	if h.columnsErr != nil {
		return nil, h.columnsErr
	}
	return append([]domain.DatabaseColumn(nil), h.columns...), nil
}

type handlerConnectionPool struct{}

func (handlerConnectionPool) Get(_ context.Context, _ uint, _ string) (svcauth.SQLConnection, error) {
	return nil, nil
}

func (handlerConnectionPool) Close(_ context.Context, _ uint) error {
	return nil
}

func (handlerConnectionPool) Shutdown(_ context.Context) error {
	return nil
}

type handlerDatabaseAuditLogger struct{}

func (h *handlerDatabaseAuditLogger) LogDatabaseCreated(_ context.Context, _ uint) {}

func newDatabaseRouter(repo *handlerDatabaseRepo, tester *handlerDatabaseTester) *gin.Engine {
	svc, err := svcauth.NewDatabaseService(repo, tester, &handlerDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		panic(err)
	}
	svc.SetConnectionProber(tester)
	svc.SetTestRateLimiter(tester)
	svc.SetSchemaInspector(tester)
	svc.SetConnectionPool(handlerConnectionPool{})
	h := httpauth.NewDatabaseHandler(svc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domainauth.UserContext{ID: 1, Active: true})
		c.Next()
	})

	admin := r.Group("/api/v1/admin")
	admin.POST("/databases", h.Create)
	admin.GET("/databases", h.List)
	admin.GET("/databases/:id", h.Get)
	admin.PUT("/databases/:id", h.Update)
	admin.DELETE("/databases/:id", h.Delete)
	admin.POST("/databases/test", h.TestConnection)
	admin.POST("/databases/:id/test", h.TestConnectionByID)
	admin.GET("/databases/:id/schemas", h.ListSchemas)
	admin.GET("/databases/:id/tables", h.ListTables)
	admin.GET("/databases/:id/columns", h.ListColumns)

	return r
}

func TestDatabaseHandler_PostReturns201WithMaskedURI(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true}, &handlerDatabaseTester{allowRate: true})

	payload := []byte(`{"database_name":"analytics","sqlalchemy_uri":"postgresql://superset:secret-pass@localhost:5432/analytics","allow_dml":true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("***")) {
		t.Fatalf("expected masked URI in response body, got: %s", w.Body.String())
	}
}

func TestDatabaseHandler_PostDuplicateNameReturns409(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true, databaseNameTaken: true}, &handlerDatabaseTester{allowRate: true})

	payload := []byte(`{"database_name":"analytics","sqlalchemy_uri":"postgresql://superset:secret-pass@localhost:5432/analytics"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_PostStrictTestFailureReturns422(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true}, &handlerDatabaseTester{err: domain.ErrDatabaseConnectionTestFailed, allowRate: true})

	payload := []byte(`{"database_name":"analytics","sqlalchemy_uri":"postgresql://superset:secret-pass@localhost:5432/analytics","strict_test":true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_PostNonAdminReturns403(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: false}, &handlerDatabaseTester{allowRate: true})

	payload := []byte(`{"database_name":"analytics","sqlalchemy_uri":"postgresql://superset:secret-pass@localhost:5432/analytics"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_TestConnectionReturns200SuccessFalse(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true}, &handlerDatabaseTester{allowRate: true, probeValue: domain.TestConnectionResult{Success: false, Driver: "postgresql", Error: "auth failed"}})

	payload := []byte(`{"sqlalchemy_uri":"postgresql://superset:secret-pass@localhost:5432/analytics"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases/test", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"success":false`)) {
		t.Fatalf("expected success false body, got %s", w.Body.String())
	}
}

func TestDatabaseHandler_TestConnectionUnknownDriverReturns422(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true}, &handlerDatabaseTester{allowRate: true, probeErr: domain.ErrUnknownDatabaseDriver})

	payload := []byte(`{"sqlalchemy_uri":"snowflake://account/warehouse"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases/test", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_TestConnectionRateLimitedReturns429(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true}, &handlerDatabaseTester{allowRate: false})

	payload := []byte(`{"sqlalchemy_uri":"postgresql://superset:secret-pass@localhost:5432/analytics"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases/test", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_TestConnectionByIDReturns200(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://superset:secret-pass@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}
	r := newDatabaseRouter(
		&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: encryptedURI}},
		&handlerDatabaseTester{allowRate: true, probeValue: domain.TestConnectionResult{Success: true, Driver: "postgresql", LatencyMS: 10, DBVersion: "PostgreSQL 15.4"}},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases/2/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_ListReturns200WithPagination(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{
		roleNames: []string{"Admin"},
		listResult: domain.DatabaseListResult{
			Items: []domain.DatabaseWithDatasetCount{{
				Database:     domain.Database{ID: 7, DatabaseName: "analytics", SQLAlchemyURI: "postgresql://superset:secret@localhost:5432/analytics", ExposeInSQLLab: true},
				DatasetCount: 5,
			}},
			Total: 1,
		},
	}, &handlerDatabaseTester{allowRate: true})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/databases?page=1&page_size=10", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"pagination\"") {
		t.Fatalf("expected pagination in body, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "***") {
		t.Fatalf("expected masked uri in body, got %s", w.Body.String())
	}
}

func TestDatabaseHandler_GetReturns404WhenNotVisible(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{roleNames: []string{"Gamma"}}, &handlerDatabaseTester{allowRate: true})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/databases/77", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_PutReturns200WithMaskedURI(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://superset:secret-pass@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	repo := &handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, DatabaseName: "analytics", SQLAlchemyURI: encryptedURI}}
	r := newDatabaseRouter(repo, &handlerDatabaseTester{allowRate: true})

	payload := []byte(`{"database_name":"analytics-updated","sqlalchemy_uri":"postgresql://superset:***@localhost:5432/analytics"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/databases/2", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "***") {
		t.Fatalf("expected masked URI in response body, got: %s", w.Body.String())
	}
	if repo.updated == nil || repo.updated.DatabaseName != "analytics-updated" {
		t.Fatalf("expected updated database name to be persisted")
	}
}

func TestDatabaseHandler_DeleteReturns204(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: "postgresql://superset:enc@localhost:5432/analytics"}}, &handlerDatabaseTester{allowRate: true})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/databases/2", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_DeleteReturns409WhenInUse(t *testing.T) {
	r := newDatabaseRouter(
		&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: "postgresql://superset:enc@localhost:5432/analytics"}, datasetCount: 3},
		&handlerDatabaseTester{allowRate: true},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/databases/2", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_ListSchemasReturns200(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://superset:secret-pass@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	r := newDatabaseRouter(
		&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: encryptedURI}},
		&handlerDatabaseTester{allowRate: true, schemas: []string{"analytics", "public"}},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/databases/2/schemas", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "public") {
		t.Fatalf("expected schemas payload, got %s", w.Body.String())
	}
}

func TestDatabaseHandler_ListTablesReturnsPaginatedData(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://superset:secret-pass@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	r := newDatabaseRouter(
		&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: encryptedURI}},
		&handlerDatabaseTester{allowRate: true, tables: []domain.DatabaseTable{{Name: "orders"}}, tablesTotal: 1},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/databases/2/tables?schema=public&page=1&page_size=10", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "orders") {
		t.Fatalf("expected tables payload, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"pagination\"") {
		t.Fatalf("expected pagination payload, got %s", w.Body.String())
	}
}

func TestDatabaseHandler_ListColumnsReturnsIsDttmMetadata(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://superset:secret-pass@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	r := newDatabaseRouter(
		&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: encryptedURI}},
		&handlerDatabaseTester{allowRate: true, columns: []domain.DatabaseColumn{{Name: "created_at", DataType: "timestamp", IsDttm: true}}},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/databases/2/columns?schema=public&table=orders", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"is_dttm\":true") {
		t.Fatalf("expected is_dttm metadata in payload, got %s", w.Body.String())
	}
}

func TestDatabaseHandler_ListSchemasForceRefreshRateLimitedReturns429(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://superset:secret-pass@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	r := newDatabaseRouter(
		&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: encryptedURI}},
		&handlerDatabaseTester{allowRate: false},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/databases/2/schemas?force_refresh=true", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_ListSchemasDatabaseUnreachableReturns502(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://superset:secret-pass@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	r := newDatabaseRouter(
		&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: encryptedURI}},
		&handlerDatabaseTester{allowRate: true, schemasErr: domain.ErrDatabaseUnreachable},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/databases/2/schemas", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseHandler_ListSchemasTimeoutReturns504(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://superset:secret-pass@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	r := newDatabaseRouter(
		&handlerDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: encryptedURI}},
		&handlerDatabaseTester{allowRate: true, schemasErr: domain.ErrDatabaseTimeout},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/databases/2/schemas", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d: %s", w.Code, w.Body.String())
	}
}
