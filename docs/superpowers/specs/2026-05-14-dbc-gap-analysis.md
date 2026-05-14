# DBC-001 to DBC-009 Gap Analysis — Database Connection Service

**Date:** 2026-05-14
**Scope:** Backend Go implementation verification against `DATABASE_CONNECTION_SERVICE_02_PHASE1.md` requirements DBC-001 through DBC-009
**Status:** Gaps identified — design spec not yet created

---

## 1. Methodology

Each requirement was verified by reading the full implementation across all four layers: domain (`domain/db`), app service (`app/db`), postgres repository (`repository/postgres`), and HTTP handler (`delivery/http/db`). Every acceptance criterion and request flow step in the spec was checked against the code.

---

## 2. Gap Summary

| # | Severity | Req | Issue |
|---|----------|-----|-------|
| 1 | HIGH | DBC-001 | No RBAC check — any authenticated user can create a database |
| 2 | HIGH | DBC-004 | No RBAC check — any authenticated user can update any database |
| 3 | HIGH | DBC-005 | No RBAC check — any authenticated user can delete any database |
| 4 | HIGH | DBC-005 | No running-queries guard before delete |
| 5 | HIGH | DBC-005 | 409 response is a bare sentinel error — does not include dataset list |
| 6 | MEDIUM | DBC-001 | Single-driver: only PostgreSQL (`pgx`) supported; MySQL, BigQuery, Snowflake, etc. return `ErrUnknownDatabaseDriver` |
| 7 | MEDIUM | DBC-002 | `SELECT version()` is PostgreSQL-specific — no multi-driver version query strategy |
| 8 | MEDIUM | DBC-004 | Missing Redis schema cache cleanup after update (pool is flushed, cache is not) |
| 9 | MEDIUM | DBC-005 | Missing Redis schema cache cleanup after delete |
| 10 | MEDIUM | DBC-007 | Schema cache is in-memory (`sync.Mutex` + `map`) instead of Redis as specified |
| 11 | MEDIUM | DBC-007 | Only `postgresSchemaInspector` — no MySQL/BigQuery/Snowflake inspector implementations |
| 12 | LOW | DBC-007 | `ListTables` hardcodes `table_type = 'BASE TABLE'` — excludes views; frontend spec expects view/table filtering |

### Phase 3 (not yet implemented — expected)

| # | Severity | Req | Status |
|---|----------|-----|--------|
| 13 | — | DBC-008 | SSH Tunnel — not started (Phase 3, P2) |
| 14 | — | DBC-009 | File Upload — not started (Phase 3, P2) |

---

## 3. Detailed Gap Analysis

### 3.1 HIGH — Missing RBAC Checks (DBC-001, DBC-004, DBC-005)

**Root Cause:** The `DatabaseService` methods `CreateDatabase`, `UpdateDatabase`, and `DeleteDatabase` do not call `repo.IsAdmin()` or verify ownership. The handler extracts the actor from context but the service never gates on role.

**Evidence:**
- `database_service.go:276` — `CreateDatabase` starts with no `IsAdmin` call
- `database_service.go:419` — `UpdateDatabase` loads the record but never checks `created_by_fk` against `actorUserID`
- `database_service.go:505` — `DeleteDatabase` loads the record but never checks ownership
- `database_service_test.go:356-370` — `TestDatabaseService_CreateDatabaseNonAdminReturnsForbidden` expects `ErrForbidden` but the fake's `IsAdmin` is never called by the service

**Spec requirement (DBC-001):** "Validate role → DB check → unique check"
**Spec requirement (DBC-004):** "Ownership check → smart password merge"
**Spec requirement (DBC-005):** "Ownership → count datasets → count running queries"

**Fix:** Add the following checks in the service layer:

```
CreateDatabase:  if !repo.IsAdmin(ctx, actorUserID) → ErrForbidden
UpdateDatabase:  if !repo.IsAdmin && existing.CreatedByFK != actorUserID → ErrForbidden
DeleteDatabase:  if !repo.IsAdmin && existing.CreatedByFK != actorUserID → ErrForbidden
```

**Files touched:**
- `backend/internal/app/db/database_service.go` — add guard checks at the top of each method
- `backend/internal/app/db/database_service_test.go` — update tests so `IsAdmin` calls are exercised

---

### 3.2 HIGH — Missing Running-Queries Guard (DBC-005)

**Root Cause:** `DeleteDatabase` only counts datasets. The spec requires a second guard: counting in-flight queries against the database.

**Evidence:** `database_service.go:514-521` — only `CountDatasetsByDatabaseID` is called; no query-counting step exists.

**Spec requirement (DBC-005):** "count datasets → count running queries → pool.Close → redis cleanup → GORM.Delete → audit"

**Fix:** Add a new repository method and guard in the delete flow:

```go
// New repository method
CountRunningQueriesByDatabaseID(ctx context.Context, databaseID uint) (int64, error)

// In DeleteDatabase
queryCount, err := s.repo.CountRunningQueriesByDatabaseID(ctx, databaseID)
if queryCount > 0 {
    return domain.ErrDatabaseInUse // or a new ErrDatabaseHasRunningQueries
}
```

**Files touched:**
- `backend/internal/domain/db/repository.go` — new interface method
- `backend/internal/repository/postgres/database_repo.go` — implement count
- `backend/internal/app/db/database_service.go` — wire guard
- `backend/internal/app/db/database_service_test.go` — test case

---

### 3.3 HIGH — 409 Without Dataset List (DBC-005)

**Root Cause:** `DeleteDatabase` returns the bare sentinel `ErrDatabaseInUse` when datasets exist. The spec requires a 409 response that includes a list of dependent datasets.

**Evidence:** `database_service.go:519-521`:
```go
if datasetCount > 0 {
    return domain.ErrDatabaseInUse
}
```

**Spec requirement (DBC-005):** "Has datasets → 409 with list."

**Fix:** Replace the flat error with a structured error type (same pattern as DS-008's `ReferencedByChartsError`):

```go
type DatabaseInUseError struct {
    Datasets []DatasetRef  `json:"datasets"`
}
```

The handler maps this to:
```json
{"error": "database is in use by 3 dataset(s)", "datasets": [{"id": 1, "table_name": "orders"}, ...]}
```

**Files touched:**
- `backend/internal/domain/db/errors.go` — new error type (or create a structured error file)
- `backend/internal/domain/db/repository.go` — new `ListDatasetsByDatabaseID` method (returns names/IDs)
- `backend/internal/repository/postgres/database_repo.go` — implement
- `backend/internal/app/db/database_service.go` — return structured error
- `backend/internal/delivery/http/db/database_handler.go` — map structured error in `handleError`
- `backend/internal/app/db/database_service_test.go` — test case

---

### 3.4 MEDIUM — Single-Driver Limitation (DBC-001, DBC-002, DBC-007)

**Root Cause:** `resolveSQLDriver` only handles two scheme names, both mapping to `pgx`.

**Evidence:** `database_service.go:167-175`:
```go
case "postgres", "postgresql":
    return "pgx", "postgresql", nil
default:
    return "", "", domain.ErrUnknownDatabaseDriver
```

**Spec requirement (DBC-001):** "RadioGroup + RadioGroupItem - DB type selector (PostgreSQL, MySQL, BigQuery, Snowflake, etc.) with logos"

**Impact:** The `CreateDatabase` flow (including the strict connection test) fails for any non-PostgreSQL database. Schema introspection (DBC-007) also only has a `postgresSchemaInspector`.

**Fix:** Add driver entries for MySQL (`go-sql-driver/mysql`), BigQuery, and Snowflake (`gosnowflake`). Add corresponding `SchemaInspector` implementations. The `SELECT version()` query in `Probe` must be parameterized per driver.

**Files touched:**
- `backend/internal/app/db/database_service.go` — extend `resolveSQLDriver` and `Probe` version query
- `backend/internal/app/db/schema_inspector.go` — add `mysqlSchemaInspector`, `bigquerySchemaInspector`, etc.
- `backend/go.mod` — new driver dependencies

---

### 3.5 MEDIUM — Missing Redis Schema Cache Cleanup (DBC-004, DBC-005)

**Root Cause:** The schema cache in DBC-007 is in-memory (`inMemorySchemaCache`), not Redis. Even if it were Redis, neither `UpdateDatabase` nor `DeleteDatabase` clears it.

**Evidence:**
- `database_service.go:207-208` — update flushes the pool but doesn't touch the schema cache
- `database_service.go:523-528` — delete flushes the pool but doesn't clear the cache
- `schema_cache_memory.go` — cache is a local `sync.Mutex` + `map[string]schemaCacheEntry`

**Spec requirement (DBC-004):** "pool.Close → redis SCAN+DEL 'schema:'+dbID+':*'"
**Spec requirement (DBC-005):** "pool.Close → redis cleanup → GORM.Delete → audit"

**Fix — part 1 (code references):** The spec directly references Redis cleanup. Since the current cache is in-memory and lives within the process, a short-term fix adds explicit cache invalidation calls. Long-term, switch to Redis and use the SCAN+DEL pattern.

**Short-term fix:**
- Add `InvalidateByPrefix(ctx, prefix string) error` to `SchemaCacheRepository`
- In `UpdateDatabase` and `DeleteDatabase`, after `poolManager.Close`, call `schemaCache.InvalidateByPrefix("schema:"+strconv.Itoa(databaseID)+":")`

**Files touched:**
- `backend/internal/domain/db/repository.go` — extend `SchemaCacheRepository` interface
- `backend/internal/app/db/schema_cache_memory.go` — implement prefix invalidation
- `backend/internal/app/db/database_service.go` — wire invalidation in update + delete flow
- `backend/internal/app/db/database_service_test.go` — test cases

---

### 3.6 MEDIUM — In-Memory Schema Cache Instead of Redis (DBC-007)

**Root Cause:** The spec explicitly states "Redis cache 10min" but the implementation uses a process-local map.

**Evidence:** `schema_cache_memory.go:1-56` — `inMemorySchemaCache` struct with `sync.RWMutex` and `map[string]schemaCacheEntry`. No Redis client anywhere in the schema cache layer.

**Spec requirement (DBC-007):** "Redis cache 10min."

**Impact:** Cache doesn't survive process restarts, doesn't share across instances, and grows unbounded (no eviction beyond TTL-based expiry on read).

**Fix:** Create a Redis-backed `SchemaCacheRepository` implementation. The existing `inMemorySchemaCache` satisfies the same interface, so this is a drop-in replacement via DI.

**Files touched:**
- `backend/internal/repository/redis/schema_cache.go` — NEW: Redis implementation
- `backend/internal/app/db/database_service.go` — accept Redis-backed cache via `NewDatabaseService` or `SetSchemaCache`
- `backend/internal/app/db/database_service_test.go` — test with Redis cache (integration) or continue with in-memory (unit)

---

### 3.7 MEDIUM — Single Inspector Implementation (DBC-007)

**Root Cause:** Only `postgresSchemaInspector` exists. The `SchemaInspector` interface is designed for multi-driver but no other implementations exist.

**Evidence:** `schema_inspector.go:24-31` — only `postgresSchemaInspector` struct; `NewDefaultSchemaInspector()` and `newDefaultSchemaInspector()` both return it.

**Spec requirement (DBC-007):** "Driver-abstracted schema discovery... Per-driver INFORMATION_SCHEMA or native queries."

**Fix:** Add `mysqlSchemaInspector` (using `information_schema` — nearly identical SQL with `?` placeholders), `bigquerySchemaInspector` (using `INFORMATION_SCHEMA` with BigQuery-specific catalog/schema conventions), and `snowflakeSchemaInspector`.

**Files touched:**
- `backend/internal/app/db/schema_inspector.go` — add new inspector implementations
- `backend/internal/app/db/database_service.go` — select inspector based on driver/backend type
- `backend/internal/app/db/database_service_test.go` — test cases per driver

---

### 3.8 LOW — ListTables Excludes Views (DBC-007)

**Root Cause:** The SQL query filters to `table_type = 'BASE TABLE'` only.

**Evidence:** `schema_inspector.go:76`:
```sql
WHERE table_schema = $1 AND table_type = 'BASE TABLE'
```

**Spec requirement (DBC-001 frontend):** "Checkbox - filter: 'Show views only' / 'Show tables only'"

**Fix:** Add a `tableType` parameter to `ListDatabaseTablesRequest` and `ListTables`. Default to `BASE TABLE` for backward compatibility; accept `VIEW` or empty (all).

**Files touched:**
- `backend/internal/domain/db/database.go` — add `TableType` field to `ListDatabaseTablesRequest`
- `backend/internal/app/db/schema_inspector.go` — accept table type parameter
- `backend/internal/app/db/database_service.go` — pass table type through
- `backend/internal/delivery/http/db/database_handler.go` — parse `table_type` query param

---

## 4. Cross-Cutting Observations

### 4.1 Redis Usage Is Inconsistent

| Area | Spec says | Implementation |
|------|-----------|----------------|
| DBC-004 schema cache flush | redis SCAN+DEL | Not implemented |
| DBC-005 schema cache cleanup | redis cleanup | Not implemented |
| DBC-007 schema cache | Redis cache 10min | in-memory map |

The schema cache layer has a clean `SchemaCacheRepository` interface, so replacing the in-memory implementation with Redis is a single swap — no service-layer changes required. However, DBC-004 and DBC-005 need code changes to call cache invalidation regardless of backend.

### 4.2 Test Gaps

| Missing test | Covers |
|-------------|--------|
| `TestCreateDatabase_NonAdmin_ReturnsForbidden` | DBC-001 RBAC (test exists but doesn't exercise the guard) |
| `TestUpdateDatabase_NonOwner_ReturnsForbidden` | DBC-004 ownership |
| `TestDeleteDatabase_NonOwner_ReturnsForbidden` | DBC-005 ownership |
| `TestDeleteDatabase_HasRunningQueries_Returns409` | DBC-005 query guard |
| `TestDeleteDatabase_HasDatasets_Returns409WithList` | DBC-005 structured error |
| `TestUpdateDatabase_FlushesSchemaCache` | DBC-004 cache cleanup |
| `TestDeleteDatabase_FlushesSchemaCache` | DBC-005 cache cleanup |
| `TestCreateDatabase_MySQLDriver` | DBC-001 multi-driver |
| `TestSchemaInspector_MySQL` | DBC-007 multi-driver |

---

## 5. Implementation Priority

```
Phase 1 (blocking — HIGH severity):
  A. RBAC checks: Add IsAdmin/ownership guards to Create, Update, Delete
  B. DBC-005: Running-queries guard
  C. DBC-005: Structured 409 error with dataset list

Phase 2 (MEDIUM severity):
  D. Multi-driver support (MySQL first — most common after PostgreSQL)
  E. Schema cache Redis implementation
  F. Schema cache invalidation on update/delete

Phase 3 (LOW severity + Phase 3 specs):
  G. DBC-007: Table type filter (views support)
  H. DBC-008: SSH Tunnel (Phase 3)
  I. DBC-009: File Upload (Phase 3)
```

---

## 6. Success Criteria

1. Non-admin users receive 403 when creating databases
2. Non-owner users receive 403 when updating/deleting databases
3. Delete with running queries returns 409 with query list
4. Delete with datasets returns 409 with `{"datasets": [...]}` in body
5. MySQL driver accepted in `CreateDatabase` and connection test
6. Schema introspection works for MySQL databases
7. Schema cache is Redis-backed (or invalidation works on update/delete)
8. All existing tests pass; new tests cover RBAC, multi-driver, cache invalidation
