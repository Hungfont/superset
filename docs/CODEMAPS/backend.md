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
GET    /api/v1/admin/databases             -> DatabaseHandler.List
GET    /api/v1/admin/databases/:id         -> DatabaseHandler.Get
GET    /api/v1/admin/databases/:id/schemas -> DatabaseHandler.ListSchemas
GET    /api/v1/admin/databases/:id/tables  -> DatabaseHandler.ListTables
GET    /api/v1/admin/databases/:id/columns -> DatabaseHandler.ListColumns
PUT    /api/v1/admin/databases/:id         -> DatabaseHandler.Update
DELETE /api/v1/admin/databases/:id         -> DatabaseHandler.Delete
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
admin routes: handler-level authorization and selective RequirePermission middleware
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
DatabaseService   -> DatabaseRepository (create/list/get/update/delete + test by config + test by id)
				 -> ConnectionPoolManager (lazy pool init, reuse, health monitor)
				 -> SchemaInspector (schemas/tables/columns via INFORMATION_SCHEMA)
				 -> SchemaCacheRepository (Redis TTL 10m for introspection payloads)
				 -> force_refresh limiter (5/min; shared limiter contract)
```

## Key Files

- `backend/cmd/api/main.go`: config load, DB/Redis init, key parsing, DI wiring, server run.
- `backend/internal/delivery/http/router.go`: `/api/v1` route graph and middleware attachment.
- `backend/internal/delivery/http/auth/*.go`: auth + user + role + permission HTTP handlers.
- `backend/internal/delivery/http/db/database_handler.go`: database create/list/get/update/delete + test-connection + schema introspection HTTP handlers.
- `backend/internal/delivery/http/middleware/jwt.go`: bearer token verification and context hydration.
- `backend/internal/app/auth/*.go`: auth/session/user/role/permission business logic.
- `backend/internal/app/db/database_service.go`: database lifecycle service, dependency wiring, and shared guard logic.
- `backend/internal/app/db/database_service_introspection.go`: DBC-007 introspection methods (schemas/tables/columns), cache read/write, force-refresh limiter, 429/502/504 error mapping.
- `backend/internal/app/db/schema_inspector.go`: PostgreSQL INFORMATION_SCHEMA inspector implementation and is_dttm mapping.
- `backend/internal/app/db/schema_cache_memory.go`: in-memory cache fallback implementation for introspection payloads.
- `backend/internal/domain/auth/entity.go`: `RegisterUser`, `User`, `Role`, `Permission`, `ViewMenu`, `PermissionView`, DTOs.
- `backend/internal/domain/db/database.go`: `Database` entity plus introspection DTOs (`DatabaseTable`, `DatabaseColumn`, list requests/responses).
- `backend/internal/domain/db/repository.go`: database repository contracts plus schema cache contract.
- `backend/internal/repository/postgres/*.go`: persistent repositories (user/register/verify/login/user-role/role/permission/database).
- `backend/internal/repository/redis/*.go`: cache/session/blocklist/rate repositories, including schema cache repository.
- `backend/configs/config.go`: env-bound configuration structs.

## Runtime Boot Sequence

```
Load env -> load config -> open Postgres -> AutoMigrate(RegisterUser, User, Role, Permission, ViewMenu, PermissionView, Database)
-> init Redis client -> parse RSA keys -> construct repos/services/handlers (including schema cache repo wiring)
-> seed default permission-view pairs -> start Gin server
```
