# SQL-005: Update and Delete Saved Query — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement update, delete, and fork operations for saved queries — backend handlers + repository methods + frontend Sheet edit panel, AlertDialog delete, and Fork button.

**Architecture:** Backend adds 3 handler methods (UpdateSavedQuery, DeleteSavedQuery, ForkSavedQuery) to the existing sqllab handler, 3 methods to the SQLLabRepository interface + postgres implementation, and 3 routes. Frontend adds a Sheet edit panel triggered from the SavedQueriesPage dropdown, an AlertDialog for delete confirmation, and API client functions. The store's `addTab` is reused to open forked queries as new tabs.

**Tech Stack:** Go (Gin, GORM, sqlparser), React 18 + TypeScript, TanStack Query v5, Zustand, shadcn/ui (Sheet, AlertDialog, Form components), Monaco Editor

---

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `backend/internal/domain/query/saved_query.go` | No change | Model is immutable per CLAUDE.md |
| `backend/internal/domain/query/sqllab_repository.go` | Modify | Add 3 interface methods |
| `backend/internal/domain/query/sqllab_types.go` | Modify | Add response types for fork |
| `backend/internal/repository/postgres/sqllab_repo.go` | Modify | Implement 3 repository methods |
| `backend/internal/delivery/http/sqllab/handler.go` | Modify | Add 3 handler methods |
| `backend/internal/delivery/http/router.go` | Modify | Register 3 routes |
| `backend/internal/delivery/http/sqllab/handler_test.go` | Modify | Add tests for new handlers |
| `frontend/src/api/sqllab.ts` | Modify | Add API client functions |
| `frontend/src/components/sqllab/SavedQueriesList.tsx` | Modify | No changes needed (depends on page) |
| `frontend/src/pages/sqllab/SavedQueriesPage.tsx` | Modify | Wire edit/delete/fork actions, add Sheet + AlertDialog |

---

### Task 1: Add `UpdateSavedQueryRequest` type

**Files:**
- Modify: `backend/internal/domain/query/sqllab_types.go` (append after line 74)

- [ ] **Step 1: Add the request type**

After `SavedQueryListParams` (line 74), add:

```go
// UpdateSavedQueryRequest is the request body for updating a saved query.
// All fields are optional — only non-nil fields are applied.
type UpdateSavedQueryRequest struct {
	Label       *string `json:"label"`
	SQL         *string `json:"sql"`
	Schema      *string `json:"schema"`
	Catalog     *string `json:"catalog"`
	Description *string `json:"description"`
	Published   *bool   `json:"published"`
	ExtraJSON   *string `json:"extra_json"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./...`
Expected: Compiles successfully

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/query/sqllab_types.go
git commit -m "feat(sql-005): add UpdateSavedQueryRequest type"
```

---

### Task 2: Add repository interface methods

**Files:**
- Modify: `backend/internal/domain/query/sqllab_repository.go:15-18`

- [ ] **Step 1: Add 3 methods to the interface**

After `GetSavedQuery` (line 18), add:

```go
	UpdateSavedQuery(ctx context.Context, sq *SavedQuery) error
	DeleteSavedQuery(ctx context.Context, id uint, userID uint) error
	ForkSavedQuery(ctx context.Context, id uint, userID uint) (*SavedQuery, error)
```

- [ ] **Step 2: Update mock in handler_test.go for compilation**

In `backend/internal/delivery/http/sqllab/handler_test.go`, add to `mockSQLLabRepo` after `GetSavedQuery` (line 144):

```go
func (m *mockSQLLabRepo) UpdateSavedQuery(_ context.Context, sq *domainquery.SavedQuery) error {
	if m.err != nil {
		return m.err
	}
	for i, existing := range m.savedQueries {
		if existing.ID == sq.ID {
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
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build ./...`
Expected: Compiles successfully

- [ ] **Step 4: Commit**

```bash
git add backend/internal/domain/query/sqllab_repository.go backend/internal/delivery/http/sqllab/handler_test.go
git commit -m "feat(sql-005): add UpdateSavedQuery, DeleteSavedQuery, ForkSavedQuery to interface"
```

---

### Task 3: Implement repository methods in postgres

**Files:**
- Modify: `backend/internal/repository/postgres/sqllab_repo.go` (append after line 206)

- [ ] **Step 1: Implement UpdateSavedQuery**

Append after `GetSavedQuery` (line 206):

```go
func (r *sqllabRepo) UpdateSavedQuery(ctx context.Context, sq *query.SavedQuery) error {
	if sq.SQL != "" {
		sq.SQLTables = extractSQLTables(sq.SQL)
	}
	return r.db.WithContext(ctx).Save(sq).Error
}

func (r *sqllabRepo) DeleteSavedQuery(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sq query.SavedQuery
		if err := tx.Where("id = ? AND created_by_fk = ?", id, userID).First(&sq).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("not found")
			}
			return fmt.Errorf("delete saved query: find: %w", err)
		}
		// Null out FK references in tab_state
		if err := tx.Model(&query.TabState{}).Where("saved_query_id = ?", id).Update("saved_query_id", gorm.Expr("NULL")).Error; err != nil {
			return fmt.Errorf("delete saved query: null tab refs: %w", err)
		}
		if err := tx.Delete(&sq).Error; err != nil {
			return fmt.Errorf("delete saved query: %w", err)
		}
		return nil
	})
}

func (r *sqllabRepo) ForkSavedQuery(ctx context.Context, id uint, userID uint) (*query.SavedQuery, error) {
	var original query.SavedQuery
	err := r.db.WithContext(ctx).Where("id = ? AND created_by_fk = ?", id, userID).First(&original).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("not found")
		}
		return nil, fmt.Errorf("fork saved query: find: %w", err)
	}

	now := time.Now()
	fork := original
	fork.ID = 0
	fork.Label = "Copy of " + original.Label
	fork.CreatedOn = now
	fork.ChangedOn = now
	fork.SQLTables = original.SQLTables // preserve extracted tables

	if err := r.db.WithContext(ctx).Create(&fork).Error; err != nil {
		return nil, fmt.Errorf("fork saved query: create: %w", err)
	}
	return &fork, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./...`
Expected: Compiles successfully

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/postgres/sqllab_repo.go
git commit -m "feat(sql-005): implement UpdateSavedQuery, DeleteSavedQuery, ForkSavedQuery in postgres repo"
```

---

### Task 4: Add handler methods

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler.go` (append after `ListSavedQueries` at line 360)

- [ ] **Step 1: Implement UpdateSavedQuery handler**

Append after `ListSavedQueries` (line 360):

```go
func (h *Handler) UpdateSavedQuery(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid saved query id"})
		return
	}

	sq, err := h.sqllabRepo.GetSavedQuery(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve saved query"})
		return
	}
	if sq == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to update this saved query"})
		return
	}

	var req domainquery.UpdateSavedQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	if req.Label != nil {
		if strings.EqualFold(*req.Label, sq.Label) {
			// same label, skip uniqueness check
		} else {
			exists, checkErr := h.sqllabRepo.LabelExists(c.Request.Context(), userCtx.ID, *req.Label)
			if checkErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to check label uniqueness"})
				return
			}
			if exists {
				c.JSON(http.StatusConflict, gin.H{"error": "duplicate_label", "message": "A saved query with this label already exists"})
				return
			}
		}
		sq.Label = *req.Label
	}
	if req.SQL != nil {
		sq.SQL = *req.SQL
	}
	if req.Schema != nil {
		sq.Schema = *req.Schema
	}
	if req.Catalog != nil {
		sq.Catalog = *req.Catalog
	}
	if req.Description != nil {
		sq.Description = *req.Description
	}
	if req.Published != nil {
		sq.Published = *req.Published
	}
	if req.ExtraJSON != nil {
		sq.ExtraJSON = *req.ExtraJSON
	}

	sq.ChangedOn = time.Now()
	sq.ChangedByFK = userCtx.ID

	if err := h.sqllabRepo.UpdateSavedQuery(c.Request.Context(), sq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update_failed", "message": "Failed to update saved query"})
		return
	}

	c.JSON(http.StatusOK, savedQueryToResponse(sq))
}
```

- [ ] **Step 2: Implement DeleteSavedQuery handler**

```go
func (h *Handler) DeleteSavedQuery(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid saved query id"})
		return
	}

	// Verify ownership — fetch before delete
	sq, err := h.sqllabRepo.GetSavedQuery(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve saved query"})
		return
	}
	if sq == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to delete this saved query"})
		return
	}

	if err := h.sqllabRepo.DeleteSavedQuery(c.Request.Context(), uint(id), userCtx.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete_failed", "message": "Failed to delete saved query"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
```

- [ ] **Step 3: Implement ForkSavedQuery handler**

```go
func (h *Handler) ForkSavedQuery(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid saved query id"})
		return
	}

	fork, err := h.sqllabRepo.ForkSavedQuery(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to fork this saved query"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fork_failed", "message": "Failed to fork saved query"})
		return
	}

	c.JSON(http.StatusCreated, savedQueryToResponse(fork))
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd backend && go build ./...`
Expected: Compiles successfully

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler.go
git commit -m "feat(sql-005): add UpdateSavedQuery, DeleteSavedQuery, ForkSavedQuery handlers"
```

---

### Task 5: Register routes

**Files:**
- Modify: `backend/internal/delivery/http/router.go:180-181`

- [ ] **Step 1: Add routes**

After line 181 (`sqlLab.GET("/saved-queries", sqllabHandler.ListSavedQueries)`), add:

```go
				sqlLab.PUT("/saved-queries/:id", sqllabHandler.UpdateSavedQuery)
				sqlLab.DELETE("/saved-queries/:id", sqllabHandler.DeleteSavedQuery)
				sqlLab.POST("/saved-queries/:id/fork", sqllabHandler.ForkSavedQuery)
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./...`
Expected: Compiles successfully

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/router.go
git commit -m "feat(sql-005): register saved query update, delete, fork routes"
```

---

### Task 6: Add backend handler tests

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler_test.go` (append after line 594)

- [ ] **Step 1: Register test routes in newSQLLabRouter**

Update `newSQLLabRouter` to include new routes (after line 192):

```go
	sqllab.PUT("/saved-queries/:id", h.UpdateSavedQuery)
	sqllab.DELETE("/saved-queries/:id", h.DeleteSavedQuery)
	sqllab.POST("/saved-queries/:id/fork", h.ForkSavedQuery)
```

- [ ] **Step 2: Add UpdateSavedQuery tests**

Append to handler_test.go:

```go
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
```

- [ ] **Step 3: Run tests**

Run: `cd backend && go test ./internal/delivery/http/sqllab/... -v -count=1`
Expected: All tests PASS (existing + new 6 tests)

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler_test.go
git commit -m "test(sql-005): add tests for update, delete, fork saved query handlers"
```

---

### Task 7: Add frontend API client functions

**Files:**
- Modify: `frontend/src/api/sqllab.ts` (append after `fetchSavedQueries` at line 162)

- [ ] **Step 1: Add update, delete, fork API functions**

Append after `fetchSavedQueries` (line 162):

```typescript
export interface UpdateSavedQueryRequest {
  label?: string;
  sql?: string;
  schema?: string;
  catalog?: string;
  description?: string;
  published?: boolean;
  extra_json?: string;
}

export async function updateSavedQuery(id: number, data: UpdateSavedQueryRequest): Promise<SavedQueryResponse> {
  return request<SavedQueryResponse>(`/api/v1/sqllab/saved-queries/${id}`, {
    method: "PUT",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}

export async function deleteSavedQuery(id: number): Promise<{ deleted: boolean }> {
  return request<{ deleted: boolean }>(`/api/v1/sqllab/saved-queries/${id}`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
}

export async function forkSavedQuery(id: number): Promise<SavedQueryResponse> {
  return request<SavedQueryResponse>(`/api/v1/sqllab/saved-queries/${id}/fork`, {
    method: "POST",
    headers: getAuthHeaders(),
  });
}
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/sqllab.ts
git commit -m "feat(sql-005): add updateSavedQuery, deleteSavedQuery, forkSavedQuery API functions"
```

---

### Task 8: Wire edit Sheet, delete AlertDialog, fork into SavedQueriesPage

**Files:**
- Modify: `frontend/src/pages/sqllab/SavedQueriesPage.tsx`

- [ ] **Step 1: Add imports**

Add to existing imports:

```typescript
import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import Editor from "@monaco-editor/react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
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
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { DataTable } from "@/components/ui/data-table";
import { Skeleton } from "@/components/ui/skeleton";
import { MoreHorizontal } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { useSqlLabStore } from "@/stores/sqlLabStore";
import {
  fetchSavedQueries,
  updateSavedQuery,
  deleteSavedQuery,
  forkSavedQuery,
  type SavedQueryResponse,
} from "@/api/sqllab";
import type { ColumnDef } from "@tanstack/react-table";
```

- [ ] **Step 2: Rewrite the component with Sheet, AlertDialog, and fork logic**

```typescript
export default function SavedQueriesPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const addTab = useSqlLabStore((s) => s.addTab);
  const [search, setSearch] = useState("");

  // Sheet state
  const [editingQuery, setEditingQuery] = useState<SavedQueryResponse | null>(null);
  const [editLabel, setEditLabel] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editSQL, setEditSQL] = useState("");
  const [editPublished, setEditPublished] = useState(false);

  // Delete dialog state
  const [deletingQuery, setDeletingQuery] = useState<SavedQueryResponse | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["saved-queries", { q: search }],
    queryFn: () => fetchSavedQueries({ q: search || undefined, limit: 50 }),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...rest }: { id: number; label?: string; description?: string; sql?: string; published?: boolean }) =>
      updateSavedQuery(id, rest),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["saved-queries"] });
      toast("Saved query updated");
      setEditingQuery(null);
    },
    onError: (error: Error) => {
      toast("Failed to update: " + error.message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteSavedQuery(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["saved-queries"] });
      toast("Saved query deleted");
      setDeletingQuery(null);
    },
    onError: (error: Error) => {
      toast("Failed to delete: " + error.message);
    },
  });

  const forkMutation = useMutation({
    mutationFn: (id: number) => forkSavedQuery(id),
    onSuccess: (forked) => {
      queryClient.invalidateQueries({ queryKey: ["saved-queries"] });
      addTab();
      const tabs = useSqlLabStore.getState().tabs;
      const newTabId = tabs[tabs.length - 1]?.id;
      if (newTabId) {
        useSqlLabStore.getState().updateTabSql(newTabId, forked.sql);
        useSqlLabStore.getState().updateTabLabel(newTabId, forked.label);
      }
      toast("Forked to new tab");
      navigate("/sqllab");
    },
    onError: (error: Error) => {
      toast("Failed to fork: " + error.message);
    },
  });

  const handleOpenEdit = (sq: SavedQueryResponse) => {
    setEditingQuery(sq);
    setEditLabel(sq.label);
    setEditDescription(sq.description || "");
    setEditSQL(sq.sql);
    setEditPublished(sq.published);
  };

  const handleSaveEdit = () => {
    if (!editingQuery) return;
    updateMutation.mutate({
      id: editingQuery.id,
      label: editLabel.trim() || undefined,
      description: editDescription.trim() || undefined,
      sql: editSQL || undefined,
      published: editPublished,
    });
  };

  const columns = useMemo<ColumnDef<SavedQueryResponse>[]>(() => [
    {
      accessorKey: "label",
      header: "Name",
      cell: ({ getValue }) => (
        <span className="font-medium">{getValue() as string}</span>
      ),
    },
    {
      accessorKey: "db_id",
      header: "Database",
      cell: ({ getValue }) => (
        <span className="text-muted-foreground text-sm">DB #{getValue() as number}</span>
      ),
    },
    {
      accessorKey: "schema",
      header: "Schema",
    },
    {
      accessorKey: "changed_on",
      header: "Modified",
      cell: ({ getValue }) => new Date(getValue() as string).toLocaleDateString(),
    },
    {
      accessorKey: "published",
      header: "Status",
      cell: ({ getValue }) =>
        getValue() ? (
          <Badge variant="secondary">Published</Badge>
        ) : (
          <Badge variant="outline">Private</Badge>
        ),
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const sq = row.original;
        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => navigate(`/sqllab?load=${sq.id}`)}>
                Load in SQL Lab
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleOpenEdit(sq)}>
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => forkMutation.mutate(sq.id)}>
                Fork
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setDeletingQuery(sq)}>
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ], [navigate, forkMutation]);

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Saved Queries</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Browse and load your saved queries
          </p>
        </div>
        <Button onClick={() => navigate("/sqllab")}>
          SQL Lab
        </Button>
      </div>

      <div className="mb-4">
        <Input
          placeholder="Search saved queries..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : (
        <DataTable data={data?.items ?? []} columns={columns} />
      )}

      {/* Edit Sheet */}
      <Sheet open={editingQuery !== null} onOpenChange={(open) => { if (!open) setEditingQuery(null); }}>
        <SheetContent side="right" className="w-[500px] sm:max-w-[500px] flex flex-col">
          <SheetHeader>
            <SheetTitle>Edit Saved Query</SheetTitle>
            <SheetDescription>
              Update the query details. Changes are saved immediately.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 py-4 overflow-y-auto">
            <div>
              <Label htmlFor="edit-label">Name</Label>
              <Input
                id="edit-label"
                value={editLabel}
                onChange={(e) => setEditLabel(e.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="edit-desc">Description</Label>
              <Textarea
                id="edit-desc"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                placeholder="What does this query do?"
                rows={3}
              />
            </div>
            <div>
              <Label htmlFor="edit-sql">SQL</Label>
              <div className="border rounded-md overflow-hidden h-[200px]">
                <Editor
                  language="sql"
                  theme="vs-dark"
                  value={editSQL}
                  onChange={(v) => setEditSQL(v || "")}
                  options={{
                    minimap: { enabled: false },
                    scrollBeyondLastLine: false,
                    lineNumbers: "on",
                    fontSize: 13,
                  }}
                />
              </div>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Label htmlFor="edit-published">Published</Label>
                <p className="text-xs text-muted-foreground">
                  Visible to all team members in your organization
                </p>
              </div>
              <Switch
                id="edit-published"
                checked={editPublished}
                onCheckedChange={setEditPublished}
              />
            </div>
          </div>
          <SheetFooter>
            <Button variant="outline" onClick={() => setEditingQuery(null)}>
              Cancel
            </Button>
            <Button onClick={handleSaveEdit} disabled={updateMutation.isPending || !editLabel.trim()}>
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      {/* Delete AlertDialog */}
      <AlertDialog open={deletingQuery !== null} onOpenChange={(open) => { if (!open) setDeletingQuery(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {deletingQuery?.label}?</AlertDialogTitle>
            <AlertDialogDescription>
              This cannot be undone. Any tabs referencing this saved query will be updated.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => { if (deletingQuery) deleteMutation.mutate(deletingQuery.id); }}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/sqllab/SavedQueriesPage.tsx
git commit -m "feat(sql-005): wire edit Sheet, delete AlertDialog, fork into SavedQueriesPage"
```

---

## Verification

After all tasks are complete:

1. **Backend tests:**
   ```bash
   cd backend && go test ./internal/delivery/http/sqllab/... -v -count=1
   ```
   Expected: All tests pass (existing + 6 new SQL-005 tests)

2. **Backend build:**
   ```bash
   cd backend && go build ./...
   ```
   Expected: No compilation errors

3. **Frontend type check:**
   ```bash
   cd frontend && npx tsc --noEmit
   ```
   Expected: No TypeScript errors

4. **Manual E2E verification:**
   - Start backend + frontend dev servers
   - Navigate to `/sqllab/saved-queries`
   - Verify: row dropdown has Edit → opens Sheet with pre-filled fields
   - Verify: change label and save → 200, list refreshes
   - Verify: Fork → creates copy, opens new SQL Lab tab with forked SQL
   - Verify: Delete → confirmation dialog → deleted, list refreshes
   - Verify: Editing another user's query → 403
   - Verify: Duplicate label on edit → 409
