<!-- Generated: 2026-04-14 | Files scanned: 120 | Token estimate: ~640 -->

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

Protected
/                                   -> HomePage

Admin area (session-protected in frontend; role enforced by backend APIs)
/admin                              -> AdminLayout
/admin/dashboard                    -> AdminDashboardPage
/admin/settings/roles               -> RolesPage
/admin/settings/roles/:id/permissions -> RolePermissionsPage
/admin/settings/users               -> UsersPage
/admin/settings/users/:id           -> UserRolesPage
/admin/settings/databases           -> DatabasesPage
/admin/settings/databases/new       -> CreateDatabasePage
/admin/settings/databases/:id       -> EditDatabasePage
/admin/settings/permissions         -> PermissionsPage

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

## State and API

```
stores/authStore.ts
  - auth/session state
  - login/logout/setSession style actions

hooks/useLogin.ts
hooks/useRegister.ts
hooks/useLogout.ts
hooks/useTokenRefresh.ts
hooks/useDatabaseIntrospection.ts
  - orchestrate API calls, redirects, and toasts

api/auth.ts + api/users.ts + api/userRoles.ts + api/roles.ts + api/permissions.ts + api/databases.ts + utils/request.ts
  - backend calls and request helpers
```

## Key Files

- `frontend/src/App.tsx`: route definitions and access controls.
- `frontend/src/main.tsx`: React Query client configuration and bootstrap.
- `frontend/src/components/ProtectedRoute.tsx`: route guard.
- `frontend/src/pages/auth/*`: login + verification views.
- `frontend/src/pages/register/*`: registration + success flow.
- `frontend/src/pages/admin/RolesPage.tsx`: AUTH-007 role CRUD screen.
- `frontend/src/pages/admin/UsersPage.tsx`: admin user CRUD and deactivate screen.
- `frontend/src/pages/admin/UserRolesPage.tsx`: AUTH-010 user-role assignment screen.
- `frontend/src/pages/admin/PermissionsPage.tsx`: AUTH-008 permission/view-menu matrix screen.
- `frontend/src/pages/admin/DatabasesPage.tsx`: database list, row actions, and delete confirmation.
- `frontend/src/pages/admin/CreateDatabasePage.tsx`: DBC wizard with create/edit mode, connection test UX (spinner, result alerts, latency badge, error details, 429 toast), and cache invalidation after save.
- `frontend/src/pages/admin/EditDatabasePage.tsx`: thin route wrapper that reuses `CreateDatabasePage` in edit mode.
- `frontend/src/pages/admin/*`: admin dashboard and settings shell.
- `frontend/src/api/databases.ts`: database API client (list/get/create/update/delete/testConnection/testConnectionById/getSchemas/getTables/getColumns).
- `frontend/src/hooks/useDatabaseIntrospection.ts`: DBC-007 query hooks using TanStack Query keys `db-schemas`, `db-tables`, `db-columns` with stale time 10 minutes.
- `frontend/src/utils/request.ts`: shared request helper with normalized errors and safe handling for 204 or empty/non-JSON responses.
- `frontend/src/stores/authStore.ts`: shared auth state.
- `frontend/src/test/setup.ts`: Vitest DOM setup.

## Build/Test Config

```
vite.config.ts      build + dev config
tailwind.config.js  utility class scan + theme settings
components.json     shadcn component registry
package.json        scripts: dev/build/test/test:coverage
```
