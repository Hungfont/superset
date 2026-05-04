<!-- Generated: 2026-05-04 | Files scanned: 120 | Token estimate: ~850 -->

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

Protected (JWT required)
GET    /api/v1/datasets                     -> DatasetHandler.ListDatasets
GET    /api/v1/datasets/:id               -> DatasetHandler.GetDataset
POST   /api/v1/datasets                   -> DatasetHandler.CreatePhysicalDataset
POST   /api/v1/datasets/virtual           -> DatasetHandler.CreateVirtualDataset
PUT    /api/v1/datasets/:id               -> DatasetHandler.UpdateDataset
PUT    /api/v1/datasets/:id/columns/:col_id -> DatasetHandler.UpdateColumn
PUT    /api/v1/datasets/:id/columns        -> DatasetHandler.BulkUpdateColumns
GET    /api/v1/datasets/:id/metrics         -> DatasetHandler.GetMetrics
POST   /api/v1/datasets/:id/metrics        -> DatasetHandler.CreateMetric
PUT    /api/v1/datasets/:id/metrics        -> DatasetHandler.BulkUpdateMetrics
PUT    /api/v1/datasets/:id/metrics/:metric_id -> DatasetHandler.UpdateMetric
DELETE /api/v1/datasets/:id/metrics/:metric_id -> DatasetHandler.DeleteMetric
DELETE /api/v1/datasets/:id               -> DatasetHandler.DeleteDataset
POST   /api/v1/datasets/:id/refresh       -> DatasetHandler.RefreshDataset
POST   /api/v1/datasets/:id/cache/flush   -> DatasetHandler.FlushCache
POST   /api/v1/query/execute              -> QueryHandler.Execute
POST   /api/v1/query/submit               -> QueryHandler.SubmitAsync (queue: Admin->critical, Alpha->default, Gamma->low)
GET    /api/v1/query/:id/status           -> QueryHandler.GetStatus
DELETE /api/v1/query/:id                  -> QueryHandler.CancelQuery
GET    /api/v1/query/:id/result           -> QueryHandler.GetResult

Protected Admin (JWT + authorization)
POST   /api/v1/admin/databases             -> DatabaseHandler.Create
GET    /api/v1/admin/databases             -> DatabaseHandler.List
GET    /api/v1/admin/databases/:id         -> DatabaseHandler.Get
GET    /api/v1/admin/databases/:id/schemas -> DatabaseHandler.ListSchemas
GET    /api/v1/admin/databases/:id/tables -> DatabaseHandler.ListTables
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

RLS Admin (admin role required)
GET    /api/v1/admin/rls                  -> RLSHandler.List
GET    /api/v1/admin/rls/:id              -> RLSHandler.Get
POST   /api/v1/admin/rls                  -> RLSHandler.Create
PUT    /api/v1/admin/rls/:id              -> RLSHandler.Update
DELETE /api/v1/admin/rls/:id              -> RLSHandler.Delete
```

## Middleware Chain

```
gin.Logger -> gin.Recovery -> JWT middleware (protected routes)
Dataset/Query routes: JWT only
Admin routes: JWT + RequirePermission middleware
RLS routes: JWT + AuthorizeAdminRole middleware
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
DatabaseService   -> DatabaseRepository + ConnectionPoolManager + SchemaInspector + SchemaCacheRepository
DatasetService    -> DatasetRepository + DatasetSyncQueue (Redis) + DatasetAsyncQueue (Redis)
QueryExecutor     -> DatabaseRepository + RLSFilterRepository + RLSInjector + QueryCacheRepository
AsyncQueryExecutor -> QueryCacheRepository + QueryQueueRepository + QueryStatusRepository + DatabaseRepository
RLSService        -> RLSFilterRepository
```

## Key Files

- `backend/cmd/api/main.go`: config load, DB/Redis init, key parsing, DI wiring, server run.
- `backend/internal/delivery/http/router.go`: `/api/v1` route graph and middleware attachment.
- `backend/internal/delivery/http/auth/*.go`: auth + user + role + permission HTTP handlers.
- `backend/internal/delivery/http/dataset/handler.go`: dataset CRUD + metrics + column management + cache operations.
- `backend/internal/delivery/http/query/handler.go`: SQL query execution (sync) + async query submission/status/result/cancel (QE-004).
- `backend/internal/delivery/http/rls/handler.go`: Row-Level Security filter CRUD.
- `backend/internal/delivery/http/db/database_handler.go`: database create/list/get/update/delete + test-connection + schema introspection HTTP handlers.
- `backend/internal/delivery/http/middleware/jwt.go`: bearer token verification and context hydration.
- `backend/internal/app/auth/*.go`: auth/session/user/role/permission business logic.
- `backend/internal/app/auth/rls_service.go`: RLS filter management service.
- `backend/internal/app/dataset/service.go`: dataset lifecycle with sync/async queue management.
- `backend/internal/app/query/executor.go`: SQL query execution with RLS filter injection and caching.
- `backend/internal/app/query/cache.go`: query result caching (QE-003), cache key generation with RLS hash + normalizeSQL, TTL from dataset config.
- `backend/internal/app/query/cache_test.go`: query result caching.
- `backend/internal/app/query/rls_injector_test.go`: RLS injection logic.
- `backend/internal/app/query/async_executor.go`: async query execution (QE-004), Redis queue routing (critical/default/low), pub/sub status events, worker polling.
- `backend/internal/app/db/database_service.go`: database lifecycle service, dependency wiring, and shared guard logic.
- `backend/internal/app/db/database_service_introspection.go`: DBC-007 introspection methods (schemas/tables/columns), cache read/write, force-refresh limiter, 429/502/504 error mapping.
- `backend/internal/app/db/schema_inspector.go`: PostgreSQL INFORMATION_SCHEMA inspector implementation and is_dttm mapping.
- `backend/internal/app/rls/service.go`: RLS filter management service.
- `backend/internal/domain/auth/entity.go`: `RegisterUser`, `User`, `Role`, `Permission`, `ViewMenu`, `PermissionView`, DTOs.
- `backend/internal/domain/db/database.go`: `Database` entity plus introspection DTOs.
- `backend/internal/domain/dataset/dataset.go`: `Dataset`, `DatasetColumn`, `DatasetMetric` entities.
- `backend/internal/domain/auth/repository.go`: repository contracts for users, roles, permissions.
- `backend/internal/domain/dataset/repository.go`: dataset repository contracts.
- `backend/internal/repository/postgres/*.go`: persistent repositories (user/register/verify/login/user-role/role/permission/database/dataset).
- `backend/internal/repository/redis/*.go`: cache/session/blocklist/rate repositories, dataset sync/async queues.
- `backend/internal/worker/column_sync.go`: background column synchronization worker.
- `backend/internal/worker/repo_wrapper.go`: repository wrapper for workers.
- `backend/configs/config.go`: env-bound configuration structs.

## Runtime Boot Sequence

```
Load env -> load config -> open Postgres -> AutoMigrate(RegisterUser, User, Role, Permission, ViewMenu, PermissionView, Database, Dataset, RLSFilter)
-> init Redis client -> parse RSA keys -> construct repos/services/handlers
-> seed default permission-view pairs -> start Gin server
```
