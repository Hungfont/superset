package auth_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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

type handlerDatabaseTester struct {
	err error
}

func (h *handlerDatabaseTester) TestConnection(_ context.Context, _ string) error {
	return h.err
}

type handlerDatabaseAuditLogger struct{}

func (h *handlerDatabaseAuditLogger) LogDatabaseCreated(_ context.Context, _ uint) {}

func newDatabaseRouter(repo *handlerDatabaseRepo, tester *handlerDatabaseTester) *gin.Engine {
	svc, err := svcauth.NewDatabaseService(repo, tester, &handlerDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		panic(err)
	}
	h := httpauth.NewDatabaseHandler(svc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domainauth.UserContext{ID: 1, Active: true})
		c.Next()
	})

	admin := r.Group("/api/v1/admin")
	admin.POST("/databases", h.Create)

	return r
}

func TestDatabaseHandler_PostReturns201WithMaskedURI(t *testing.T) {
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true}, &handlerDatabaseTester{})

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
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true, databaseNameTaken: true}, &handlerDatabaseTester{})

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
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: true}, &handlerDatabaseTester{err: domain.ErrDatabaseConnectionTestFailed})

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
	r := newDatabaseRouter(&handlerDatabaseRepo{isAdmin: false}, &handlerDatabaseTester{})

	payload := []byte(`{"database_name":"analytics","sqlalchemy_uri":"postgresql://superset:secret-pass@localhost:5432/analytics"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/databases", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
