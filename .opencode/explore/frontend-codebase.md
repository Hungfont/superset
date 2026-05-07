1. FRONTEND LOCATION & TECH STACK
Root directory: D:\superset\frontend
Build tool: Vite (config: D:\superset\frontend\vite.config.ts), using @vitejs/plugin-react-swc for Fast Refresh. Path alias @ maps to ./src. Vite dev server proxies /api to backend (default http://localhost:3000).
Key config files:
- package.json — all dependencies
- tsconfig.json — TypeScript config
- components.json — shadcn/ui configuration
- tailwind.config.js — Tailwind CSS
- postcss.config.js — PostCSS
- index.html — entry HTML
Core dependencies (from package.json):
Category	Libraries
Framework	React 18.3.1, React Router DOM 6.26.2
State Management	Zustand 5.0.0 (persisted), TanStack Query 5.56.2
UI Components	shadcn/ui (Radix UI primitives), class-variance-authority, clsx, tailwind-merge, tailwindcss-animate
Forms/Validation	react-hook-form 7.72.1, @hookform/resolvers 3.10.0, Zod 3.25.76
Icons	lucide-react 0.462.0
Tables	@tanstack/react-table 8.21.3
Toast	sonner 2.0.7
Command palette	cmdk 1.1.1
Testing	Vitest 2.1.1, Testing Library (React + user-event + jest-dom), jsdom
---
2. DIRECTORY STRUCTURE
D:\superset\frontend\
  index.html
  package.json
  vite.config.ts
  tsconfig.json
  tailwind.config.js
  postcss.config.js
  components.json          # shadcn/ui config
  src/
    main.tsx                # Entry point: QueryClientProvider wrapping App
    App.tsx                 # BrowserRouter + all route definitions
    index.css               # Tailwind + CSS variables
    api/                    # API client modules (service layer)
      auth.ts / auth.test.ts
      databases.ts / databases.test.ts
      datasets.ts / datasets.test.ts
      queries.ts            # *** Query execution API ***
      roles.ts / roles.test.ts
      users.ts / users.test.ts
      userRoles.ts / userRoles.test.ts
      permissions.ts / permissions.test.ts
      rlsFilters.ts         # RLS filter CRUD API
    components/
      ProtectedRoute.tsx / .test.tsx   # Auth guard wrapper
      query/                           # *** Query-related UI components ***
        QueryBadges.tsx                # CacheBadge, RLSBadge, QueryStatusBadge,
        QueryBadges.test.tsx           #   RunButton, RunAsyncButton, CancelButton,
                                       #   AsyncStatusBadge, AsyncProgressBar, QueueBadge
      ui/                              # shadcn/ui primitive components (26 files)
        alert.tsx, alert-dialog.tsx, badge.tsx, button.tsx, card.tsx,
        checkbox.tsx, command.tsx, data-table.tsx, dialog.tsx,
        dropdown-menu.tsx, form.tsx, input.tsx, label.tsx, popover.tsx,
        progress.tsx, scroll-area.tsx, select.tsx, separator.tsx,
        sheet.tsx, skeleton.tsx, sonner.tsx, stepper.tsx, switch.tsx,
        tabs.tsx, textarea.tsx, tooltip.tsx
    hooks/                              # Custom React hooks
      use-toast.ts                      # Toast hook (sonner wrapper)
      useDatabaseIntrospection.ts       # TanStack Query hooks for DB schemas/tables/columns
      useLoading.ts                     # Generic loading state manager
      useLogin.ts                       # Login mutation (TanStack Query)
      useLogout.ts                      # Logout mutation (TanStack Query)
      useRegister.ts                    # Registration mutation
      useTokenRefresh.ts                # Proactive JWT refresh hook
    lib/
      utils.ts                          # cn() utility (clsx + tailwind-merge)
      api/
        client.ts                       # Authenticated fetch with automatic 401 refresh
      utils/
        database.ts                     # URI builder/parser/masker
      validations/                      # Zod schemas
        database.ts                     # Create/update database form schemas
        dataset.ts                      # Dataset form schemas
        login.ts                        # Login form schema
        register.ts / register.test.ts
        rls.ts                          # RLS filter schema
    pages/
      auth/
        LoginPage.tsx / LoginPage.test.tsx
        VerifyPage.tsx                  # Email verification
      home/
        HomePage.tsx                    # Landing page after login
      register/
        RegisterPage.tsx / .test.tsx
        RegisterSuccessPage.tsx
      sqllab/
        SQLLabPage.tsx                  # *** SQL Lab (query editor + results) ***
      explore/
        ExplorePage.tsx                 # Chart builder (ECharts)
      admin/
        AdminLayout.tsx                 # Admin layout with sidebar
        AdminDashboardPage.tsx
        DatabasesPage.tsx / .test.tsx   # List databases
        CreateDatabasePage.tsx / .test.tsx
        EditDatabasePage.tsx
        DatasetsPage.tsx                # List datasets
        RolesPage.tsx / .test.tsx
        PermissionsPage.tsx / .test.tsx
        RolePermissionsPage.tsx / .test.tsx
        UserRolesPage.tsx / .test.tsx
        UsersPage.tsx / .test.tsx
      datasets/
        CreateDatasetPage.tsx / .test.tsx
        EditDatasetPage.tsx
        MetricsTab.tsx / .test.tsx
        ColumnsTab.tsx
      security/
        RLSFiltersPage.tsx              # RLS filter management
    stores/                             # Zustand stores
      authStore.ts                      # User auth state, JWT token, persisted to localStorage
      sqlLabStore.ts                    # *** SQL Lab tab state (tabs, results, status) ***
    test/
      setup.ts                          # Vitest/jest-dom setup, localStorage mock, ResizeObserver mock
    utils/
      request.ts                        # fetch wrapper with error handling
---
3. QUERY-RELATED FILES (Components, Hooks, Stores, API)
a) API Layer — D:\superset\frontend\src\api\queries.ts
The central query execution API module. Exports queriesApi object with these methods:
Method	Endpoint
execute(data)	POST /api/v1/query/execute
submit(data)	POST /api/v1/query/submit
getStatus(queryId)	GET /api/v1/query/:id/status
cancel(queryId)	DELETE /api/v1/query/:id
getHistory(params?)	GET /api/v1/query/history
getResult(queryId)	GET /api/v1/query/:id/result
Key TypeScript interfaces: ExecuteQueryRequest, ExecuteQueryResponse, SubmitQueryRequest, SubmitQueryResponse, QueryStatusResponse, QueryColumn, QueryMeta.
b) Zustand Store — D:\superset\frontend\src\stores\sqlLabStore.ts
Manages all SQL Lab tab state. Created with create() from Zustand (no persistence middleware).
Interface SqlLabTab:
- id, title, sql, databaseId, schema
- result: QueryResult | null (data, columns, from_cache, query meta)
- status: "idle" | "running" | "success" | "error"
- error: string | null
- Async fields: asyncQueryId, asyncStatus, asyncQueue
Store actions: addTab, removeTab, setActiveTab, updateTabSql, updateTabDatabase, setTabResult, setTabStatus, setTabError, setDatabaseId, setAsyncState, setAsyncResult, clearAsyncState.
c) Query UI Components — D:\superset\frontend\src\components\query\QueryBadges.tsx
Exported components used by SQL Lab and Explore pages:
Component	Props
CacheBadge	fromCache, durationMs, cachedAt?, ttlSeconds?, onForceRefresh
RLSBadge	executedSql, originalSql
QueryStatusBadge	status
RunButton	onClick, disabled, isRunning
RunAsyncButton	onClick, disabled, isRunning, isQueued
CancelButton	onClick, disabled
AsyncStatusBadge	status
AsyncProgressBar	status
QueueBadge	queue
Tested in QueryBadges.test.tsx.
d) Custom Hooks
useDatabaseIntrospection.ts — Three TanStack Query hooks:
- useDatabaseSchemasQuery(databaseId?, forceRefresh?) — fetches schema list
- useDatabaseTablesQuery(databaseId?, schema?, page?, pageSize?, forceRefresh?) — fetches tables for a schema
- useDatabaseColumnsQuery(databaseId?, schema?, table?, forceRefresh?) — fetches columns for a table
All use queryKey patterns like ["db-schemas", databaseId, forceRefresh] with staleTime: 10 minutes.
useLoading.ts — Generic loading state manager (not TanStack Query). Provides withLoading wrapper, isLoading checker, startLoading/stopLoading for keyed loading counts.
---
4. SQL LAB RELATED FILES
File	Role
D:\superset\frontend\src\pages\sqllab\SQLLabPage.tsx	Main SQL Lab page — 633 lines. Multi-tab SQL editor with synchronous and asynchronous execution, WebSocket support, polling, browser notifications. Uses three useMutation hooks: executeMutation, submitAsyncMutation, cancelMutation. Auto-detects when to switch to async mode (if last query > 5s).
D:\superset\frontend\src\stores\sqlLabStore.ts	Zustand store managing all tabs, results, and async state
D:\superset\frontend\src\api\queries.ts	Query execution API (execute, submit, status, cancel, history, result)
D:\superset\frontend\src\components\query\QueryBadges.tsx	All query status/cache/RLS/progress UI components used in SQL Lab
Key behaviors in SQLLabPage.tsx:
- Auto-async threshold: Queries running > 5s auto-switch to async mode on next run
- Polling: 2s interval polling via setInterval + queriesApi.getStatus()
- WebSocket: Connects to ws://<host>/ws/query/<queryId>?token=<jwt> for real-time updates
- Notifications: Browser Notification API for query completion/failure/timeout
- RLS display: Compares executed_sql vs original sql to show RLS badge
- Timeout handling: Server sends timeout_at — client validates it and stops polling
---
5. ROUTE DEFINITIONS
Defined in D:\superset\frontend\src\App.tsx using React Router DOM v6:
PUBLIC ROUTES (no auth required):
  /login                   → LoginPage
  /register                → RegisterPage
  /register/success        → RegisterSuccessPage
  /auth/verify             → VerifyPage (email verification)
PROTECTED ROUTES (authenticated):
  /                        → HomePage
  /sqllab                  → SQLLabPage
  /explore                 → ExplorePage
ADMIN ROUTES (/admin/* all under ProtectedRoute + AdminLayout):
  /admin/dashboard                              → AdminDashboardPage
  /admin/settings/roles                         → RolesPage
  /admin/settings/roles/:id/permissions          → RolePermissionsPage
  /admin/settings/users                         → UsersPage
  /admin/settings/users/:id                     → UserRolesPage
  /admin/settings/databases                     → DatabasesPage
  /admin/settings/databases/new                 → CreateDatabasePage
  /admin/settings/databases/:id                 → EditDatabasePage
  /admin/settings/datasets                      → DatasetsPage
  /admin/settings/datasets/new                  → CreateDatasetPage
  /admin/settings/datasets/:id/edit             → EditDatasetPage
  /admin/settings/permissions                   → PermissionsPage
  /admin/security/rls                           → RLSFiltersPage
FALLBACK:
  *                          → Navigate to /login
ProtectedRoute component wraps protected routes — checks isAuthenticated from auth store, redirects to /login with state.from for return navigation. Also activates useTokenRefresh() for proactive JWT refresh.
---
6. STATE MANAGEMENT PATTERNS
Dual approach: Zustand + TanStack Query
Zustand (zustand 5.0.0)
Used for client-side state that needs to be shared across components without request caching:
1. authStore.ts (useAuthStore)
   - Persisted to localStorage via zustand/middleware (persist middleware)
   - Stores: user, accessToken, isAuthenticated, refreshTimer
   - Actions: setAuth, clearAuth, setAccessToken, setRefreshTimer
   - Used by API modules for auth headers, by hooks for login/logout, by ProtectedRoute
2. sqlLabStore.ts (useSqlLabStore)
   - Not persisted (in-memory only)
   - Manages SQL Lab tabs, active tab, SQL content, query results, async state
   - Used exclusively by SQLLabPage.tsx
Usage pattern in API modules:
const accessToken = useAuthStore.getState().accessToken;  // synchronous get
TanStack Query (@tanstack/react-query 5.56.2)
Used for server state — data fetching, caching, mutations:
- QueryClient created in main.tsx with staleTime: 30s, retry disabled for 401s
- Queries used in pages like useQuery({ queryKey: ["databases"], queryFn: ... })
- Mutations used for write operations: useMutation({ mutationFn: queriesApi.execute, onSuccess: ... })
- Custom hooks wrap TanStack Query: useDatabaseSchemasQuery, useDatabaseTablesQuery, useDatabaseColumnsQuery
Third pattern: apiFetch in D:\superset\frontend\src\lib\api\client.ts
A lower-level authenticated fetch client with:
- Automatic Bearer token injection
- Proactive JWT expiry detection (parse JWT exp claim)
- 401 response → silent refresh via POST /api/v1/auth/refresh
- Request queuing during refresh (only one refresh in-flight)
- Auto logout + redirect on second failure
Note: The API modules in src/api/ do NOT use this apiFetch — they use the simpler request() from utils/request.ts directly and manually attach Bearer headers via useAuthStore.getState().accessToken. The apiFetch client exists in lib/api/client.ts but does not appear to be imported anywhere yet.
---
7. COMPONENT LIBRARY (shadcn/ui)
Confirmed: shadcn/ui (configured in components.json)
- Style: default, base color: slate, CSS variables enabled
- Components in src/components/ui/ — 26 Radix-based primitives styled with Tailwind + CVA
- Import alias: @/components/ui
- Custom non-shadcn components: data-table.tsx (uses @tanstack/react-table), stepper.tsx
8. TEST SETUP
- Test runner: Vitest (configured in vite.config.ts — globals: true, environment: "jsdom")
- Setup file: src/test/setup.ts (jest-dom matchers, localStorage mock, ResizeObserver mock)
- Coverage threshold: 80% lines/functions/branches/statements (v8 provider)
- Tests are co-located with source files (*.test.ts, *.test.tsx)
- 15 test files found across api/, components/, hooks/, lib/validations/, pages/