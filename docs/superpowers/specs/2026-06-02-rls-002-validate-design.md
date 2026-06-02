# RLS-002: RLS Clause Validation & Live Preview

**Date:** 2026-06-02
**Status:** Draft
**Priority:** P0
**Phase:** Phase 2 - Core
**API Route:** `POST /api/v1/rls/validate`
**Frontend Route:** `/security/rls` (inline inside create/edit Dialog)

---

## 1. Overview

Two-phase clause validation callable independently from save, used by the admin
form for real-time feedback before a filter is committed.

- **Phase 1 — Syntax (always executed):** `sqlparser.ParseExpr(clause)` — validates
  the clause is a syntactically valid SQL boolean expression. No DB connection needed.
- **Phase 2 — Runtime (optional):** Renders template vars, builds probe SQL
  `SELECT 1 FROM schema.table WHERE (clause) LIMIT 0`, executes via connection pool
  with 5s timeout. `LIMIT 0` ensures zero rows returned — no data exposure.

---

## 2. Backend

### 2.1 Route

- `POST /api/v1/rls/validate` — JWT-protected, **not** Admin-only.
- Router placement: under `protected` group (JWT middleware), **outside** the `admin/rls` sub-group.

### 2.2 Request / Response

```go
// domain/auth
type ValidateRequest struct {
    Clause       string `json:"clause" binding:"required"`
    DatabaseID   *uint  `json:"database_id"`       // nil → Phase 1 only
    TableName    string `json:"table_name"`
    Schema       string `json:"schema"`
    TestUserID   *int   `json:"test_user_id"`      // nil → Phase 1 only
    TestUsername string `json:"test_username"`
}

type ValidateResult struct {
    IsValid        bool   `json:"is_valid"`
    Phase          string `json:"phase"`                       // "syntax" | "runtime"
    RenderedClause string `json:"rendered_clause,omitempty"`
    Error          string `json:"error,omitempty"`
    ErrorPosition  *int   `json:"error_position,omitempty"`
}
```

### 2.3 Service Changes

**`RLSService`** — new deps via constructor:

```go
type RLSService struct {
    repo        domain.RLSFilterRepository
    rdb         *redis.Client
    dbRepo      domain.DatabaseRepository         // DB fetch URI
    poolManager DatabaseConnectionPool            // DB connection pool
}
```

**New method:**

```go
func (s *RLSService) Validate(ctx context.Context, uc domain.UserContext, req domain.ValidateRequest) (domain.ValidateResult, int, error)
// Returns (result, httpStatus, error)
```

### 2.4 Request Flow

```
1. Rate Limit
   key = "rls:rate:validate:" + strconv.Itoa(uc.UserID)
   cnt, _ := s.rdb.Incr(ctx, key).Result()
   if cnt == 1: s.rdb.Expire(ctx, key, 60*time.Second)
   if cnt > 60: return 429 {error:"Rate limit exceeded"}

2. Phase 1 — Syntax (always)
   _, err := sqlparser.ParseExpr(req.Clause)
   if err != nil:
       pos := extractPosition(err)
       return 200 {is_valid:false, phase:"syntax", error:err.Error(), error_position:pos}

3. Phase 2 gate — database_id + test_user_id + table_name required
   if req.DatabaseID == nil || req.TestUserID == nil || req.TableName == "":
       return 200 {is_valid:true, phase:"syntax", rendered_clause:req.Clause}

4. Render template vars
   rendered := strings.NewReplacer(
       "{{current_user_id}}", strconv.Itoa(*req.TestUserID),
       "{{current_username}}", req.TestUsername,
   ).Replace(req.Clause)

5. Build probe SQL (pgx Identifier for quoting — project uses pgx driver)
   probeSQL := fmt.Sprintf(
       "SELECT 1 FROM %s.%s WHERE (%s) LIMIT 0",
       pgx.Identifier{req.Schema}.Sanitize(),
       pgx.Identifier{req.TableName}.Sanitize(),
       rendered,
   )

6. Get DB connection
   db, _ := s.dbRepo.GetDatabaseByID(ctx, *req.DatabaseID)
   decryptedURI := crypto.DecryptSQLAlchemyURIPassword(db.SQLAlchemyURI, encryptionKey)
   conn, err := s.poolManager.Get(ctx, *req.DatabaseID, decryptedURI)
   if err != nil: return 500 {error:"Connection pool unavailable"}
   ctx5s, cancel := context.WithTimeout(ctx, 5*time.Second)
   defer cancel()
   _, err = conn.ExecContext(ctx5s, probeSQL)

7. Return
   if err != nil: return 200 {is_valid:false, phase:"runtime", error:err.Error()}
   return 200 {is_valid:true, phase:"runtime", rendered_clause:rendered}
```

### 2.5 extractPosition

```go
func extractPosition(err error) *int {
    re := regexp.MustCompile(`position (\d+)`)
    matches := re.FindStringSubmatch(err.Error())
    if len(matches) >= 2 {
        pos, _ := strconv.Atoi(matches[1])
        return &pos
    }
    return nil
}
```

### 2.6 Security

| Measure | What it prevents |
|---------|-----------------|
| `pgx.Identifier.Sanitize()` on schema + table_name | Table reference injection in probe SQL |
| `strconv.Itoa(req.TestUserID)` — typed int | `{{current_user_id}}` never accepts arbitrary string |
| `LIMIT 0` on probe | Zero rows returned — no data exposure |
| Rate limit 60 req/min per user | Timing-oracle for row existence / brute force |
| Gin `ShouldBindJSON` with `*int` | `test_user_id:"42 OR 1=1"` → 400 parse error |

### 2.7 Error Responses

| Status | Condition |
|--------|-----------|
| 200 | Any validation result (valid or invalid in body) |
| 400 | Malformed request body |
| 429 | Rate limit exceeded (60/min per user) |
| 500 | Connection pool unavailable for runtime phase |

### 2.8 Acceptance Criteria

1. `POST {clause:"org_id = {{current_user_id}}", test_user_id:42, test_username:"alice"}` → 200 `{is_valid:true, phase:"syntax", rendered_clause:"org_id = 42"}` (no DB fields → syntax-only)
2. `POST {clause:"org_id = AND"}` → 200 `{is_valid:false, phase:"syntax", error:"syntax error near 'AND'", error_position:10}`
3. `POST {clause:"nonexistent_col = 1", database_id:1, table_name:"orders", schema:"public", test_user_id:42}` → 200 `{is_valid:false, phase:"runtime", error:"column \"nonexistent_col\" does not exist"}`
4. `POST {clause:"org_id = {{current_user_id}}", database_id:1, table_name:"orders", schema:"public", test_user_id:42}` → probe on real DB → 200 `{is_valid:true, phase:"runtime", rendered_clause:"org_id = 42"}`
5. `POST {test_user_id:"42 OR 1=1", ...}` → 400 (Go `*int` parse failure)
6. 61st call within 60s → 429 `{error:"Rate limit exceeded"}`

---

## 3. Frontend

### 3.1 Location

No new route. Inline inside the existing create/edit `Dialog` on `/security/rls`,
attached to the clause `Textarea` FormField.

### 3.2 Components (all shadcn/ui)

| Component | Details |
|-----------|---------|
| `Textarea` (clause) | `font-mono text-sm min-h-[120px]` — border reflects validation: default gray, valid `ring-2 ring-green-500`, invalid `ring-2 ring-destructive` |
| `FormMessage` | Shows syntax error text below Textarea |
| `Button` ("Validate Clause") | ShieldCheck icon, `variant="outline" size="sm"`, right-aligned below Textarea |
| `Badge` (inline status) | Next to Validate btn: "Syntax OK" (green) / "Runtime OK" (green) / "Syntax Error" (red) / "Runtime Error" (red) — only after validation attempted |
| `Skeleton` (1-line, 80px) | Replaces badge during mutation |
| `Alert` (success) | Green border, ShieldCheck: "Clause is valid · Rendered as: org_id = 42" |
| `Alert` (error) | Destructive, ShieldOff: phase label + error message |
| `Select` ("Test as user") | Populated via `GET /api/v1/admin/users?page_size=50`. Placeholder "Select user to test template vars..." |
| `Select` ("Test against table") | Populated from selected datasets. Placeholder "Select table for runtime probe..." |
| `Tooltip` (on Validate btn) | "Select a test user and target table to enable runtime validation." (when disabled) |

### 3.3 State

```typescript
// Local state
const [validationResult, setValidationResult] = useState<ValidateResult | null>(null);
const [testUserID, setTestUserID] = useState<number | null>(null);
const [testTableName, setTestTableName] = useState<string>("");
const [testSchema, setTestSchema] = useState<string>("");

// Mutation
const validateMutation = useMutation({
    mutationFn: (body: ValidateRequest) =>
        fetch("/api/v1/rls/validate", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
        }).then((r) => r.json()),
    onSuccess: (r) => setValidationResult(r),
});

// Debounced auto-validate (Phase 1 — syntax only)
const debouncedClause = useDebounce(clauseValue, 1500);
useEffect(() => {
    if (debouncedClause) {
        setValidationResult(null);
        validateMutation.mutate({ clause: debouncedClause });
    }
}, [debouncedClause]);
```

### 3.4 UX

- **Auto-validate**: Keystroke → 1500ms debounce → `POST {clause}` → border + FormMessage
- **Validate Clause button**: Manual click → Phase 2. Shows `Loader2 animate-spin` + disabled
- **Template chips**: `{{current_user_id}}` / `{{current_username}}` badges above Textarea → click inserts → triggers debounced re-validation
- **Success Alert**: Shows `rendered_clause` with substituted values. Fades in on success
- **Error Alert**: Phase label + message. Fades out on clause change
- **Clause change**: `validationResult=null`, border resets, FormMessage clears, Alert fades (300ms opacity)

### 3.5 Accessibility

- Textarea: `aria-label="SQL WHERE clause"`, `aria-invalid={validationResult && !validationResult.is_valid}`, `aria-describedby="clause-validation-message"`
- Validation Alert: `role="alert"`
- Validate button: `aria-label="Validate SQL clause syntax and runtime"`

### 3.6 API Types

```typescript
interface ValidateRequest {
    clause: string;
    database_id?: number;
    table_name?: string;
    schema?: string;
    test_user_id?: number;
    test_username?: string;
}
interface ValidateResult {
    is_valid: boolean;
    phase: "syntax" | "runtime";
    rendered_clause?: string;
    error?: string;
    error_position?: number;
}
```

---

## 4. Files Changed

### Backend

| File | Change |
|------|--------|
| `backend/internal/domain/auth/entity.go` | Add `ValidateRequest`, `ValidateResult` structs |
| `backend/internal/app/auth/rls_service.go` | Add `Validate()` method; inject `*redis.Client` + `DatabaseRepository` + `DatabaseConnectionPool` |
| `backend/internal/delivery/http/rls/handler.go` | Add `Validate` handler, extend `service` interface |
| `backend/internal/delivery/http/router.go` | Add `POST /api/v1/rls/validate` to protected group |

### Frontend

| File | Change |
|------|--------|
| `frontend/src/api/rlsFilters.ts` | Add `ValidateRequest`/`ValidateResult` types + `validateClause()` API method |
| `frontend/src/pages/security/RLSFiltersPage.tsx` | Add validation state, `useMutation`, `useDebounce`, test user Select, test table Select, Validate button, result Alert/Badge |

### Deps

- `github.com/jackc/pgx/v5` — already in `go.mod`. Use `pgx.Identifier{}.Sanitize()` for identifier quoting.

---

## 5. Router Integration

```go
// protected group (JWT required, any authenticated user)
protected.POST("/rls/validate", rlsHandler.Validate)

// existing admin group (Admin role required)
admin.GET("/rls", rlsHandler.List)
admin.POST("/rls", rlsHandler.Create)
```

Produces: `POST /api/v1/rls/validate` — matches spec.
