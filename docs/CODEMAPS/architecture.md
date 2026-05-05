<!-- Generated: 2026-05-05 | Files scanned: 180 | Token estimate: ~620 -->

# Architecture

## System Overview

```
Browser (React + Vite)
  -> Frontend router/guards (ProtectedRoute)
    -> REST API /api/v1/*
      -> Go service (Gin)
        -> App services (auth, role, permission, database, dataset, query, rls)
          -> Postgres (users, register_users, roles, permissions, view_menus, permission_views, dbs, datasets, columns, metrics, rls_filters)
          -> Redis (jwt blocklist, refresh sessions, role cache, rate limits, schema introspection cache, dataset sync/async queues, query result cache)
          -> External databases via pooled SQL connections for schema/table/column discovery and query execution
          -> SMTP (verification email)
```

## Primary Request Flows

```
Register:
POST /api/v1/auth/register
  -> RegisterHandler -> RegisterService -> RegisterUserRepository + SMTPSender

Verify:
GET /api/v1/auth/verify
  -> VerifyHandler -> VerifyService -> VerifyRepository

Login/Session:
POST /api/v1/auth/login
  -> LoginHandler -> LoginService -> LoginRepository + RateLimitRepository + RefreshRepository
POST /api/v1/auth/refresh
  -> RefreshHandler -> RefreshService -> RefreshRepository + UserRepository
POST /api/v1/auth/logout
  -> LogoutHandler -> LogoutService -> JWTRepository + RefreshRepository

Dataset Management:
GET/POST/PUT/DELETE /api/v1/datasets...
  -> JWTMiddleware -> DatasetHandler -> DatasetService -> DatasetRepository + SyncQueue + AsyncQueue

SQL Query Execution:
POST /api/v1/query/execute
  -> JWTMiddleware -> QueryHandler -> QueryExecutor -> DatabaseRepository + RLSInjector + RLSFilterRepository -> cache results

RLS Filter Management:
GET/POST/PUT/DELETE /api/v1/admin/rls (admin role required)
  -> JWTMiddleware + AuthorizeAdminRole -> RLSHandler -> RLSService -> RLSFilterRepository

Protected RBAC:
GET/POST/PUT/DELETE /api/v1/admin/roles...
  -> JWTMiddleware -> RoleHandler -> RoleService -> RoleRepository + RoleCacheRepository

Permission Management:
GET/POST /api/v1/admin/permissions
GET/POST /api/v1/admin/view-menus
GET/POST/DELETE /api/v1/admin/permission-views
  -> JWTMiddleware -> PermissionHandler -> PermissionService -> PermissionRepository + RoleCacheRepository

Database Schema Introspection (DBC-007):
GET /api/v1/admin/databases/:id/schemas
GET /api/v1/admin/databases/:id/tables?schema=...
GET /api/v1/admin/databases/:id/columns?schema=...&table=...
  -> JWTMiddleware -> DatabaseHandler -> DatabaseService
    -> ConnectionPoolManager.Get
    -> Schema cache lookup (Redis, TTL 10m)
    -> SchemaInspector (INFORMATION_SCHEMA)
    -> force_refresh=true bypasses cache (rate limited: 5/min)
```

## Backend Layer Map

```
cmd/api/main.go                     bootstrap, config, DI wiring
internal/delivery/http/router.go    route groups + middleware hookup
internal/delivery/http/auth/*       request/response adapters
internal/delivery/http/dataset/*  dataset CRUD, metrics, columns handlers
internal/delivery/http/query/*    SQL query execution handler
internal/delivery/http/rls/*       Row-Level Security filter handler
internal/delivery/http/middleware/* JWT validation + context injection + admin role authorization
internal/app/auth/*                 business workflows
internal/app/db/*                   database lifecycle + connection pool + schema introspection
internal/app/dataset/*              dataset lifecycle + sync/async queue management
internal/app/query/*                 SQL executor + RLS injection + caching
internal/app/rls/*                   RLS filter service
internal/domain/auth/*              entities + interfaces + contracts
internal/domain/db/*                database contracts and introspection DTOs
internal/domain/dataset/*            dataset/column/metric entities
internal/repository/postgres/*      durable persistence
internal/repository/redis/*         cache/session/blocklist/rate storage + dataset queues
internal/worker/*                   background workers (column sync)
```

## Frontend Structure

```
src/main.tsx   React root + QueryClientProvider
src/App.tsx    public routes + protected routes + admin routes
src/stores/*   auth state + SQL Lab state
src/hooks/*    login/register/logout/refresh workflow hooks
src/pages/*    auth, home, sqllab, datasets, security, and admin views
src/api/*      backend API clients + queries, rls filters
```

## Reference Docs

- Sequence diagrams: `docs/diagram/sequence/`
- Data details: `docs/db/`
