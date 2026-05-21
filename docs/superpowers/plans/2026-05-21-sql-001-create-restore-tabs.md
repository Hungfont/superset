# SQL-001: Create and Restore SQL Lab Editor Tabs — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement tab state persistence for SQL Lab — users can create editor tabs saved to `tab_state` and restored on page load.

**Architecture:** New backend handler + repository for `/api/v1/sqllab/tabs` CRUD. Frontend gets resizable 3-pane layout with shadcn components, Monaco editor replacing textarea, and Zustand store hydration from API on mount.

**Tech Stack:** Go/Gin/GORM (backend), React 18/TanStack Query v5/Zustand/shadcn/ui/Monaco Editor (frontend)

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `backend/internal/domain/query/tab_state.go` | Read-only | TabState model (exists, do NOT modify per CLAUDE.md) |
| `backend/internal/repository/postgres/sqllab_repo.go` | **Create** | TabStateRepository — Create, ListByUser, GetByID |
| `backend/internal/domain/query/sqllab_repository.go` | **Create** | Repository interface definition |
| `backend/internal/domain/query/sqllab_types.go` | **Create** | Request/response types for SQL Lab handler |
| `backend/internal/delivery/http/sqllab/handler.go` | **Create** | HTTP handler — CreateTab, ListTabs, GetTab |
| `backend/internal/delivery/http/router.go` | Modify | Add sqllab routes + handler param |
| `backend/cmd/api/main.go` | Modify | Wire sqllabRepo, sqllabHandler |
| `frontend/src/api/sqllab.ts` | **Create** | API client — fetch tabs, create tab |
| `frontend/src/stores/sqlLabStore.ts` | Modify | Add initTabs action, isDirty, latestQueryStatus |
| `frontend/src/pages/sqllab/SQLLabPage.tsx` | Modify | Resizable 3-pane layout, Monaco editor, restored tabs |

---

### Task 1: Create SQL Lab repository interface and types

**Files:**
- Create: `backend/internal/domain/query/sqllab_repository.go`
- Create: `backend/internal/domain/query/sqllab_types.go`

- [ ] **Step 1: Write the repository interface**

```go
// backend/internal/domain/query/sqllab_repository.go
package query

import "context"

type SQLLabRepository interface {
	Create(ctx context.Context, tab *TabState) error
	ListByUser(ctx context.Context, userID uint) ([]*TabState, error)
	GetByID(ctx context.Context, id uint, userID uint) (*TabState, error)
}
```

- [ ] **Step 2: Write request/response types**

```go
// backend/internal/domain/query/sqllab_types.go
package query

type CreateTabRequest struct {
	DbID       uint   `json:"db_id" binding:"required"`
	Schema     string `json:"schema"`
	Catalog    string `json:"catalog"`
	SQL        string `json:"sql"`
	QueryLimit int    `json:"query_limit"`
}

type TabResponse struct {
	ID                  uint   `json:"id"`
	Label               string `json:"label"`
	DbID                uint   `json:"db_id"`
	Schema              string `json:"schema"`
	Catalog             string `json:"catalog"`
	SQL                 string `json:"sql"`
	Active              bool   `json:"active"`
	QueryLimit          int    `json:"query_limit"`
	LatestQueryID       string `json:"latest_query_id"`
	LatestQueryStatus   string `json:"latest_query_status"`
	HideLeftBar         bool   `json:"hide_left_bar"`
	CreatedOn           string `json:"created_on"`
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/query/sqllab_repository.go backend/internal/domain/query/sqllab_types.go
git commit -m "feat: add SQLLabRepository interface and tab types"
```

---

### Task 2: Implement TabStateRepository (Postgres)

**Files:**
- Create: `backend/internal/repository/postgres/sqllab_repo.go`

- [ ] **Step 1: Write the repository implementation**

```go
// backend/internal/repository/postgres/sqllab_repo.go
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	query "superset/auth-service/internal/domain/query"
)

type sqllabRepo struct {
	db *gorm.DB
}

func NewSQLLabRepository(db *gorm.DB) query.SQLLabRepository {
	return &sqllabRepo{db: db}
}

func (r *sqllabRepo) Create(ctx context.Context, tab *query.TabState) error {
	return r.db.WithContext(ctx).Create(tab).Error
}

func (r *sqllabRepo) ListByUser(ctx context.Context, userID uint) ([]*query.TabState, error) {
	var tabs []*query.TabState
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND active = ?", userID, true).
		Order("created_on ASC").
		Find(&tabs).Error
	if err != nil {
		return nil, fmt.Errorf("listing tabs by user: %w", err)
	}
	return tabs, nil
}

func (r *sqllabRepo) GetByID(ctx context.Context, id uint, userID uint) (*query.TabState, error) {
	var tab query.TabState
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&tab).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting tab by id: %w", err)
	}
	return &tab, nil
}

var _ query.SQLLabRepository = (*sqllabRepo)(nil)
```

You need to add `"errors"` and `"gorm.io/gorm"` imports.

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./internal/repository/postgres/
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/postgres/sqllab_repo.go
git commit -m "feat: implement TabStateRepository with Create, ListByUser, GetByID"
```

---

### Task 3: Implement SQL Lab HTTP handler

**Files:**
- Create: `backend/internal/delivery/http/sqllab/handler.go`

- [ ] **Step 1: Write the handler**

```go
// backend/internal/delivery/http/sqllab/handler.go
package sqllab

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	query "superset/auth-service/internal/domain/query"
	domdb "superset/auth-service/internal/domain/database"
)

type Handler struct {
	sqllabRepo   query.SQLLabRepository
	databaseRepo domdb.DatabaseRepository
}

func NewHandler(sqllabRepo query.SQLLabRepository, databaseRepo domdb.DatabaseRepository) *Handler {
	return &Handler{sqllabRepo: sqllabRepo, databaseRepo: databaseRepo}
}

func (h *Handler) CreateTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	var req query.CreateTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	visible, err := h.databaseRepo.IsVisibleToUser(c.Request.Context(), req.DbID, userCtx.ID)
	if err != nil || !visible {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "db_not_visible", "message": "Database not accessible"})
		return
	}

	label, err := h.generateLabel(c.Request.Context(), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}

	tab := &query.TabState{
		UserID:     userCtx.ID,
		DbID:       req.DbID,
		Schema:     req.Schema,
		Catalog:    req.Catalog,
		SQL:        req.SQL,
		QueryLimit: req.QueryLimit,
		Label:      label,
		Active:     true,
	}

	if err := h.sqllabRepo.Create(c.Request.Context(), tab); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": tab.ID, "label": tab.Label, "active": true})
}

func (h *Handler) ListTabs(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	tabs, err := h.sqllabRepo.ListByUser(c.Request.Context(), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}

	resp := make([]query.TabResponse, 0, len(tabs))
	for _, t := range tabs {
		resp = append(resp, tabToResponse(t))
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	tab, err := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}
	if tab == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
		return
	}

	c.JSON(http.StatusOK, tabToResponse(tab))
}

func (h *Handler) generateLabel(ctx context.Context, userID uint) (string, error) {
	tabs, err := h.sqllabRepo.ListByUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("counting untitled tabs: %w", err)
	}

	maxN := 0
	prefix := "Untitled Query "
	for _, t := range tabs {
		if len(t.Label) > len(prefix) && t.Label[:len(prefix)] == prefix {
			if n, err := strconv.Atoi(t.Label[len(prefix):]); err == nil && n > maxN {
				maxN = n
			}
		}
	}
	return fmt.Sprintf("%s%d", prefix, maxN+1), nil
}

func getUserContext(c *gin.Context) (domain.UserContext, bool) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return domain.UserContext{}, false
	}
	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return domain.UserContext{}, false
	}
	return userCtx, true
}

func tabToResponse(t *query.TabState) query.TabResponse {
	return query.TabResponse{
		ID:            t.ID,
		Label:         t.Label,
		DbID:          t.DbID,
		Schema:        t.Schema,
		Catalog:       t.Catalog,
		SQL:           t.SQL,
		Active:        t.Active,
		QueryLimit:    t.QueryLimit,
		LatestQueryID: t.LatestQueryID,
		HideLeftBar:   t.HideLeftBar,
		CreatedOn:     t.CreatedOn.Format("2006-01-02T15:04:05Z07:00"),
	}
}
```

You need to add missing imports. For `ctx context.Context`, add `"context"` to imports. For `domain.UserContext`, use the correct import path which is `"superset/auth-service/internal/domain"` — check an existing handler for the exact path.

- [ ] **Step 2: Check correct imports from existing handlers**

Read `backend/internal/delivery/http/query/handler.go` and find the exact import path for `domain.UserContext` and `domdb.DatabaseRepository`.

- [ ] **Step 3: Fix imports and verify compilation**

```bash
cd backend && go build ./internal/delivery/http/sqllab/
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler.go
git commit -m "feat: implement SQL Lab handler with CreateTab, ListTabs, GetTab"
```

---

### Task 4: Wire routes and handler in router.go

**Files:**
- Modify: `backend/internal/delivery/http/router.go`

- [ ] **Step 1: Add sqllab handler parameter to NewRouter**

Add to the function signature, after the last existing handler parameter (`wsHandler *httpquery.WSHandler`):

```go
	sqllabHandler *httpsqllab.Handler,
```

After the last existing handler parameter in the function signature. The closing `) *gin.Engine {` should come right after.

- [ ] **Step 2: Add routes inside the protected group**

After all existing routes inside the `protected` group, add:

```go
		sqllab := protected.Group("/sqllab")
		{
			sqllab.POST("/tabs", sqllabHandler.CreateTab)
			sqllab.GET("/tabs", sqllabHandler.ListTabs)
			sqllab.GET("/tabs/:id", sqllabHandler.GetTab)
		}
```

- [ ] **Step 3: Add the import**

```go
	httpsqllab "superset/auth-service/internal/delivery/http/sqllab"
```

Add this among the other handler imports at the top of the file.

- [ ] **Step 4: Verify compilation**

```bash
cd backend && go build ./internal/delivery/http/
```
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/router.go
git commit -m "feat: add sqllab tabs routes and handler wiring"
```

---

### Task 5: Wire dependencies in main.go

**Files:**
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Create sqllab repository**

After the `queryRepo` creation line (around line 105), add:

```go
	sqllabRepo := repopostgres.NewSQLLabRepository(db)
```

- [ ] **Step 2: Create sqllab handler**

After all handler creation lines (after `wsHandler := ...`), add:

```go
	sqllabHandler := httpsqllab.NewHandler(sqllabRepo, databaseRepo)
```

- [ ] **Step 3: Find the import for httpsqllab**

```go
	httpsqllab "superset/auth-service/internal/delivery/http/sqllab"
```

Add among the other handler imports.

- [ ] **Step 4: Pass sqllabHandler to NewRouter**

Add `sqllabHandler,` to the `NewRouter(...)` call arguments.

- [ ] **Step 5: Verify compilation**

```bash
cd backend && go build ./cmd/api/
```
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add backend/cmd/api/main.go
git commit -m "feat: wire sqllab repo and handler in main"
```

---

### Task 6: Create frontend API client for SQL Lab

**Files:**
- Create: `frontend/src/api/sqllab.ts`

- [ ] **Step 1: Write the API client**

```typescript
// frontend/src/api/sqllab.ts
import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

function getAuthHeaders(): Record<string, string> {
  const token = useAuthStore.getState().accessToken;
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

export interface TabStateResponse {
  id: number;
  label: string;
  db_id: number;
  schema: string;
  catalog: string;
  sql: string;
  active: boolean;
  query_limit: number;
  latest_query_id: string;
  latest_query_status: string;
  hide_left_bar: boolean;
  created_on: string;
}

export interface CreateTabRequest {
  db_id: number;
  schema?: string;
  catalog?: string;
  sql?: string;
  query_limit?: number;
}

export interface CreateTabResponse {
  id: number;
  label: string;
  active: boolean;
}

export async function fetchTabs(): Promise<TabStateResponse[]> {
  return request<TabStateResponse[]>("/api/v1/sqllab/tabs", {
    method: "GET",
    headers: getAuthHeaders(),
  });
}

export async function createTab(data: CreateTabRequest): Promise<CreateTabResponse> {
  return request<CreateTabResponse>("/api/v1/sqllab/tabs", {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}

export async function fetchTab(id: number): Promise<TabStateResponse> {
  return request<TabStateResponse>(`/api/v1/sqllab/tabs/${id}`, {
    method: "GET",
    headers: getAuthHeaders(),
  });
}
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit src/api/sqllab.ts
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/sqllab.ts
git commit -m "feat: add SQL Lab API client for tabs endpoints"
```

---

### Task 7: Update Zustand store with initTabs and new fields

**Files:**
- Modify: `frontend/src/stores/sqlLabStore.ts`

- [ ] **Step 1: Add new fields to SqlLabTab interface**

After the existing fields in `SqlLabTab`, add:

```typescript
  isDirty?: boolean;
  latestQueryStatus?: string;
```

- [ ] **Step 2: Add the initTabs action**

Add to `SqlLabState` interface:

```typescript
  initTabs: (tabs: TabStateFromAPI[]) => void;
```

And add the type alias near the top of the file (after imports):

```typescript
interface TabStateFromAPI {
  id: number;
  label: string;
  db_id: number;
  schema: string;
  sql: string;
  query_limit: number;
  latest_query_status: string;
}
```

- [ ] **Step 3: Implement initTabs in the store creator**

Add inside the `set(` call in the `create<SqlLabState>` function:

```typescript
  initTabs: (tabs) =>
    set((state) => ({
      tabs: tabs.map((t) => {
        const existing = state.tabs.find((et) => et.id === String(t.id));
        if (existing) return existing;
        return {
          id: String(t.id),
          title: t.label,
          sql: t.sql || "",
          databaseId: t.db_id,
          schema: t.schema || "public",
          catalog: t.catalog,
          result: null,
          status: "idle" as const,
          error: null,
          isDirty: false,
          latestQueryStatus: t.latest_query_status,
        };
      }),
      activeTabId: state.activeTabId || (tabs.length > 0 ? String(tabs[0].id) : null),
      databaseId: state.databaseId || (tabs.length > 0 ? tabs[0].db_id : null),
    })),
```

- [ ] **Step 4: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/sqlLabStore.ts
git commit -m "feat: add initTabs action and isDirty/latestQueryStatus fields to sqlLabStore"
```

---

### Task 8: Install new frontend dependencies

**Files:**
- Modify: `frontend/package.json` (via npm install)

- [ ] **Step 1: Install Monaco editor**

```bash
cd frontend && npm install @monaco-editor/react
```

- [ ] **Step 2: Install shadcn Resizable component**

```bash
cd frontend && npx shadcn@latest add resizable
```

- [ ] **Step 3: Install shadcn Collapsible component**

```bash
cd frontend && npx shadcn@latest add collapsible
```

- [ ] **Step 4: Install shadcn DropdownMenu component (if not present)**

First check if `frontend/src/components/ui/dropdown-menu.tsx` exists. If not:

```bash
cd frontend && npx shadcn@latest add dropdown-menu
```

- [ ] **Step 5: Verify all packages installed**

```bash
cd frontend && ls node_modules/@monaco-editor/react
```
Expected: directory exists.

```bash
cd frontend && ls src/components/ui/resizable.tsx src/components/ui/collapsible.tsx src/components/ui/dropdown-menu.tsx
```
Expected: all files exist.

- [ ] **Step 6: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/components/ui/resizable.tsx frontend/src/components/ui/collapsible.tsx
git commit -m "feat: install Monaco editor, shadcn resizable and collapsible components"
```

---

### Task 9: Rewrite SQLLabPage with resizable layout, Monaco, tab restore

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx`

- [ ] **Step 1: Read the current file for reference before rewriting**

Read `frontend/src/pages/sqllab/SQLLabPage.tsx` to understand the current full structure.

- [ ] **Step 2: Rewrite SQLLabPage**

Write the new component. The key changes:
1. Wrap layout in `ResizablePanelGroup` with 3 panels (left: schema browser, center: editor+results, right: optional)
2. On mount, call `useQuery("sqllab-tabs", fetchTabs)` → `initTabs` to hydrate store
3. If empty response, auto-create a default tab via `createTab` mutation
4. Replace `<textarea>` with `<Editor>` from `@monaco-editor/react`
5. Tab bar uses shadcn `Tabs` as before but adds status icon from `latestQueryStatus`
6. Auto-save: `useEffect` watching `sql` → 1000ms debounce → (placeholder for SQL-002, log for now)

```tsx
// frontend/src/pages/sqllab/SQLLabPage.tsx
import { useEffect, useCallback, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Editor from "@monaco-editor/react";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/hooks/use-toast";
import { useSqlLabStore } from "@/stores/sqlLabStore";
import { fetchTabs, createTab } from "@/api/sqllab";
import { executeQuery } from "@/api/queries";
import { Plus, X, Play, Loader2 } from "lucide-react";

export default function SQLLabPage() {
  const { tabs, activeTabId, addTab, removeTab, setActiveTab, updateTabSql, initTabs, setTabResult, setTabStatus, setTabError } =
    useSqlLabStore();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { isLoading } = useQuery({
    queryKey: ["sqllab-tabs"],
    queryFn: fetchTabs,
    staleTime: 0,
    onSuccess: (data) => {
      if (data.length === 0) {
        createFirstTab();
      } else {
        initTabs(data);
      }
    },
  });

  const createMutation = useMutation({
    mutationFn: createTab,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["sqllab-tabs"] }),
  });

  const executeMutation = useMutation({
    mutationFn: executeQuery,
    onSuccess: (data, variables) => {
      const tab = tabs.find((t) => t.id === activeTabId);
      if (tab) {
        setTabResult(tab.id, data);
        setTabStatus(tab.id, "success");
      }
    },
    onError: (error: Error) => {
      if (activeTabId) {
        setTabError(activeTabId, error.message);
        setTabStatus(activeTabId, "error");
      }
    },
  });

  const createFirstTab = useCallback(() => {
    createMutation.mutate({ db_id: 1 }); // default db_id
  }, []);

  const activeTab = tabs.find((t) => t.id === activeTabId);

  // Auto-save: debounce 1000ms on sql change (placeholder for SQL-002)
  const prevSqlRef = useRef<string | undefined>();
  useEffect(() => {
    if (!activeTab || activeTab.sql === prevSqlRef.current) return;
    prevSqlRef.current = activeTab.sql;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      // SQL-002: PUT /api/v1/sqllab/tabs/:id with sql
      console.log("[auto-save] tab", activeTab.id, "sql length:", activeTab.sql.length);
    }, 1000);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [activeTab?.sql]);

  const handleRun = useCallback(() => {
    if (!activeTab || !activeTab.databaseId || !activeTab.sql.trim()) return;
    setTabStatus(activeTab.id, "running");
    executeMutation.mutate({
      database_id: activeTab.databaseId,
      sql: activeTab.sql,
      schema: activeTab.schema,
      limit: 100,
    });
  }, [activeTab, executeMutation]);

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="h-screen">
      <ResizablePanelGroup direction="horizontal" className="h-full">
        {/* Left: Schema Browser (placeholder for SQL-006) */}
        <ResizablePanel defaultSize={20} minSize={15} maxSize={30}>
          <div className="h-full border-r p-3">
            <h3 className="text-sm font-semibold mb-2">Schema Browser</h3>
            <Skeleton className="h-4 w-full mb-2" />
            <Skeleton className="h-4 w-3/4 mb-2" />
            <Skeleton className="h-4 w-5/6 mb-2" />
            <Skeleton className="h-4 w-2/3 mb-2" />
            <Skeleton className="h-4 w-4/5" />
          </div>
        </ResizablePanel>

        <ResizableHandle withHandle />

        {/* Center: Tabs + Editor + Results */}
        <ResizablePanel defaultSize={80}>
          <ResizablePanelGroup direction="vertical" className="h-full">
            {/* Top: Tabs + Editor */}
            <ResizablePanel defaultSize={55}>
              <div className="flex flex-col h-full">
                {/* Tab Bar */}
                <div className="flex items-center border-b px-2 py-1 gap-1 overflow-x-auto" role="tablist" aria-label="SQL Editor Tabs">
                  {tabs.map((tab) => (
                    <div
                      key={tab.id}
                      role="tab"
                      aria-selected={tab.id === activeTabId}
                      className={`flex items-center gap-1 px-3 py-1.5 text-sm rounded-t-md cursor-pointer border border-b-0 whitespace-nowrap ${
                        tab.id === activeTabId
                          ? "bg-background border-border text-foreground"
                          : "bg-muted/50 border-transparent text-muted-foreground hover:bg-muted"
                      }`}
                      onClick={() => setActiveTab(tab.id)}
                    >
                      {tab.latestQueryStatus === "success" && (
                        <span className="w-2 h-2 rounded-full bg-green-500" />
                      )}
                      {tab.latestQueryStatus === "failed" && (
                        <span className="w-2 h-2 rounded-full bg-red-500" />
                      )}
                      {tab.latestQueryStatus === "running" && (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      )}
                      {tab.isDirty && !tab.latestQueryStatus && (
                        <span className="text-yellow-500">&bull;</span>
                      )}
                      <span>{tab.title}</span>
                      <button
                        className="ml-1 rounded-sm hover:bg-muted-foreground/20 p-0.5"
                        onClick={(e) => { e.stopPropagation(); removeTab(tab.id); }}
                        aria-label={`Close ${tab.title}`}
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 flex-shrink-0"
                    onClick={() => createMutation.mutate({ db_id: tabs[0]?.databaseId || 1 })}
                    aria-label="New tab"
                  >
                    <Plus className="h-4 w-4" />
                  </Button>
                </div>

                {/* Toolbar */}
                {activeTab && (
                  <div className="flex items-center gap-2 px-3 py-1.5 border-b bg-muted/20">
                    <Button size="sm" onClick={handleRun} disabled={executeMutation.isPending} aria-label="Run Query (Ctrl+Enter)">
                      <Play className="h-3.5 w-3.5 mr-1" />
                      Run
                    </Button>
                    {activeTab.databaseId && (
                      <Badge variant="outline" className="text-xs">
                        {activeTab.databaseId}
                      </Badge>
                    )}
                    {activeTab.schema && (
                      <Badge variant="outline" className="text-xs">
                        {activeTab.schema}
                      </Badge>
                    )}
                  </div>
                )}

                {/* Monaco Editor */}
                <div className="flex-1 min-h-0">
                  {activeTab && (
                    <Editor
                      height="100%"
                      language="sql"
                      theme="vs-dark"
                      value={activeTab.sql}
                      onChange={(value) =>
                        updateTabSql(activeTab.id, value || "")
                      }
                      options={{
                        minimap: { enabled: false },
                        lineNumbers: "on",
                        fontSize: 13,
                        wordWrap: "on",
                      }}
                      aria-label="SQL Editor"
                      loading={<Skeleton className="h-full w-full" />}
                    />
                  )}
                </div>
              </div>
            </ResizablePanel>

            <ResizableHandle withHandle />

            {/* Bottom: Results */}
            <ResizablePanel defaultSize={45}>
              <div className="h-full border-t">
                <Tabs defaultValue="results" className="h-full flex flex-col">
                  <TabsList className="px-3 pt-2 border-b rounded-none justify-start bg-transparent">
                    <TabsTrigger value="results">Results</TabsTrigger>
                    <TabsTrigger value="details">Query Details</TabsTrigger>
                  </TabsList>
                  <TabsContent value="results" className="flex-1 overflow-auto p-0">
                    {activeTab?.status === "running" && (
                      <div className="flex items-center justify-center h-full py-8">
                        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                      </div>
                    )}
                    {activeTab?.status === "error" && (
                      <div className="p-4 text-destructive text-sm">{activeTab.error}</div>
                    )}
                    {activeTab?.status === "success" && activeTab.result?.data && (
                      <ResultDataTable data={activeTab.result.data} columns={activeTab.result.columns} />
                    )}
                    {(!activeTab || activeTab.status === "idle") && (
                      <div className="flex items-center justify-center h-full py-8 text-muted-foreground text-sm">
                        Run a query to see results
                      </div>
                    )}
                  </TabsContent>
                  <TabsContent value="details" className="p-4">
                    <p className="text-sm text-muted-foreground">Query details will appear here after execution.</p>
                  </TabsContent>
                </Tabs>
              </div>
            </ResizablePanel>
          </ResizablePanelGroup>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}

function ResultDataTable({ data, columns }: { data: Record<string, unknown>[]; columns: { name: string }[] }) {
  if (!data.length) return null;
  const cols = columns.map((c) => c.name);
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/50">
            {cols.map((col) => (
              <th key={col} className="px-3 py-2 text-left font-medium whitespace-nowrap">
                {col}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.slice(0, 100).map((row, i) => (
            <tr key={i} className="border-b hover:bg-muted/30">
              {cols.map((col) => (
                <td key={col} className="px-3 py-1.5 whitespace-nowrap">
                  {String(row[col] ?? "")}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

**Important notes for the implementer:**
- `onSuccess` on `useQuery` is deprecated in TanStack Query v5. Use a `useEffect` watching `data` instead.
- The `executeQuery` import and its types must match what exists in `@/api/queries`. Check that file for the exact import.
- The `Collapsible` import is included for the schema browser placeholder.
- Replace the `ResultDataTable` with the existing `DataTable` component once confirmed working.

- [ ] **Step 2: Fix the deprecated `onSuccess` in useQuery**

Replace the `useQuery` call so it uses `useEffect` for side effects:

```tsx
  const { data: tabsData, isLoading } = useQuery({
    queryKey: ["sqllab-tabs"],
    queryFn: fetchTabs,
    staleTime: 0,
  });

  useEffect(() => {
    if (!tabsData) return;
    if (tabsData.length === 0) {
      createFirstTab();
    } else {
      initTabs(tabsData);
    }
  }, [tabsData]);
```

- [ ] **Step 3: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```
Expected: no errors. Fix any type issues.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat: rewrite SQLLabPage with resizable panels, Monaco editor, and tab restore from API"
```

---

### Task 10: End-to-end verification

**Files:** None (verification only)

- [ ] **Step 1: Start the backend**

```bash
cd backend && go run ./cmd/api/
```
Expected: server starts without errors, and `/api/v1/sqllab/tabs` routes are registered.

- [ ] **Step 2: Start the frontend**

```bash
cd frontend && npm run dev
```
Expected: dev server starts.

- [ ] **Step 3: Test the golden path in browser**

1. Open `http://localhost:5173/sqllab`
2. Verify: Tabs load from API (or auto-create default tab if empty)
3. Verify: Monaco editor renders, can type SQL
4. Verify: Resizable panels work — drag handles to resize
5. Verify: Click "+" to create new tab
6. Verify: Close tab with × button
7. Verify: Run query and see results in bottom panel

- [ ] **Step 4: Test API directly with curl**

```bash
curl -X POST http://localhost:8080/api/v1/sqllab/tabs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"db_id": 1}'
```
Expected: `201 {"id": 1, "label": "Untitled Query 1", "active": true}`

```bash
curl http://localhost:8080/api/v1/sqllab/tabs \
  -H "Authorization: Bearer <token>"
```
Expected: `200 [{...}]` with created tabs.

- [ ] **Step 5: Fix any issues found**

Track issues found, fix them, re-verify.

---

## Out of Scope (not in this plan)
- SQL-002: PUT /api/v1/sqllab/tabs/:id auto-save endpoint
- SQL-003: Tab close/delete/soft-close endpoints
- SQL-004: Saved queries CRUD
- SQL-006: Schema browser data (only layout placeholder)
- Result persistence on tab restore
- Tab sharing between users
- Double-click inline rename (future UX polish)
- Right-click context menu on tabs (future UX polish)
