# DS-001 to DS-008 Gap Fixes — Design Spec

**Date:** 2026-05-14
**Scope:** Backend Go + frontend TypeScript fixes for 12 identified gaps
**Spec version:** 1.0

---

## 1. Problem Summary

12 gaps/issues found during verification of the Dataset Service implementation against `DATASET_SERVICE_03_PHASE1.md` requirements DS-001 through DS-008.

| # | Severity | Req | Issue |
|---|----------|-----|-------|
| 1 | HIGH | DS-002 | `validate_sql=true` is a no-op — no LIMIT 0 execution, no immediate column population |
| 2 | HIGH | DS-002 | No `sqlparser.Parse()` — uses regex for SQL validation |
| 3 | HIGH | DS-008 | `force=true` doesn't delete charts |
| 4 | HIGH | DS-008 | Missing cascade for `rls_filter_tables` and `tagged_object` |
| 5 | HIGH | DS-008 | No Redis qcache cleanup on delete |
| 6 | MEDIUM | DS-004/005/006 | Partial update bug: empty strings always written as updates |
| 7 | MEDIUM | DS-005 | No `sqlparser.ParseExpr()` — uses regex for expression validation |
| 8 | MEDIUM | DS-006 | Delete metric doesn't check chart references |
| 9 | MEDIUM | DS-008 | 409 doesn't return chart list |
| 10 | LOW | DS-002 | Virtual dataset uses `/datasets/virtual` instead of `/datasets` with `sql` field |
| 11 | LOW | DS-004 | Update response returns stale `dataset.Name` instead of updated table_name |
| 12 | LOW | DS-006 | Frontend `CreateMetricPayload` missing `description` and `extra` fields |

---

## 2. Architecture

No new packages. All changes fit within the existing layered architecture:

```
Handler (delivery/http/dataset) → Service (app/dataset) → Repository (domain + postgres + redis)
```

A new file `backend/internal/app/dataset/sql_validator.go` is added for the LIMIT 0 execution logic. A new Go dependency `github.com/xwb1989/sqlparser` is added.

---

## 3. Fix 1 — Partial Update Bug (pointer DTOs)

### 3.1 Root Cause

Flat `string` / `bool` / `int` fields in update DTOs cannot distinguish "field not provided" from "field intentionally set to empty/zero." The repo mapping logic writes empty strings unconditionally:

```go
// current — always writes description
if req.Description != "" || req.Description == "" {
    updates["description"] = req.Description  // "" overwrites existing
}
```

### 3.2 Fix

Change optional fields in update DTOs to Go pointer types. `nil` = skip, non-nil = apply the value (even if zero).

**`UpdateDatasetMetadataRequest` changes:**
```
TableName           string → *string
Description         string → *string
MainDttmCol         string → *string
SQL                 string → *string
```

**`UpdateColumnRequest` changes:**
```
VerboseName      string → *string
Description      string → *string
PythonDateFormat string → *string
Expression       string → *string
ColumnType       string → *string
```

**`UpdateMetricRequest` changes:**
```
MetricName           string → *string
VerboseName          string → *string
MetricType           string → *string
Expression           string → *string
D3Format             string → *string
WarningText          string → *string
CertifiedBy          string → *string
CertificationDetails string → *string
```

### 3.3 Repo changes

Pattern changes from:
```go
if req.Description != "" || req.Description == "" {
    updates["description"] = req.Description
}
```
To:
```go
if req.Description != nil {
    updates["description"] = *req.Description
}
```

Applied in `UpdateDatasetMetadata`, `UpdateColumn`, `BulkUpdateColumns`, `UpdateMetric`.

### 3.4 Service validation changes

Existing code that checks `if req.MainDttmCol != ""` becomes `if req.MainDttmCol != nil && *req.MainDttmCol != ""`.
Existing code that checks `if req.SQL != ""` becomes `if req.SQL != nil && *req.SQL != ""`.

### 3.5 Files touched

| File | Change |
|------|--------|
| `backend/internal/domain/dataset/dataset.go` | DTO pointer changes |
| `backend/internal/repository/postgres/dataset_repo.go` | nil-check update mapping |
| `backend/internal/app/dataset/service.go` | nil-check validation |
| `backend/internal/app/dataset/service_test.go` | Update test call sites |
| `backend/internal/delivery/http/dataset/handler_test.go` | Update test payloads |

---

## 4. Fix 2 — DS-002: Real validate_sql + sqlparser

### 4.1 sqlparser integration

**New dependency:** `github.com/xwb1989/sqlparser`

**Replacements:**

| Current | Replacement |
|---------|-------------|
| `selectPattern` regex (service.go:151-154) | `sqlparser.Parse(sql)` → type-assert `*sqlparser.Select` |
| `isValidSQLExpression()` regex (service.go:566-581) | `sqlparser.ParseExpr(expr)` |

**DS-002 SQL validation in `CreateVirtualDataset`:**
```go
stmt, err := sqlparser.Parse(sql)
if err != nil {
    return nil, domain.ErrSQLNotSelect  // or a new ErrSQLParseError
}
if _, ok := stmt.(*sqlparser.Select); !ok {
    return nil, domain.ErrSQLNotSelect
}
```

**DS-005 expression validation in `validateColumnRequest`:**
```go
if req.Expression != nil && *req.Expression != "" {
    if _, err := sqlparser.ParseExpr(*req.Expression); err != nil {
        return domain.ErrInvalidExpression
    }
}
```

### 4.2 validate_sql=true execution

When `CreateVirtualDatasetRequest.ValidateSQL` is true, the service:
1. Parses SQL with sqlparser (as above)
2. Executes `SELECT * FROM (<sql>) AS _validate LIMIT 0` via the database connection pool
3. Extracts column metadata from `rows.ColumnTypes()`
4. Creates column rows in the database inline
5. Returns columns populated in the response

**New interface dependency for Service:**
```go
type sqlExecutor interface {
    ExecLimitZero(ctx context.Context, databaseID uint, sql string) ([]domain.Column, error)
}
```

**New file: `backend/internal/app/dataset/sql_validator.go`**

Contains `sqlExecutor` implementation that uses the connection pool manager to get a DB connection, execute the LIMIT 0 query, and extract column info.

**NewService changes:** `sqlExecutor` follows the same nil-safe pattern as `SyncQueue` — if nil, a noop implementation is used that returns `nil, nil` (validation skipped, falls through to async sync). Similarly, the `cacheFlusher` for DS-008 delete cache cleanup defaults to a noop if nil.

**Response:**
```go
type CreateVirtualDatasetResponse struct {
    ID             uint     `json:"id"`
    TableName      string   `json:"table_name"`
    BackgroundSync bool     `json:"background_sync"`
    Columns        []Column `json:"columns,omitempty"` // populated when validate_sql=true
}
```

### 4.3 Files touched

| File | Change |
|------|--------|
| `backend/internal/app/dataset/service.go` | Replace regex with sqlparser; add validate_sql flow |
| `backend/internal/app/dataset/sql_validator.go` | NEW — LIMIT 0 executor |
| `backend/internal/domain/dataset/dataset.go` | Add `ValidateSQLResult` fields if needed |
| `backend/internal/app/dataset/service_test.go` | Tests for validate_sql path |
| `backend/go.mod` | Add `github.com/xwb1989/sqlparser` |

---

## 5. Fix 3 — DS-008: Delete gaps

### 5.1 force=true deletes charts

In `Service.DeleteDataset`, when `req.Force && isAdmin && chartCount > 0`:
```go
if req.Force && isAdmin {
    if err := s.repo.DeleteChartsByDatasetID(ctx, id); err != nil {
        return nil, fmt.Errorf("deleting charts: %w", err)
    }
}
```

**New repository method:**
```go
DeleteChartsByDatasetID(ctx context.Context, datasetID uint) error
```

Implementation deletes rows from `slices` where `datasource_id` matches the dataset ID (string comparison).

### 5.2 Cascade tables

Add to the delete transaction in `DeleteDataset` (dataset_repo.go):
```go
tx.Table("rls_filter_tables").Where("table_id = ?", id).Delete(nil)
tx.Table("tagged_object").Where("object_id = ? AND object_type = 'table'", id).Delete(nil)
```

Execution order: `rls_filter_tables` → `tagged_object` → `table_columns` → `sql_metrics` → `tables`.

### 5.3 Redis qcache cleanup

After successful delete transaction, scan and delete Redis keys:

```go
pattern := fmt.Sprintf("qcache:%s:*", dataset.Perm)
iter := redisClient.Scan(ctx, 0, pattern, 0).Iterator()
var keys []string
for iter.Next(ctx) {
    keys = append(keys, iter.Val())
}
if len(keys) > 0 {
    redisClient.Del(ctx, keys...)
}
```

The `Service` needs a new dependency: `cacheFlusher interface { FlushPattern(ctx context.Context, pattern string) (int64, error) }`. The existing Redis infrastructure (used by `FlushCache`) is adapted.

### 5.4 409 with chart list

Replace the simple sentinel `ErrDatasetReferencedByCharts` with a structured error:

```go
type ReferencedByChartsError struct {
    Charts []ChartRef
}

func (e *ReferencedByChartsError) Error() string {
    return fmt.Sprintf("dataset is referenced by %d chart(s)", len(e.Charts))
}
```

In `handler.go`:
```go
var refErr *domain.ReferencedByChartsError
if errors.As(err, &refErr) {
    c.JSON(http.StatusConflict, gin.H{
        "error":  refErr.Error(),
        "charts": refErr.Charts,
    })
}
```

**New repository method:**
```go
GetChartsByDatasetID(ctx context.Context, datasetID uint) ([]ChartRef, error)
```

### 5.5 Files touched

| File | Change |
|------|--------|
| `backend/internal/domain/dataset/errors.go` | Add `ReferencedByChartsError` type |
| `backend/internal/domain/dataset/repository.go` | Add `DeleteChartsByDatasetID`, `GetChartsByDatasetID` |
| `backend/internal/repository/postgres/dataset_repo.go` | Implement new methods + cascade additions |
| `backend/internal/app/dataset/service.go` | force delete + cache cleanup logic |
| `backend/internal/delivery/http/dataset/handler.go` | Structured 409 error handling |
| `backend/internal/app/dataset/service_test.go` | Tests for force delete, 409 with charts |

---

## 6. Fix 4 — DS-006: Delete metric chart reference check

In `Service.DeleteMetric`, before deleting:
```go
chartCount, chartRefs, err := s.repo.CountChartsByMetricName(ctx, datasetID, metric.MetricName)
if err != nil {
    return nil, fmt.Errorf("checking chart references: %w", err)
}
if chartCount > 0 {
    warnings = buildMetricWarning(chartRefs)
}
```

**New repository method:**
```go
CountChartsByMetricName(ctx context.Context, datasetID uint, metricName string) (int64, []ChartRef, error)
```

Implementation scans `slices.params` JSON for the metric name (string contains check on the params JSON blob).

### Files touched

| File | Change |
|------|--------|
| `backend/internal/domain/dataset/repository.go` | Add `CountChartsByMetricName` |
| `backend/internal/repository/postgres/dataset_repo.go` | Implement method |
| `backend/internal/app/dataset/service.go` | Wire metric reference check |
| `backend/internal/app/dataset/service_test.go` | Test delete with chart references |

---

## 7. Fix 5 — Route Consolidation

### 7.1 Backend

Remove `POST /datasets/virtual` from `router.go`. Modify the `CreatePhysicalDataset` handler (or create a unified `CreateDataset` handler) to inspect the request body:

```go
func (h *Handler) CreateDataset(c *gin.Context) {
    // Try parsing as physical first
    var physReq domain.CreatePhysicalDatasetRequest
    if err := c.ShouldBindJSON(&physReq); err != nil {
        // ...
    }

    // If sql field present, route to virtual
    // This requires raw body inspection or a discriminated union approach
}
```

Recommended approach: use a discriminator DTO:

```go
type CreateDatasetDiscriminator struct {
    SQL string `json:"sql"`
}

var disc CreateDatasetDiscriminator
c.ShouldBindJSON(&disc)
if disc.SQL != "" {
    h.CreateVirtualDataset(c) // internally delegates
} else {
    h.CreatePhysicalDataset(c)
}
```

The frontend `datasetsApi.createVirtualDataset` already posts to `/api/v1/datasets/virtual` — route the handler to the unified endpoint.

### 7.2 Frontend

Change `createVirtualDataset` endpoint from `/api/v1/datasets/virtual` to `/api/v1/datasets`.

### 7.3 Files touched

| File | Change |
|------|--------|
| `backend/internal/delivery/http/router.go` | Remove `/datasets/virtual` route; add unified handler |
| `backend/internal/delivery/http/dataset/handler.go` | Add `CreateDataset` discriminator handler |
| `frontend/src/api/datasets.ts` | Change `createVirtualDataset` URL |

---

## 8. Fix 6 — Misc Low-Priority Items

### 8.1 DS-004: Stale table_name in update response

In `Service.UpdateDatasetMetadata`:
```go
tableName := dataset.Name
if req.TableName != nil && *req.TableName != "" {
    tableName = *req.TableName
}
return &domain.UpdateDatasetMetadataResponse{
    ID:        id,
    TableName: tableName,
    ...
}
```

**File:** `service.go` only.

### 8.2 DS-006: Frontend CreateMetricPayload

Add `description` and `extra` fields to the TypeScript interface:

```typescript
export interface CreateMetricPayload {
    metric_name: string;
    verbose_name?: string;
    metric_type: string;
    expression: string;
    description?: string;   // added
    extra?: string;          // added
    d3_format?: string;
    warning_text?: string;
    is_restricted?: boolean;
    certified_by?: string;
    certification_details?: string;
}
```

**File:** `frontend/src/api/datasets.ts` only.

---

## 9. Testing Strategy

### 9.1 Backend unit tests (service_test.go)

| Test | Covers |
|------|--------|
| `TestCreateVirtualDataset_ValidateSQL_PopulatesColumns` | DS-002 validate_sql=true |
| `TestCreateVirtualDataset_NonSelect_RejectedByParser` | DS-002 sqlparser |
| `TestCreateVirtualDataset_Semicolon_RejectedByParser` | DS-002 |
| `TestDeleteDataset_Force_DeletesCharts` | DS-008 force |
| `TestDeleteDataset_Referenced_Returns409WithCharts` | DS-008 chart list |
| `TestDeleteDataset_AdminForce_DeletesCascade` | DS-008 cascade |
| `TestDeleteMetric_ReferencedByCharts_ReturnsWarnings` | DS-006 metric refs |
| `TestUpdateDataset_Partial_NilSkipsFields` | DS-004 partial update |
| `TestUpdateColumn_InvalidExpression_ParseExprFails` | DS-005 sqlparser |
| `TestUpdateDataset_TableNameReflectsChange` | DS-004 stale name |

### 9.2 Backend HTTP tests (handler_test.go)

| Test | Covers |
|------|--------|
| `TestCreateDataset_DiscriminatesVirtualBySQLField` | DS-002 route |
| `TestCreateDataset_DiscriminatesPhysical` | DS-001 route |
| `TestDeleteDataset_409ReturnsChartList` | DS-008 structured error |

### 9.3 Frontend tests

- Update existing `datasetsApi` test to verify `createVirtualDataset` calls `/api/v1/datasets` instead of `/datasets/virtual`.

---

## 10. Implementation Order (Dependency Graph)

```
Phase 1 (no deps between them — can be parallel):
  A. Fix 1: Pointer DTOs for partial update (DS-004, DS-005, DS-006)
  B. Fix 2a: sqlparser integration (DS-002, DS-005)
  C. Fix 6: Misc low-priority (DS-004 stale name, DS-006 frontend)

Phase 2 (depends on Phase 1A + 1B):
  D. Fix 2b: validate_sql=true LIMIT 0 execution (DS-002)
  E. Fix 4: Delete metric chart reference check (DS-006)

Phase 3 (depends on Phase 1A):
  F. Fix 3: DS-008 delete gaps (force, cascade, cache, chart list)
  G. Fix 5: Route consolidation (DS-002)

Phase 4 (integration):
  H. Run full test suite, fix regressions
  I. Verify all 12 items against spec
```

---

## 11. Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Pointer DTO breaks JSON serialization | Low | Go's `encoding/json` handles `*string` natively — nil omits, non-nil includes |
| sqlparser rejects valid SQL | Medium | Thorough testing with known Superset SQL patterns; fallback to regex for unrecognized syntax |
| LIMIT 0 execution fails on some DBs | Low | Already used in column sync worker — same path is proven |
| Chart delete cascade misses foreign keys | Low | Check `slices` table schema for FK constraints; use explicit WHERE |
| Redis client not available in delete path | Low | Make cache flush best-effort (log warning on failure, don't block delete) |

---

## 12. Success Criteria

1. `validate_sql=true` creates virtual dataset with columns populated inline (201 + columns array)
2. `sqlparser.Parse()` rejects non-SELECT and SQL with syntax errors (422)
3. `DELETE /datasets/:id?force=true` as Admin deletes charts then dataset (204)
4. Delete cascade includes `rls_filter_tables` and `tagged_object` rows
5. Delete triggers Redis qcache key removal matching `qcache:<perm>:*`
6. Partial PUT updates don't overwrite unset fields with zero values
7. `PUT /datasets/:id/columns/:col_id` validates expression via `sqlparser.ParseExpr()`
8. Delete metric returns chart warnings when metric is used in charts
9. 409 on delete includes `charts:[...]` array in response body
10. `POST /api/v1/datasets` with `sql` field creates virtual; without creates physical
11. PUT response reflects updated `table_name` if provided
12. Frontend `CreateMetricPayload` includes `description` and `extra` fields
13. All existing tests pass; new tests cover all changes
