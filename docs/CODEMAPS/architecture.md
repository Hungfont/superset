<!-- Generated: 2026-04-14 | Files scanned: 120 | Token estimate: ~520 -->

# Architecture

## System Overview

```
Browser (React + Vite)
  -> Frontend router/guards (ProtectedRoute)
    -> REST API /api/v1/*
      -> Go service (Gin)
        -> App services (auth, role, permission, database)
          -> Postgres (users, register_users, roles, permissions, view_menus, permission_views, dbs)
          -> Redis (jwt blocklist, refresh sessions, role cache, rate limits, schema introspection cache)
          -> External databases via pooled SQL connections for schema/table/column discovery
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
internal/delivery/http/middleware/* JWT validation + context injection
internal/app/auth/*                 business workflows
internal/app/db/*                   database lifecycle + connection pool + schema introspection
internal/domain/auth/*              entities + interfaces + contracts
internal/domain/db/*                database contracts and introspection DTOs
internal/repository/postgres/*      durable persistence
internal/repository/redis/*         cache/session/blocklist/rate storage
```

## Frontend Structure

```
src/main.tsx   React root + QueryClientProvider
src/App.tsx    public routes + protected routes + admin routes
src/stores/*   auth state
src/hooks/*    login/register/logout/refresh workflow hooks
src/pages/*    auth, home, and admin views (dashboard/roles/permissions)
```

## Reference Docs

- Sequence diagrams: `docs/diagram/sequence/`
- Data details: `docs/db/`
