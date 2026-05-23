package sqllab

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "superset/auth-service/internal/domain/auth"
	domdb "superset/auth-service/internal/domain/db"
	domainquery "superset/auth-service/internal/domain/query"

	"github.com/gin-gonic/gin"
)

// ---- mock SQLLab repo ----

type mockSQLLabRepo struct {
	tabs map[uint]*domainquery.TabState
	err  error
}

func (m *mockSQLLabRepo) Create(_ context.Context, tab *domainquery.TabState) error {
	if m.err != nil {
		return m.err
	}
	tab.ID = uint(len(m.tabs) + 1)
	m.tabs[tab.ID] = tab
	return nil
}
func (m *mockSQLLabRepo) ListByUser(_ context.Context, _ uint) ([]*domainquery.TabState, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []*domainquery.TabState
	for _, t := range m.tabs {
		out = append(out, t)
	}
	return out, nil
}
func (m *mockSQLLabRepo) GetByID(_ context.Context, id uint, _ uint) (*domainquery.TabState, error) {
	if m.err != nil {
		return nil, m.err
	}
	t, ok := m.tabs[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}
func (m *mockSQLLabRepo) Update(_ context.Context, tab *domainquery.TabState) error {
	if m.err != nil {
		return m.err
	}
	m.tabs[tab.ID] = tab
	return nil
}

// ---- mock Database repo (must satisfy full DatabaseRepository interface) ----

type mockDatabaseRepo struct{}

func (m *mockDatabaseRepo) IsAdmin(_ context.Context, _ uint) (bool, error)                       { return false, nil }
func (m *mockDatabaseRepo) GetRoleNamesByUser(_ context.Context, _ uint) ([]string, error)         { return nil, nil }
func (m *mockDatabaseRepo) DatabaseNameExists(_ context.Context, _ string) (bool, error)           { return false, nil }
func (m *mockDatabaseRepo) ListDatabases(_ context.Context, _ domdb.DatabaseListFilters) (domdb.DatabaseListResult, error) {
	return domdb.DatabaseListResult{}, nil
}
func (m *mockDatabaseRepo) GetVisibleDatabaseByID(_ context.Context, _ uint, _ domdb.DatabaseVisibilityScope, _ uint) (*domdb.DatabaseWithDatasetCount, error) {
	return nil, nil
}
func (m *mockDatabaseRepo) GetDatabaseByID(_ context.Context, _ uint) (*domdb.Database, error) {
	return nil, nil
}
func (m *mockDatabaseRepo) CreateDatabase(_ context.Context, _ *domdb.Database) error { return nil }
func (m *mockDatabaseRepo) UpdateDatabase(_ context.Context, _ *domdb.Database) error { return nil }
func (m *mockDatabaseRepo) DeleteDatabase(_ context.Context, _ uint) error             { return nil }
func (m *mockDatabaseRepo) CountDatasetsByDatabaseID(_ context.Context, _ uint) (int64, error) {
	return 0, nil
}
func (m *mockDatabaseRepo) CountRunningQueriesByDatabaseID(_ context.Context, _ uint) (int64, error) {
	return 0, nil
}
func (m *mockDatabaseRepo) ListDatasetsByDatabaseID(_ context.Context, _ uint) ([]domdb.DatasetRef, error) {
	return nil, nil
}

// ---- helper ----

func newSQLLabRouter(repo *mockSQLLabRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(repo, &mockDatabaseRepo{})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user", domain.UserContext{ID: 1, Active: true})
		c.Next()
	})
	sqllab := r.Group("/api/v1/sqllab")
	sqllab.PUT("/tabs/:id", h.UpdateTab)
	return r
}

// ---- tests ----

func TestUpdateTab_SQLExceeds64KB_Returns422(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 1, DbID: 1, Label: "test"}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	bigSQL := make([]byte, 65537)
	for i := range bigSQL {
		bigSQL[i] = 'x'
	}
	body := `{"sql":"` + string(bigSQL) + `"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("sql_too_large")) {
		t.Fatalf("expected sql_too_large in body, got %s", w.Body.String())
	}
}

func TestUpdateTab_DifferentUser_Returns403(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 999, DbID: 1, Label: "other user's tab"}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	body := `{"label":"hacked"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTab_PartialUpdate_OnlyChangedFields(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 1, DbID: 1, Label: "original", SQL: "SELECT 1", Schema: "public"}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	body := `{"label":"renamed"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	updated := repo.tabs[uint(1)]
	if updated.Label != "renamed" {
		t.Fatalf("expected label 'renamed', got '%s'", updated.Label)
	}
	if updated.SQL != "SELECT 1" {
		t.Fatalf("SQL should remain unchanged, got '%s'", updated.SQL)
	}
	if updated.Schema != "public" {
		t.Fatalf("schema should remain unchanged, got '%s'", updated.Schema)
	}
}

func TestUpdateTab_AllFieldsIncludingNew(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 1, DbID: 1, Label: "old", SQL: "", Schema: "", Catalog: "", QueryLimit: 0, LatestQueryID: nil, HideLeftBar: false, ExtraJSON: ""}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	body := `{"label":"new","sql":"SELECT 2","schema":"private","catalog":"cat","query_limit":500,"db_id":2,"latest_query_id":"q-123","hide_left_bar":true,"extra_json":"{\"key\":\"val\"}"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	updated := repo.tabs[uint(1)]
	if updated.Label != "new" {
		t.Fatalf("label mismatch: %s", updated.Label)
	}
	if updated.SQL != "SELECT 2" {
		t.Fatalf("sql mismatch: %s", updated.SQL)
	}
	if updated.Schema != "private" {
		t.Fatalf("schema mismatch: %s", updated.Schema)
	}
	if updated.Catalog != "cat" {
		t.Fatalf("catalog mismatch: %s", updated.Catalog)
	}
	if updated.QueryLimit != 500 {
		t.Fatalf("query_limit mismatch: %d", updated.QueryLimit)
	}
	if updated.DbID != 2 {
		t.Fatalf("db_id mismatch: %d", updated.DbID)
	}
	if updated.LatestQueryID == nil || *updated.LatestQueryID != "q-123" {
		t.Fatalf("latest_query_id mismatch")
	}
	if !updated.HideLeftBar {
		t.Fatalf("hide_left_bar should be true")
	}
	if updated.ExtraJSON != `{"key":"val"}` {
		t.Fatalf("extra_json mismatch: %s", updated.ExtraJSON)
	}
}

func TestUpdateTab_66KB_SQL_Allowed(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 1, DbID: 1, Label: "test"}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	sql := make([]byte, 65536) // exactly 64KB = 65536 bytes, allowed
	for i := range sql {
		sql[i] = 'x'
	}
	body := `{"sql":"` + string(sql) + `"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for exactly 64KB SQL, got %d: %s", w.Code, w.Body.String())
	}
}
