# QE-008 Query Cost Estimation — Design

## Overview

Add `POST /api/v1/query/estimate` to run `EXPLAIN` without executing the query, returning planner cost estimates. Supports PostgreSQL now; BigQuery/Snowflake/MySQL drivers plug in later.

## Architecture

```
FE (SQLLab toolbar: Estimate button + debounce auto-fire)
  → POST /api/v1/query/estimate {sql, database_id}
  → handler.Estimate()
    → Redis rate limit check (30 req/60s per user)
    → databaseRepo.GetDatabaseByID → sqlalchemy_uri
    → ParseSQLAlchemyURI → extract scheme (driver type)
    → NewEstimator(driver) → estimator.Estimate(ctx, sql, conn)
    → poolMgr.GetPinned(ctx, dbID, uri) → *sql.Conn
    → Return EstimateResult
```

## Files

### New
- `backend/internal/app/query/cost_estimator.go` — Estimator interface + dispatcher + EstimateResult type
- `backend/internal/app/query/cost_pg.go` — PostgreSQL EXPLAIN (FORMAT JSON)
- `backend/internal/app/query/cost_unsupported.go` — Fallback (supported: false)
- `frontend/src/hooks/useEstimate.ts` — useDebounce + useMutation
- `frontend/src/components/query/EstimatePopover.tsx` — Popover with Card/Skeleton/Alert

### Modified
- `backend/internal/domain/query/entity.go` — +EstimateRequest, EstimateResult
- `backend/internal/delivery/http/query/handler.go` — +rdb, dbRepo, poolMgr fields; +Estimate() method
- `backend/internal/delivery/http/router.go` — +POST /api/v1/query/estimate route
- `backend/cmd/api/main.go` — wire rdb, dbRepo, poolMgr into queryHandler
- `frontend/src/api/queries.ts` — +estimate()
- `frontend/src/stores/sqlLabStore.ts` — +estimate field on SqlLabTab
- `frontend/src/components/query/QueryBadges.tsx` — +EstimateBadge
- `frontend/src/pages/sqllab/SQLLabPage.tsx` — toolbar button, auto-debounce, popover

## Types

```go
// domain/query/entity.go

type EstimateRequest struct {
    SQL        string `json:"sql" binding:"required"`
    DatabaseID uint   `json:"database_id" binding:"required"`
}

type EstimateResult struct {
    Supported        bool    `json:"supported"`
    Driver           string  `json:"driver,omitempty"`
    TotalCost        float64 `json:"total_cost,omitempty"`
    EstimatedRows    int64   `json:"estimated_rows,omitempty"`
    BytesProcessed   int64   `json:"bytes_processed,omitempty"`
    EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
}
```

## Estimator Interface

```go
type Estimator interface {
    Estimate(ctx context.Context, sql string, conn dbpool.SQLConnection) (*EstimateResult, error)
}

func NewEstimator(driver string) Estimator
```

Dispatches: `postgresql` → postgresEstimator, all else → unsupportedEstimator.

## API Contract

```
POST /api/v1/query/estimate
Auth: JWT required

// PostgreSQL:
200 {"supported":true, "driver":"postgresql", "total_cost":1250.5, "estimated_rows":50000}

// Unsupported:
200 {"supported":false}

// Rate limited:
429 {"error":"rate_limited", "retry_after":45}

// Invalid SQL:
422 {"error":"invalid_sql", "message":"..."}
```

## Rate Limiting

```go
key := "rate:estimate:" + strconv.FormatUint(uint64(userID), 10)
count, _ := rdb.Incr(ctx, key).Result()
if count == 1 { rdb.Expire(ctx, key, 60*time.Second) }
if count > 30 { return 429 }
```

## PostgreSQL EXPLAIN

```sql
EXPLAIN (FORMAT JSON) SELECT * FROM ...
```

Parse `[0].Plan["Total Cost"]` and `[0].Plan["Plan Rows"]` from the JSON array.

## Frontend

- Button: "Estimate Cost" with Zap icon, variant=outline, in SQLLab toolbar
- Visibility: only when `selectedDB.backend` ∈ ["postgresql","bigquery","snowflake","mysql"]
- Debounce: `useDebounce(sql, 2000)` triggers estimateMutation automatically
- Popover: Card with metrics, Skeleton during loading, Alert "Estimate only. Actual execution may differ."
- Badge: "~50k rows" (PostgreSQL) or "~$0.005" (BigQuery)

## Error Handling

| Code | Condition | Body |
|------|-----------|------|
| 200 | Unsupported DB | `{"supported":false}` |
| 400 | Invalid request body | `{"error":"invalid_request"}` |
| 401 | Missing/invalid JWT | `{"error":"unauthorized"}` |
| 422 | EXPLAIN failed (bad SQL) | `{"error":"invalid_sql"}` |
| 429 | >30 req/60s | `{"error":"rate_limited","retry_after":45}` |
| 500 | Internal error | `{"error":"internal_error"}` |

## Testing

### Backend
- `cost_pg_test.go` — parse EXPLAIN JSON output
- `cost_unsupported_test.go` — returns supported:false
- `handler_test.go` — estimate endpoint: rate limit 429, invalid SQL 422, missing auth 401, unsupported DB 200, valid PostgreSQL 200

### Frontend
- `useEstimate.test.ts` — debounce + mutation
- `EstimatePopover.test.tsx` — loading, metrics display
- `SQLLabPage.test.tsx` — button visibility per DB backend
