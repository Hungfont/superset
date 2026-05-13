<!-- Generated: 2026-05-13 | Files scanned: ~250 | Token estimate: ~850 -->

# Frontend Codemap

Entry point: `frontend/src/main.tsx`
Stack: React 18 + TypeScript + Vite + React Query + React Router + Zustand + Tailwind + shadcn/ui

## Route Tree (`src/App.tsx`)

```
Public
/login                              -> LoginPage
/register                           -> RegisterPage
/register/success                   -> RegisterSuccessPage
/auth/verify                        -> VerifyPage

Protected (session required)
/                                   -> HomePage
/sqllab                            -> SQLLabPage
/explore                            -> ExplorePage

Admin (authorization enforced by backend APIs)
* /admin                              -> AdminLayout
* /admin/dashboard                    -> AdminDashboardPage
* /admin/settings/roles               -> RolesPage
* /admin/settings/roles/:id/permissions -> RolePermissionsPage
* /admin/settings/users               -> UsersPage
* /admin/settings/users/:id           -> UserRolesPage
* /admin/settings/databases           -> DatabasesPage
* /admin/settings/databases/new       -> CreateDatabasePage
* /admin/settings/databases/:id       -> EditDatabasePage
* /admin/settings/datasets            -> DatasetsPage (admin)
* /admin/settings/datasets/new        -> CreateDatasetPage
* /admin/settings/datasets/:id/edit   -> EditDatasetPage
* /admin/settings/permissions         -> PermissionsPage
* /admin/security/rls                 -> RLSFiltersPage

Fallback
* -> redirect /login
```

## Component/Flow Map

```
main.tsx
  -> QueryClientProvider
    -> App (BrowserRouter)
      -> ProtectedRoute (auth gate)
      -> pages/*
      -> Toaster (sonner)
```

## Component Inventory (`src/components/query/`)

| Component | Purpose |
|-----------|---------|
| `QueryBadges.tsx` | CacheBadge (cached/live + force refresh), RLSBadge, QueryStatusBadge (idle/running/success/error), RunButton, AsyncStatusBadge (pending/running/done/failed/stopped), RunAsyncButton, CancelButton, AsyncProgressBar, QueueBadge |
| `QueryHistoryTable.tsx` | Paginated query history with SQL search, status filter, Run Again / Load SQL / Download actions, admin Clear History (QE-007) |
| `DownloadButton.tsx` | Large result download via JWT token in query string and window.open (QE-005) |
| `WsStatusBadge.tsx` | WebSocket connection status (connected/reconnecting/disconnected) per query (QE-005) |

## API Calling Conventions

### Two HTTP clients:
- **`request`** (`utils/request.ts`): Simple fetch wrapper. Manual header control.
- **`apiFetch`** (`lib/api/client.ts`): Auto-bearer token + refresh handling.

### When to use which:
- Use **`request`** when you need manual control over headers or don't need auto-refresh.
- Use **`apiFetch`** for most authenticated API calls (auto-bearer + refresh).

### Pattern for API functions:
```typescript
import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}
```

## State and API

```
stores/authStore.ts
  - auth/session state
  - login/logout/setSession style actions

stores/sqlLabStore.ts
  - SQL Lab multi-tab state
  - query results cache, async query state
  - from_cache badge support (QE-003)
  - force refresh capability
  - async query status polling (QE-004)
  - download URL for large results (QE-005)

stores/wsStore.ts
  - WebSocket connection state per query ID (QE-005)
  - subscribe/unsubscribe with auto-reconnect (exponential backoff, max 3 retries)
  - fallback to polling notification after retries exhausted

hooks/useLogin.ts
hooks/useRegister.ts
hooks/useLogout.ts
hooks/useTokenRefresh.ts
hooks/useDatabaseIntrospection.ts
hooks/useLoading.ts
  - orchestrate API calls, redirects, and toasts

api/auth.ts + api/users.ts + api/userRoles.ts + api/roles.ts + api/permissions.ts + api/databases.ts + api/datasets.ts + api/queries.ts + api/rlsFilters.ts + utils/request.ts
  - query execution with from_cache badge support (QE-003)
  - async query submit/status/result/cancel (QE-004/006)
  - query history list/delete (QE-007)
  - result download via token
  - backend calls and request helpers
```

## Key Files

- `frontend/src/App.tsx`: route definitions and access controls.
- `frontend/src/main.tsx`: React Query client configuration and bootstrap.
- `frontend/src/components/ProtectedRoute.tsx`: route guard.
- `frontend/src/pages/auth/*`: login + verification views.
- `frontend/src/pages/register/*`: registration + success flow.
- `frontend/src/pages/home/HomePage.tsx`: main dashboard page.
- `frontend/src/pages/sqllab/SQLLabPage.tsx`: SQL editor with query execution, from_cache badge (QE-003), async query support (QE-004), WebSocket streaming (QE-005), cancel button (QE-006), query history panel (QE-007).
- `frontend/src/pages/explore/ExplorePage.tsx`: data exploration view.
- `frontend/src/pages/datasets/*`: dataset list, create, edit with ColumnsTab and MetricsTab.
- `frontend/src/pages/security/RLSFiltersPage.tsx`: Row-Level Security filter management.
- `frontend/src/pages/admin/RolesPage.tsx`: role CRUD screen.
- `frontend/src/pages/admin/RolePermissionsPage.tsx`: role-permission assignment.
- `frontend/src/pages/admin/UsersPage.tsx`: admin user CRUD and deactivate screen.
- `frontend/src/pages/admin/UserRolesPage.tsx`: user-role assignment screen.
- `frontend/src/pages/admin/PermissionsPage.tsx`: permission/view-menu matrix screen.
- `frontend/src/pages/admin/DatabasesPage.tsx`: database list, row actions, and delete confirmation.
- `frontend/src/pages/admin/CreateDatabasePage.tsx`: database wizard with connection test and cache invalidation.
- `frontend/src/pages/admin/EditDatabasePage.tsx`: thin route wrapper reusing CreateDatabasePage.
- `frontend/src/pages/admin/DatasetsPage.tsx`: admin datasets management.
- `frontend/src/pages/admin/AdminLayout.tsx`: admin area layout shell.
- `frontend/src/pages/admin/AdminDashboardPage.tsx`: admin dashboard.
- `frontend/src/components/query/QueryBadges.tsx`: cache/RLS/status/async/cancel badges and buttons (QE-003/005/006).
- `frontend/src/components/query/QueryHistoryTable.tsx`: paginated query history with filters and admin clear (QE-007).
- `frontend/src/components/query/DownloadButton.tsx`: large result download via token (QE-005).
- `frontend/src/components/query/WsStatusBadge.tsx`: WebSocket connection display (QE-005).
- `frontend/src/api/databases.ts`: database API client.
- `frontend/src/api/datasets.ts`: dataset CRUD + metrics API client.
- `frontend/src/api/queries.ts`: query execution API client (sync + async: submit/status/result/cancel), history list/delete (QE-007), result download, WebSocket event type definitions.
- `frontend/src/api/rlsFilters.ts`: RLS filter API client.
- `frontend/src/stores/authStore.ts`: shared auth state.
- `frontend/src/stores/sqlLabStore.ts`: SQL Lab state (tabs, queries, results, async, download URL).
- `frontend/src/stores/wsStore.ts`: WebSocket connection store (QE-005) -- per-query WS lifecycle, auto-reconnect with backoff, fallback flag.
- `frontend/src/utils/request.ts`: shared request helper.
- `frontend/src/test/setup.ts`: Vitest DOM setup.

## Build/Test Config

```
vite.config.ts      build + dev config
tailwind.config.js  utility class scan + theme settings
components.json     shadcn component registry
package.json        scripts: dev/build/test/test:coverage
```
