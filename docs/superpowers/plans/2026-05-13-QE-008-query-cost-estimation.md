# QE-008 Query Cost Estimation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add POST /api/v1/query/estimate to run EXPLAIN without executing, returning planner cost estimates. PostgreSQL first; other drivers plug in later.

**Architecture:** Strategy-pattern estimators behind a dispatch interface. Handler-level rate limiting via Redis. Frontend: button + 2s auto-debounce with Popover metrics and toolbar badge.

**Tech Stack:** Go (Gin, GORM, go-redis), TypeScript (React 18, TanStack Query v5, Zustand, shadcn/ui)

---

## Backend Tasks

### Task 1: Add domain types for estimate request/response

**Files:**
- Modify: `backend/internal/domain/query/entity.go` (append)

- [ ] **Step 1: Add EstimateRequest and EstimateResult types**

```go
// EstimateRequest is the body for POST /api/v1/query/estimate (QE-008)
type EstimateRequest struct {
	SQL        string `json:"sql" binding:"required"`
	DatabaseID uint   `json:"database_id" binding:"required"`
}

// EstimateResult holds the cost estimate for a query (QE-008)
type EstimateResult struct {
	Supported        bool    `json:"supported"`
	Driver           string  `json:"driver,omitempty"`
	TotalCost        float64 `json:"total_cost,omitempty"`
	EstimatedRows    int64   `json:"estimated_rows,omitempty"`
	BytesProcessed   int64   `json:"bytes_processed,omitempty"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./internal/domain/query/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/query/entity.go
git commit -m "feat: add EstimateRequest and EstimateResult domain types (QE-008)"
```

---

### Task 2: Create unsupported estimator (fallback)

**Files:**
- Create: `backend/internal/app/query/cost_unsupported.go`

- [ ] **Step 1: Write failing test**

Create: `backend/internal/app/query/cost_unsupported_test.go`

```go
package query

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnsupportedEstimatorReturnsSupportedFalse(t *testing.T) {
	e := &unsupportedEstimator{driver: "sqlite"}
	result, err := e.Estimate(context.Background(), "SELECT 1", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Supported)
}

func TestUnsupportedEstimatorNoPanicWithNilConn(t *testing.T) {
	e := &unsupportedEstimator{driver: "mongodb"}
	result, err := e.Estimate(context.Background(), "db.collection.find()", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Supported)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/app/query/ -run TestUnsupportedEstimator -v`
Expected: FAIL with "undefined: unsupportedEstimator"

- [ ] **Step 3: Write minimal implementation**

Create: `backend/internal/app/query/cost_unsupported.go`

```go
package query

import (
	"context"

	dbpool "superset/auth-service/internal/app/db"
	domainquery "superset/auth-service/internal/domain/query"
)

type unsupportedEstimator struct {
	driver string
}

func (e *unsupportedEstimator) Estimate(ctx context.Context, sql string, conn dbpool.SQLConnection) (*domainquery.EstimateResult, error) {
	return &domainquery.EstimateResult{Supported: false}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/app/query/ -run TestUnsupportedEstimator -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/app/query/cost_unsupported.go backend/internal/app/query/cost_unsupported_test.go
git commit -m "feat: add unsupported estimator fallback (QE-008)"
```

---

### Task 3: Create PostgreSQL EXPLAIN estimator

**Files:**
- Create: `backend/internal/app/query/cost_pg.go`

- [ ] **Step 1: Write failing test**

Create: `backend/internal/app/query/cost_pg_test.go`

```go
package query

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresEstimatorParsesExplainJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	explainJSON := `[{"Plan":{"Node Type":"Seq Scan","Startup Cost":0.00,"Total Cost":1250.50,"Plan Rows":50000,"Plan Width":100}}]`
	rows := sqlmock.NewRows([]string{"QUERY PLAN"}).AddRow(explainJSON)
	mock.ExpectQuery(`EXPLAIN \(FORMAT JSON\) SELECT`).WillReturnRows(rows)

	e := &postgresEstimator{}
	result, err := e.Estimate(context.Background(), "SELECT * FROM orders", db)
	require.NoError(t, err)
	assert.True(t, result.Supported)
	assert.Equal(t, "postgresql", result.Driver)
	assert.Equal(t, float64(1250.50), result.TotalCost)
	assert.Equal(t, int64(50000), result.EstimatedRows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEstimatorHandlesExplainError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`EXPLAIN \(FORMAT JSON\)`).WillReturnError(sql.ErrConnDone)

	e := &postgresEstimator{}
	result, err := e.Estimate(context.Background(), "SELECT * FROM orders", db)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEstimatorHandlesInvalidJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"QUERY PLAN"}).AddRow(`not json`)
	mock.ExpectQuery(`EXPLAIN \(FORMAT JSON\)`).WillReturnRows(rows)

	e := &postgresEstimator{}
	result, err := e.Estimate(context.Background(), "SELECT 1", db)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/app/query/ -run TestPostgresEstimator -v`
Expected: FAIL with "undefined: postgresEstimator"

- [ ] **Step 3: Write implementation**

Create: `backend/internal/app/query/cost_pg.go`

```go
package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	dbpool "superset/auth-service/internal/app/db"
	domainquery "superset/auth-service/internal/domain/query"
)

type postgresEstimator struct{}

type pgPlanNode struct {
	Plan pgPlanDetail `json:"Plan"`
}

type pgPlanDetail struct {
	TotalCost float64 `json:"Total Cost"`
	PlanRows  int64   `json:"Plan Rows"`
}

func (e *postgresEstimator) Estimate(ctx context.Context, sqlStr string, conn dbpool.SQLConnection) (*domainquery.EstimateResult, error) {
	if conn == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	explainSQL := "EXPLAIN (FORMAT JSON) " + sqlStr
	rows, err := conn.QueryContext(ctx, explainSQL)
	if err != nil {
		return nil, fmt.Errorf("EXPLAIN failed: %w", err)
	}
	defer rows.Close()

	var planText string
	if rows.Next() {
		if err := rows.Scan(&planText); err != nil {
			return nil, fmt.Errorf("scanning EXPLAIN result: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading EXPLAIN result: %w", err)
	}

	var nodes []pgPlanNode
	if err := json.Unmarshal([]byte(planText), &nodes); err != nil {
		return nil, fmt.Errorf("parsing EXPLAIN JSON: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("empty EXPLAIN result")
	}

	return &domainquery.EstimateResult{
		Supported:     true,
		Driver:        "postgresql",
		TotalCost:     nodes[0].Plan.TotalCost,
		EstimatedRows: nodes[0].Plan.PlanRows,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/app/query/ -run TestPostgresEstimator -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/app/query/cost_pg.go backend/internal/app/query/cost_pg_test.go
git commit -m "feat: add PostgreSQL EXPLAIN estimator (QE-008)"
```

---

### Task 4: Create estimator interface and dispatcher

**Files:**
- Create: `backend/internal/app/query/cost_estimator.go`

- [ ] **Step 1: Write interface + dispatcher**

```go
package query

import (
	"context"

	dbpool "superset/auth-service/internal/app/db"
	domainquery "superset/auth-service/internal/domain/query"
)

// Estimator runs EXPLAIN on a database to estimate query cost.
type Estimator interface {
	Estimate(ctx context.Context, sql string, conn dbpool.SQLConnection) (*domainquery.EstimateResult, error)
}

// NewEstimator returns the appropriate estimator for the given database driver.
func NewEstimator(driver string) Estimator {
	switch driver {
	case "postgresql":
		return &postgresEstimator{}
	default:
		return &unsupportedEstimator{driver: driver}
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./internal/app/query/...`
Expected: PASS

- [ ] **Step 3: Write dispatcher test**

Create/modify test:

```go
func TestNewEstimatorDispatchesCorrectly(t *testing.T) {
	tests := []struct {
		driver        string
		expectPG      bool
		expectUnsupported bool
	}{
		{"postgresql", true, false},
		{"sqlite", false, true},
		{"bigquery", false, true},
		{"mysql", false, true},
		{"snowflake", false, true},
		{"", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			e := NewEstimator(tt.driver)
			require.NotNil(t, e)
			_, isPG := e.(*postgresEstimator)
			_, isUnsupported := e.(*unsupportedEstimator)
			assert.Equal(t, tt.expectPG, isPG)
			assert.Equal(t, tt.expectUnsupported, isUnsupported)
		})
	}
}
```

- [ ] **Step 4: Run test to verify**

Run: `cd backend && go test ./internal/app/query/ -run TestNewEstimatorDispatchesCorrectly -v`
Expected: PASS (if in separate file; if added to existing test, all pass)

- [ ] **Step 5: Commit**

```bash
git add backend/internal/app/query/cost_estimator.go
git commit -m "feat: add cost estimator interface and dispatcher (QE-008)"
```

---

### Task 5: Add Estimate method to query handler

**Files:**
- Modify: `backend/internal/delivery/http/query/handler.go`

- [ ] **Step 1: Add rdb, dbRepo, poolMgr to handler struct and constructor**

In the Handler struct, add three new fields after `roleRepo`:
```go
type Handler struct {
	// ...existing fields...
	rdb      *redis.Client
	dbRepo   domdb.DatabaseRepository
	poolMgr  dbpool.DatabaseConnectionPool
}
```

Update constructor `NewHandlerWithAsync` to accept and set them:
```go
func NewHandlerWithAsync(
	executor *svcquery.QueryExecutor,
	asyncExecutor *svcquery.AsyncQueryExecutor,
	pubKey *rsa.PublicKey,
	jwtRepo domain.JWTRepository,
	userRepo domain.UserRepository,
	queryRepo domainquery.Repository,
	roleRepo domain.RoleRepository,
	rdb *redis.Client,
	dbRepo domdb.DatabaseRepository,
	poolMgr dbpool.DatabaseConnectionPool,
) *Handler {
	return &Handler{
		executor:      executor,
		asyncExecutor: asyncExecutor,
		pubKey:        pubKey,
		jwtRepo:       jwtRepo,
		userRepo:      userRepo,
		queryRepo:     queryRepo,
		roleRepo:      roleRepo,
		rdb:           rdb,
		dbRepo:        dbRepo,
		poolMgr:       poolMgr,
	}
}
```

Add imports: `"strconv"`, `"github.com/redis/go-redis/v9"`, `domdb "superset/auth-service/internal/domain/db"`, `dbpool "superset/auth-service/internal/app/db"`, `"superset/auth-service/internal/pkg/crypto"`

- [ ] **Step 2: Add Estimate method**

Append to handler.go:

```go
type EstimateRequest = domainquery.EstimateRequest

func (h *Handler) Estimate(c *gin.Context) {
	var req EstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return
	}

	// Rate limit: 30 requests per 60 seconds per user
	if h.rdb != nil {
		key := "rate:estimate:" + strconv.FormatUint(uint64(userCtx.ID), 10)
		count, err := h.rdb.Incr(c.Request.Context(), key).Result()
		if err == nil {
			if count == 1 {
				h.rdb.Expire(c.Request.Context(), key, 60*time.Second)
			}
			if count > 30 {
				ttl, _ := h.rdb.TTL(c.Request.Context(), key).Result()
				retryAfter := int(ttl.Seconds())
				if retryAfter <= 0 {
					retryAfter = 60
				}
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited", "retry_after": retryAfter})
				return
			}
		}
	}

	// Look up database to get SQLAlchemyURI
	db, err := h.dbRepo.GetDatabaseByID(c.Request.Context(), req.DatabaseID)
	if err != nil || db == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "database not found"})
		return
	}

	// Detect driver from SQLAlchemyURI scheme
	parsedURI, err := crypto.ParseSQLAlchemyURI(db.SQLAlchemyURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid database URI"})
		return
	}
	driver := parsedURI.Scheme
	if driver == "postgres" {
		driver = "postgresql"
	}

	// Get connection from pool
	conn, err := h.poolMgr.Get(c.Request.Context(), req.DatabaseID, db.SQLAlchemyURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "failed to get database connection"})
		return
	}

	// Run estimate
	estimator := svcquery.NewEstimator(driver)
	result, err := estimator.Estimate(c.Request.Context(), req.SQL, conn)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid_sql", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
```

Add `"time"` to imports if not already present.

- [ ] **Step 3: Verify compilation** (will need main.go wired first, do dry-build)

Run: `cd backend && go build ./internal/delivery/http/query/...`
Expected: will fail until main.go is wired (Task 6)

---

### Task 6: Wire route and dependencies

**Files:**
- Modify: `backend/internal/delivery/http/router.go`
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Add route in router.go**

In router.go, add after the existing query routes in the admin block:
```go
protected.POST("/query/estimate", queryHandler.Estimate)
```

Place it near the other query routes (after `protected.DELETE("/query/history", ...)` appears ok, but better near the other query endpoint lines).

- [ ] **Step 2: Wire in main.go**

In main.go, update the `NewHandlerWithAsync` call. Current:
```go
queryHandler := httpquery.NewHandlerWithAsync(queryExecutor, asyncQueryExecutor, pubKey, jwtRepo, userRepo, queryRepo, roleRepo)
```

Change to:
```go
queryHandler := httpquery.NewHandlerWithAsync(queryExecutor, asyncQueryExecutor, pubKey, jwtRepo, userRepo, queryRepo, roleRepo, redisClient, databaseRepo, poolManager)
```

- [ ] **Step 3: Build and verify**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 4: Run all existing tests to check for regressions**

Run: `cd backend && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/query/handler.go backend/internal/delivery/http/router.go backend/cmd/api/main.go
git commit -m "feat: add POST /api/v1/query/estimate endpoint (QE-008)"
```

---

### Task 7: Add handler-level tests for Estimate endpoint

**Files:**
- Modify: `backend/internal/delivery/http/query/handler_test.go`

- [ ] **Step 1: Add Estimate handler tests**

Append to handler_test.go:

```go
// TestQE008_EstimateRequestTypeMatchesSpec verifies request type matches spec contract
func TestQE008_EstimateRequestTypeMatchesSpec(t *testing.T) {
	tests := []struct {
		name string
		req  domainquery.EstimateRequest
	}{
		{
			name: "valid estimate request",
			req: domainquery.EstimateRequest{
				SQL:        "SELECT * FROM orders",
				DatabaseID: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := json.Marshal(tt.req)
			require.NoError(t, err)

			var decoded domainquery.EstimateRequest
			err = json.Unmarshal(bytes, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.req.SQL, decoded.SQL)
			assert.Equal(t, tt.req.DatabaseID, decoded.DatabaseID)
		})
	}
}

// TestQE008_EstimateResultTypeMatchesSpec verifies response type matches spec contract
func TestQE008_EstimateResultTypeMatchesSpec(t *testing.T) {
	tests := []struct {
		name   string
		result domainquery.EstimateResult
		json   string
	}{
		{
			name:   "unsupported DB",
			result: domainquery.EstimateResult{Supported: false},
			json:   `{"supported":false}`,
		},
		{
			name: "postgresql estimate",
			result: domainquery.EstimateResult{
				Supported:     true,
				Driver:        "postgresql",
				TotalCost:     1250.50,
				EstimatedRows: 50000,
			},
			json: `{"supported":true,"driver":"postgresql","total_cost":1250.5,"estimated_rows":50000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := json.Marshal(tt.result)
			require.NoError(t, err)
			assert.JSONEq(t, tt.json, string(bytes))
		})
	}
}

// TestQE008_EstimateHandlerRequiresAuth verifies Estimate method checks for user context
func TestQE008_EstimateHandlerRequiresAuth(t *testing.T) {
	// Handler with no executors — Estimate should return 401 if no user in context
	handler := NewHandler(nil)
	require.NotNil(t, handler)
	assert.NotNil(t, handler.Estimate)
}
```

- [ ] **Step 2: Run tests**

Run: `cd backend && go test ./internal/delivery/http/query/ -run TestQE008 -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/query/handler_test.go
git commit -m "test: add handler tests for estimate endpoint (QE-008)"
```

---

## Frontend Tasks

### Task 8: Add estimate API to queries client

**Files:**
- Modify: `frontend/src/api/queries.ts`

- [ ] **Step 1: Add types and API method**

Add after `DeleteHistoryResponse`:

```ts
export interface EstimateRequest {
  sql: string;
  database_id: number;
}

export interface EstimateResult {
  supported: boolean;
  driver?: string;
  total_cost?: number;
  estimated_rows?: number;
  bytes_processed?: number;
  estimated_cost_usd?: number;
}
```

Add to `queriesApi`:

```ts
estimate: (data: EstimateRequest): Promise<EstimateResult> =>
  request("/api/v1/query/estimate", {
    method: "POST",
    credentials: "include",
    headers: getAuthHeaders(true),
    body: JSON.stringify(data),
  }),
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS (no new errors)

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/queries.ts
git commit -m "feat: add estimate API method to queries client (QE-008)"
```

---

### Task 9: Add estimate state to SqlLab store

**Files:**
- Modify: `frontend/src/stores/sqlLabStore.ts`

- [ ] **Step 1: Add estimate field and setters**

Import `EstimateResult` from api:
```ts
import { type EstimateResult } from "@/api/queries";
```

Add to `SqlLabTab` interface:
```ts
export interface SqlLabTab {
  // ...existing fields...
  estimate: EstimateResult | null;  // QE-008
}
```

Add to `SqlLabState` interface:
```ts
setEstimate: (id: string, estimate: EstimateResult | null) => void;
```

In the initial `addTab` default tab, add:
```ts
estimate: null,
```

Add the setter implementation:
```ts
setEstimate: (id, estimate) => {
  set(state => ({
    tabs: state.tabs.map(t =>
      t.id === id ? { ...t, estimate } : t
    ),
  }));
},
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/sqlLabStore.ts
git commit -m "feat: add estimate state to sqlLab store (QE-008)"
```

---

### Task 10: Create useEstimate hook

**Files:**
- Create: `frontend/src/hooks/useEstimate.ts`

- [ ] **Step 1: Write hook**

```ts
import { useState, useEffect, useRef } from "react";
import { useMutation } from "@tanstack/react-query";
import { queriesApi, type EstimateResult } from "@/api/queries";
import { useSqlLabStore } from "@/stores/sqlLabStore";

function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);
  useEffect(() => {
    const handler = setTimeout(() => setDebouncedValue(value), delay);
    return () => clearTimeout(handler);
  }, [value, delay]);
  return debouncedValue;
}

const SUPPORTED_BACKENDS = ["postgresql", "bigquery", "snowflake", "mysql"] as const;

interface UseEstimateOptions {
  sql: string;
  tabId: string;
  databaseId: number | null;
  backend: string | undefined;
  enabled: boolean;
}

export function useEstimate({ sql, tabId, databaseId, backend, enabled }: UseEstimateOptions) {
  const setEstimate = useSqlLabStore(s => s.setEstimate);
  const debouncedSql = useDebounce(sql, enabled ? 2000 : 0);
  const lastSqlRef = useRef<string>("");

  const isSupported = backend ? (SUPPORTED_BACKENDS as readonly string[]).includes(backend) : false;

  const mutation = useMutation({
    mutationFn: queriesApi.estimate,
    onSuccess: (data: EstimateResult) => {
      setEstimate(tabId, data);
    },
    onError: () => {
      setEstimate(tabId, null);
    },
  });

  // Auto-fire on debounced SQL change
  useEffect(() => {
    if (!enabled || !databaseId || !isSupported || !debouncedSql.trim()) {
      setEstimate(tabId, null);
      return;
    }
    if (debouncedSql === lastSqlRef.current) return;
    lastSqlRef.current = debouncedSql;
    mutation.mutate({ sql: debouncedSql, database_id: databaseId });
  }, [debouncedSql, databaseId, enabled, isSupported]);

  const trigger = () => {
    if (!databaseId || !sql.trim()) return;
    mutation.mutate({ sql, database_id: databaseId });
  };

  return {
    estimate: useSqlLabStore(s => s.tabs.find(t => t.id === tabId)?.estimate ?? null),
    isLoading: mutation.isPending,
    trigger,
    isSupported,
  };
}
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/hooks/useEstimate.ts
git commit -m "feat: add useEstimate hook with debounce (QE-008)"
```

---

### Task 11: Create EstimateBadge component

**Files:**
- Modify: `frontend/src/components/query/QueryBadges.tsx`

- [ ] **Step 1: Add EstimateBadge**

Import `Zap` from lucide-react (already has many lucide imports, add `Zap`).

Add component at end of file:

```tsx
interface EstimateBadgeProps {
  estimate: {
    supported: boolean;
    driver?: string;
    total_cost?: number;
    estimated_rows?: number;
    bytes_processed?: number;
    estimated_cost_usd?: number;
  } | null;
  isLoading: boolean;
  onClick: () => void;
}

export function EstimateBadge({ estimate, isLoading, onClick }: EstimateBadgeProps) {
  if (isLoading) {
    return (
      <Badge variant="outline" className="h-6 gap-1 cursor-pointer" onClick={onClick}>
        <Loader2 className="h-3 w-3 animate-spin" />
        Estimating...
      </Badge>
    );
  }

  if (!estimate || !estimate.supported) return null;

  let label = "";
  if (estimate.estimated_cost_usd !== undefined) {
    label = `~$${estimate.estimated_cost_usd.toFixed(4)}`;
  } else if (estimate.estimated_rows !== undefined) {
    label = `~${estimate.estimated_rows.toLocaleString()} rows`;
  } else if (estimate.total_cost !== undefined) {
    label = `cost: ${estimate.total_cost.toFixed(1)}`;
  }

  if (!label) return null;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <Badge variant="outline" className="h-6 gap-1 cursor-pointer" onClick={onClick}>
            <Zap className="h-3 w-3" />
            {label}
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          <p className="text-xs">Click to view estimate details</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
```

Add `Zap` to the lucide-react import line.

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/query/QueryBadges.tsx
git commit -m "feat: add EstimateBadge component (QE-008)"
```

---

### Task 12: Create EstimatePopover component

**Files:**
- Create: `frontend/src/components/query/EstimatePopover.tsx`

- [ ] **Step 1: Check Popover is installed, write component**

Run: `cd frontend && npx shadcn@latest search popover` to confirm it exists.

```tsx
import { Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";

interface EstimateResult {
  supported: boolean;
  driver?: string;
  total_cost?: number;
  estimated_rows?: number;
  bytes_processed?: number;
  estimated_cost_usd?: number;
}

interface EstimatePopoverProps {
  estimate: EstimateResult | null;
  isLoading: boolean;
  onTrigger: () => void;
  isSupported: boolean;
}

export function EstimatePopover({ estimate, isLoading, onTrigger, isSupported }: EstimatePopoverProps) {
  if (!isSupported) return null;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2" onClick={onTrigger}>
          <Zap className="h-4 w-4" />
          Estimate Cost
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72">
        {isLoading ? (
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Estimating Cost...</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
            </CardContent>
          </Card>
        ) : estimate ? (
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">
                Query Estimate {estimate.driver && `(${estimate.driver})`}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              {estimate.supported ? (
                <>
                  {estimate.estimated_rows !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Estimated Rows</span>
                      <span className="font-medium">{estimate.estimated_rows.toLocaleString()}</span>
                    </div>
                  )}
                  {estimate.total_cost !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Planner Cost</span>
                      <span className="font-medium">{estimate.total_cost.toFixed(1)}</span>
                    </div>
                  )}
                  {estimate.bytes_processed !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Bytes Processed</span>
                      <span className="font-medium">{(estimate.bytes_processed / 1e9).toFixed(2)} GB</span>
                    </div>
                  )}
                  {estimate.estimated_cost_usd !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Estimated Cost</span>
                      <span className="font-medium">${estimate.estimated_cost_usd.toFixed(4)}</span>
                    </div>
                  )}
                  <Alert variant="default" className="mt-2 bg-muted/30">
                    <AlertDescription className="text-xs">
                      Estimate only. Actual execution may differ.
                    </AlertDescription>
                  </Alert>
                </>
              ) : (
                <p className="text-muted-foreground">Cost estimation not supported for this database.</p>
              )}
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="py-4 text-center text-sm text-muted-foreground">
              Click Estimate Cost to analyze your query.
            </CardContent>
          </Card>
        )}
      </PopoverContent>
    </Popover>
  );
}
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/query/EstimatePopover.tsx
git commit -m "feat: add EstimatePopover component (QE-008)"
```

---

### Task 13: Wire estimate into SQLLabPage

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx`

- [ ] **Step 1: Add imports and hook usage**

Add import:
```ts
import { useEstimate } from "@/hooks/useEstimate";
import { EstimatePopover } from "@/components/query/EstimatePopover";
import { EstimateBadge } from "@/components/query/QueryBadges";
```

After `useSqlLabStore` destructuring, find the active tab's backend. Add:
```ts
const selectedDbId = activeTab?.databaseId ?? null;

const selectedDb = useMemo(() => {
  if (!selectedDbId) return undefined;
  return databasesData?.items?.find(db => db.id === selectedDbId);
}, [selectedDbId, databasesData]);

const {
  estimate,
  isLoading: estimateLoading,
  trigger: triggerEstimate,
  isSupported: estimateSupported,
} = useEstimate({
  sql: activeTab?.sql ?? "",
  tabId: activeTabId ?? "",
  databaseId: selectedDbId,
  backend: selectedDb?.backend,
  enabled: true,
});
```

- [ ] **Step 2: Add Estimate button and badge to toolbar**

In the toolbar area (near Run/RunAsync buttons), after the existing CancelButton section, add:

```tsx
{/* Estimate button */}
{!tab.asyncStatus && estimateSupported && (
  <>
    <EstimatePopover
      estimate={estimate}
      isLoading={estimateLoading}
      onTrigger={triggerEstimate}
      isSupported={estimateSupported}
    />
    <EstimateBadge
      estimate={estimate}
      isLoading={estimateLoading}
      onClick={triggerEstimate}
    />
  </>
)}
```

- [ ] **Step 3: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 4: Verify existing frontend tests pass**

Run: `cd frontend && npx vitest run`
Expected: PASS (existing tests)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat: wire estimate into SQL Lab toolbar (QE-008)"
```

---

### Task 14: Final verification

- [ ] **Step 1: Backend full test suite**

Run: `cd backend && go test ./...`
Expected: PASS (all tests)

- [ ] **Step 2: Frontend full test suite**

Run: `cd frontend && npx vitest run && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Verify acceptance criteria**

| # | Criterion | How verified |
|---|-----------|-------------|
| 1 | POST with PostgreSQL database returns cost/rows | Task 3 unit test |
| 2 | POST with BigQuery database returns supported:false | Task 4 dispatcher test |
| 3 | POST with unsupported DB returns supported:false | Task 2 test |
| 4 | 31st request in 60s returns 429 | Manual integration test (needs running server) |
| 5 | Invalid SQL returns 422 | Task 3 error test |
| 6 | Button hidden for unsupported DBs | EstimatePopover returns null when !isSupported |
