# DBC Gap Remediation — Design Spec

**Date:** 2026-05-14
**Source:** `docs/superpowers/specs/2026-05-14-dbc-gap-analysis.md`
**Scope:** All 15 gaps across 3 phases, 13 files

---

## 1. RBAC Guards (Phase 1 — HIGH)

### CreateDatabase

```
if !repo.IsAdmin(ctx, actorUserID) → ErrForbidden
```

Added at top of `CreateDatabase` in `database_service.go`, after parameter validation but before any business logic.

### UpdateDatabase

```
if !repo.IsAdmin && existing.CreatedByFK != actorUserID → ErrForbidden
```

Added after fetching existing record, before applying updates.

### DeleteDatabase

```
if !repo.IsAdmin && existing.CreatedByFK != actorUserID → ErrForbidden
```

Added after fetching existing record, before dataset count.

**Files:** `database_service.go`
**Tests:** Existing `TestDatabaseService_CreateDatabaseNonAdminReturnsForbidden` passes. New `TestUpdateDatabase_NonOwner_ReturnsForbidden`, `TestDeleteDatabase_NonOwner_ReturnsForbidden`.

---

## 2. Running-Queries Guard (Phase 1 — HIGH)

New domain error:

```go
var ErrDatabaseHasRunningQueries = errors.New("database has running queries")
```

New repository method:

```go
CountRunningQueriesByDatabaseID(ctx context.Context, databaseID uint) (int64, error)
```

In `DeleteDatabase`, after dataset count:

```go
queryCount, err := s.repo.CountRunningQueriesByDatabaseID(ctx, databaseID)
if queryCount > 0 {
    return domain.ErrDatabaseHasRunningQueries
}
```

Handler maps `ErrDatabaseHasRunningQueries` → 409.

**Files:** `errors.go`, `repository.go`, `database_repo.go`, `database_service.go`, `database_handler.go`
**Tests:** `TestDeleteDatabase_HasRunningQueries_Returns409`

---

## 3. Structured 409 Error (Phase 1 — HIGH)

New types:

```go
type DatasetRef struct {
    ID        uint   `json:"id"`
    TableName string `json:"table_name"`
}

type DatabaseInUseError struct {
    Datasets []DatasetRef `json:"datasets"`
}

func (e *DatabaseInUseError) Error() string {
    return fmt.Sprintf("database is in use by %d dataset(s)", len(e.Datasets))
}
```

New repository method:

```go
ListDatasetsByDatabaseID(ctx context.Context, databaseID uint) ([]DatasetRef, error)
```

In `DeleteDatabase`, replace bare `ErrDatabaseInUse` with `&DatabaseInUseError{Datasets: datasets}`.

Handler `handleError` matches `*DatabaseInUseError` and serializes:

```json
{"error": "database is in use by 3 dataset(s)", "datasets": [{"id": 1, "table_name": "orders"}]}
```

**Files:** `errors.go`, `repository.go`, `database_repo.go`, `database_service.go`, `database_handler.go`
**Tests:** `TestDeleteDatabase_HasDatasets_Returns409WithList`

---

## 4. Remove Plaintext URI Log (Phase 2 — MEDIUM)

Delete line 118 in `database_service.go`:

```go
log.Println("[database_prober] sqlalchemyURI:" + sqlalchemyURI)
```

**Files:** `database_service.go`
**Tests:** `TestProbe_DoesNotLogPlaintextURI`

---

## 5. Multi-Driver Support (Phase 2 — MEDIUM)

### resolveSQLDriver

Add entries:

```go
case "mysql":
    return "mysql", "mysql", nil
case "bigquery":
    return "bigquery", "bigquery", nil
case "snowflake":
    return "snowflake", "snowflake", nil
```

### Probe version query

Parameterize `SELECT version()` per driver:
- PostgreSQL: `SELECT version()`
- MySQL: `SELECT VERSION()`
- BigQuery: `SELECT CURRENT_VERSION()`
- Snowflake: `SELECT CURRENT_VERSION()`

### New drivers in go.mod

- `github.com/go-sql-driver/mysql`
- `github.com/snowflakedb/gosnowflake`
- BigQuery uses standard `database/sql` via existing BigQuery driver

**Files:** `database_service.go`, `go.mod`
**Tests:** `TestCreateDatabase_MySQLDriver`

---

## 6. Multi-Driver Schema Inspectors (Phase 2 — MEDIUM)

New structs in `schema_inspector.go`:

### mysqlSchemaInspector

Uses `information_schema` — same SQL as PostgreSQL but `?` placeholders instead of `$1`. `table_type` filter supports `BASE TABLE` and `VIEW`. Column type mapping: MySQL types (`VARCHAR`, `INT`, `TEXT`, etc.) → generic types.

### bigquerySchemaInspector

Uses `INFORMATION_SCHEMA` with BigQuery catalog/schema conventions. Dataset = schema in BigQuery terms.

### snowflakeSchemaInspector

Uses `INFORMATION_SCHEMA` with Snowflake-specific column types (`TEXT`, `NUMBER`, `FLOAT`, etc.).

### Inspector selection

Factory or switch on `Database.Backend` (driver name) to return the correct inspector.

**Files:** `schema_inspector.go`, `database_service.go`
**Tests:** `TestSchemaInspector_MySQL`

---

## 7. Schema Cache Invalidation (Phase 2 — MEDIUM)

### Interface extension

Add to `SchemaCacheRepository` in `repository.go`:

```go
InvalidateByPrefix(ctx context.Context, prefix string) error
```

### In-memory implementation

`inMemorySchemaCache.InvalidateByPrefix` iterates keys, deletes those with matching prefix.

### Redis implementation

New file `repository/redis/database_schema_cache_repo.go`:
- 10min TTL on all keys
- `InvalidateByPrefix` uses `SCAN` + `DEL` pattern
- Drop-in replacement via interface

### Wiring

In `UpdateDatabase`: after `poolManager.Close`, call `schemaCache.InvalidateByPrefix("schema:"+strconv.Itoa(databaseID)+":")`.

In `DeleteDatabase`: same call after pool close, before GORM delete.

**Files:** `repository.go`, `schema_cache_memory.go`, `database_schema_cache_repo.go` (new), `database_service.go`
**Tests:** `TestUpdateDatabase_FlushesSchemaCache`, `TestDeleteDatabase_FlushesSchemaCache`

---

## 8. Audit on Delete (Phase 3 — LOW)

### Interface extension

```go
type DatabaseAuditLogger interface {
    LogDatabaseCreated(ctx context.Context, databaseID uint)
    LogDatabaseDeleted(ctx context.Context, databaseID uint)
}
```

### Call site

In `DeleteDatabase`, after successful `repo.DeleteDatabase`:

```go
s.auditLogger.LogDatabaseDeleted(ctx, databaseID)
```

### Noop & fake

Update `noopDatabaseAuditLogger` and `fakeDatabaseAuditLogger` with the new method.

**Files:** `database_service.go`, `database_service_test.go`
**Tests:** `TestDeleteDatabase_AuditLogCalled`

---

## 9. Views Support (Phase 3 — LOW)

### Domain

Add to `ListDatabaseTablesRequest`:

```go
TableType string // "BASE TABLE", "VIEW", or "" for all
```

### Schema inspector

Parameterize SQL:

```sql
WHERE table_schema = ? AND table_type = ?
```

If `TableType` is empty, omit the `table_type` filter entirely.

### Handler

Parse `table_type` query parameter, pass through to service.

**Files:** `database.go`, `schema_inspector.go`, `database_service_introspection.go`, `database_handler.go`
**Tests:** coverage for VIEW and empty (all) filter

---

## 10. Handler Test Compile Fix (Phase 3 — LOW)

Add `GetPinned` stub to `handlerConnectionPool` in `database_handler_test.go`:

```go
func (handlerConnectionPool) GetPinned(_ context.Context, _ uint) (svcauth.SQLConnection, error) {
    return nil, nil
}
```

**Files:** `database_handler_test.go`

---

## 11. File Summary

| File | Change |
|------|--------|
| `domain/db/errors.go` | +3 errors, +2 types |
| `domain/db/repository.go` | +3 methods |
| `domain/db/database.go` | +1 field |
| `app/db/database_service.go` | RBAC guards, structured error, query guard, log removal, driver expansion, cache invalidation, audit |
| `app/db/schema_inspector.go` | +3 inspector impls, view filtering |
| `app/db/schema_cache_memory.go` | +InvalidateByPrefix |
| `app/db/database_service_introspection.go` | table_type passthrough |
| `app/db/database_service_test.go` | +8 new tests |
| `delivery/http/db/database_handler.go` | 2 new error mappings, query param |
| `delivery/http/db/database_handler_test.go` | +GetPinned stub, +2 new tests |
| `repository/postgres/database_repo.go` | +2 methods |
| `repository/redis/database_schema_cache_repo.go` | NEW: Redis cache |
| `go.mod` | +3 driver deps |

## 12. Success Criteria

1. Non-admin users receive 403 when creating databases
2. Non-owner users receive 403 when updating/deleting databases
3. Delete with running queries returns 409
4. Delete with datasets returns `{"error": "...", "datasets": [...]}`
5. MySQL/BigQuery/Snowflake drivers accepted in CreateDatabase + connection test
6. Schema introspection works for all four drivers
7. Schema cache invalidated on update/delete; Redis-backed
8. All existing tests pass; new tests cover RBAC, multi-driver, cache invalidation
9. No plaintext credentials in logs
10. Handler tests compile and pass
