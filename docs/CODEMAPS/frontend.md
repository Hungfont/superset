<!-- Generated: 2026-05-04 | Files scanned: 120 | Token estimate: ~780 -->

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
/datasets                          -> ListDatasetsPage
/datasets/new                      -> CreateDatasetPage
/datasets/:id                      -> EditDatasetPage

Admin (role-checked in backend)
* /admin                              -> AdminLayout
* /admin/dashboard                    -> AdminDashboardPage
* /admin/settings/roles               -> RolesPage
* /admin/settings/roles/:id/permissions -> RolePermissionsPage
* /admin/settings/users               -> UsersPage
* /admin/settings/users/:id           -> UserRolesPage
* /admin/settings/databases           -> DatabasesPage
* /admin/settings/databases/new       -> CreateDatabasePage
* /admin/settings/databases/:id       -> EditDatabasePage
* /admin/settings/permissions         -> PermissionsPage
* /admin/settings/rls                 -> RLSFiltersPage

Fallback
* -> redirect /login
```

## Component/Flow Map

```
main.tsx
  -> QueryClientProvider
    -> App
      -> ProtectedRoute (auth + role gate)
      -> pages/*
      -> Toaster
```

## API Calling Conventions

### Two HTTP clients:
- **`request`** (`utils/request.ts`): Simple fetch wrapper for public/unauthenticated endpoints. Does not add Authorization header.
- **`apiFetch`** (`lib/api/client.ts`): Adds Bearer token automatically, handles token refresh. Use for authenticated endpoints.

### Pattern for new API files:
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

export const myApi = {
  fetch: () => request("/api/endpoint", {
    method: "GET",
    credentials: "include",
    headers: getAuthHeaders(),
  }),
};
```

### When to use which:
- Use **`request`** when you need manual control over headers or don't need auth.
- Use **`apiFetch`** for most authenticated API calls (auto-bearer + refresh).

## State and API

```
stores/authStore.ts
  - auth/session state
  - login/logout/setSession style actions

stores/sqlLabStore.ts
  - SQL Lab query state
  - query history, results cache
  - from_cache badge display (QE-003)
  - force refresh capability
  - async query status polling (QE-004)

hooks/useLogin.ts
hooks/useRegister.ts
hooks/useLogout.ts
hooks/useTokenRefresh.ts
hooks/useDatabaseIntrospection.ts
hooks/useAsyncQuery.ts (QE-004)
hooks/useLoading.ts
  - orchestrate API calls, redirects, and toasts

api/auth.ts + api/users.ts + api/userRoles.ts + api/roles.ts + api/permissions.ts + api/databases.ts + api/datasets.ts + api/queries.ts + api/rlsFilters.ts + utils/request.ts
  - query execution with from_cache badge support (QE-003)
  - async query submit/status/result/cancel (QE-004)
  - backend calls and request helpers
```

## Key Files

- `frontend/src/App.tsx`: route definitions and access controls.
- `frontend/src/main.tsx`: React Query client configuration and bootstrap.
- `frontend/src/components/ProtectedRoute.tsx`: route guard.
- `frontend/src/pages/auth/*`: login + verification views.
- `frontend/src/pages/register/*`: registration + success flow.
- `frontend/src/pages/home/HomePage.tsx`: main dashboard page.
- `frontend/src/pages/sqllab/SQLLabPage.tsx`: SQL editor with query execution, from_cache badge (QE-003), async query support (QE-004).
- `frontend/src/pages/datasets/*`: dataset list, create, edit pages.
- `frontend/src/pages/security/RLSFiltersPage.tsx`: Row-Level Security filter management.
- `frontend/src/pages/admin/RolesPage.tsx`: role CRUD screen.
- `frontend/src/pages/admin/UsersPage.tsx`: admin user CRUD and deactivate screen.
- `frontend/src/pages/admin/UserRolesPage.tsx`: user-role assignment screen.
- `frontend/src/pages/admin/PermissionsPage.tsx`: permission/view-menu matrix screen.
- `frontend/src/pages/admin/DatabasesPage.tsx`: database list, row actions, and delete confirmation.
- `frontend/src/pages/admin/CreateDatabasePage.tsx`: database wizard with connection test and cache invalidation.
- `frontend/src/pages/admin/EditDatabasePage.tsx`: thin route wrapper reusing CreateDatabasePage.
- `frontend/src/pages/admin/AdminLayout.tsx`: admin area layout shell.
- `frontend/src/pages/admin/AdminDashboardPage.tsx`: admin dashboard.
- `frontend/src/api/databases.ts`: database API client.
- `frontend/src/api/datasets.ts`: dataset CRUD + metrics API client.
- `frontend/src/api/queries.ts`: query execution API client (sync + async: submit/status/result/cancel), cache flush.
- `frontend/src/api/rlsFilters.ts`: RLS filter API client.
- `frontend/src/hooks/useDatabaseIntrospection.ts`: schema introspection query hooks.
- `frontend/src/hooks/useAsyncQuery.ts`: async query submission, status polling, result retrieval, cancellation (QE-004).
- `frontend/src/stores/authStore.ts`: shared auth state.
- `frontend/src/stores/sqlLabStore.ts`: SQL Lab state (queries, results, history).
- `frontend/src/utils/request.ts`: shared request helper.
- `frontend/src/test/setup.ts`: Vitest DOM setup.

## Build/Test Config

```
vite.config.ts      build + dev config
tailwind.config.js  utility class scan + theme settings
components.json     shadcn component registry
package.json        scripts: dev/build/test/test:coverage
```
