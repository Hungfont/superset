# SQL-001: Create and Restore SQL Lab Editor Tabs — Design

**Date:** 2026-05-21
**Spec:** docs/requirement/SQLLAB_SERVICE_05_PHASE2.md
**Dependencies:** AUTH-004 (user context), DBC-001 (db_id visibility)

## Overview

Implement tab state persistence for SQL Lab. Users can create editor tabs that are saved to the `tab_state` table and restored on page load across devices/browsers. Editor state only (SQL, database, schema, label) — results are NOT persisted on restore.

## Backend

### API Endpoints

| Method | Endpoint | Purpose | Request Body | Response |
|--------|----------|---------|-------------|----------|
| `POST` | `/api/v1/sqllab/tabs` | Create a new tab | `{db_id, schema?, catalog?, sql?, query_limit?}` | `201 {id, label, active:true}` |
| `GET` | `/api/v1/sqllab/tabs` | List user's active tabs | — | `200 [{id, label, db_id, schema, sql, active, latest_query_id, latest_query_status, ...}]` |
| `GET` | `/api/v1/sqllab/tabs/:id` | Get single tab with latest query | — | `200 {id, label, db_id, schema, sql, latest_query: {...}}` |

Error responses: 422 (db_id not visible), 404 (not found).

### Handler Logic

**CreateTab:** Extract userCtx → validate db_id visibility → auto-label "Untitled Query N" (count existing `Untitled Query%` labels + 1) → GORM.Create with Active=true → return 201.

**ListTabs:** Extract userCtx → GORM.Where("user_id=? AND active=true").Preload("LatestQuery").Order("created_on ASC").Find → return 200 with latest_query.status for badge display.

**GetTab:** Extract userCtx + parse :id → GORM.Where("id=? AND user_id=?").Preload("LatestQuery").First → 404 if not found → return 200.

### Repository

New `TabStateRepository` interface in `backend/internal/repository/postgres/sqllab_repo.go`:
- `Create(ctx, *TabState) error`
- `ListByUser(ctx, userID uint) ([]*TabState, error)` — with Preload("LatestQuery")
- `GetByID(ctx, id uint, userID uint) (*TabState, error)`

### Routes

Inside the existing `protected` group in router.go:
```
POST   /api/v1/sqllab/tabs     → sqllabHandler.CreateTab
GET    /api/v1/sqllab/tabs     → sqllabHandler.ListTabs
GET    /api/v1/sqllab/tabs/:id → sqllabHandler.GetTab
```

### Wire-Up (main.go)

- `sqllabRepo := repopostgres.NewSQLLabRepository(db)`
- `sqllabHandler := httpsqllab.NewHandler(sqllabRepo, databaseRepo)`
- New parameter on `NewRouter(...)` for sqllabHandler

## Frontend

### Route

`/sqllab` — existing route, no change to App.tsx.

### Layout

3-pane `ResizablePanelGroup` with `ResizablePanel` + `ResizableHandle`:
- **Left:** Schema Browser (placeholder structure for SQL-006)
- **Center:** Tab Bar + Monaco Editor + Results Panel
- **Right:** Optional detail panel

### Components (all shadcn/ui)

**Tab Bar:**
- `Tabs` + `TabsList` — horizontal tab strip
- `TabsTrigger` per tab — label + status icon (running/success/error) + × close
- `Button` ("+", size=icon) — add new tab
- `DropdownMenu` on right-click — Close, Close All, Rename
- Double-click tab → inline `Input` rename, blur saves

**Schema Browser (left panel):**
- `ScrollArea` — scrollable tree
- `Collapsible` + `CollapsibleTrigger` + `CollapsibleContent` — schema > tables > columns
- `Input` + Search icon — schema search
- `Select` (schema) — schema switcher
- `Skeleton` × 5 — loading state

**Editor Panel (center):**
- Monaco Editor (`@monaco-editor/react`) — SQL mode, dark theme, line numbers
- `Button` ("Run", size=sm) with Play icon — execute
- `Button` ("Run All") — execute full editor content
- `Select` (limit) — 100/1000/10000 row limit
- `Button` ("Save") with Save icon — opens saved query Dialog
- `Badge` (db_name, schema_name) — connection indicator

**Results Panel (bottom):**
- `Tabs` — [Results | Query Details | Saved Queries]
- `DataTable` (TanStack Table v8) — sort + pagination
- `Button` ("Download CSV/Excel") — export DropdownMenu
- `Alert` (destructive) — query errors
- `Skeleton` — loading state
- `Badge` (from_cache, latency_ms, rows_count) — metadata

### State (Zustand sqlLabStore)

New fields on `SqlLabTab`:
- `isDirty: boolean` — unsaved indicator for tab label dot
- `latestQueryStatus?: string` — badge display from preloaded latest_query

New action:
- `initTabs(tabs: TabStateFromAPI[])` — bulk-hydrate store from GET /tabs response. Maps `tab_state.id` → tab `id`, `tab_state.label` → tab `title`, `tab_state.db_id` → tab `databaseId`.

### Data Fetching (TanStack Query)

- `useQuery({ queryKey:["sqllab-tabs"], queryFn: ()=>fetch("/api/v1/sqllab/tabs") })` — on page mount, hydrate store via `initTabs`. If empty response, auto-create one empty tab.
- `useMutation({ mutationFn: (data)=>fetch("/api/v1/sqllab/tabs", {method:"POST"}) })` — create tab
- Auto-save: `useEffect` watching `sql` → 1000ms debounce → `PUT /sqllab/tabs/:id`. Schema change → immediate PUT.

### UX Behaviors

- **Page load:** GET tabs → restore all. First tab auto-selected.
- **Tab strip:** label + status icon + × close. Right-click context menu. Double-click inline rename.
- **Editor:** Monaco SQL mode, Ctrl+Enter = Run (selection if exists, else all).
- **Schema browser:** click column → insert at cursor in Monaco.
- **Results:** DataTable sticky header, virtual scroll for 10k+ rows.
- **Cache badge:** Green "Cached (42ms)" or gray "Live (234ms)".
- **Auto-save:** Silent, 1000ms debounced. Dirty dot (•) on unsaved tabs. Network error → toast warning.
- **Empty tabs on mount:** if GET returns empty, POST create one default tab.

### Accessibility

- Monaco: `aria-label="SQL Editor"`
- Tab strip: `role="tablist" aria-label="SQL Editor Tabs"`
- Run button: `aria-label="Run Query (Ctrl+Enter)"`

## Files

### New
- `backend/internal/repository/postgres/sqllab_repo.go`
- `backend/internal/delivery/http/sqllab/handler.go`
- `frontend/src/api/sqllab.ts`

### Modified
- `backend/internal/delivery/http/router.go` — sqllab routes + handler param
- `backend/cmd/api/main.go` — wire sqllabRepo, sqllabHandler
- `frontend/src/stores/sqlLabStore.ts` — initTabs, isDirty, latestQueryStatus
- `frontend/src/pages/sqllab/SQLLabPage.tsx` — Monaco, resizable panels, updated tab bar

## Out of Scope
- SQL-002 (auto-save endpoint — tab state update via PUT /tabs/:id)
- SQL-003 (soft close, delete, reopen closed tabs)
- SQL-004 (saved queries CRUD)
- SQL-006 (schema browser data — only layout placeholder)
- Result persistence on tab restore
- Tab sharing between users
