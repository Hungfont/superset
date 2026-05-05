<!-- Generated: 2026-05-05 | Files scanned: 180 | Token estimate: ~620 -->

# Data Codemap

## Postgres Entities (AutoMigrate)

Source: `backend/internal/domain/auth/entity.go` + `backend/internal/domain/dataset/dataset.go`, bootstrapped in `backend/cmd/api/main.go`

### `ab_register_user`

- Purpose: pending registrations before email verification.
- Key columns: `id`, `first_name`, `last_name`, `username` (unique), `email` (unique), `password` (bcrypt), `registration_hash` (unique), `created_at`.

### `ab_user`

- Purpose: activated user accounts.
- Key columns: `id`, `first_name`, `last_name`, `username` (unique), `email` (unique), `password`, `active`, `login_count`, `last_login`, `created_on`, `changed_on`.

### `ab_role`

- Purpose: RBAC role catalog.
- Key columns: `id`, `name` (unique).

### `ab_permission`

- Purpose: RBAC action catalog (for example: `can_read`, `can_write`).
- Key columns: `id`, `name` (unique).

### `ab_view_menu`

- Purpose: RBAC resource/menu catalog (for example: `Dashboard`, `Chart`).
- Key columns: `id`, `name` (unique).

### `ab_permission_view`

- Purpose: permission-to-view mapping matrix used by role assignments.
- Key columns: `id`, `permission_id`, `view_menu_id`, unique composite (`permission_id`, `view_menu_id`).
- API list enrichment: GET `/api/v1/admin/permission-views` joins `ab_permission` and `ab_view_menu` to return UI display fields `permission_name` and `view_menu_name`.

### `ab_permission_view_role`

- Purpose: role-to-permission_view join table used for assignment checks.
- Key columns: `role_id`, `permission_view_id`.

### `row_level_security_filters`

- Purpose: Row-Level Security filter rules.
- Key columns: `id`, `name` (unique), `filter_type` (Regular/Base), `clause`, `group_key`, `description`, `created_by_fk`, `changed_by_fk`, `created_on`, `changed_on`.

### `rls_filter_roles`

- Purpose: RLS filter to role mapping.
- Key columns: `rls_id`, `role_id`.

### `rls_filter_tables`

- Purpose: RLS filter to dataset/datasource mapping.
- Key columns: `rls_id`, `datasource_id`, `datasource_type`, `table_name`, `database_name`.

### `dbs`

- Purpose: configured database connections (SQLAlchemy URI stored).
- Key columns: `id`, `database_name` (unique), `sqlalchemy_uri`, `password`, `allow_dml`, `expose_in_sqllab`, `allow_run_async`, `allow_file_upload`, `created_by_fk`, `created_on`, `changed_on`.

### `tables` (ab_dataset)

- Purpose: virtual or physical datasets backed by databases.
- Key columns: `id`, `table_name`, `schema`, `database_id`, `sql`, `perm`, `description`, `main_dttm_col`, `cache_timeout`, `filter_select_enabled`, `normalize_columns`, `is_featured`, `created_by_fk`, `changed_by_fk`, `created_on`, `changed_on`.

### `table_columns` (ab_dataset_column)

- Purpose: column metadata for datasets.
- Key columns: `id`, `table_id`, `column_name`, `type`, `is_dttm`, `is_active`, `verbose_name`, `description`, `filterable`, `groupby`, `python_date_format`, `expression`, `column_type`, `exported`.

### `sql_metrics` (ab_dataset_metric)

- Purpose: metrics defined on datasets.
- Key columns: `id`, `table_id`, `metric_name`, `verbose_name`, `metric_type`, `expression`, `description`, `d3format`, `warning_text`, `is_restricted`, `extra`, `certified_by`, `certification_details`, `created_on`, `changed_on`, `created_by_fk`, `changed_by_fk`.

## Redis Key Spaces

- `jwt:blacklist:<jti>`: revoked access-token JTIs.
- `refresh:<token>`: refresh token -> user ID mapping.
- `user_tokens:<userID>`: set of active refresh tokens for logout-all operations.
- `user:<userID>`: cached user context for JWT middleware hydration.
- `rate:login:<ip>`: short-window login attempt throttling counter.
- `failed_login:<username>`: failed login count for lockout policy.
- `lockout:<username>`: active lockout marker with TTL.
- `rbac:*`: RBAC cache namespace invalidated on role changes.
- `schema:<dbID>:schemas`: cached schema list for one configured database (TTL 10 minutes).
- `schema:<dbID>:<schema>:tables:<page>:<pageSize>`: cached paginated table list (TTL 10 minutes).
- `schema:<dbID>:<schema>:<table>:columns`: cached column metadata list (TTL 10 minutes).
- `dataset_sync:<datasetID>`: sync queue for dataset refresh (Redis list).
- `dataset_async:<datasetID>`: async queue for background dataset operations (Redis list).
- `query:<queryID>`: cached query results (TTL configurable).
- `qcache:<cacheKey>`: query result cache (QE-003), 10MB max size, RLS hash + normalizeSQL as key, TTL from dataset config.
- `queue:query:<priority>`: async query queue (QE-004), priority: critical/default/low based on user role.
- `query:status:<queryID>`: async query status (pending/running/completed/failed/cancelled), pub/sub events.
- `query:cancel:<queryID>`: async query cancellation request flag.

## Data Flow Summary

```
register -> ab_register_user
verify   -> move/activate into ab_user
login    -> read ab_user + write refresh/rate keys
logout   -> write jwt:blacklist + delete refresh session
roles    -> read/write ab_role + invalidate Redis rbac:* namespace
permissions/view-menus -> read/write ab_permission + ab_view_menu + invalidate Redis rbac:* namespace
permission-views -> read/write ab_permission_view, join ab_permission + ab_view_menu for display names, check ab_permission_view_role usage, invalidate Redis rbac:* namespace
database schema introspection -> read dbs (connection config), open/reuse pool, query external DB INFORMATION_SCHEMA, cache payload under schema:* keys, bypass cache when force_refresh=true (rate limited)
datasets -> read/write ab_dataset + push to sync/async Redis queues for column/metric sync
queries -> execute SQL against dataset's database, inject RLS filters, cache results (QE-003: qcache: prefix, 10MB max, force refresh bypasses cache)
async queries -> submit to queue by priority (QE-004: Admin->critical, Alpha->default, Gamma->low), worker polls and processes, status via pub/sub, result retrieval via GET, cancellation via DELETE
rls filters -> read/write ab_rls_filter, used in query execution pipeline
```

## Domain Types Used in API

- `RegisterRequest`, `LoginRequest`, `RefreshRequest`, `LogoutRequest`
- `UserContext` (middleware-injected actor)
- `Role`, `UpsertRoleRequest`, `RoleListItem`
- `Permission`, `ViewMenu`, `PermissionView`
- `UpsertPermissionRequest`, `UpsertViewMenuRequest`, `CreatePermissionViewRequest`
- `Database`, `DatabaseDetail`, `DatabaseListItem`, `CreateDatabaseRequest`, `UpdateDatabaseRequest`
- `ListDatabaseTablesRequest`, `ListDatabaseColumnsRequest`
- `DatabaseTable`, `DatabaseTableListResponse`, `DatabaseColumn`
- `TestConnectionResult`, `TestDatabaseConnectionRequest`
- `Dataset`, `DatasetDetail`, `DatasetWithCounts`, `CreatePhysicalDatasetRequest`, `CreateVirtualDatasetRequest`, `UpdateDatasetMetadataRequest`
- `Column`, `UpdateColumnRequest`, `BulkUpdateColumnRequest`
- `SqlMetric`, `CreateMetricRequest`, `UpdateMetricRequest`, `BulkUpdateMetricsRequest`
- `RLSFilter`, `RLSFilterResponse`, `CreateRLSFilterRequest`, `UpdateRLSFilterRequest`, `RLSFilterListResult`

## Extended Docs

- Service-level DB docs: `docs/db/`
- Backend repository map: `docs/CODEMAPS/backend.md`
