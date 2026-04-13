<!-- Generated: 2026-04-13 | Files scanned: 120 | Token estimate: ~680 -->

# Backend Codemap

Entry point: `backend/cmd/api/main.go`  
Module: `superset/auth-service`  
Runtime: Go 1.25 + Gin + GORM + Redis

## Route Map

```
Public
POST /api/v1/auth/register                 -> RegisterHandler.Register
GET  /api/v1/auth/verify                   -> VerifyHandler.Verify
POST /api/v1/auth/login                    -> LoginHandler.Login
POST /api/v1/auth/refresh                  -> RefreshHandler.Refresh
POST /api/v1/auth/logout                   -> LogoutHandler.Logout

Protected (JWTMiddleware)
GET    /api/v1/roles                       -> RoleHandler.List
POST   /api/v1/roles                       -> RoleHandler.Create
PUT    /api/v1/roles/:id                   -> RoleHandler.Update
DELETE /api/v1/roles/:id                   -> RoleHandler.Delete
```

## Middleware Chain

```
gin.Logger -> gin.Recovery -> route group middleware
protected routes: JWTMiddleware(pubKey, jwtRepo, userRepo)
```

## Service to Repository Mapping

```
RegisterService -> RegisterUserRepository + SMTPSender
VerifyService   -> VerifyRepository
LoginService    -> LoginRepository + RateLimitRepository + RefreshRepository
RefreshService  -> RefreshRepository + UserRepository
LogoutService   -> JWTRepository + RefreshRepository
RoleService     -> RoleRepository + RoleCacheRepository
```

## Key Files

- `backend/cmd/api/main.go`: config load, DB/Redis init, key parsing, DI wiring, server run.
- `backend/internal/delivery/http/router.go`: `/api/v1` route graph and middleware attachment.
- `backend/internal/delivery/http/auth/*.go`: auth + role HTTP handlers.
- `backend/internal/delivery/http/middleware/jwt.go`: bearer token verification and context hydration.
- `backend/internal/app/auth/*.go`: auth/session/role business logic.
- `backend/internal/domain/auth/entity.go`: `RegisterUser`, `User`, `Role`, DTOs.
- `backend/internal/domain/auth/repository.go`: repository contracts.
- `backend/internal/repository/postgres/*.go`: persistent repositories (user/register/verify/login/role).
- `backend/internal/repository/redis/*.go`: cache/session/blocklist/rate repositories.
- `backend/configs/config.go`: env-bound configuration structs.

## Runtime Boot Sequence

```
Load env -> load config -> open Postgres -> AutoMigrate(RegisterUser, User, Role)
-> init Redis client -> parse RSA keys -> construct repos/services/handlers -> start Gin server
```
