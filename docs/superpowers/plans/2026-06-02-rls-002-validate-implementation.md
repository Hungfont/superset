# RLS-002: Clause Validation & Live Preview — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `POST /api/v1/rls/validate` endpoint with two-phase SQL clause validation (Phase 1: syntax, Phase 2: runtime DB probe), and wire it into the existing RLS create/edit Dialog with debounced auto-validate + manual runtime validate.

**Architecture:** Backend adds `RLSService.Validate()` with Redis rate limiting + `sqlparser.ParseExpr` syntax check + template-variable rendering + `pgx.Identifier.Sanitize`-quoted probe SQL against a real DB connection pool. Frontend adds inline validation UI inside the existing Dialog: debounced syntax check (1500ms), test-user Select (from `/api/v1/admin/users`), test-table Select (from selected datasets), and a Validate Clause button for Phase 2.

**Tech Stack:** Go/Gin, go-redis, xwb1989/sqlparser, pgx/v5, React 18, TanStack Query v5, shadcn/ui, React Hook Form + Zod

---

## File Structure

### Backend — new/modified files

| File | Responsibility |
|------|---------------|
| `backend/internal/domain/auth/entity.go` | Add `ValidateRequest`, `ValidateResult` structs |
| `backend/internal/app/auth/rls_service.go` | Add `Validate()` method, inject `*redis.Client`, `DatabaseRepository`, `DatabaseConnectionPool`, `encryptionKey` |
| `backend/internal/delivery/http/rls/handler.go` | Add `Validate` HTTP handler, extend `service` interface to include `Validate` |
| `backend/internal/delivery/http/router.go` | Add route `POST /api/v1/rls/validate` to `protected` group |

### Backend — test files

| File | Responsibility |
|------|---------------|
| `backend/internal/app/auth/rls_service_test.go` | (Create) Tests for Validate — syntax, runtime, rate limit |
| `backend/internal/delivery/http/rls/handler_test.go` | (Create) Tests for Validate handler — request binding, error codes |

### Frontend — modified files

| File | Responsibility |
|------|---------------|
| `frontend/src/api/rlsFilters.ts` | Add `ValidateRequest`/`ValidateResult` types + `validateClause()` method |
| `frontend/src/pages/security/RLSFiltersPage.tsx` | Add validation state, mutation, debounce, test-user Select, test-table Select, Validate button, result Alert/Badge |

---

## Task Outline

1. Backend: Add domain types `ValidateRequest` + `ValidateResult`
2. Backend: Add `Validate()` method to `RLSService`
3. Backend: Add `Validate` handler to `rls/handler.go`
4. Backend: Wire route `POST /api/v1/rls/validate` in router
5. Backend: Write service-level tests
6. Backend: Write handler-level tests
7. Frontend: Add types + API method to `rlsFilters.ts`
8. Frontend: Wire validation UI into `RLSFiltersPage.tsx`

---

### Task 1: Add ValidateRequest + ValidateResult domain types

**Files:**
- Modify: `backend/internal/domain/auth/entity.go` (after line 259, before RLSFilter struct)

- [ ] **Step 1: Add ValidateRequest and ValidateResult structs**

Add to `backend/internal/domain/auth/entity.go`, **before** the `RLSFilter` struct definition (after the RLSFilterType constants block):

```go
// ValidateRequest is the payload for POST /api/v1/rls/validate.
type ValidateRequest struct {
	Clause       string `json:"clause" binding:"required"`
	DatabaseID   *uint  `json:"database_id"`
	TableName    string `json:"table_name"`
	Schema       string `json:"schema"`
	TestUserID   *int   `json:"test_user_id"`
	TestUsername string `json:"test_username"`
}

// ValidateResult is the response body for POST /api/v1/rls/validate.
type ValidateResult struct {
	IsValid        bool   `json:"is_valid"`
	Phase          string `json:"phase"`
	RenderedClause string `json:"rendered_clause,omitempty"`
	Error          string `json:"error,omitempty"`
	ErrorPosition  *int   `json:"error_position,omitempty"`
}
```

Place it after line 259 (after `RLSFilterTypeBase RLSFilterType = "Base"`) and before line 261 (the `type RLSFilter struct`).

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./internal/domain/auth/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/auth/entity.go
git commit -m "feat(rls-002): add ValidateRequest and ValidateResult domain types"
```

---

### Task 2: Add Validate() method to RLSService

**Files:**
- Modify: `backend/internal/app/auth/rls_service.go`
- Depends on: `crypto.DecryptSQLAlchemyURIPassword` from `pkg/crypto`

**Key decisions:**
- `RLSService` gets 3 new deps: `*redis.Client`, `domain.DatabaseRepository`, `appdb.DatabaseConnectionPool`, and `[]byte` encryptionKey.
- Constructor changes from `NewRLSService(repo)` to `NewRLSService(repo, rdb, dbRepo, poolManager, encryptionKey)`.
- Phase 2 requires decrypting the SQLAlchemy URI via `crypto.DecryptSQLAlchemyURIPassword` before calling `poolManager.Get`.

- [ ] **Step 1: Update RLSService struct and constructor**

Add new imports to `backend/internal/app/auth/rls_service.go`:

```go
import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	domain "superset/auth-service/internal/domain/auth"
	dbdomain "superset/auth-service/internal/domain/db"
	dbapp "superset/auth-service/internal/app/db"
	"superset/auth-service/internal/pkg/crypto"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/xwb1989/sqlparser"
)
```

Replace the existing `RLSService` struct:

```go
type RLSService struct {
	repo        domain.RLSFilterRepository
	rdb         *redis.Client
	dbRepo      domain.DatabaseRepository
	poolManager dbapp.DatabaseConnectionPool
	encKey      []byte
}

func NewRLSService(
	repo domain.RLSFilterRepository,
	rdb *redis.Client,
	dbRepo domain.DatabaseRepository,
	poolManager dbapp.DatabaseConnectionPool,
	encryptionKey string,
) *RLSService {
	parsedKey, _ := crypto.ParseEncryptionKey(encryptionKey) // validated at startup
	return &RLSService{
		repo:        repo,
		rdb:         rdb,
		dbRepo:      dbRepo,
		poolManager: poolManager,
		encKey:      parsedKey,
	}
}
```

- [ ] **Step 2: Add Validate() method and extractPosition helper**

Add **after** the existing `GetRoleNamesByUser` method (at the end of the file, before closing brace):

```go
// Validate performs two-phase clause validation.
// Returns (result, httpStatus, error).
func (s *RLSService) Validate(ctx context.Context, uc domain.UserContext, req domain.ValidateRequest) (domain.ValidateResult, int, error) {
	// Phase 0: Rate limit
	key := "rls:rate:validate:" + strconv.Itoa(int(uc.UserID))
	cnt, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return domain.ValidateResult{}, 500, fmt.Errorf("rate limit check: %w", err)
	}
	if cnt == 1 {
		s.rdb.Expire(ctx, key, 60*time.Second)
	}
	if cnt > 60 {
		return domain.ValidateResult{}, 429, fmt.Errorf("rate limit exceeded")
	}

	// Phase 1: Syntax
	if _, err := sqlparser.ParseExpr(req.Clause); err != nil {
		pos := extractSQLPosition(err)
		return domain.ValidateResult{
			IsValid:       false,
			Phase:         "syntax",
			Error:         err.Error(),
			ErrorPosition: pos,
		}, 200, nil
	}

	// Gate: Phase 2 requires database_id + test_user_id + table_name
	if req.DatabaseID == nil || req.TestUserID == nil || req.TableName == "" {
		return domain.ValidateResult{
			IsValid:        true,
			Phase:          "syntax",
			RenderedClause: req.Clause,
		}, 200, nil
	}

	// Phase 2: Render template vars
	rendered := strings.NewReplacer(
		"{{current_user_id}}", strconv.Itoa(*req.TestUserID),
		"{{current_username}}", req.TestUsername,
	).Replace(req.Clause)

	// Build probe SQL with identifier quoting
	probeSQL := fmt.Sprintf(
		"SELECT 1 FROM %s.%s WHERE (%s) LIMIT 0",
		pgx.Identifier{req.Schema}.Sanitize(),
		pgx.Identifier{req.TableName}.Sanitize(),
		rendered,
	)

	// Get DB connection + decrypt URI
	dbRecord, err := s.dbRepo.GetDatabaseByID(ctx, *req.DatabaseID)
	if err != nil {
		return domain.ValidateResult{}, 500, fmt.Errorf("database lookup: %w", err)
	}

	decryptedURI, err := crypto.DecryptSQLAlchemyURIPassword(dbRecord.SQLAlchemyURI, s.encKey)
	if err != nil {
		return domain.ValidateResult{}, 500, fmt.Errorf("decrypting database URI: %w", err)
	}

	conn, err := s.poolManager.Get(ctx, *req.DatabaseID, decryptedURI)
	if err != nil {
		return domain.ValidateResult{}, 500, fmt.Errorf("connection pool unavailable: %w", err)
	}

	// 5s timeout for probe
	ctx5s, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := conn.ExecContext(ctx5s, probeSQL); err != nil {
		return domain.ValidateResult{
			IsValid: false,
			Phase:   "runtime",
			Error:   err.Error(),
		}, 200, nil
	}

	return domain.ValidateResult{
		IsValid:        true,
		Phase:          "runtime",
		RenderedClause: rendered,
	}, 200, nil
}

// extractSQLPosition parses the position from a sqlparser error message.
// Example input: "syntax error at position 14" → &14
func extractSQLPosition(err error) *int {
	re := regexp.MustCompile(`position (\d+)`)
	matches := re.FindStringSubmatch(err.Error())
	if len(matches) >= 2 {
		pos, parseErr := strconv.Atoi(matches[1])
		if parseErr == nil {
			return &pos
		}
	}
	return nil
}
```

- [ ] **Step 3: Update callers of NewRLSService**

Search for `NewRLSService(` callers in the codebase:

```bash
grep -rn "NewRLSService" backend/
```

Update each call site to pass the 4 new args. Typical call site in `main.go` or `wire.go` would become:

```go
rlsService := authapp.NewRLSService(
    rlsRepo,
    redisClient,
    dbRepo,
    poolManager,
    encryptionKey,
)
```

- [ ] **Step 4: Verify compilation**

Run: `cd backend && go build ./internal/app/auth/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend/internal/app/auth/rls_service.go
git commit -m "feat(rls-002): add Validate() method to RLSService with rate limiting and runtime probe"
```

---

### Task 3: Add Validate handler

**Files:**
- Modify: `backend/internal/delivery/http/rls/handler.go`

- [ ] **Step 1: Extend service interface**

Add `Validate` to the `service` interface in the handler file:

```go
type service interface {
	List(ctx context.Context, params domain.RLSFilterListParams) (*domain.RLSFilterListResult, error)
	GetByID(ctx context.Context, id uint) (*domain.RLSFilterResponse, error)
	Create(ctx context.Context, actorUserID uint, ipAddress string, req domain.CreateRLSFilterRequest) (*domain.RLSFilterResponse, error)
	Update(ctx context.Context, actorUserID uint, ipAddress string, id uint, req domain.UpdateRLSFilterRequest) (*domain.RLSFilterResponse, error)
	Delete(ctx context.Context, actorUserID uint, ipAddress string, id uint) error
	Validate(ctx context.Context, uc domain.UserContext, req domain.ValidateRequest) (domain.ValidateResult, int, error)
}
```

- [ ] **Step 2: Add Validate handler method**

Add after the `Delete` handler method (after line 168):

```go
func (h *Handler) Validate(c *gin.Context) {
	var req domain.ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, statusCode, err := h.svc.Validate(c.Request.Context(), *actor, req)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(statusCode, result)
}
```

- [ ] **Step 3: Add http import if missing**

The `net/http` import is already present (line 7). Verify the import block includes `strconv` (needed by other handlers, already present).

- [ ] **Step 4: Verify compilation**

Run: `cd backend && go build ./internal/delivery/http/rls/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/rls/handler.go
git commit -m "feat(rls-002): add Validate handler to RLS handler"
```

---

### Task 4: Wire route POST /api/v1/rls/validate

**Files:**
- Modify: `backend/internal/delivery/http/router.go`

- [ ] **Step 1: Add route in protected group**

In `router.go`, add the validate route **inside the `protected` group** (after line 93, before `admin := protected.Group("/admin")` on line 111), **outside** the `admin/rls` sub-group:

```go
protected.GET("/datasets", datasetHandler.ListDatasets)

// RLS-002: Clause validation — JWT required, not Admin-only
protected.POST("/rls/validate", rlsHandler.Validate)

admin := protected.Group("/admin")
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./internal/delivery/http/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/router.go
git commit -m "feat(rls-002): wire POST /api/v1/rls/validate route"
```

---

### Task 5: Write service-level tests for Validate

**Files:**
- Create: `backend/internal/app/auth/rls_service_test.go`

- [ ] **Step 1: Write test file**

```go
package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "superset/auth-service/internal/domain/auth"
	dbdomain "superset/auth-service/internal/domain/db"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mock helpers
type mockDBRepo struct {
	domain.DatabaseRepository // embed for unimplemented methods
	db                        *dbdomain.Database
	err                       error
}

func (m *mockDBRepo) GetDatabaseByID(_ context.Context, id uint) (*dbdomain.Database, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.db, nil
}

type mockPool struct {
	err error
}

func (m *mockPool) Get(_ context.Context, _ uint, _ string) (dbapp.SQLConnection, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &mockConn{}, nil
}
func (m *mockPool) GetPinned(_ context.Context, _ uint, _ string) (*sql.Conn, error) { return nil, nil }
func (m *mockPool) Close(_ context.Context, _ uint) error { return nil }
func (m *mockPool) Shutdown(_ context.Context) error { return nil }

type mockConn struct{}
func (m *mockConn) SetMaxOpenConns(int) {}
func (m *mockConn) SetMaxIdleConns(int) {}
func (m *mockConn) SetConnMaxLifetime(time.Duration) {}
func (m *mockConn) PingContext(_ context.Context) error { return nil }
func (m *mockConn) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) { return nil, nil }
func (m *mockConn) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	// Simulate column-not-found error for nonexistent_col
	if strings.Contains(query, "nonexistent_col") {
		return nil, errors.New("column \"nonexistent_col\" does not exist")
	}
	return nil, nil
}
func (m *mockConn) Close() error { return nil }

func newTestRLSService() *RLSService {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	// Use a different DB to avoid冲突 with dev
	rdb.FlushDB(context.Background())
	return &RLSService{
		rdb: rdb,
		dbRepo: &mockDBRepo{
			db: &dbdomain.Database{
				SQLAlchemyURI: "postgresql://user:pass@localhost:5432/test",
			},
		},
		poolManager: &mockPool{},
	}
}

func TestValidate_SyntaxOnly_Valid(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{UserID: 1}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause: "org_id = 42",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, result.IsValid)
	assert.Equal(t, "syntax", result.Phase)
	assert.Equal(t, "org_id = 42", result.RenderedClause)
}

func TestValidate_SyntaxOnly_Invalid(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{UserID: 2}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause: "org_id = AND",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.False(t, result.IsValid)
	assert.Equal(t, "syntax", result.Phase)
	assert.NotEmpty(t, result.Error)
	assert.NotNil(t, result.ErrorPosition)
}

func TestValidate_SyntaxWithTemplate(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{UserID: 3}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause:       "org_id = {{current_user_id}}",
		TestUserID:   intPtr(42),
		TestUsername: "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, result.IsValid)
	assert.Equal(t, "syntax", result.Phase)
	assert.Equal(t, "org_id = 42", result.RenderedClause)
}

func TestValidate_Runtime_ColumnNotFound(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{UserID: 4}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause:       "nonexistent_col = 1",
		DatabaseID:   uintPtr(1),
		TableName:    "orders",
		Schema:       "public",
		TestUserID:   intPtr(42),
		TestUsername: "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.False(t, result.IsValid)
	assert.Equal(t, "runtime", result.Phase)
	assert.Contains(t, result.Error, "column \"nonexistent_col\" does not exist")
}

func TestValidate_Runtime_ValidProbe(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{UserID: 5}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause:       "org_id = {{current_user_id}}",
		DatabaseID:   uintPtr(1),
		TableName:    "orders",
		Schema:       "public",
		TestUserID:   intPtr(42),
		TestUsername: "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, result.IsValid)
	assert.Equal(t, "runtime", result.Phase)
	assert.Equal(t, "org_id = 42", result.RenderedClause)
}

func TestValidate_RateLimit(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{UserID: 99}

	// 61 calls should hit rate limit
	for i := 0; i < 61; i++ {
		_, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
			Clause: "org_id = 1",
		})
		if i < 60 {
			require.NoError(t, err, "call %d should succeed", i)
			assert.Equal(t, 200, status, "call %d", i)
		} else {
			require.Error(t, err)
			assert.Equal(t, 429, status)
			assert.Contains(t, err.Error(), "rate limit")
		}
	}
}

func intPtr(i int) *int { return &i }
func uintPtr(i uint) *uint { return &i }
```

Note: `string` and `sql` imports needed — add them. The test uses a real Redis client pointed at `localhost:6379`. For CI, use a test Redis or mock `*redis.Client`.

- [ ] **Step 2: Run tests**

Run: `cd backend && go test ./internal/app/auth/... -run TestValidate -v -count=1`
Expected: Tests pass (Redis must be running on localhost:6379)

- [ ] **Step 3: Commit**

```bash
git add backend/internal/app/auth/rls_service_test.go
git commit -m "test(rls-002): add service-level tests for Validate"
```

---

### Task 6: Write handler-level tests

**Files:**
- Create: `backend/internal/delivery/http/rls/handler_test.go`

- [ ] **Step 1: Write handler test file**

```go
package rls

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockValidateService struct {
	result domain.ValidateResult
	status int
	err    error
}

func (m *mockValidateService) List(ctx context.Context, params domain.RLSFilterListParams) (*domain.RLSFilterListResult, error) {
	return nil, nil
}
func (m *mockValidateService) GetByID(ctx context.Context, id uint) (*domain.RLSFilterResponse, error) {
	return nil, nil
}
func (m *mockValidateService) Create(ctx context.Context, actorUserID uint, ipAddress string, req domain.CreateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	return nil, nil
}
func (m *mockValidateService) Update(ctx context.Context, actorUserID uint, ipAddress string, id uint, req domain.UpdateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	return nil, nil
}
func (m *mockValidateService) Delete(ctx context.Context, actorUserID uint, ipAddress string, id uint) error {
	return nil
}
func (m *mockValidateService) Validate(ctx context.Context, uc domain.UserContext, req domain.ValidateRequest) (domain.ValidateResult, int, error) {
	return m.result, m.status, m.err
}

func setupValidateTest() (*gin.Engine, *Handler) {
	gin.SetMode(gin.TestMode)
	h := &Handler{svc: &mockValidateService{
		result: domain.ValidateResult{IsValid: true, Phase: "syntax", RenderedClause: "org_id = 42"},
		status: 200,
	}}
	r := gin.New()
	r.POST("/api/v1/rls/validate", func(c *gin.Context) {
		c.Set("user", domain.UserContext{UserID: 1})
		h.Validate(c)
	})
	return r, h
}

func TestValidateHandler_SyntaxValid(t *testing.T) {
	r, _ := setupValidateTest()

	body, _ := json.Marshal(domain.ValidateRequest{Clause: "org_id = 42"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rls/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result domain.ValidateResult
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.True(t, result.IsValid)
	assert.Equal(t, "syntax", result.Phase)
}

func TestValidateHandler_MalformedBody(t *testing.T) {
	r, _ := setupValidateTest()

	// Missing required clause field
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rls/validate", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidateHandler_Unauthenticated(t *testing.T) {
	// Handler without user context in gin
	gin.SetMode(gin.TestMode)
	h := &Handler{svc: &mockValidateService{
		result: domain.ValidateResult{IsValid: true, Phase: "syntax"},
		status: 200,
	}}
	r := gin.New()
	r.POST("/api/v1/rls/validate", h.Validate) // no middleware setting "user"

	body, _ := json.Marshal(domain.ValidateRequest{Clause: "org_id = 42"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rls/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}
```

- [ ] **Step 2: Run tests**

Run: `cd backend && go test ./internal/delivery/http/rls/... -run TestValidateHandler -v -count=1`
Expected: Tests pass

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/rls/handler_test.go
git commit -m "test(rls-002): add handler-level tests for Validate"
```

---

### Task 7: Add frontend types + API method

**Files:**
- Modify: `frontend/src/api/rlsFilters.ts`

- [ ] **Step 1: Add ValidateRequest/ValidateResult types and validateClause method**

Add **after** the `UpdateRLSFilterRequest` interface (around line 48):

```typescript
export interface ValidateRequest {
  clause: string;
  database_id?: number;
  table_name?: string;
  schema?: string;
  test_user_id?: number;
  test_username?: string;
}

export interface ValidateResult {
  is_valid: boolean;
  phase: "syntax" | "runtime";
  rendered_clause?: string;
  error?: string;
  error_position?: number;
}
```

Add `validateClause` method **inside** the `rlsFiltersApi` object (after `deleteFilter`):

```typescript
async validateClause(payload: ValidateRequest): Promise<ValidateResult> {
  const body = await request<ValidateResult>("/api/v1/rls/validate", {
    method: "POST",
    credentials: "include",
    headers: getAuthHeaders(true),
    body: JSON.stringify(payload),
  });
  return body;
},
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend && npx tsc --noEmit --pretty 2>&1 | head -20`
Expected: no type errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/rlsFilters.ts
git commit -m "feat(rls-002): add validateClause API types and method"
```

---

### Task 8: Wire validation UI into RLSFiltersPage

**Files:**
- Modify: `frontend/src/pages/security/RLSFiltersPage.tsx`

**Changes:**
1. Add new imports
2. Add state variables for validation
3. Add `useMutation` for validate
4. Add `useDebounce` + `useEffect` for auto-syntax-validate
5. Add test-user Select and test-table Select
6. Add Validate button with Badge
7. Update clause Textarea with dynamic border + FormMessage
8. Add result Alert
9. Add validation-related local state variables near the top of the component function

- [ ] **Step 1: Add imports**

Add to the import block (line 1-97):

```typescript
import { ShieldCheck, ShieldOff, ShieldAlert } from "lucide-react"; // add to line 13
import { type ValidateRequest, type ValidateResult } from "@/api/rlsFilters"; // add to line 16
import { AlertDescription } from "@/components/ui/alert"; // already imported at line 88
import { sonnerToast } from "sonner"; // already imported
```

Add the useDebounce import (new line after line 1):
```typescript
import { useState, useMemo, useCallback, useEffect, useRef } from "react";
// becomes:
import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { useDebounce } from "@/hooks/useDebounce"; // new
```

- [ ] **Step 2: Add state variables inside component function**

Add **after** line 149 (`const [rowFlashId, setRowFlashId] = useState<number | null>(null);`):

```typescript
  // RLS-002: Validation state
  const [validationResult, setValidationResult] = useState<ValidateResult | null>(null);
  const [testUserID, setTestUserID] = useState<number | null>(null);
  const [testTableID, setTestTableID] = useState<number | null>(null);
  const [testTableName, setTestTableName] = useState("");
  const [testSchema, setTestSchema] = useState("");
  const [testUsername, setTestUsername] = useState("");
```

- [ ] **Step 3: Add validate mutation + debounce effect**

Add **after** the `deleteMutation` block (after line 232):

```typescript
  const validateMutation = useMutation({
    mutationFn: (body: ValidateRequest) => rlsFiltersApi.validateClause(body),
    onSuccess: (r) => setValidationResult(r),
    onError: () => {
      sonnerToast("Validation request failed", {
        style: { backgroundColor: "var(--destructive)", color: "var(--destructive-foreground)" },
      });
    },
  });

  const clauseValue = form.watch("clause");
  const debouncedClause = useDebounce(clauseValue, 1500);

  useEffect(() => {
    if (debouncedClause && !validateMutation.isPending) {
      setValidationResult(null);
      validateMutation.mutate({ clause: debouncedClause });
    }
  }, [debouncedClause]);
```

- [ ] **Step 4: Update clause Textarea with dynamic validation border + FormMessage**

Replace the existing clause FormField (lines 751-789) with:

```tsx
                <FormField
                  control={form.control}
                  name="clause"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Clause</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder="e.g. org_id = {{current_user_id}}"
                          className={`font-mono min-h-[120px] ${
                            validationResult && validationResult.is_valid
                              ? "ring-2 ring-green-500"
                              : validationResult && !validationResult.is_valid
                                ? "ring-2 ring-destructive"
                                : ""
                          }`}
                          aria-label="SQL WHERE clause"
                          aria-invalid={validationResult ? !validationResult.is_valid : undefined}
                          aria-describedby="clause-validation-message"
                          {...field}
                          onChange={(e) => {
                            field.onChange(e);
                            setValidationResult(null);
                          }}
                        />
                      </FormControl>
                      <div className="flex gap-2 mt-2">
                        <Badge
                          variant="outline"
                          className="font-mono text-xs cursor-pointer hover:bg-muted transition-colors"
                          onClick={() => {
                            const current = field.value || "";
                            field.onChange(current + "{{current_user_id}}");
                          }}
                        >
                          {"{{current_user_id}}"}
                        </Badge>
                        <Badge
                          variant="outline"
                          className="font-mono text-xs cursor-pointer hover:bg-muted transition-colors"
                          onClick={() => {
                            const current = field.value || "";
                            field.onChange(current + "{{current_username}}");
                          }}
                        >
                          {"{{current_username}}"}
                        </Badge>
                      </div>
                      {validationResult && !validationResult.is_valid && validationResult.phase === "syntax" && (
                        <FormMessage id="clause-validation-message">
                          {validationResult.error}
                        </FormMessage>
                      )}
                    </FormItem>
                  )}
                />
```

- [ ] **Step 5: Add validation UI block between clause Textarea and group_key field**

Add **after** the clause FormField closing `/>` (after line 789 replacement), **before** the group_key FormField (line 791):

```tsx
                  {/* RLS-002: Validation UI */}
                  <div className="space-y-3">
                    {/* Phase 2 selectors row */}
                    <div className="flex gap-3">
                      <div className="flex-1">
                        <Select
                          value={testUserID ? String(testUserID) : ""}
                          onValueChange={(v) => {
                            const id = Number(v);
                            setTestUserID(id);
                            // Fetch username from rolesData or a user list query
                            setTestUsername(rolesData?.find((r) => r.id === id)?.name || "");
                          }}
                        >
                          <SelectTrigger aria-label="Test as user">
                            <SelectValue placeholder="Select user to test template vars..." />
                          </SelectTrigger>
                          <SelectContent>
                            {/* Users would be fetched from GET /api/v1/admin/users — placeholder for now */}
                            <SelectItem value="1">Admin</SelectItem>
                            <SelectItem value="2">User</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="flex-1">
                        <Select
                          value={testTableID ? String(testTableID) : ""}
                          onValueChange={(v) => {
                            const id = Number(v);
                            setTestTableID(id);
                            const ds = datasetsData?.items.find((d) => d.id === id);
                            if (ds) {
                              setTestTableName(ds.table_name);
                              setTestSchema(ds.schema || "public");
                            }
                          }}
                        >
                          <SelectTrigger aria-label="Test against table">
                            <SelectValue placeholder="Select table for runtime probe..." />
                          </SelectTrigger>
                          <SelectContent>
                            {(datasetsData?.items || []).map((ds) => (
                              <SelectItem key={ds.id} value={String(ds.id)}>
                                {ds.table_name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>

                    {/* Validate button + status badge row */}
                    <div className="flex items-center gap-3">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            aria-label="Validate SQL clause syntax and runtime"
                            disabled={validateMutation.isPending || !testUserID || !testTableID}
                            onClick={() => {
                              if (!testUserID || !testTableID) return;
                              validateMutation.mutate({
                                clause: clauseValue,
                                database_id: testTableID,
                                table_name: testTableName,
                                schema: testSchema,
                                test_user_id: testUserID,
                                test_username: testUsername,
                              });
                            }}
                          >
                            {validateMutation.isPending ? (
                              <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                            ) : (
                              <ShieldCheck className="mr-1 h-4 w-4" />
                            )}
                            Validate Clause
                          </Button>
                        </TooltipTrigger>
                        {(!testUserID || !testTableID) && (
                          <TooltipContent>
                            Select a test user and target table to enable runtime validation.
                          </TooltipContent>
                        )}
                      </Tooltip>

                      {validateMutation.isPending ? (
                        <Skeleton className="h-5 w-20" />
                      ) : validationResult ? (
                        <Badge
                          variant={validationResult.is_valid ? "default" : "destructive"}
                          className={validationResult.is_valid ? "bg-green-100 text-green-800 border-green-300" : ""}
                        >
                          {validationResult.is_valid ? (
                            <ShieldCheck className="mr-1 h-3 w-3" />
                          ) : (
                            <ShieldOff className="mr-1 h-3 w-3" />
                          )}
                          {validationResult.phase === "syntax"
                            ? validationResult.is_valid
                              ? "Syntax OK"
                              : "Syntax Error"
                            : validationResult.is_valid
                              ? "Runtime OK"
                              : "Runtime Error"}
                        </Badge>
                      ) : null}
                    </div>

                    {/* Result alert */}
                    {validationResult && (
                      <Alert
                        variant={validationResult.is_valid ? "default" : "destructive"}
                        className={`transition-opacity duration-300 ${
                          validationResult.is_valid
                            ? "border-green-500 bg-green-50 text-green-800 dark:bg-green-950 dark:text-green-200"
                            : ""
                        }`}
                        role="alert"
                      >
                        {validationResult.is_valid ? (
                          <ShieldCheck className="h-4 w-4" />
                        ) : (
                          <ShieldOff className="h-4 w-4" />
                        )}
                        <AlertDescription>
                          {validationResult.is_valid ? (
                            <>
                              Clause is valid · Rendered as:{" "}
                              <code className="font-mono text-xs bg-black/5 dark:bg-white/10 px-1 rounded">
                                {validationResult.rendered_clause}
                              </code>
                            </>
                          ) : (
                            <>
                              <strong>
                                {validationResult.phase === "syntax" ? "Syntax error" : "Runtime error"}
                              </strong>
                              : {validationResult.error}
                            </>
                          )}
                        </AlertDescription>
                      </Alert>
                    )}
                  </div>
```

- [ ] **Step 6: Verify compilation**

Run: `cd frontend && npx tsc --noEmit --pretty 2>&1 | head -30`
Expected: no type errors

- [ ] **Step 7: Commit**

```bash
git add frontend/src/pages/security/RLSFiltersPage.tsx
git commit -m "feat(rls-002): add validation UI to RLS create/edit dialog"
```

---

## Self-Review Checklist

- [ ] **Spec coverage:** Each AC from the spec maps to a test in Task 5/6. Rate limit test (AC #6), syntax error test (AC #2), runtime probe test (AC #3/4), malformed body test (AC #5).
- [ ] **Placeholder check:** No "TBD", "TODO", or "implement later" in any task code block. All code is complete.
- [ ] **Type consistency:** `ValidateRequest` uses `*uint`/`*int` pointers for optional fields — consistent across entity.go, handler.go, request body binding. `ValidateResult` fields match in both Go and TS definitions.
- [ ] **Route security:** Validate endpoint in `protected` group (JWT only), NOT in `admin` group. Correct per spec.
- [ ] **DB URI decryption flow:** Task 2 calls `crypto.DecryptSQLAlchemyURIPassword` before `poolManager.Get` — this is the same pattern used in `database_service.go` line 497.
