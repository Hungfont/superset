<!-- Generated: 2026-05-13 | Files scanned: ~250 | Token estimate: ~950 -->

# Backend Codemap

Entry point: `backend/cmd/api/main.go`
Module: `superset/auth-service`
Runtime: Go 1.25 + Gin + GORM + Redis + gorilla/websocket

## Route Map

```
Public
POST /api/v1/auth/register                 -> RegisterHandler.Register
GET  /api/v1/auth/verify                   -> VerifyHandler.Verify
POST /api/v1/auth/login                    -> LoginHandler.Login
POST /api/v1/auth/refresh                  -> RefreshHandler.Refresh
POST /api/v1/auth/logout                   -> LogoutHandler.Logout

WebSocket (QE-005 — JWT in query string)
GET  /ws/query/:query_id                   -> WSHandler.Handle

Protected (JWT required)
GET    /api/v1/query/history               -> QueryHandler.ListHistory (QE-007)
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
DELETE /api/v1/query/:id                  -> QueryHandler.Cancel (QE-006)
GET    /api/v1/query/:id/result           -> QueryHandler.GetResult (QE-007: ownership check)
GET    /api/v1/query/:id/result/download  -> QueryHandler.GetResultByToken (QE-005: JWT in query string)

Protected Admin (JWT + RequirePermission where applicable)
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

Query History (QE-007)
DELETE /api/v1/query/history               -> QueryHandler.DeleteHistory (admin only)

RLS Admin (admin role required)
GET    /api/v1/admin/rls                  -> RLSHandler.List
GET    /api/v1/admin/rls/:id              -> RLSHandler.Get
POST   /api/v1/admin/rls                  -> RLSHandler.Create
PUT    /api/v1/admin/rls/:id              -> RLSHandler.Update
DELETE /api/v1/admin/rls/:id              -> RLSHandler.Delete
```

## Middleware Chain

```
gin.Logger -> gin.Recovery
  -> /ws/*       : middleware.ValidateJWTFromQuery (token in query string)
  -> /api/v1/query/:id/result/download: middleware.ValidateJWTFromQuery (token in query string)
  -> /api/v1/*    : JWT middleware (bearer token, protected routes)
    -> /api/v1/admin/permissions: JWT + RequirePermission("can_read", "Permission")
    -> /api/v1/admin/view-menus: JWT + RequirePermission("can_read", "ViewMenu")
    -> /api/v1/admin/permission-views: JWT + RequirePermission("can_read", "PermissionView")
    -> /api/v1/admin/rls/*: JWT + AuthorizeAdminRole
    -> /api/v1/query/history (DELETE): JWT + AuthorizeAdminRole
```

## Service to Repository Mapping

```
RegisterService       -> RegisterUserRepository + SMTPSender
VerifyService         -> VerifyRepository
LoginService          -> LoginRepository + RateLimitRepository + RefreshRepository
RefreshService        -> RefreshRepository + UserRepository
LogoutService         -> JWTRepository + RefreshRepository
UserService           -> UserAdminRepository + RoleCacheRepository
UserRoleService       -> UserRoleRepository + RoleCacheRepository
RoleService           -> RoleRepository + RoleCacheRepository
PermissionService     -> PermissionRepository + RoleCacheRepository
DatabaseService       -> DatabaseRepository + ConnectionPoolManager + SchemaInspector + SchemaCacheRepository
DatasetService        -> DatasetRepository + DatasetAsyncQueue (Redis)
QueryExecutor         -> DatabaseRepository + RLSFilterRepository + RLSInjector + QueryCacheRepository + ConnectionPoolManager
AsyncQueryExecutor    -> QueryCacheRepository + QueryQueueRepository + QueryStatusRepository + DatabaseRepository + QueryExecutor
RLSService            -> RLSFilterRepository
WSHandler (QE-005)    -> JWTRepository + UserRepository + RoleNameProvider + query.Repository + Redis subscribe
QueryHandler (QE-007) -> query.Repository + RoleRepository
QueryWorker           -> QueryExecutor + AsyncQueryExecutor + Redis BRPOP
ColumnSyncWorker      -> DatasetRepoWrapper + DatabaseService + ConnectionPoolManager
```

## Key Files

- `backend/cmd/api/main.go`: config load, DB/Redis init, key parsing, DI wiring (including WS handler and query worker), pg_trgm GIN index creation (QE-007), server run.
- `backend/internal/delivery/http/router.go`: `/api/v1` route graph, `/ws/query/:query_id` endpoint (QE-005), middleware attachment.
- `backend/internal/delivery/http/auth/*.go`: auth + user + role + permission HTTP handlers.
- `backend/internal/delivery/http/dataset/handler.go`: dataset CRUD + metrics + column management + cache operations.
- `backend/internal/delivery/http/query/handler.go`: SQL query execution (sync + async submit/status/result/cancel), query history list/delete (QE-007), result download by token (QE-005).
- `backend/internal/delivery/http/query/websocket.go`: WebSocket handler (QE-005); JWT auth via query string, Redis pub/sub for status events, heartbeat ping/pong, ownership check.
- `backend/internal/delivery/http/rls/handler.go`: Row-Level Security filter CRUD.
- `backend/internal/delivery/http/db/database_handler.go`: database create/list/get/update/delete + test-connection + schema introspection HTTP handlers.
- `backend/internal/delivery/http/middleware/jwt.go`: bearer token and query-string JWT validation + context hydration.
- `backend/internal/delivery/http/middleware/authorize.go`: admin role authorization middleware.
- `backend/internal/app/auth/*.go`: auth/session/user/role/permission business logic.
- `backend/internal/app/auth/rls_service.go`: RLS filter management service.
- `backend/internal/app/dataset/service.go`: dataset lifecycle with async queue management.
- `backend/internal/app/query/executor.go`: SQL query execution with RLS filter injection and caching.
- `backend/internal/app/query/async_executor.go`: async query execution (QE-004), Redis queue routing (critical/default/low), pub/sub status events, cancellation (QE-006).
- `backend/internal/app/db/database_service.go`: database lifecycle service, dependency wiring, and shared guard logic.
- `backend/internal/app/db/database_service_introspection.go`: DBC-007 introspection methods (schemas/tables/columns), cache read/write, force-refresh limiter.
- `backend/internal/app/db/schema_inspector.go`: PostgreSQL INFORMATION_SCHEMA inspector implementation.
- `backend/internal/app/db/schema_cache_memory.go`: in-memory schema cache (TTL 10m) for introspection results.
- `backend/internal/domain/auth/entity.go`: `RegisterUser`, `User`, `Role`, `Permission`, `ViewMenu`, `PermissionView`, DTOs.
- `backend/internal/domain/db/database.go`: `Database` entity plus introspection DTOs.
- `backend/internal/domain/dataset/dataset.go`: `Dataset`, `DatasetColumn`, `DatasetMetric` entities.
- `backend/internal/domain/query/entity.go`: `Query` (full Superset query model), `ExecuteRequest`, `ExecuteResponse`, `ExecuteMeta`, `QueryTask`, `AsyncSubmitRequest/Response`, `QueryStatusResponse`, `ListFilter`, `HistoryResponseItem`, `HistoryResponse`, `DeleteHistoryResponse`.
- `backend/internal/domain/query/saved_query.go`: `SavedQuery` entity (saved_query table).
- `backend/internal/domain/query/tab_state.go`: `TabState` entity (tab_state table) -- SQL Lab tab persistence.
- `backend/internal/domain/query/table_schema.go`: `TableSchema` entity (table_schema table) -- expanded schema state.
- `backend/internal/domain/query/repository.go`: query storage contract (Create, GetByID, GetByClientID, Update, List, UpdateStatusConditional, ListHistory, DeleteOlderThan).
- `backend/internal/domain/auth/repository.go`: repository contracts for users, roles, permissions.
- `backend/internal/domain/dataset/repository.go`: dataset repository contracts.
- `backend/internal/repository/postgres/*.go`: persistent repositories (user/register/verify/login/user-role/role/permission/database/dataset/rls_filter/query).
- `backend/internal/repository/redis/*.go`: cache/session/blocklist/rate repositories, dataset sync/async queues, schema cache, role cache.
- `backend/internal/worker/query_worker.go`: async query worker (QE-004); FIFO queue consumption with BRPOP; delegates execution to AsyncQueryExecutor; handles cancellation via Redis flag.
- `backend/internal/worker/column_sync.go`: background column synchronization worker.
- `backend/configs/config.go`: env-bound configuration structs.

## Runtime Boot Sequence

```
Load env -> load config -> open Postgres -> AutoMigrate(RegisterUser, User, Role, Permission, ViewMenu, PermissionView, Database, Dataset, Query, SavedQuery, TabState, TableSchema)
  -> CREATE EXTENSION pg_trgm + GIN index on query.sql (QE-007)
  -> init Redis client -> parse RSA keys -> construct repos/services/handlers
  -> seed default permission-view pairs
  -> start ColumnSyncWorker -> start QueryWorker
  -> start Gin server
```
