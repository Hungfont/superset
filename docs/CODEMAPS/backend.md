<!-- Generated: 2026-04-14 | Files scanned: 120 | Token estimate: ~680 -->

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
POST   /api/v1/admin/databases             -> DatabaseHandler.Create
POST   /api/v1/admin/databases/test        -> DatabaseHandler.TestConnection
POST   /api/v1/admin/databases/:id/test    -> DatabaseHandler.TestConnectionByID

GET    /api/v1/admin/users                 -> UserHandler.List
GET    /api/v1/admin/users/:id             -> UserHandler.Get
POST   /api/v1/admin/users                 -> UserHandler.Create
PUT    /api/v1/admin/users/:id             -> UserHandler.Update
DELETE /api/v1/admin/users/:id             -> UserHandler.Delete

GET    /api/v1/admin/users/:id/roles       -> UserRoleHandler.List
PUT    /api/v1/admin/users/:id/roles       -> UserRoleHandler.Set

GET    /api/v1/admin/roles                 -> RoleHandler.List
POST   /api/v1/admin/roles                 -> RoleHandler.Create
PUT    /api/v1/admin/roles/:id             -> RoleHandler.Update
DELETE /api/v1/admin/roles/:id             -> RoleHandler.Delete
GET    /api/v1/admin/roles/:id/permissions -> RoleHandler.ListPermissions
PUT    /api/v1/admin/roles/:id/permissions -> RoleHandler.SetPermissions
POST   /api/v1/admin/roles/:id/permissions/add -> RoleHandler.AddPermissions
DELETE /api/v1/admin/roles/:id/permissions/:pv_id -> RoleHandler.RemovePermission

GET    /api/v1/admin/permissions           -> PermissionHandler.ListPermissions
POST   /api/v1/admin/permissions           -> PermissionHandler.CreatePermission
GET    /api/v1/admin/view-menus            -> PermissionHandler.ListViewMenus
POST   /api/v1/admin/view-menus            -> PermissionHandler.CreateViewMenu
GET    /api/v1/admin/permission-views      -> PermissionHandler.ListPermissionViews
POST   /api/v1/admin/permission-views      -> PermissionHandler.CreatePermissionView
DELETE /api/v1/admin/permission-views/:id  -> PermissionHandler.DeletePermissionView
```

## Middleware Chain

```
gin.Logger -> gin.Recovery -> route group middleware
protected routes: JWTMiddleware(pubKey, jwtRepo, userRepo)
admin routes: AuthorizeAdminRole(roleRepo)
```

## Service to Repository Mapping

```
RegisterService -> RegisterUserRepository + SMTPSender
VerifyService   -> VerifyRepository
LoginService    -> LoginRepository + RateLimitRepository + RefreshRepository
RefreshService  -> RefreshRepository + UserRepository
LogoutService   -> JWTRepository + RefreshRepository
UserService     -> UserAdminRepository + RoleCacheRepository
UserRoleService -> UserRoleRepository + RoleCacheRepository
RoleService     -> RoleRepository + RoleCacheRepository
PermissionService -> PermissionRepository + RoleCacheRepository
DatabaseService   -> DatabaseRepository (create + test by config + test by id, timeout/rate limit/error sanitization)
```

## Key Files

- `backend/cmd/api/main.go`: config load, DB/Redis init, key parsing, DI wiring, server run.
- `backend/internal/delivery/http/router.go`: `/api/v1` route graph and middleware attachment.
- `backend/internal/delivery/http/auth/*.go`: auth + user + role + permission HTTP handlers.
- `backend/internal/delivery/http/db/database_handler.go`: database create + test-connection HTTP handlers.
- `backend/internal/delivery/http/middleware/jwt.go`: bearer token verification and context hydration.
- `backend/internal/app/auth/*.go`: auth/session/user/role/permission business logic.
- `backend/internal/app/db/database_service.go`: database create/test business logic with 5s timeout probe flow.
- `backend/internal/domain/auth/entity.go`: `RegisterUser`, `User`, `Role`, `Permission`, `ViewMenu`, `PermissionView`, DTOs.
- `backend/internal/domain/db/database.go`: `Database` entity and request/response DTOs for DB connection APIs.
- `backend/internal/domain/db/repository.go`: database repository contracts.
- `backend/internal/repository/postgres/*.go`: persistent repositories (user/register/verify/login/user-role/role/permission/database).
- `backend/internal/repository/redis/*.go`: cache/session/blocklist/rate repositories.
- `backend/configs/config.go`: env-bound configuration structs.

## Runtime Boot Sequence

```
Load env -> load config -> open Postgres -> AutoMigrate(RegisterUser, User, Role, Permission, ViewMenu, PermissionView, Database)
-> init Redis client -> parse RSA keys -> construct repos/services/handlers
-> seed default permission-view pairs -> start Gin server
```
