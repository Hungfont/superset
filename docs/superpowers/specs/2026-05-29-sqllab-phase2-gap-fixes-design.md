# SQL Lab Phase 2 — Gap Fixes Design

2026-05-29 | 9 gaps across backend (2) and frontend (7)

## Backend Gaps

### Gap B1 — SQL-001: `latest_query` as proper object

**Problem:** `TabResponse.LatestQueryStatus` is a flat string populated via ExtraJSON
round-trip hack. Spec expects a structured `latest_query` object.

**Fix:** Add `LatestQueryResponse` struct, change `TabResponse`, update repo JOIN
SELECT, update `tabToResponse`. No model changes.

**Changes:**

1. `domain/query/sqllab_types.go` — add struct:
```go
type LatestQueryResponse struct {
    ID           string `json:"id"`
    Status       string `json:"status"`
    Rows         int    `json:"rows"`
    ErrorMessage string `json:"error_message,omitempty"`
}
```
Replace `LatestQueryStatus string` with `LatestQuery *LatestQueryResponse` in
`TabResponse`.

2. `domain/query/sqllab_repository.go` — extend `TabWithQueryStatus` to join full
query fields: `QueryID`, `QueryStatus`, `QueryRows`, `QueryErrorMessage`.

3. `repository/postgres/sqllab_repo.go` — change SELECT from `query.status` to
`query.id AS query_id, query.status AS query_status, query.rows AS query_rows,
query.error_message AS query_error_message`. Populate new fields. Remove
ExtraJSON hack from both `GetByID` and `ListByUser`.

4. `delivery/http/sqllab/handler.go` — `tabToResponse`: populate
`LatestQuery` from join fields directly (no ExtraJSON parsing).

**Breaking:** Frontend must read `latest_query.status` instead of
`latest_query_status`.

---

### Gap B2 — SQL-004: Published saved queries org-visible

**Problem:** `ListSavedQueries` and `GetSavedQuery` filter `created_by_fk = ?`
only — no path to see other users' published queries.

**Constraint:** No TenantID on `ab_user` model. No model changes allowed.
This is a single-org system (all users share same scope). "Org-visible" =
visible to all authenticated users.

**Fix:** Expand WHERE to `created_by_fk = ? OR published = true` in both
`ListSavedQueries` and `GetSavedQuery`. Also fork should allow forking
published queries from other users.

**Changes:**

1. `repository/postgres/sqllab_repo.go`:
   - `ListSavedQueries` (line 168): `WHERE created_by_fk = ? OR published = true`
   - `GetSavedQuery` (line 201): `WHERE id = ? AND (created_by_fk = ? OR published = true)`
   - `ForkSavedQuery` (line 239): same expansion

2. Label uniqueness remains per-user (not affected).

3. Delete/Update remain owner-only (ownership check after fetch).

---

## Frontend Gaps

All changes in `SQLLabPage.tsx` unless noted.

### Gap F1 — SQL-001: Limit Select dropdown

**Missing:** No UI to set `query_limit` on tab.

**Fix:**
1. Add `queryLimit: number` to `SqlLabTab` in `sqlLabStore.ts` (default 1000)
2. Add `<Select>` in editor toolbar area (between db selector and schema badge):
```tsx
<Select value={String(activeTab.queryLimit ?? 1000)}
  onValueChange={(v) => updateTabQueryLimit(activeTabId!, Number(v))}>
  <SelectTrigger className="w-24 h-8 text-xs">
    <SelectValue />
  </SelectTrigger>
  <SelectContent>
    <SelectItem value="100">100</SelectItem>
    <SelectItem value="1000">1000</SelectItem>
    <SelectItem value="10000">10000</SelectItem>
  </SelectContent>
</Select>
```
3. Pass `limit: activeTab.queryLimit` in `executeMutation.mutate` and
   `submitAsyncMutation.mutate` payloads in `handleRun` and `handleRunAsync`
4. Auto-save limit changes via existing `useAutoSaveTab` debounce

---

### Gap F2 — SQL-001: "Run All" button

**Missing:** Only one Run button that's selection-aware. Spec requires separate
"Run All" that always runs entire editor content.

**Fix:** Add second `RunButton` variant next to existing Run button.
- Existing: runs selection if exists, else all (current Ctrl+Enter behavior)
- New "Run All": always runs `editorRef.current.getValue()` ignoring selection
- Use `Keyboard` icon (or `Play` with distinct variant)
- Label: "Run All"
- `onClick`: `handleRun(editorRef.current?.getValue())`

---

### Gap F3 — SQL-001: 3rd detail pane

**Missing:** Spec calls for optional 3rd detail pane ("optional detail"). Current
2-pane layout.

**Fix:** Add collapsible 3rd `<ResizablePanel>` at right side of outer
`ResizablePanelGroup`. Default collapsed (size=0). Toggle via button in toolbar
(`PanelRightOpen` icon when hidden, `PanelRightClose` when shown). Content:
current query metadata — executed SQL, row count, cache status, timings, RLS
info. Only visible when `activeTab?.lastResult` exists.

---

### Gap F4 — SQL-001: db+schema Badge

**Missing:** No connection indicator showing current db_name and schema.

**Fix:** Add `<Badge variant="outline">` in editor toolbar:
```tsx
{selectedDb && (
  <Badge variant="outline" className="gap-1 text-xs">
    <Database className="h-3 w-3" />
    {selectedDb.database_name}
    {activeTab?.schema && <> / {activeTab.schema}</>}
  </Badge>
)}
```

---

### Gap F5 — SQL-001: "Query Details" sub-tab

**Missing:** Results sub-tab shows "History" (runs `QueryHistoryTable`). Spec
requires "Query Details" showing current result metadata.

**Fix:** Rename tab trigger "History" → "Query Details". Replace content with
detail view of current result: timings (start_time, end_time, duration), row
count, cache status ("Cached (42ms)" / "Live (234ms)"), RLS info, error message
if present. Use shadcn `DescriptionList`-like layout (dl/dt/dd via divs).

---

### Gap F6 — SQL-005: "In use by N tabs" badge

**Missing:** Delete saved query dialog doesn't show how many tabs reference it.
Needs backend endpoint.

**Fix:**
- Backend: `GET /api/v1/sqllab/saved-queries/:id/usage` → `{ tab_count: N }`.
  Query `SELECT COUNT(*) FROM tab_state WHERE saved_query_id = ?`. Handler in
  `sqllab/handler.go`. Route in router.
- Frontend `SavedQueriesPage.tsx`: fetch count when delete dialog opens.
  Show `<Badge variant="secondary">In use by {count} tab(s)</Badge>` in
  `AlertDialog` body before delete button. If count > 0, add info text
  "These tabs will reference a deleted query."

---

### Gap F7 — SQL-007: Monaco autocomplete provider

**Missing:** Entire feature — no `registerCompletionItemProvider`, no API call,
no suggestion mapping.

**Fix:**
1. Add `api/sqllab.ts` function: `autocomplete(params)` → POST
   `/api/v1/sqllab/autocomplete`
2. In `handleEditorMount` in `SQLLabPage.tsx`, register completion provider:
```ts
monaco.languages.registerCompletionItemProvider("sql", {
  triggerCharacters: [".", " ", "\n"],
  provideCompletionItems: async (model, position) => {
    const word = model.getWordUntilPosition(position);
    const prefix = model.getValueInRange({
      startLineNumber: position.lineNumber, startColumn: 1,
      endLineNumber: position.lineNumber, endColumn: position.column,
    });
    // Call POST /api/v1/sqllab/autocomplete
    // Map AutocompleteSuggestion[] → monaco.languages.CompletionItem[]
    // Map types: keyword→Keyword, schema→Module, table→Class, column→Field, function→Function
  }
});
```
3. Debounce trigger: only call API after 200ms of no typing
4. `cache_miss` handling: show dismissible `<Alert>` below editor
5. Auto-hide alert when next request returns `cache_miss: false`

---

## Files Changed

| File | Gaps |
|------|------|
| `backend/internal/domain/query/sqllab_types.go` | B1 |
| `backend/internal/domain/query/sqllab_repository.go` | B1 |
| `backend/internal/repository/postgres/sqllab_repo.go` | B1, B2, F6 |
| `backend/internal/delivery/http/sqllab/handler.go` | B1, F6 |
| `backend/internal/delivery/http/router.go` | F6 |
| `frontend/src/stores/sqlLabStore.ts` | F1 |
| `frontend/src/pages/sqllab/SQLLabPage.tsx` | F1, F2, F3, F4, F5, F7 |
| `frontend/src/pages/sqllab/SavedQueriesPage.tsx` | F6 |
| `frontend/src/api/sqllab.ts` | F7 |

## Testing

- Backend B1: Update `tabToResponse` test to verify `LatestQuery` object
- Backend B2: Test published query visible to other user; unpublished not visible
- Backend F6: Test usage endpoint returns correct tab count
- Frontend: Manual verification via dev server + browser for all 7 UI gaps
