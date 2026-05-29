# SQL Lab Phase 2 — Gap Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 9 gaps (2 backend, 7 frontend) identified in SQL Lab Service Phase 2 verification audit.

**Architecture:** Backend: extend `TabWithQueryStatus` join to carry full query metadata, expand saved-query visibility to published, add usage-count endpoint. Frontend: add UI controls (limit Select, Run All, detail pane, Badges, "Query Details" tab), Monaco autocomplete provider, and in-use count badge.

**Tech Stack:** Go 1.22 (Gin, GORM, sqlparser), React 18 (TypeScript, TanStack Query v5, Zustand, Monaco Editor, shadcn/ui)

---

### Task 1: Backend B1 — Extend `LatestQuery` to proper object

**Files:**
- Modify: `backend/internal/domain/query/sqllab_types.go:26-40`
- Modify: `backend/internal/domain/query/sqllab_repository.go:30-34`
- Modify: `backend/internal/repository/postgres/sqllab_repo.go:51-95`
- Modify: `backend/internal/delivery/http/sqllab/handler.go:750-773`

- [ ] **Step 1: Add `LatestQueryResponse` struct and update `TabResponse`**

In `backend/internal/domain/query/sqllab_types.go`, after line 24 (`UpdateTabRequest`):

```go
// LatestQueryResponse is a compact view of the linked query for tab badge display.
type LatestQueryResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Rows         int    `json:"rows"`
	ErrorMessage string `json:"error_message,omitempty"`
}
```

Replace `TabResponse` (lines 27-40):

```go
type TabResponse struct {
	ID            uint                 `json:"id"`
	Label         string               `json:"label"`
	DbID          uint                 `json:"db_id"`
	Schema        string               `json:"schema"`
	Catalog       string               `json:"catalog"`
	SQL           string               `json:"sql"`
	Active        bool                 `json:"active"`
	QueryLimit    int                  `json:"query_limit"`
	LatestQueryID *string              `json:"latest_query_id,omitempty"`
	LatestQuery   *LatestQueryResponse `json:"latest_query,omitempty"`
	HideLeftBar   bool                 `json:"hide_left_bar"`
	CreatedOn     string               `json:"created_on"`
}
```

- [ ] **Step 2: Extend `TabWithQueryStatus` join struct**

In `backend/internal/domain/query/sqllab_repository.go`, replace `TabWithQueryStatus` (lines 31-34):

```go
type TabWithQueryStatus struct {
	TabState
	QueryID           *string `gorm:"column:query_id" json:"query_id,omitempty"`
	QueryStatus       *string `gorm:"column:query_status" json:"query_status,omitempty"`
	QueryRows         *int    `gorm:"column:query_rows" json:"query_rows,omitempty"`
	QueryErrorMessage *string `gorm:"column:query_error_message" json:"query_error_message,omitempty"`
}
```

- [ ] **Step 3: Update repo SELECT queries**

In `backend/internal/repository/postgres/sqllab_repo.go`:

Replace the `ListByUser` SELECT (line 54):
```go
Select("tab_state.*, query.id AS query_id, query.status AS query_status, query.rows AS query_rows, query.error_message AS query_error_message").
```

Replace the `GetByID` SELECT (line 80):
```go
Select("tab_state.*, query.id AS query_id, query.status AS query_status, query.rows AS query_rows, query.error_message AS query_error_message").
```

Replace the `ListByUser` ExtraJSON hack (lines 67-70):
```go
tabs := make([]*query.TabState, 0, len(rows))
for _, r := range rows {
    t := r.TabState
    if r.QueryID != nil {
        t.ExtraJSON = packLatestQueryExtra(r.QueryID, r.QueryStatus, r.QueryRows, r.QueryErrorMessage)
    }
    tabs = append(tabs, &t)
}
return tabs, nil
```

Replace the `GetByID` ExtraJSON hack (lines 91-93):
```go
t := row.TabState
if row.QueryID != nil {
    t.ExtraJSON = packLatestQueryExtra(row.QueryID, row.QueryStatus, row.QueryRows, row.QueryErrorMessage)
}
return &t, nil
```

Add helper after `extractSQLTables` (appx line 45):

```go
func packLatestQueryExtra(id, status *string, rows *int, errMsg *string) string {
	r := 0
	if rows != nil {
		r = *rows
	}
	em := ""
	if errMsg != nil {
		em = *errMsg
	}
	j, _ := json.Marshal(map[string]interface{}{
		"query_id":        id,
		"query_status":    status,
		"query_rows":      r,
		"query_error_message": em,
	})
	return string(j)
}
```

Add `"encoding/json"` to imports.

- [ ] **Step 4: Update `tabToResponse` to populate `LatestQuery`**

In `backend/internal/delivery/http/sqllab/handler.go`, replace `tabToResponse` (lines 750-773):

```go
func tabToResponse(t *domainquery.TabState) domainquery.TabResponse {
	resp := domainquery.TabResponse{
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
	if t.ExtraJSON != "" {
		var extra struct {
			QueryID           *string `json:"query_id"`
			QueryStatus       *string `json:"query_status"`
			QueryRows         int     `json:"query_rows"`
			QueryErrorMessage string `json:"query_error_message"`
		}
		if json.Unmarshal([]byte(t.ExtraJSON), &extra) == nil && extra.QueryID != nil {
			resp.LatestQuery = &domainquery.LatestQueryResponse{
				ID:           *extra.QueryID,
				Status:       stringPtrVal(extra.QueryStatus),
				Rows:         extra.QueryRows,
				ErrorMessage: extra.QueryErrorMessage,
			}
		}
	}
	return resp
}

func stringPtrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
```

- [ ] **Step 5: Run backend tests**

```bash
cd backend && go test ./internal/delivery/http/sqllab/... -v -run "TestTab|TestCreate"
```

Expected: all tab-related tests pass.

- [ ] **Step 6: Update frontend API types for new response shape**

In `frontend/src/api/sqllab.ts`, replace `TabStateResponse` (lines 12-25):

```typescript
export interface TabStateResponse {
  id: number;
  label: string;
  db_id: number;
  schema: string;
  catalog: string;
  sql: string;
  active: boolean;
  query_limit: number;
  latest_query_id: string | null;
  latest_query?: {
    id: string;
    status: string;
    rows: number;
    error_message?: string;
  } | null;
  hide_left_bar: boolean;
  created_on: string;
}
```

In `frontend/src/stores/sqlLabStore.ts`, update `initTabs` (line 156) to read from new path:

```typescript
latestQueryStatus: t.latest_query?.status,
```

- [ ] **Step 7: Commit**

```bash
git add backend/internal/domain/query/sqllab_types.go backend/internal/domain/query/sqllab_repository.go backend/internal/repository/postgres/sqllab_repo.go backend/internal/delivery/http/sqllab/handler.go frontend/src/api/sqllab.ts frontend/src/stores/sqlLabStore.ts
git commit -m "fix(sql-001): return latest_query as structured object instead of flat status string"
```

---

### Task 2: Backend B2 — Published saved queries org-visible

**Files:**
- Modify: `backend/internal/repository/postgres/sqllab_repo.go:167-168,199-209,237-239`

- [ ] **Step 1: Expand `ListSavedQueries` WHERE clause**

In `backend/internal/repository/postgres/sqllab_repo.go`, change line 168 from:

```go
q := r.db.WithContext(ctx).Model(&query.SavedQuery{}).Where("created_by_fk = ?", userID)
```

To:

```go
q := r.db.WithContext(ctx).Model(&query.SavedQuery{}).Where("created_by_fk = ? OR published = true", userID)
```

- [ ] **Step 2: Expand `GetSavedQuery` WHERE clause**

Change line 201 from:

```go
err := r.db.WithContext(ctx).Where("id = ? AND created_by_fk = ?", id, userID).First(&sq).Error
```

To:

```go
err := r.db.WithContext(ctx).Where("id = ? AND (created_by_fk = ? OR published = true)", id, userID).First(&sq).Error
```

- [ ] **Step 3: Expand `ForkSavedQuery` WHERE clause**

Change line 239 from:

```go
err := r.db.WithContext(ctx).Where("id = ? AND created_by_fk = ?", id, userID).First(&original).Error
```

To:

```go
err := r.db.WithContext(ctx).Where("id = ? AND (created_by_fk = ? OR published = true)", id, userID).First(&original).Error
```

Note: fork still creates copy owned by requesting user — only read access is expanded.

- [ ] **Step 4: Run backend tests**

```bash
cd backend && go test ./internal/delivery/http/sqllab/... -v -run "TestSavedQuery"
```

Expected: saved query tests pass; others' published queries visible.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/postgres/sqllab_repo.go
git commit -m "fix(sql-004): make published saved queries visible to all users, allow forking published queries"
```

---

### Task 3: Backend F6 — Saved query usage count endpoint

**Files:**
- Modify: `backend/internal/domain/query/sqllab_types.go` (add types)
- Modify: `backend/internal/domain/query/sqllab_repository.go` (add interface method)
- Modify: `backend/internal/repository/postgres/sqllab_repo.go` (add implementation)
- Modify: `backend/internal/delivery/http/sqllab/handler.go` (add handler)
- Modify: `backend/internal/delivery/http/router.go` (add route)

- [ ] **Step 1: Add response type**

In `backend/internal/domain/query/sqllab_types.go`, after `ExpandTableResponse` (appx line 116):

```go
// SavedQueryUsageResponse is returned by GET /saved-queries/:id/usage.
type SavedQueryUsageResponse struct {
	TabCount int `json:"tab_count"`
}
```

- [ ] **Step 2: Add repository interface method**

In `backend/internal/domain/query/sqllab_repository.go`, after `ForkSavedQuery` line (appx line 21):

```go
	CountTabReferences(ctx context.Context, savedQueryID uint) (int64, error)
```

- [ ] **Step 3: Implement repository method**

In `backend/internal/repository/postgres/sqllab_repo.go`, after `ForkSavedQuery` (after line 259):

```go
func (r *sqllabRepo) CountTabReferences(ctx context.Context, savedQueryID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&query.TabState{}).
		Where("saved_query_id = ? AND active = true", savedQueryID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count tab references: %w", err)
	}
	return count, nil
}
```

- [ ] **Step 4: Add handler**

In `backend/internal/delivery/http/sqllab/handler.go`, after `ForkSavedQuery` handler, add:

```go
func (h *Handler) GetSavedQueryUsage(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid saved query id"})
		return
	}

	// Check visibility (owner or published)
	sq, err := h.sqllabRepo.GetSavedQuery(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve saved query"})
		return
	}
	if sq == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Saved query not found"})
		return
	}

	count, err := h.sqllabRepo.CountTabReferences(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to count tab references"})
		return
	}

	c.JSON(http.StatusOK, domainquery.SavedQueryUsageResponse{TabCount: int(count)})
}
```

- [ ] **Step 5: Add route**

In `backend/internal/delivery/http/router.go`, after line 185 (`sqlLab.POST("/saved-queries/:id/fork"`):

```go
				sqlLab.GET("/saved-queries/:id/usage", sqllabHandler.GetSavedQueryUsage)
```

- [ ] **Step 6: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/domain/query/sqllab_types.go backend/internal/domain/query/sqllab_repository.go backend/internal/repository/postgres/sqllab_repo.go backend/internal/delivery/http/sqllab/handler.go backend/internal/delivery/http/router.go
git commit -m "feat(sql-005): add saved query usage count endpoint (GET /saved-queries/:id/usage)"
```

---

### Task 4: Frontend F1 — Limit Select dropdown

**Files:**
- Modify: `frontend/src/stores/sqlLabStore.ts:48-67,139-164`
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx:618-635,1056-1095`

- [ ] **Step 1: Add `queryLimit` to `SqlLabTab` and store methods**

In `frontend/src/stores/sqlLabStore.ts`, add field to `SqlLabTab` interface (after line 66):

```typescript
  queryLimit: number;
```

Add to store actions interface (after `clearTabDirty` line 91):

```typescript
  updateTabQueryLimit: (id: string, limit: number) => void;
```

Add default in `addTab` (after `estimate: null,` line 132):

```typescript
          queryLimit: 1000,
```

Add default in `initTabs` (after `isDirty: false,` line 155):

```typescript
          queryLimit: t.query_limit || 1000,
```

Add action implementation (after `clearTabDirty`, appx line 250):

```typescript
  updateTabQueryLimit: (id, limit) =>
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? { ...t, queryLimit: limit, isDirty: true } : t
      ),
    })),
```

- [ ] **Step 2: Add `updateTabQueryLimit` import in SQLLabPage**

In `frontend/src/pages/sqllab/SQLLabPage.tsx`, add `updateTabQueryLimit` to the store destructuring (line 121-138):

```typescript
    updateTabQueryLimit,
  } = useSqlLabStore();
```

- [ ] **Step 3: Add limit Select in toolbar**

In `SQLLabPage.tsx`, insert after the RunAsyncButton block (after line 1041, before the estimate section):

```tsx
                    <Select
                      value={String(tab.queryLimit ?? 1000)}
                      onValueChange={(v) => {
                        if (activeTabId) updateTabQueryLimit(activeTabId, Number(v));
                      }}
                    >
                      <SelectTrigger className="w-[90px] h-8 text-xs">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="100">100</SelectItem>
                        <SelectItem value="1000">1000</SelectItem>
                        <SelectItem value="10000">10000</SelectItem>
                      </SelectContent>
                    </Select>
```

- [ ] **Step 4: Pass `limit` in execute and async mutations**

In `handleRun` (line 627), add `limit` to payload:

```typescript
      executeMutation.mutate({
        database_id: activeTab.databaseId,
        sql: sqlToRun,
        catalog: activeTab.catalog,
        tab_name: activeTab.title,
        sql_editor_id: activeTab.sqlEditorId,
        limit: activeTab.queryLimit,
      });
```

In `handleRunAsync` (line 661), add `limit` to payload:

```typescript
    submitAsyncMutation.mutate({
      database_id: activeTab.databaseId,
      sql: sqlToRun,
      catalog: activeTab.catalog,
      tab_name: activeTab.title,
      limit: activeTab.queryLimit,
    });
```

- [ ] **Step 5: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/stores/sqlLabStore.ts frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat(sql-001): add query limit Select dropdown (100/1000/10000) in editor toolbar"
```

---

### Task 5: Frontend F2 — "Run All" button

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx:1026-1035`

- [ ] **Step 1: Add "Run All" button next to existing Run button**

In `SQLLabPage.tsx`, after the existing `RunButton` (line 1034) and before `RunAsyncButton` (line 1036):

```tsx
                        <RunButton
                          onClick={() => {
                            const editor = editorRef.current;
                            if (!editor) return;
                            const fullSql = editor.getValue();
                            handleRun(fullSql);
                          }}
                          disabled={!tab.databaseId || !tab.sql || isRunning || isAsyncRunning}
                          isRunning={isRunning}
                          label="Run All"
                        />
```

Update `RunButton` in `frontend/src/components/query/QueryBadges.tsx` (lines 148-170) to accept optional `label` prop:

```typescript
interface RunButtonProps {
  onClick: () => void;
  disabled: boolean;
  isRunning: boolean;
  label?: string;
}

export function RunButton({ onClick, disabled, isRunning, label }: RunButtonProps) {
  if (isRunning) {
    return (
      <Button disabled size="sm" className="gap-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        Running...
      </Button>
    );
  }

  return (
    <Button onClick={onClick} disabled={disabled} size="sm" variant={label ? "outline" : "default"} className="gap-2">
      <Play className="h-4 w-4" />
      {label ?? "Run"}
    </Button>
  );
}
```

The default `Run` button stays selection-aware (Ctrl+Enter). The new `Run All` always executes full editor content.

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx frontend/src/components/query/QueryBadges.tsx
git commit -m "feat(sql-001): add Run All button that always executes full editor content"
```

---

### Task 6: Frontend F3 — 3rd detail pane

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx:814-843`

- [ ] **Step 1: Add resize panel for detail pane**

After the existing 2-pane `ResizablePanelGroup` (the main horizontal one at line 816), add a 3rd `ResizablePanel` after the center panel's closing `</ResizablePanel>` (after line 1225). Add a toggle button state:

In state declarations (after line 616):

```typescript
  const [showDetailPane, setShowDetailPane] = useState(false);
```

Add import for `PanelRightOpen`, `PanelRightClose` at top:

```typescript
import { PanelRightOpen, PanelRightClose } from "lucide-react";
```

In the toolbar (after Save button line 1078, before CacheBadge line 1079):

```tsx
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowDetailPane(v => !v)}
                      className="h-8 w-8 p-0"
                    >
                      {showDetailPane ? <PanelRightClose className="h-4 w-4" /> : <PanelRightOpen className="h-4 w-4" />}
                    </Button>
```

After the center `</ResizablePanel>` closing (before `</ResizablePanelGroup>` line 1226), insert:

```tsx
        {showDetailPane && (
          <>
            <ResizableHandle withHandle />
            <ResizablePanel defaultSize={20} minSize={10} maxSize={35}>
              <div className="flex flex-col h-full p-4 overflow-auto">
                <h3 className="text-sm font-medium mb-2">Query Details</h3>
                {activeTab?.result?.query ? (
                  <div className="space-y-2 text-sm">
                    {activeTab.result.query.start_time && (
                      <div>
                        <span className="text-muted-foreground">Started: </span>
                        {new Date(activeTab.result.query.start_time).toLocaleString()}
                      </div>
                    )}
                    {activeTab.result.query.end_time && (
                      <div>
                        <span className="text-muted-foreground">Finished: </span>
                        {new Date(activeTab.result.query.end_time).toLocaleString()}
                      </div>
                    )}
                    {activeTab.result.query.start_time && activeTab.result.query.end_time && (
                      <div>
                        <span className="text-muted-foreground">Duration: </span>
                        {calculateDurationMs(activeTab.result.query.start_time, activeTab.result.query.end_time)}ms
                      </div>
                    )}
                    <div>
                      <span className="text-muted-foreground">Rows: </span>
                      {activeTab.result.query.rows.toLocaleString()}
                    </div>
                    <div>
                      <span className="text-muted-foreground">Cache: </span>
                      {activeTab.result.from_cache ? "Yes" : "No"}
                    </div>
                    {activeTab.result.query.executed_sql && activeTab.result.query.executed_sql !== activeTab.result.query.sql && (
                      <div>
                        <span className="text-muted-foreground">RLS: </span>
                        <span className="text-orange-600">Applied</span>
                      </div>
                    )}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No query results yet</p>
                )}
              </div>
            </ResizablePanel>
          </>
        )}
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat(sql-001): add optional 3rd detail pane with query metadata"
```

---

### Task 7: Frontend F4 — db+schema Badge

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx:1056-1095`

- [ ] **Step 1: Add connection indicator Badge**

Import `Database` at top (line 2):

```typescript
import { Plus, X, Database, PanelRightOpen, PanelRightClose } from "lucide-react";
```

(Note: `PanelRightOpen`/`PanelRightClose` already added in Task 6. Add `Database` only.)

In the editor toolbar (after the Save button line 1078, before the detail pane toggle added in Task 6):

```tsx
                    {selectedDb && (
                      <Badge variant="outline" className="gap-1 text-xs h-8">
                        <Database className="h-3 w-3" />
                        {selectedDb.database_name}
                        {activeTab?.schema && <> / {activeTab.schema}</>}
                      </Badge>
                    )}
```

The `Badge` import already exists — verify `import { Badge } from "@/components/ui/badge"` is present. If not, add it.

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat(sql-001): add db_name/schema Badge as connection indicator"
```

---

### Task 8: Frontend F5 — Rename "History" to "Query Details"

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx:1138-1205`

- [ ] **Step 1: Rename tab trigger and replace content**

Change the TabsList trigger (line 1141) from:

```tsx
                      <TabsTrigger value="history">History</TabsTrigger>
```

To:

```tsx
                      <TabsTrigger value="details">Query Details</TabsTrigger>
```

Replace the TabsContent from "history" (lines 1189-1206) with:

```tsx
                    <TabsContent value="details" className="flex-1 min-h-0 mt-2 overflow-auto">
                      {tab.result?.query ? (
                        <div className="space-y-2 text-sm">
                          <div className="grid grid-cols-2 gap-2">
                            {tab.result.query.start_time && (
                              <>
                                <span className="text-muted-foreground">Started</span>
                                <span>{new Date(tab.result.query.start_time).toLocaleString()}</span>
                              </>
                            )}
                            {tab.result.query.end_time && (
                              <>
                                <span className="text-muted-foreground">Finished</span>
                                <span>{new Date(tab.result.query.end_time).toLocaleString()}</span>
                              </>
                            )}
                            {tab.result.query.start_time && tab.result.query.end_time && (
                              <>
                                <span className="text-muted-foreground">Duration</span>
                                <span>{calculateDurationMs(tab.result.query.start_time, tab.result.query.end_time)}ms</span>
                              </>
                            )}
                            <span className="text-muted-foreground">Rows returned</span>
                            <span>{tab.result.query.rows.toLocaleString()}</span>
                            <span className="text-muted-foreground">Cache</span>
                            <span>{tab.result.from_cache ? `Cached` : "Live"}</span>
                            {tab.result.query.limit > 0 && (
                              <>
                                <span className="text-muted-foreground">Row limit</span>
                                <span>{tab.result.query.limit.toLocaleString()}</span>
                              </>
                            )}
                            {tab.result.query.status && (
                              <>
                                <span className="text-muted-foreground">Status</span>
                                <span>{tab.result.query.status}</span>
                              </>
                            )}
                          </div>
                          {tab.result.query.executed_sql && tab.result.query.executed_sql !== tab.result.query.sql && (
                            <div>
                              <span className="text-muted-foreground text-xs">RLS filters applied</span>
                            </div>
                          )}
                          {tab.result.query.sql && (
                            <details className="mt-2">
                              <summary className="text-xs text-muted-foreground cursor-pointer">Executed SQL</summary>
                              <pre className="text-xs mt-1 p-2 bg-muted rounded whitespace-pre-wrap max-h-48 overflow-auto">{tab.result.query.executed_sql || tab.result.query.sql}</pre>
                            </details>
                          )}
                        </div>
                      ) : (
                        <p className="text-sm text-muted-foreground text-center py-12">
                          Run a query to see details
                        </p>
                      )}
                    </TabsContent>
```

Remove `QueryHistoryTable` import if unused: remove line 63 `import { QueryHistoryTable } from "@/components/query/QueryHistoryTable";` and its usage.

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "fix(sql-001): rename History sub-tab to Query Details with result metadata view"
```

---

### Task 9: Frontend F6 — "In use by N tabs" badge on delete

**Files:**
- Modify: `frontend/src/api/sqllab.ts` (add usage API)
- Modify: `frontend/src/pages/sqllab/SavedQueriesPage.tsx:298-315`

- [ ] **Step 1: Add usage API function**

In `frontend/src/api/sqllab.ts`, before the Schema Browser section (before line 196):

```typescript
export interface SavedQueryUsageResponse {
  tab_count: number;
}

export async function getSavedQueryUsage(id: number): Promise<SavedQueryUsageResponse> {
  return request<SavedQueryUsageResponse>(`/api/v1/sqllab/saved-queries/${id}/usage`, {
    method: "GET",
    headers: getAuthHeaders(),
  });
}
```

- [ ] **Step 2: Wire usage count into delete dialog**

In `frontend/src/pages/sqllab/SavedQueriesPage.tsx`, add state (after `deletingQuery` state):

```typescript
  const [inUseCount, setInUseCount] = useState<number | null>(null);
```

Add import:

```typescript
import { getSavedQueryUsage } from "@/api/sqllab";
```

(Add to existing sqllab API import on line 46.)

Update the `onOpenChange` handler of the delete AlertDialog (line 298) to fetch usage count:

```tsx
      <AlertDialog
        open={deletingQuery !== null}
        onOpenChange={(open) => {
          if (!open) { setDeletingQuery(null); setInUseCount(null); }
          if (open && deletingQuery?.id) {
            getSavedQueryUsage(deletingQuery.id).then(r => setInUseCount(r.tab_count)).catch(() => setInUseCount(null));
          }
        }}
      >
```

Replace the `AlertDialogDescription` (line 303):

```tsx
            <AlertDialogDescription>
              <span>This cannot be undone. Any tabs referencing this saved query will be updated.</span>
              {inUseCount !== null && inUseCount > 0 && (
                <Badge variant="secondary" className="ml-2 align-middle">
                  In use by {inUseCount} tab{inUseCount !== 1 ? "s" : ""}
                </Badge>
              )}
            </AlertDialogDescription>
```

- [ ] **Step 3: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/sqllab.ts frontend/src/pages/sqllab/SavedQueriesPage.tsx
git commit -m "feat(sql-005): show In use by N tabs badge on saved query delete dialog"
```

---

### Task 10: Frontend F7 — Monaco autocomplete provider

**Files:**
- Modify: `frontend/src/api/sqllab.ts` (add autocomplete API)
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx:637-650`

- [ ] **Step 1: Add autocomplete API function and types**

In `frontend/src/api/sqllab.ts`, after the usage section added in Task 9 (before Schema Browser section):

```typescript
export interface AutocompleteRequest {
  word: string;
  prefix: string;
  db_id: number;
  schema: string;
}

export interface AutocompleteSuggestion {
  text: string;
  type: "keyword" | "schema" | "table" | "column" | "function";
  score: number;
  detail: string;
}

export interface AutocompleteResponse {
  suggestions: AutocompleteSuggestion[];
  cache_miss: boolean;
}

export async function autocomplete(data: AutocompleteRequest): Promise<AutocompleteResponse> {
  return request<AutocompleteResponse>("/api/v1/sqllab/autocomplete", {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}
```

- [ ] **Step 2: Register completion provider in `handleEditorMount`**

Add import near `OnMount` (line 4):

```typescript
import type { editor, languages } from "monaco-editor";
```

Also import:

```typescript
import { autocomplete } from "@/api/sqllab";
```

Replace `handleEditorMount` (lines 637-650):

```typescript
  const handleEditorMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;

    // Ctrl+Enter run shortcut
    editor.addAction({
      id: "run-query",
      label: "Run Query",
      keybindings: [2048 | 3],
      run: () => {
        const selection = editor.getSelection();
        const selectedText = selection ? editor.getModel()?.getValueInRange(selection)?.trim() : null;
        const sql = selectedText || activeTab?.sql || "";
        handleRun(sql);
      },
    });

    // Autocomplete provider
    let debounceTimer: ReturnType<typeof setTimeout> | null = null;

    monaco.languages.registerCompletionItemProvider("sql", {
      triggerCharacters: [".", " ", "\n", "("],
      provideCompletionItems: async (model, position) => {
        const word = model.getWordUntilPosition(position);
        const prefix = model.getValueInRange({
          startLineNumber: position.lineNumber,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });

        if (!word.word || word.word.length < 1) {
          return { suggestions: [] };
        }

        const tab = tabs.find(t => t.id === activeTabId);
        if (!tab?.databaseId) {
          // Return SQL keywords only when no DB selected
          return { suggestions: SQL_KEYWORDS.map(k => ({
            label: k,
            kind: monaco.languages.CompletionItemKind.Keyword,
            insertText: k,
            detail: "keyword",
            sortText: "3" + k,
          })) };
        }

        try {
          const resp = await autocomplete({
            word: word.word,
            prefix,
            db_id: tab.databaseId,
            schema: tab.schema || "public",
          });

          if (resp.cache_miss) {
            setCacheMissAlert(true);
          } else {
            setCacheMissAlert(false);
          }

          return {
            suggestions: resp.suggestions.map(s => {
              const kindMap: Record<string, languages.CompletionItemKind> = {
                keyword: monaco.languages.CompletionItemKind.Keyword,
                schema: monaco.languages.CompletionItemKind.Module,
                table: monaco.languages.CompletionItemKind.Class,
                column: monaco.languages.CompletionItemKind.Field,
                function: monaco.languages.CompletionItemKind.Function,
              };
              const scorePrefix = String(Math.max(0, 99 - s.score)).padStart(2, "0");
              return {
                label: s.text,
                kind: kindMap[s.type] ?? monaco.languages.CompletionItemKind.Text,
                insertText: s.text,
                detail: `${s.type} · ${s.detail || ""}`,
                sortText: scorePrefix + s.text,
              };
            }),
          };
        } catch {
          return { suggestions: [] };
        }
      },
    });
  };
```

- [ ] **Step 3: Add SQL keywords constant and cache_miss alert state**

Add after the `POLLING_INTERVAL_MS` constant (line 78):

```typescript
const SQL_KEYWORDS = [
  "SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "ON",
  "AND", "OR", "NOT", "IN", "LIKE", "BETWEEN", "IS", "NULL", "AS", "ORDER",
  "BY", "GROUP", "HAVING", "LIMIT", "OFFSET", "INSERT", "INTO", "VALUES",
  "UPDATE", "SET", "DELETE", "CREATE", "TABLE", "ALTER", "DROP", "INDEX",
  "DISTINCT", "UNION", "ALL", "CASE", "WHEN", "THEN", "ELSE", "END",
  "EXISTS", "WITH", "COUNT", "SUM", "AVG", "MIN", "MAX", "CAST", "COALESCE",
  "ASC", "DESC", "TRUE", "FALSE", "PRIMARY", "KEY", "FOREIGN", "REFERENCES",
  "CASCADE", "DEFAULT", "CHECK", "UNIQUE", "CONSTRAINT", "TRUNCATE",
];
```

Add state for cache miss alert (after other state declarations, appx line 616):

```typescript
  const [cacheMissAlert, setCacheMissAlert] = useState(false);
```

- [ ] **Step 4: Render cache_miss Alert**

In the editor area, after the error `Alert` (after line 1136), add:

```tsx
                  {cacheMissAlert && (
                    <Alert variant="default" className="mt-2 bg-blue-50 border-blue-200">
                      <AlertDescription className="text-blue-800 flex items-center justify-between">
                        <span>Schema not loaded yet. Autocomplete showing SQL keywords only.</span>
                        <Button variant="ghost" size="sm" onClick={() => setCacheMissAlert(false)} className="h-6 px-2">
                          <X className="h-3 w-3" />
                        </Button>
                      </AlertDescription>
                    </Alert>
                  )}
```

- [ ] **Step 5: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors (may require `npm install monaco-editor` if types missing).

- [ ] **Step 6: Commit**

```bash
git add frontend/src/api/sqllab.ts frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat(sql-007): add Monaco autocomplete provider with SQL keywords and schema suggestions"
```

---

## Verification Checklist

After all tasks complete, verify end-to-end:

1. **Backend tests:** `cd backend && go test ./internal/delivery/http/sqllab/... -v`
2. **Backend build:** `cd backend && go build ./...`
3. **Frontend build:** `cd frontend && npx tsc --noEmit`
4. **Manual smoke test:** Start app, open /sqllab, verify:
   - Limit dropdown changes row limit
   - Run All button executes full editor
   - Detail pane toggle works
   - db+schema badge shows in toolbar
   - Results sub-tab "Query Details" shows metadata
   - Delete saved query shows "In use by N" badge
   - Autocomplete triggers in Monaco (Ctrl+Space or dot)
   - Tab response includes `latest_query` object (check Network tab)
   - Published query from user A visible to user B
