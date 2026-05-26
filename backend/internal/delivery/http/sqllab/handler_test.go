package sqllab

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"net/http/httptest"
	"testing"
	"time"

	domain "superset/auth-service/internal/domain/auth"
	domdb "superset/auth-service/internal/domain/db"
	domainquery "superset/auth-service/internal/domain/query"

	"github.com/gin-gonic/gin"
)

// ---- mock SQLLab repo ----

type mockSQLLabRepo struct {
	tabs         map[uint]*domainquery.TabState
	savedQueries []*domainquery.SavedQuery
	err          error
}

func (m *mockSQLLabRepo) Create(_ context.Context, tab *domainquery.TabState) error {
	if m.err != nil {
		return m.err
	}
	tab.ID = uint(len(m.tabs) + 1)
	m.tabs[tab.ID] = tab
	return nil
}
func (m *mockSQLLabRepo) ListByUser(_ context.Context, userID uint, includeClosed bool) ([]*domainquery.TabState, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []*domainquery.TabState
	for _, t := range m.tabs {
		if t.UserID != userID {
			continue
		}
		if !includeClosed && !t.Active {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
func (m *mockSQLLabRepo) GetByID(_ context.Context, id uint, userID uint) (*domainquery.TabState, error) {
	if m.err != nil {
		return nil, m.err
	}
	t, ok := m.tabs[id]
	if !ok {
		return nil, nil
	}
	if t.UserID != userID {
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
func (m *mockSQLLabRepo) CloseTab(_ context.Context, id uint, userID uint) error {
	if m.err != nil {
		return m.err
	}
	t, ok := m.tabs[id]
	if !ok || t.UserID != userID || !t.Active {
		return fmt.Errorf("not found")
	}
	t.Active = false
	return nil
}
func (m *mockSQLLabRepo) CloseAllTabs(_ context.Context, userID uint, exceptID *uint) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	var closed int64
	for _, t := range m.tabs {
		if t.UserID != userID || !t.Active {
			continue
		}
		if exceptID != nil && t.ID == *exceptID {
			continue
		}
		t.Active = false
		closed++
	}
	return closed, nil
}
func (m *mockSQLLabRepo) HardDelete(_ context.Context, id uint, userID uint) error {
	if m.err != nil {
		return m.err
	}
	t, ok := m.tabs[id]
	if !ok || t.UserID != userID {
		return fmt.Errorf("not found")
	}
	delete(m.tabs, id)
	return nil
}

func (m *mockSQLLabRepo) CreateSavedQuery(_ context.Context, sq *domainquery.SavedQuery) error {
	if m.err != nil {
		return m.err
	}
	sq.ID = uint(len(m.savedQueries) + 1)
	m.savedQueries = append(m.savedQueries, sq)
	return nil
}
func (m *mockSQLLabRepo) LabelExists(_ context.Context, userID uint, label string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	for _, sq := range m.savedQueries {
		if strings.EqualFold(sq.Label, label) && sq.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockSQLLabRepo) ListSavedQueries(_ context.Context, _ uint, _ domainquery.SavedQueryListParams) ([]*domainquery.SavedQuery, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.savedQueries, int64(len(m.savedQueries)), nil
}
func (m *mockSQLLabRepo) GetSavedQuery(_ context.Context, id uint, userID uint) (*domainquery.SavedQuery, error) {
	for _, sq := range m.savedQueries {
		if sq.ID == id && sq.UserID == userID {
			return sq, nil
		}
	}
	return nil, nil
}
func (m *mockSQLLabRepo) UpdateSavedQuery(_ context.Context, sq *domainquery.SavedQuery) error {
	if m.err != nil {
		return m.err
	}
	for i, existing := range m.savedQueries {
		if existing.ID == sq.ID && existing.UserID == sq.UserID {
			m.savedQueries[i] = sq
			return nil
		}
	}
	return fmt.Errorf("not found")
}
func (m *mockSQLLabRepo) DeleteSavedQuery(_ context.Context, id uint, userID uint) error {
	if m.err != nil {
		return m.err
	}
	for i, sq := range m.savedQueries {
		if sq.ID == id && sq.UserID == userID {
			m.savedQueries = append(m.savedQueries[:i], m.savedQueries[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found")
}
func (m *mockSQLLabRepo) ForkSavedQuery(_ context.Context, id uint, userID uint) (*domainquery.SavedQuery, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, sq := range m.savedQueries {
		if sq.ID == id && sq.UserID == userID {
			copy_ := *sq
			copy_.ID = uint(len(m.savedQueries) + 1)
			copy_.Label = "Copy of " + sq.Label
			m.savedQueries = append(m.savedQueries, &copy_)
			return &copy_, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockSQLLabRepo) FindSchemaState(_ context.Context, _ uint) ([]domainquery.TableSchema, error) {
	return nil, nil
}
func (m *mockSQLLabRepo) UpsertSchemaState(_ context.Context, _ *domainquery.TableSchema) error {
	return nil
}
func (m *mockSQLLabRepo) UpdateSchemaStateCollapsed(_ context.Context, _ uint, _ string) error {
	return nil
}
func (m *mockSQLLabRepo) DeleteSchemaStateByTab(_ context.Context, _ uint) error {
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
	h := NewHandler(repo, &mockDatabaseRepo{}, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user", domain.UserContext{ID: 1, Active: true})
		c.Next()
	})
	sqllab := r.Group("/api/v1/sqllab")
	sqllab.PUT("/tabs/:id", h.UpdateTab)
	sqllab.PUT("/tabs/:id/close", h.CloseTab)
	sqllab.DELETE("/tabs", h.CloseAllTabs)
	sqllab.DELETE("/tabs/:id", h.HardDeleteTab)
	sqllab.GET("/tabs", h.ListTabs)
	sqllab.POST("/saved-queries", h.CreateSavedQuery)
	sqllab.GET("/saved-queries", h.ListSavedQueries)
	sqllab.PUT("/saved-queries/:id", h.UpdateSavedQuery)
	sqllab.DELETE("/saved-queries/:id", h.DeleteSavedQuery)
	sqllab.POST("/saved-queries/:id/fork", h.ForkSavedQuery)
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

// ---- CloseTab tests ----

func TestCloseTab_ActiveTab_Returns200(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 1, DbID: 1, Label: "test", Active: true}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1/close", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"closed":true`)) {
		t.Fatalf("expected closed:true, got %s", w.Body.String())
	}
	if repo.tabs[1].Active {
		t.Fatal("tab should be inactive after close")
	}
}

func TestCloseTab_NotOwner_Returns404(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 999, DbID: 1, Label: "other", Active: true}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1/close", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCloseTab_NotFound_Returns404(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/999/close", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ---- CloseAllTabs tests ----

func TestCloseAllTabs_ClosesMultiple_ReturnsCount(t *testing.T) {
	tabs := map[uint]*domainquery.TabState{
		1: {ID: 1, UserID: 1, DbID: 1, Label: "a", Active: true},
		2: {ID: 2, UserID: 1, DbID: 1, Label: "b", Active: true},
		3: {ID: 3, UserID: 1, DbID: 1, Label: "c", Active: true},
	}
	repo := &mockSQLLabRepo{tabs: tabs}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/tabs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"closed":3`)) {
		t.Fatalf("expected closed:3, got %s", w.Body.String())
	}
	for _, tab := range tabs {
		if tab.Active {
			t.Fatalf("tab %d should be inactive", tab.ID)
		}
	}
}

func TestCloseAllTabs_ExceptID_ExcludesActive(t *testing.T) {
	tabs := map[uint]*domainquery.TabState{
		1: {ID: 1, UserID: 1, DbID: 1, Label: "a", Active: true},
		2: {ID: 2, UserID: 1, DbID: 1, Label: "b", Active: true},
		3: {ID: 3, UserID: 1, DbID: 1, Label: "c", Active: true},
	}
	repo := &mockSQLLabRepo{tabs: tabs}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/tabs?except_id=2", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"closed":2`)) {
		t.Fatalf("expected closed:2, got %s", w.Body.String())
	}
	if !tabs[2].Active {
		t.Fatal("tab 2 should remain active (excluded)")
	}
	if tabs[1].Active || tabs[3].Active {
		t.Fatal("tabs 1 and 3 should be inactive")
	}
}

func TestCloseAllTabs_NoneOpen_ReturnsZero(t *testing.T) {
	tabs := map[uint]*domainquery.TabState{
		1: {ID: 1, UserID: 1, DbID: 1, Label: "a", Active: false},
	}
	repo := &mockSQLLabRepo{tabs: tabs}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/tabs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"closed":0`)) {
		t.Fatalf("expected closed:0, got %s", w.Body.String())
	}
}

// ---- HardDeleteTab tests ----

func TestHardDeleteTab_OwnTab_Returns204(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 1, DbID: 1, Label: "test", Active: true}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/tabs/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if _, exists := repo.tabs[1]; exists {
		t.Fatal("tab should be deleted from repo")
	}
}

func TestHardDeleteTab_NotOwner_Returns404(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 999, DbID: 1, Label: "other", Active: true}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/tabs/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHardDeleteTab_NotFound_Returns404(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/tabs/999", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ---- ListTabs include_closed test ----

func TestListTabs_IncludeClosed_ReturnsAll(t *testing.T) {
	tabs := map[uint]*domainquery.TabState{
		1: {ID: 1, UserID: 1, DbID: 1, Label: "active", Active: true, CreatedOn: time.Now()},
		2: {ID: 2, UserID: 1, DbID: 1, Label: "closed", Active: false, CreatedOn: time.Now()},
	}
	repo := &mockSQLLabRepo{tabs: tabs}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sqllab/tabs?include_closed=true", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"active"`)) {
		t.Fatal("should include active tab")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"closed"`)) {
		t.Fatal("should include closed tab")
	}
}

func TestUpdateTab_64KB_SQL_Allowed(t *testing.T) {
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

// ---- SavedQuery tests ----

func TestCreateSavedQuery_ValidRequest_Returns201(t *testing.T) {
	repo := &mockSQLLabRepo{
		tabs: map[uint]*domainquery.TabState{},
	}
	router := newSQLLabRouter(repo)

	body := `{"db_id":1,"label":"My Query","sql":"SELECT * FROM users"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/saved-queries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"label":"My Query"`)) {
		t.Fatalf("expected label in response, got %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"sql_tables"`)) {
		t.Fatalf("expected sql_tables in response, got %s", w.Body.String())
	}
}

func TestCreateSavedQuery_DuplicateLabel_Returns409(t *testing.T) {
	dupe := &domainquery.SavedQuery{ID: 1, Label: "My Query", DbID: 1, UserID: 1}
	repo := &mockSQLLabRepo{
		tabs:         map[uint]*domainquery.TabState{},
		savedQueries: []*domainquery.SavedQuery{dupe},
	}
	router := newSQLLabRouter(repo)

	body := `{"db_id":1,"label":"My Query","sql":"SELECT 1"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/saved-queries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSavedQuery_MissingDbID_Returns400(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	body := `{"label":"X","sql":"SELECT 1"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/saved-queries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListSavedQueries_EmptyList_Returns200(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sqllab/saved-queries", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"items"`)) {
		t.Fatalf("expected items array, got %s", w.Body.String())
	}
}

// ---- SavedQuery update/delete/fork tests ----

func TestUpdateSavedQuery_OwnQuery_Returns200(t *testing.T) {
	sq := &domainquery.SavedQuery{ID: 1, Label: "Original", DbID: 1, UserID: 1, SQL: "SELECT 1", CreatedByFK: 1, ChangedByFK: 1}
	repo := &mockSQLLabRepo{savedQueries: []*domainquery.SavedQuery{sq}, tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	body := `{"label":"Updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/saved-queries/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"label":"Updated"`)) {
		t.Fatalf("expected updated label, got %s", w.Body.String())
	}
}

func TestUpdateSavedQuery_NotOwner_Returns403(t *testing.T) {
	sq := &domainquery.SavedQuery{ID: 1, Label: "Other", DbID: 1, UserID: 999, CreatedByFK: 999}
	repo := &mockSQLLabRepo{savedQueries: []*domainquery.SavedQuery{sq}, tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	body := `{"label":"Hacked"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/saved-queries/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSavedQuery_DuplicateLabel_Returns409(t *testing.T) {
	sq1 := &domainquery.SavedQuery{ID: 1, Label: "First", DbID: 1, UserID: 1, CreatedByFK: 1}
	sq2 := &domainquery.SavedQuery{ID: 2, Label: "Second", DbID: 1, UserID: 1, CreatedByFK: 1}
	repo := &mockSQLLabRepo{savedQueries: []*domainquery.SavedQuery{sq1, sq2}, tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	body := `{"label":"Second"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/saved-queries/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteSavedQuery_OwnQuery_Returns200(t *testing.T) {
	sq := &domainquery.SavedQuery{ID: 1, Label: "Delete Me", DbID: 1, UserID: 1, CreatedByFK: 1}
	repo := &mockSQLLabRepo{savedQueries: []*domainquery.SavedQuery{sq}, tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/saved-queries/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(repo.savedQueries) != 0 {
		t.Fatal("saved query should be deleted")
	}
}

func TestDeleteSavedQuery_NotOwner_Returns403(t *testing.T) {
	sq := &domainquery.SavedQuery{ID: 1, Label: "Other", DbID: 1, UserID: 999, CreatedByFK: 999}
	repo := &mockSQLLabRepo{savedQueries: []*domainquery.SavedQuery{sq}, tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/saved-queries/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestForkSavedQuery_OwnQuery_Returns201(t *testing.T) {
	sq := &domainquery.SavedQuery{ID: 1, Label: "Original", DbID: 1, UserID: 1, SQL: "SELECT * FROM users", CreatedByFK: 1, ChangedByFK: 1}
	repo := &mockSQLLabRepo{savedQueries: []*domainquery.SavedQuery{sq}, tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/saved-queries/1/fork", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"label":"Copy of Original"`)) {
		t.Fatalf("expected forked label, got %s", w.Body.String())
	}
	if len(repo.savedQueries) != 2 {
		t.Fatalf("expected 2 saved queries, got %d", len(repo.savedQueries))
	}
}

func TestForkSavedQuery_NotOwner_Returns403(t *testing.T) {
	sq := &domainquery.SavedQuery{ID: 1, Label: "Other", DbID: 1, UserID: 999, CreatedByFK: 999}
	repo := &mockSQLLabRepo{savedQueries: []*domainquery.SavedQuery{sq}, tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/saved-queries/1/fork", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
