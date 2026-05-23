# SQL-003: Close and Delete Tabs — Design Spec

**Date:** 2026-05-23
**Phase:** 2 (depends on SQL-001)
**Priority:** P0

---

## 1. Summary

Soft-close tabs (active=false, retained for recovery), hard-delete tabs (permanent, cascades table_schema), bulk close-all/close-others, and reopen from a "Recently Closed" sheet. All UX actions soft-close; hard delete is API-only.

---

## 2. Backend

### 2.1 Repository Interface

File: `backend/internal/domain/query/sqllab_repository.go`

Add to `SQLLabRepository`:

```go
CloseTab(ctx context.Context, id uint, userID uint) error
CloseAllTabs(ctx context.Context, userID uint, exceptID *uint) (int64, error)
ReopenTab(ctx context.Context, id uint, userID uint) error
HardDelete(ctx context.Context, id uint, userID uint) error
```

Modify `ListByUser`:
```go
ListByUser(ctx context.Context, userID uint, includeClosed bool) ([]*TabState, error)
```

### 2.2 Repository Implementation

File: `backend/internal/repository/postgres/sqllab_repo.go`

**CloseTab** — `UPDATE tab_state SET active=false WHERE id=? AND user_id=? AND active=true`. Returns error on RowsAffected==0.

**CloseAllTabs** — `UPDATE tab_state SET active=false WHERE user_id=? AND active=true` with optional `AND id != ?` for exceptID. Returns RowsAffected count.

**ReopenTab** — `UPDATE tab_state SET active=true, changed_on=NOW() WHERE id=? AND user_id=? AND active=false`. Returns error on RowsAffected==0.

**HardDelete** — Transaction: verify ownership, `DELETE FROM table_schema WHERE tab_state_id=?`, then `DELETE FROM tab_state WHERE id=? AND user_id=?`. Returns error on not found.

**ListByUser** — Extended with `includeClosed bool` parameter. When false, adds `WHERE active=true`. Existing LEFT JOIN query unchanged.

### 2.3 Request/Response Types

File: `backend/internal/domain/query/sqllab_types.go`

```go
type CloseAllTabsRequest struct {
    ExceptID *uint `json:"except_id"`
}
```

### 2.4 Handler

File: `backend/internal/delivery/http/sqllab/handler.go`

All handlers follow the existing pattern: `getUserContext(c)`, parse params, enforce ownership, call repo, return JSON.

#### CloseTab — `PUT /api/v1/sqllab/tabs/:id/close`
1. Parse `:id` param, get user context
2. `repo.CloseTab(ctx, id, userID)`
3. On error: check ownership via `repo.GetByID` → 403 if owned by another user, 404 if not found
4. On success: 200 `{"closed": true}`

#### CloseAllTabs — `POST /api/v1/sqllab/tabs/close-all`
1. Bind JSON body (CloseAllTabsRequest, except_id optional)
2. `repo.CloseAllTabs(ctx, userID, req.ExceptID)`
3. Success: 200 `{"closed": N}`

#### ReopenTab — `PUT /api/v1/sqllab/tabs/:id/reopen`
1. Parse `:id`, get user context
2. `repo.ReopenTab(ctx, id, userID)`
3. Same ownership check as CloseTab for 403/404
4. Success: 200 `{"reopened": true, "id": N, "label": "..."}`

#### HardDeleteTab — `DELETE /api/v1/sqllab/tabs/:id`
1. Parse `:id`, get user context
2. `repo.HardDelete(ctx, id, userID)`
3. Same ownership check for 403/404
4. Success: 204 No Content

#### ListTabs — Modified
1. Parse query param `include_closed` (default false)
2. `repo.ListByUser(ctx, userID, includeClosed)`
3. Response unchanged (returns active tabs by default, all tabs when include_closed=true)

### 2.5 Router

File: `backend/internal/delivery/http/router.go`

New routes in the `sqlLab` group:
```go
sqlLab.PUT("/tabs/:id/close", sqllabHandler.CloseTab)
sqlLab.POST("/tabs/close-all", sqllabHandler.CloseAllTabs)
sqlLab.PUT("/tabs/:id/reopen", sqllabHandler.ReopenTab)
sqlLab.DELETE("/tabs/:id", sqllabHandler.HardDeleteTab)
```

### 2.6 Error Matrix

| Scenario | Status | Body |
|----------|--------|------|
| Close active tab (own) | 200 | `{"closed": true}` |
| Close tab owned by another | 403 | `{"error": "forbidden"}` |
| Close already-closed or nonexistent | 404 | `{"error": "not_found"}` |
| Close all (N closed) | 200 | `{"closed": N}` |
| Close all (none open) | 200 | `{"closed": 0}` |
| Close others (N closed) | 200 | `{"closed": N}` |
| Reopen own closed tab | 200 | `{"reopened": true, "id": N, "label": "..."}` |
| Reopen another user's tab | 403 | `{"error": "forbidden"}` |
| Reopen nonexistent tab | 404 | `{"error": "not_found"}` |
| Hard delete own tab | 204 | empty |
| Hard delete another user's | 403 | `{"error": "forbidden"}` |
| Hard delete nonexistent | 404 | `{"error": "not_found"}` |
| Missing JWT | 401 | `{"error": "unauthorized"}` |
| Invalid id param | 400 | `{"error": "invalid_id"}` |

### 2.7 Tests

File: `backend/internal/delivery/http/sqllab/handler_test.go`

~12 new test cases following the existing mock-repo + Gin test router pattern:

- CloseTab: success 200, not-owner 403, not-found 404
- CloseAllTabs: multiple closed, except_id exclusion, zero open
- ReopenTab: success 200, not-owner 403, not-found 404
- HardDeleteTab: success 204, not-owner 403, not-found 404
- ListTabs with include_closed: returns both active and closed

---

## 3. Frontend

### 3.1 API Layer

File: `frontend/src/api/sqllab.ts`

Four new functions:

```ts
closeTab(id: number): Promise<{ closed: boolean }>
closeAllTabs(exceptId?: number): Promise<{ closed: number }>
reopenTab(id: number): Promise<{ reopened: boolean; id: number; label: string }>
```

Modify `fetchTabs` to accept optional `include_closed` parameter. When true, appends `?include_closed=true` to the URL.

### 3.2 Zustand Store

File: `frontend/src/stores/sqlLabStore.ts`

- Rename `removeTab` → `removeTabFromState` (local-only cleanup, called from mutation onSuccess)
- Rename `closeAllTabs` → `clearTabsState` (local-only, same pattern)
- Reopen does not need a new store action — it uses `invalidateQueries` which triggers the existing `initTabs` flow via the `useEffect` watching `tabsData`

### 3.3 TanStack Query Mutations

In `SQLLabPage.tsx`, added alongside the existing `createTabMutation`:

- `closeTabMutation` — onSuccess calls `removeTabFromState(id)`, invalidates query
- `closeAllTabsMutation` — onSuccess invalidates query, shows toast with count
- `reopenTabMutation` — onSuccess invalidates query, closes sheet
- `useQuery(["sqllab-tabs", "closed"], fetchClosedTabs, { enabled: isSheetOpen })` — lazy query for Sheet

### 3.4 UX Flows

#### Close single tab (X button / "Close" context menu)

```
User clicks X or "Close"
  → tab.status === "running" OR tab.isDirty?
    YES → AlertDialog confirmation
      Cancel → nothing
      Close → closeTabMutation.mutate(id)
    NO → closeTabMutation.mutate(id) immediately
```

Confirmation reasons:
- "running" — "A query is still running on this tab. Closing will not stop the query — it will continue on the server."
- "dirty" — "This tab has unsaved changes."
- "both" — combined message for both conditions
- X button only shown when `tabs.length > 1`
- "Close" context menu item disabled when `tabs.length <= 1`

#### Close All

```
User right-clicks → "Close All"
  → AlertDialog: "Close all N tabs?"
    Cancel → nothing
    Close All → closeAllTabsMutation.mutate()
```

#### Close Others

```
User right-clicks → "Close Others"
  → closeAllTabsMutation.mutate(activeTabId)
  → No confirmation dialog (not destructive to active tab)
```

#### Reopen Closed Tab

```
User presses Ctrl+Shift+T or clicks "Reopen Closed Tab" in context menu
  → Sheet slides open from right
  → useQuery fetches tabs with include_closed=true
  → Lists closed tabs with label, closed date, "Reopen" button
  → Click "Reopen" → reopenTabMutation.mutate(id)
  → On success: sheet closes, tab reappears in main tab bar
```

### 3.5 Components

| Component | shadcn/ui | Usage |
|-----------|-----------|-------|
| AlertDialog | `alert-dialog.tsx` | Close confirmation for dirty/running tabs and "Close All" |
| Sheet | `sheet.tsx` | "Recently Closed" list, slides from right, 400px wide |
| DropdownMenu | `dropdown-menu.tsx` | Right-click context menu (add "Close Others", "Reopen Closed Tab") |
| Sonner | `sonner.tsx` | Toast notifications for success/error |

All components are already installed in `components/ui/`. No new files.

### 3.6 Keyboard Shortcut

Ctrl+Shift+T opens the "Recently Closed" Sheet. Implemented via a `keydown` event listener in a `useEffect`.

### 3.7 Files Changed (Frontend)

| File | Changes |
|------|---------|
| `api/sqllab.ts` | 4 new API functions, modify fetchTabs signature |
| `stores/sqlLabStore.ts` | Rename 2 actions, add 1 new action |
| `pages/sqllab/SQLLabPage.tsx` | 3 mutations, 1 query, AlertDialog, Sheet, keyboard handler, modified X button + context menu |

Zero new files.

---

## 4. Files Changed (Complete List)

| File | Layer | Changes |
|------|-------|---------|
| `backend/internal/domain/query/sqllab_repository.go` | Interface | 4 new methods, modify ListByUser |
| `backend/internal/domain/query/sqllab_types.go` | Types | CloseAllTabsRequest |
| `backend/internal/repository/postgres/sqllab_repo.go` | Repo | 4 implementations, modify ListByUser |
| `backend/internal/delivery/http/sqllab/handler.go` | Handler | 5 handler methods, modify ListTabs |
| `backend/internal/delivery/http/router.go` | Router | 4 new routes |
| `backend/internal/delivery/http/sqllab/handler_test.go` | Tests | ~12 test cases |
| `frontend/src/api/sqllab.ts` | API | 4 new functions, modify fetchTabs |
| `frontend/src/stores/sqlLabStore.ts` | Store | Rename 2, add 1 action |
| `frontend/src/pages/sqllab/SQLLabPage.tsx` | Page | Mutations, Sheet, AlertDialog, keyboard shortcut |
