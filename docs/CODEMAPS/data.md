<!-- Generated: 2026-05-13 | Files scanned: ~250 | Token estimate: ~750 -->

# Data Codemap

## Postgres Entities (AutoMigrate)

Source: `backend/internal/domain/auth/entity.go`, `backend/internal/domain/dataset/dataset.go`, `backend/internal/domain/query/entity.go`, bootstrapped in `backend/cmd/api/main.go`

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

- Purpose: RBAC action catalog (e.g., `can_read`, `can_write`).
- Key columns: `id`, `name` (unique).

### `ab_view_menu`

- Purpose: RBAC resource/menu catalog (e.g., `Dashboard`, `Chart`).
- Key columns: `id`, `name` (unique).

### `ab_permission_view`

- Purpose: permission-to-view mapping matrix used by role assignments.
- Key columns: `id`, `permission_id`, `view_menu_id`, unique composite (`permission_id`, `view_menu_id`).

### `ab_permission_view_role`

- Purpose: role-to-permission_view join table.
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

- Purpose: configured database connections.
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

### `query`

- Purpose: query execution records (Superset query model).
- Key columns: `id` (varchar UUID), `client_id` (varchar UUID), `database_id`, `user_id`, `status`, `tab_name`, `sql_editor_id`, `schema`, `catalog`, `sql` (text), `select_sql` (text), `executed_sql` (text), `limit`, `limiting_factor`, `select_as_cta`, `select_as_cta_used`, `progress`, `rows`, `error_message` (text), `results_key`, `start_time`, `start_running_time`, `end_time`, `end_result_backend_time`, `tmp_table_name`, `tracking_url`, `tmp_schema_name`, `cached_data` (text), `is_saved`, `extra_json` (text), `tenant_id`, `changed_on`, `created_at`, `updated_at`.
- QE-007: GIN index on `sql` column via `pg_trgm` for ILIKE search performance.

### `saved_query`

- Purpose: persisted/saved user queries.
- Key columns: `id`, `db_id`, `user_id`, `label`, `schema`, `catalog`, `sql` (text), `description` (text), `sql_tables` (text), `extra_json` (text), `published`, `created_on`, `changed_on`, `created_by_fk`, `changed_by_fk`, `tags` (text).

### `tab_state`

- Purpose: SQL Lab tab persistence (editor state).
- Key columns: `id`, `user_id`, `db_id`, `schema`, `catalog`, `label`, `active`, `sql` (text), `query_limit`, `latest_query_id`, `hide_left_bar`, `saved_query_id`, `created_on`, `changed_on`, `created_by_fk`, `changed_by_fk`, `extra_json` (text).

### `table_schema`

- Purpose: expanded schema/table state in SQL Lab.
- Key columns: `id`, `tab_state_id`, `db_id`, `schema`, `catalog`, `table`, `description` (text), `expanded`, `created_on`, `changed_on`.

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
- `queue:query:<priority>`: async query queue (QE-004), priority levels: critical/default/low based on user role.
- `query:status:<queryID>`: async query status (pending/running/completed/failed/cancelled), Redis pub/sub channel (QE-004/005).
- `query:cancel:<queryID>`: async query cancellation request flag (QE-006).
- `schema_cache:<dbID>:<cacheKey>`: in-memory schema cache with TTL 10m.

## Data Flow Summary

```
register -> ab_register_user
verify   -> move/activate into ab_user
login    -> read ab_user + write refresh/rate keys
logout   -> write jwt:blacklist + delete refresh session
roles    -> read/write ab_role + invalidate Redis rbac:* namespace
permissions/view-menus -> read/write ab_permission + ab_view_menu + invalidate Redis rbac:* namespace
permission-views -> read/write ab_permission_view, join ab_permission + ab_view_menu for display names, invalidate rbac:*
database schema introspection -> read dbs, open pool, query external DB INFORMATION_SCHEMA, cache under schema:* keys, bypass cache on force_refresh=true (rate limited)
datasets -> read/write ab_dataset + push to sync/async Redis queues for column/metric sync
queries -> execute SQL against dataset's database, inject RLS filters, cache results (QE-003: qcache:* prefix, 10MB max)
async queries -> submit by priority (QE-004), worker BRPOP processes, pub/sub status events (QE-005), WebSocket streaming, cancellation via Redis flag (QE-006)
query history -> read query JOIN dbs with GIN-indexed ILIKE search, delete older than N days (QE-007)
saved queries -> read/write saved_query, linked to user and database
tab state -> read/write tab_state (SQL Lab editor state) + table_schema (expanded table metadata)
rls filters -> read/write ab_rls_filter, referenced during query execution pipeline
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
- `Query`, `ExecuteRequest`, `ExecuteResponse`, `ExecuteMeta`, `QueryTask`
- `AsyncSubmitRequest`, `AsyncSubmitResponse`, `QueryStatusResponse`
- `ListFilter`, `HistoryResponseItem`, `HistoryResponse`, `DeleteHistoryResponse`
- `SavedQuery`, `TabState`, `TableSchema`

## Extended Docs

- Service-level DB docs: `docs/db/`
- Backend repository map: `docs/CODEMAPS/backend.md`
