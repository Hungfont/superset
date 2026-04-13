<!-- Generated: 2026-04-13 | Files scanned: 120 | Token estimate: ~520 -->

# Architecture

## System Overview

```
Browser (React + Vite)
  -> Frontend router/guards (ProtectedRoute)
    -> REST API /api/v1/*
      -> Go service (Gin)
        -> App services (auth, role)
          -> Postgres (users, register_users, roles)
          -> Redis (jwt blocklist, refresh sessions, role cache, rate limits)
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
GET/POST/PUT/DELETE /api/v1/roles...
  -> JWTMiddleware -> RoleHandler -> RoleService -> RoleRepository + RoleCacheRepository
```

## Backend Layer Map

```
cmd/api/main.go                     bootstrap, config, DI wiring
internal/delivery/http/router.go    route groups + middleware hookup
internal/delivery/http/auth/*       request/response adapters
internal/delivery/http/middleware/* JWT validation + context injection
internal/app/auth/*                 business workflows
internal/domain/auth/*              entities + interfaces + contracts
internal/repository/postgres/*      durable persistence
internal/repository/redis/*         cache/session/blocklist/rate storage
```

## Frontend Structure

```
src/main.tsx   React root + QueryClientProvider
src/App.tsx    public routes + protected routes + admin routes
src/stores/*   auth state
src/hooks/*    login/register/logout/refresh workflow hooks
src/pages/*    auth, home, admin/settings views
```

## Reference Docs

- Sequence diagrams: `docs/diagram/sequence/`
- Data details: `docs/db/`
