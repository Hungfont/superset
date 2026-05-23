# SQL-003: Close and Delete Tabs — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add soft-close, close-all, close-others, reopen, and hard-delete for SQL Lab tabs with confirmation dialogs and a "Recently Closed" recovery sheet.

**Architecture:** Backend adds 4 repository methods and 4 API routes following the existing handler pattern (getUserContext → parse params → ownership check → repo call → JSON response). Frontend adds TanStack Query mutations and wires them into the existing Zustand store + shadcn/ui components — no new files, all changes are surgical edits to existing files.

**Tech Stack:** Go + GORM + Gin (backend), React + TypeScript + TanStack Query + Zustand + shadcn/ui (frontend)

---

### Task 1: Add repository interface methods

**Files:**
- Modify: `backend/internal/domain/query/sqllab_repository.go`

- [ ] **Step 1: Add new method signatures to SQLLabRepository**

Add these method signatures to the `SQLLabRepository` interface (the existing `TabWithQueryStatus` struct stays unchanged):

```go
CloseTab(ctx context.Context, id uint, userID uint) error
CloseAllTabs(ctx context.Context, userID uint, exceptID *uint) (int64, error)
ReopenTab(ctx context.Context, id uint, userID uint) error
HardDelete(ctx context.Context, id uint, userID uint) error
```

And update `ListByUser` signature from `(ctx context.Context, userID uint)` to `(ctx context.Context, userID uint, includeClosed bool)`.

- [ ] **Step 2: Verify compilation catches the missing implementations**

Run: `cd backend && go build ./...`
Expected: compilation errors in `postgres/sqllab_repo.go` and `sqllab/handler_test.go` because the mock doesn't implement the new methods yet.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/query/sqllab_repository.go
git commit -m "feat(sqllab): add CloseTab, CloseAllTabs, ReopenTab, HardDelete to repository interface

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 2: Add CloseAllTabsRequest type

**Files:**
- Modify: `backend/internal/domain/query/sqllab_types.go`

- [ ] **Step 1: Add CloseAllTabsRequest at the end of the file**

Append this new type after the existing `TabResponse` struct:

```go
// CloseAllTabsRequest is the request body for closing all tabs.
// ExceptID, when set, excludes that tab from being closed (used for "Close Others").
type CloseAllTabsRequest struct {
	ExceptID *uint `json:"except_id"`
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/domain/query/sqllab_types.go
git commit -m "feat(sqllab): add CloseAllTabsRequest type

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 3: Implement repository methods (postgres)

**Files:**
- Modify: `backend/internal/repository/postgres/sqllab_repo.go`

- [ ] **Step 1: Update imports and struct (no changes to imports/struct)**

The existing imports already have `context`, `errors`, `fmt`, `gorm.io/gorm`, and the domain package. Just need to add `"time"` for ReopenTab timestamp.

- [ ] **Step 2: Update ListByUser to accept includeClosed parameter**

Replace the existing `ListByUser` method (lines 24-44) with:

```go
func (r *sqllabRepo) ListByUser(ctx context.Context, userID uint, includeClosed bool) ([]*query.TabState, error) {
	q := r.db.WithContext(ctx).
		Table("tab_state").
		Select("tab_state.*, query.status AS latest_query_status").
		Joins("LEFT JOIN query ON query.id = tab_state.latest_query_id").
		Where("tab_state.user_id = ?", userID)
	if !includeClosed {
		q = q.Where("tab_state.active = ?", true)
	}
	var rows []*query.TabWithQueryStatus
	err := q.Order("tab_state.created_on ASC").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("listing tabs by user: %w", err)
	}
	tabs := make([]*query.TabState, 0, len(rows))
	for _, r := range rows {
		t := r.TabState
		if r.LatestQueryStatus != nil {
			t.ExtraJSON = `{"latest_query_status":"` + *r.LatestQueryStatus + `"}`
		}
		tabs = append(tabs, &t)
	}
	return tabs, nil
}
```

- [ ] **Step 3: Add CloseTab, CloseAllTabs, ReopenTab, HardDelete methods**

Add at the end of the file, before the `var _` compile-time check:

```go
func (r *sqllabRepo) CloseTab(ctx context.Context, id uint, userID uint) error {
	result := r.db.WithContext(ctx).
		Model(&query.TabState{}).
		Where("id = ? AND user_id = ? AND active = ?", id, userID, true).
		Update("active", false)
	if result.Error != nil {
		return fmt.Errorf("close tab: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *sqllabRepo) CloseAllTabs(ctx context.Context, userID uint, exceptID *uint) (int64, error) {
	q := r.db.WithContext(ctx).
		Model(&query.TabState{}).
		Where("user_id = ? AND active = ?", userID, true)
	if exceptID != nil {
		q = q.Where("id != ?", *exceptID)
	}
	result := q.Update("active", false)
	if result.Error != nil {
		return 0, fmt.Errorf("close all tabs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *sqllabRepo) ReopenTab(ctx context.Context, id uint, userID uint) error {
	result := r.db.WithContext(ctx).
		Model(&query.TabState{}).
		Where("id = ? AND user_id = ? AND active = ?", id, userID, false).
		Updates(map[string]interface{}{
			"active":     true,
			"changed_on": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("reopen tab: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *sqllabRepo) HardDelete(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tab query.TabState
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&tab).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("not found")
			}
			return fmt.Errorf("hard delete: find tab: %w", err)
		}
		if err := tx.Where("tab_state_id = ?", id).Delete(&query.TableSchema{}).Error; err != nil {
			return fmt.Errorf("hard delete: table_schema: %w", err)
		}
		if err := tx.Delete(&tab).Error; err != nil {
			return fmt.Errorf("hard delete: tab: %w", err)
		}
		return nil
	})
}
```

- [ ] **Step 4: Add time to imports**

Replace the import block:

```go
import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	query "superset/auth-service/internal/domain/query"
)
```

- [ ] **Step 5: Verify compilation**

Run: `cd backend && go build ./...`
Expected: succeeds (no errors).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/postgres/sqllab_repo.go
git commit -m "feat(sqllab): implement CloseTab, CloseAllTabs, ReopenTab, HardDelete in postgres repo

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 4: Add handler methods

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler.go`

- [ ] **Step 1: Add CloseTab handler**

Add after the `UpdateTab` method, before `resolveVisibilityScope`:

```go
func (h *Handler) CloseTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	err = h.sqllabRepo.CloseTab(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		tab, getErr := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
		if getErr != nil || tab == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to close this tab"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"closed": true})
}
```

- [ ] **Step 2: Add CloseAllTabs handler**

```go
func (h *Handler) CloseAllTabs(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	var req domainquery.CloseAllTabsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// empty body or missing except_id is fine — use defaults
		req = domainquery.CloseAllTabsRequest{}
	}

	closed, err := h.sqllabRepo.CloseAllTabs(c.Request.Context(), userCtx.ID, req.ExceptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to close tabs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"closed": closed})
}
```

- [ ] **Step 3: Add ReopenTab handler**

```go
func (h *Handler) ReopenTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	err = h.sqllabRepo.ReopenTab(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		tab, getErr := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
		if getErr != nil || tab == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to reopen this tab"})
		return
	}

	tab, _ := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	label := ""
	if tab != nil {
		label = tab.Label
	}

	c.JSON(http.StatusOK, gin.H{"reopened": true, "id": id, "label": label})
}
```

- [ ] **Step 4: Add HardDeleteTab handler**

```go
func (h *Handler) HardDeleteTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	err = h.sqllabRepo.HardDelete(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		tab, getErr := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
		if getErr != nil || tab == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to delete this tab"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
```

- [ ] **Step 5: Modify ListTabs to support include_closed query parameter**

Replace the existing `ListTabs` method (lines 79-97) with:

```go
func (h *Handler) ListTabs(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	includeClosed := c.Query("include_closed") == "true"

	tabs, err := h.sqllabRepo.ListByUser(c.Request.Context(), userCtx.ID, includeClosed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to list tabs"})
		return
	}

	resp := make([]domainquery.TabResponse, 0, len(tabs))
	for _, t := range tabs {
		resp = append(resp, tabToResponse(t))
	}

	c.JSON(http.StatusOK, resp)
}
```

- [ ] **Step 6: Verify compilation**

Run: `cd backend && go build ./...`
Expected: succeeds.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler.go
git commit -m "feat(sqllab): add CloseTab, CloseAllTabs, ReopenTab, HardDeleteTab handlers

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 5: Add routes

**Files:**
- Modify: `backend/internal/delivery/http/router.go`

- [ ] **Step 1: Add four new routes to the sqlLab group**

In the `sqlLab` route group (lines 171-177), add the new routes:

```go
sqlLab := protected.Group("/sqllab")
{
	sqlLab.POST("/tabs", sqllabHandler.CreateTab)
	sqlLab.GET("/tabs", sqllabHandler.ListTabs)
	sqlLab.GET("/tabs/:id", sqllabHandler.GetTab)
	sqlLab.PUT("/tabs/:id", sqllabHandler.UpdateTab)
	sqlLab.PUT("/tabs/:id/close", sqllabHandler.CloseTab)
	sqlLab.POST("/tabs/close-all", sqllabHandler.CloseAllTabs)
	sqlLab.PUT("/tabs/:id/reopen", sqllabHandler.ReopenTab)
	sqlLab.DELETE("/tabs/:id", sqllabHandler.HardDeleteTab)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./...`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/router.go
git commit -m "feat(sqllab): register close, close-all, reopen, delete tab routes

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 6: Update mock repo for backward compatibility

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler_test.go`

- [ ] **Step 1: Add new mock methods so existing tests compile**

Add these methods to `mockSQLLabRepo` after the `Update` method:

```go
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
func (m *mockSQLLabRepo) ReopenTab(_ context.Context, id uint, userID uint) error {
	if m.err != nil {
		return m.err
	}
	t, ok := m.tabs[id]
	if !ok || t.UserID != userID || t.Active {
		return fmt.Errorf("not found")
	}
	t.Active = true
	t.ChangedOn = time.Now()
	return nil
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
```

- [ ] **Step 2: Remove the old ListByUser without includeClosed**

Delete the old `ListByUser` mock method (lines 32-41) since the new one replaces it.

- [ ] **Step 3: Add imports for fmt and time**

Add to the import block:
```go
import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domain "superset/auth-service/internal/domain/auth"
	domdb "superset/auth-service/internal/domain/db"
	domainquery "superset/auth-service/internal/domain/query"

	"github.com/gin-gonic/gin"
)
```

- [ ] **Step 4: Verify compilation and existing tests pass**

Run: `cd backend && go build ./... && go test ./internal/delivery/http/sqllab/... -v`
Expected: all 5 existing tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler_test.go
git commit -m "test(sqllab): update mock repo with new interface methods, keep existing tests green

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 7: Write backend handler tests

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler_test.go`

- [ ] **Step 1: Update the test router helper to register all routes**

Replace `newSQLLabRouter` with a version that registers all routes:

```go
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
	sqllab.PUT("/tabs/:id/close", h.CloseTab)
	sqllab.POST("/tabs/close-all", h.CloseAllTabs)
	sqllab.PUT("/tabs/:id/reopen", h.ReopenTab)
	sqllab.DELETE("/tabs/:id", h.HardDeleteTab)
	sqllab.GET("/tabs", h.ListTabs)
	return r
}
```

- [ ] **Step 2: Add CloseTab tests**

Add at the end of the file:

```go
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

func TestCloseTab_NotOwner_Returns403(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 999, DbID: 1, Label: "other", Active: true}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1/close", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
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
```

- [ ] **Step 3: Add CloseAllTabs tests**

```go
func TestCloseAllTabs_ClosesMultiple_ReturnsCount(t *testing.T) {
	tabs := map[uint]*domainquery.TabState{
		1: {ID: 1, UserID: 1, DbID: 1, Label: "a", Active: true},
		2: {ID: 2, UserID: 1, DbID: 1, Label: "b", Active: true},
		3: {ID: 3, UserID: 1, DbID: 1, Label: "c", Active: true},
	}
	repo := &mockSQLLabRepo{tabs: tabs}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/tabs/close-all", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"closed":3`)) {
		t.Fatalf("expected closed:3, got %s", w.Body.String())
	}
	for _, t := range tabs {
		if t.Active {
			t.Fatalf("tab %d should be inactive", t.ID)
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

	body := `{"except_id":2}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/tabs/close-all", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
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
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/tabs/close-all", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"closed":0`)) {
		t.Fatalf("expected closed:0, got %s", w.Body.String())
	}
}
```

- [ ] **Step 4: Add ReopenTab tests**

```go
func TestReopenTab_ClosedTab_Returns200(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 1, DbID: 1, Label: "test", Active: false}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1/reopen", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"reopened":true`)) {
		t.Fatalf("expected reopened:true, got %s", w.Body.String())
	}
	if !repo.tabs[1].Active {
		t.Fatal("tab should be active after reopen")
	}
}

func TestReopenTab_NotOwner_Returns403(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 999, DbID: 1, Label: "other", Active: false}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/1/reopen", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReopenTab_NotFound_Returns404(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/sqllab/tabs/999/reopen", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 5: Add HardDeleteTab tests**

```go
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

func TestHardDeleteTab_NotOwner_Returns403(t *testing.T) {
	tab := &domainquery.TabState{ID: 1, UserID: 999, DbID: 1, Label: "other", Active: true}
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{1: tab}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sqllab/tabs/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
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
```

- [ ] **Step 6: Add ListTabs include_closed test**

```go
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
```

- [ ] **Step 7: Run all tests**

Run: `cd backend && go test ./internal/delivery/http/sqllab/... -v`
Expected: all 17 tests pass (5 existing + 12 new).

- [ ] **Step 8: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler_test.go
git commit -m "test(sqllab): add handler tests for close, close-all, reopen, hard-delete, include_closed

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 8: Add frontend API functions

**Files:**
- Modify: `frontend/src/api/sqllab.ts`

- [ ] **Step 1: Add new API functions and modify fetchTabs**

Append these new API functions at the end of the file:

```ts
export async function closeTab(id: number): Promise<{ closed: boolean }> {
  return request<{ closed: boolean }>(`/api/v1/sqllab/tabs/${id}/close`, {
    method: "PUT",
    headers: getAuthHeaders(),
  });
}

export async function closeAllTabs(exceptId?: number): Promise<{ closed: number }> {
  return request<{ closed: number }>("/api/v1/sqllab/tabs/close-all", {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify(exceptId !== undefined ? { except_id: exceptId } : {}),
  });
}

export async function reopenTab(id: number): Promise<{ reopened: boolean; id: number; label: string }> {
  return request<{ reopened: boolean; id: number; label: string }>(`/api/v1/sqllab/tabs/${id}/reopen`, {
    method: "PUT",
    headers: getAuthHeaders(),
  });
}
```

Modify `fetchTabs` to accept an optional `include_closed` parameter. Change its signature from:

```ts
export async function fetchTabs(): Promise<TabStateResponse[]> {
  return request<TabStateResponse[]>("/api/v1/sqllab/tabs", {
```

To:

```ts
interface FetchTabsOptions {
  include_closed?: boolean;
}

export async function fetchTabs(options?: FetchTabsOptions): Promise<TabStateResponse[]> {
  const params = options?.include_closed ? "?include_closed=true" : "";
  return request<TabStateResponse[]>(`/api/v1/sqllab/tabs${params}`, {
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: any errors are pre-existing, not from our changes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/sqllab.ts
git commit -m "feat(sqllab): add closeTab, closeAllTabs, reopenTab API functions

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 9: Rename Zustand store actions

**Files:**
- Modify: `frontend/src/stores/sqlLabStore.ts`

- [ ] **Step 1: Rename removeTab → removeTabFromState and closeAllTabs → clearTabsState**

In the `SqlLabState` interface, replace:
```ts
  removeTab: (id: string) => void;
  closeAllTabs: () => void;
```
with:
```ts
  removeTabFromState: (id: string) => void;
  clearTabsState: () => void;
```

In the store creation, replace the implementation of `removeTab` and `closeAllTabs`:

Replace:
```ts
  removeTab: (id) => {
    set(state => {
      const newTabs = state.tabs.filter(t => t.id !== id);
      const newActiveId =
        state.activeTabId === id
          ? newTabs[0]?.id ?? null
          : state.activeTabId;
      return { tabs: newTabs, activeTabId: newActiveId };
    });
  },

  closeAllTabs: () => set({ tabs: [], activeTabId: null }),
```

With:
```ts
  removeTabFromState: (id) => {
    set(state => {
      const newTabs = state.tabs.filter(t => t.id !== id);
      const newActiveId =
        state.activeTabId === id
          ? newTabs[0]?.id ?? null
          : state.activeTabId;
      return { tabs: newTabs, activeTabId: newActiveId };
    });
  },

  clearTabsState: () => set({ tabs: [], activeTabId: null }),
```

- [ ] **Step 2: Verify the file has no TypeScript errors**

Run: `cd frontend && npx tsc --noEmit src/stores/sqlLabStore.ts`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/sqlLabStore.ts
git commit -m "refactor(sqllab): rename removeTab→removeTabFromState, closeAllTabs→clearTabsState

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 10: Wire frontend mutations and UI in SQLLabPage

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx`

- [ ] **Step 1: Add new imports**

Add to the import block (the existing import on line 52 needs updating, and new shadcn imports):

Replace line 52:
```ts
import { fetchTabs, createTab as createTabApi } from "@/api/sqllab";
```
With:
```ts
import { fetchTabs, createTab as createTabApi, closeTab, closeAllTabs, reopenTab } from "@/api/sqllab";
```

Add new shadcn/ui imports after the existing ContextMenu import block (after line 29). Only AlertDialog, Sheet, and ScrollArea are needed — ContextMenu is already imported and handles right-click menus:

```ts
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ScrollArea } from "@/components/ui/scroll-area";
```

- [ ] **Step 2: Destructure renamed store actions and add new state**

Find the destructuring line in the component body that extracts store actions. It's around line 660 area. Let me check the exact code...

From the exploration, the store destructuring has `removeTab` and `closeAllTabs`. We need to update these to use the new names. Also add `databaseId`.

The destructured actions from `useSqlLabStore` should be updated. Find the line similar to:
```ts
const { tabs, activeTabId, removeTab, closeAllTabs, ... } = useSqlLabStore();
```
Replace `removeTab` with `removeTabFromState` and `closeAllTabs` with `clearTabsState`.

- [ ] **Step 3: Add mutation hooks**

Add after the `createTabMutation` (after line 698):

```ts
  const closeTabMutation = useMutation({
    mutationFn: closeTab,
    onSuccess: (_, id) => {
      removeTabFromState(String(id));
      queryClient.invalidateQueries({ queryKey: ["sqllab-tabs"] });
      toast({ title: "Tab closed" });
    },
    onError: () => {
      toast({ title: "Failed to close tab", variant: "destructive" });
    },
  });

  const closeAllTabsMutation = useMutation({
    mutationFn: closeAllTabs,
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ["sqllab-tabs"] });
      toast({ title: `Closed ${result.closed} tab${result.closed !== 1 ? "s" : ""}` });
    },
    onError: () => {
      toast({ title: "Failed to close tabs", variant: "destructive" });
    },
  });

  const reopenTabMutation = useMutation({
    mutationFn: reopenTab,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sqllab-tabs"] });
      setIsClosedSheetOpen(false);
      toast({ title: "Tab reopened" });
    },
    onError: () => {
      toast({ title: "Failed to reopen tab", variant: "destructive" });
    },
  });
```

- [ ] **Step 4: Add "Recently Closed" Sheet state and query**

Add state and query after the mutation hooks (after Step 3's code):

```ts
  const [isClosedSheetOpen, setIsClosedSheetOpen] = useState(false);
  const [confirmClose, setConfirmClose] = useState<{
    tabId: string;
    reason: "dirty" | "running" | "both";
  } | null>(null);

  const { data: closedTabsData } = useQuery({
    queryKey: ["sqllab-tabs", "closed"],
    queryFn: () => fetchTabs({ include_closed: true }),
    enabled: isClosedSheetOpen,
  });

  const closedTabs = (closedTabsData ?? []).filter(t => !t.active);
```

- [ ] **Step 5: Add Ctrl+Shift+T keyboard handler**

Add a useEffect after the existing `useEffect` blocks:

```ts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.shiftKey && e.key === "T") {
        e.preventDefault();
        setIsClosedSheetOpen(true);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);
```

- [ ] **Step 6: Update the X button onClick handler**

Replace the X button `onClick` (lines 844-847) — change from calling `removeTab` to using the new close flow:

Replace:
```tsx
                          onClick={e => {
                            e.stopPropagation();
                            removeTab(tab.id);
                          }}
```

With:
```tsx
                          onClick={e => {
                            e.stopPropagation();
                            if (tab.status === "running" || tab.isDirty) {
                              const reason =
                                tab.status === "running" && tab.isDirty ? "both"
                                : tab.status === "running" ? "running"
                                : "dirty";
                              setConfirmClose({ tabId: tab.id, reason });
                            } else {
                              closeTabMutation.mutate(Number(tab.id));
                            }
                          }}
```

And the same for the `onKeyDown` handler (lines 848-853):

Replace:
```tsx
                          onKeyDown={e => {
                            if (e.key === "Enter" || e.key === " ") {
                              e.stopPropagation();
                              removeTab(tab.id);
                            }
                          }}
```

With:
```tsx
                          onKeyDown={e => {
                            if (e.key === "Enter" || e.key === " ") {
                              e.stopPropagation();
                              if (tab.status === "running" || tab.isDirty) {
                                const reason =
                                  tab.status === "running" && tab.isDirty ? "both"
                                  : tab.status === "running" ? "running"
                                  : "dirty";
                                setConfirmClose({ tabId: tab.id, reason });
                              } else {
                                closeTabMutation.mutate(Number(tab.id));
                              }
                            }
                          }}
```

- [ ] **Step 7: Update the context menu "Close" item**

Replace line 879:
```tsx
                    onClick={() => removeTab(tab.id)}
```
With:
```tsx
                    onClick={() => {
                      if (tab.status === "running" || tab.isDirty) {
                        const reason =
                          tab.status === "running" && tab.isDirty ? "both"
                          : tab.status === "running" ? "running"
                          : "dirty";
                        setConfirmClose({ tabId: tab.id, reason });
                      } else {
                        closeTabMutation.mutate(Number(tab.id));
                      }
                    }}
```

- [ ] **Step 8: Update the context menu "Close All" item and add "Close Others" + "Reopen Closed Tab"**

Replace lines 884-886:

```tsx
                  <ContextMenuItem onClick={() => closeAllTabs()}>
                    Close All
                  </ContextMenuItem>
```

With (also add `confirmCloseAll` state in the Step 4 block):

```tsx
                  <ContextMenuItem
                    onClick={() => closeAllTabsMutation.mutate(activeTabId ? Number(activeTabId) : undefined)}
                    disabled={tabs.length <= 1}
                  >
                    Close Others
                  </ContextMenuItem>
                  <ContextMenuItem onClick={() => setConfirmCloseAll(true)}>
                    Close All
                  </ContextMenuItem>
                  <ContextMenuSeparator />
                  <ContextMenuItem onClick={() => setIsClosedSheetOpen(true)}>
                    Reopen Closed Tab
                  </ContextMenuItem>
```

- [ ] **Step 9: Add the AlertDialog for close confirmation**

Add at the end of the JSX return block, just before the closing `</div>` of the outer `return`:

```tsx
      <AlertDialog open={confirmClose !== null} onOpenChange={(open) => !open && setConfirmClose(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Close tab?</AlertDialogTitle>
            <AlertDialogDescription>
              {confirmClose?.reason === "running" && "A query is still running on this tab. Closing will not stop the query — it will continue on the server."}
              {confirmClose?.reason === "dirty" && "This tab has unsaved changes."}
              {confirmClose?.reason === "both" && "This tab has unsaved changes and a running query."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => {
              if (confirmClose) closeTabMutation.mutate(Number(confirmClose.tabId));
              setConfirmClose(null);
            }}>
              Close anyway
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={confirmCloseAll} onOpenChange={setConfirmCloseAll}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Close all tabs?</AlertDialogTitle>
            <AlertDialogDescription>
              Close all {tabs.length} tabs?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => {
              closeAllTabsMutation.mutate();
              clearTabsState();
              setConfirmCloseAll(false);
            }}>
              Close All
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Sheet open={isClosedSheetOpen} onOpenChange={setIsClosedSheetOpen}>
        <SheetContent side="right" className="w-[400px]">
          <SheetHeader>
            <SheetTitle>Recently Closed</SheetTitle>
            <SheetDescription>Tabs closed in the last 30 days</SheetDescription>
          </SheetHeader>
          <ScrollArea className="flex-1 mt-4">
            {closedTabs.length === 0 && (
              <p className="text-sm text-muted-foreground text-center py-8">
                No recently closed tabs
              </p>
            )}
            {closedTabs.map(tab => (
              <div key={tab.id} className="flex items-center justify-between py-2 border-b">
                <div>
                  <p className="font-medium text-sm">{tab.label}</p>
                  <p className="text-xs text-muted-foreground">
                    {new Date(tab.created_on).toLocaleDateString()}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => reopenTabMutation.mutate(tab.id)}
                  disabled={reopenTabMutation.isPending}
                >
                  Reopen
                </Button>
              </div>
            ))}
          </ScrollArea>
        </SheetContent>
      </Sheet>
```

- [ ] **Step 10: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: no new errors (pre-existing errors OK).

- [ ] **Step 11: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat(sqllab): wire close, close-all, reopen mutations with AlertDialog and Recently Closed Sheet

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 11: End-to-end verification

**Files:** None (testing only)

- [ ] **Step 1: Run all backend tests**

Run: `cd backend && go test ./internal/delivery/http/sqllab/... -v`
Expected: all tests pass.

- [ ] **Step 2: Run backend compilation check**

Run: `cd backend && go build ./...`
Expected: no errors.

- [ ] **Step 3: Verify frontend TypeScript**

Run: `cd frontend && npx tsc --noEmit`
Expected: no new type errors.

---

## Complete Route Summary

| Method | Route | Handler | Purpose |
|--------|-------|---------|---------|
| POST | `/api/v1/sqllab/tabs` | CreateTab | Create new tab |
| GET | `/api/v1/sqllab/tabs` | ListTabs | List tabs (`?include_closed=true` for all) |
| GET | `/api/v1/sqllab/tabs/:id` | GetTab | Get single tab |
| PUT | `/api/v1/sqllab/tabs/:id` | UpdateTab | Update tab fields |
| **PUT** | `/api/v1/sqllab/tabs/:id/close` | **CloseTab** | **Soft-close single tab** |
| **POST** | `/api/v1/sqllab/tabs/close-all` | **CloseAllTabs** | **Close all/others** |
| **PUT** | `/api/v1/sqllab/tabs/:id/reopen` | **ReopenTab** | **Reopen soft-closed tab** |
| **DELETE** | `/api/v1/sqllab/tabs/:id` | **HardDeleteTab** | **Permanently delete tab** |
