# SQL-002: Auto-Save Tab SQL and Editor State

**Status:** Design approved
**Depends on:** SQL-001 (tab must exist)
**Priority:** P0, Phase 2

## Summary

Auto-save editor state (SQL, schema, label, db_id) to the backend via PUT. Transparent to the user — no save button. Dirty indicator on tab label. Latest query linkage after query execution.

## Backend: UpdateTab Gaps

### Request type expansion (`sqllab_types.go`)

Add to `UpdateTabRequest`:
- `DbID *uint` (`json:"db_id"`)
- `LatestQueryID *string` (`json:"latest_query_id"`)
- `HideLeftBar *bool` (`json:"hide_left_bar"`
- `ExtraJSON *string` (`json:"extra_json"`)

### SQL size validation (handler.go)

After `ShouldBindJSON`, before applying fields: if `req.SQL != nil && len(*req.SQL) > 65536` → 422 `{"error":"sql_too_large","message":"SQL exceeds 64KB limit"}`.

### Ownership enforcement

`GetByID` already scopes by user. If tab is nil, return 403 `{"error":"forbidden","message":"Not authorized to update this tab"}` instead of 404.

### Field application

Add the 4 new fields to the if-nil-then-set block:

```go
if req.DbID != nil     { tab.DbID = *req.DbID }
if req.LatestQueryID != nil { tab.LatestQueryID = *req.LatestQueryID }
if req.HideLeftBar != nil   { tab.HideLeftBar = *req.HideLeftBar }
if req.ExtraJSON != nil     { tab.ExtraJSON = *req.ExtraJSON }
```

## Frontend: `useAutoSaveTab` Hook

New file: `frontend/src/hooks/useAutoSaveTab.ts`

### Triggers

| Watch | Debounce | PUT payload |
|-------|----------|-------------|
| `tab.sql` | 1000ms | `{ sql }` |
| `tab.title` | 1000ms | `{ label }` |
| `tab.schema` | none (immediate) | `{ schema }` |

### Exports

```typescript
function useAutoSaveTab(tabId: string | null, tab: SqlLabTab | undefined): {
  linkLatestQuery: (queryId: string) => void
}
```

### Error handling

Failed PUT → `toast("Failed to save tab. Check connection.")`.

### Dirty tracking

- `updateTabSql` sets `isDirty: true` (currently only `updateTabLabel` does)
- Successful PUT clears `isDirty` to `false`

## Frontend: SQLLabPage.tsx Changes

1. **Replace inline auto-save**: Remove lines 696–710, call `useAutoSaveTab(activeTabId, activeTab)` instead.
2. **Query linkage**: In `executeMutation.onSuccess` and async query completion paths, call `linkLatestQuery(queryId)`.
3. **Dirty indicator**: In the tab trigger, prepend `•` when `tab.isDirty`:
   ```tsx
   {tab.isDirty && <span className="text-amber-500 mr-1">•</span>}
   ```

## Files Changed

| File | Change |
|------|--------|
| `backend/internal/domain/query/sqllab_types.go` | Add 4 fields to UpdateTabRequest |
| `backend/internal/delivery/http/sqllab/handler.go` | SQL validation, 403, apply new fields |
| `frontend/src/hooks/useAutoSaveTab.ts` | **New** — extracted auto-save hook |
| `frontend/src/pages/sqllab/SQLLabPage.tsx` | Replace inline effect, add query linkage, dirty indicator |
| `frontend/src/stores/sqlLabStore.ts` | Set `isDirty: true` in `updateTabSql` |
| `frontend/src/api/sqllab.ts` | Add new fields to `UpdateTabRequest` type |

## Acceptance Criteria

- Auto-save fires on sql/label change (debounced 1s) and schema change (immediate)
- SQL > 64KB → 422
- Non-owner → 403
- Tab label shows `•` when dirty
- Query completion links `latest_query_id`
- Network error → toast warning
