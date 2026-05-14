# DBC Gap Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 15 gaps identified in the DBC-001 through DBC-007 gap analysis across 3 phases.

**Architecture:** Follows existing clean layers — domain (`domain/db`), app service (`app/db`), postgres repo (`repository/postgres`), Redis repo (`repository/redis`), HTTP handler (`delivery/http/db`). All changes are surgical additions to existing files. Package names: `app/db` = `package auth`, tests = `package auth_test`, handler = `package auth`, handler tests = `package auth_test`.

**Tech Stack:** Go, GORM, Gin, pgx (PostgreSQL), go-sql-driver/mysql, gosnowflake, go-redis

---

### Task 1: Domain Layer — New Errors and Types (Phase 1 prep)

**Files:**
- Modify: `backend/internal/pkg/autherrors/errors.go`
- Modify: `backend/internal/domain/db/errors.go`
- Modify: `backend/internal/domain/db/database.go`
- Modify: `backend/internal/domain/db/repository.go`

- [ ] **Step 1: Add `"fmt"` import to `database.go`**

In `backend/internal/domain/db/database.go`, change the import block from:
```go
import "time"
```
To:
```go
import (
	"fmt"
	"time"
)
```

- [ ] **Step 3: Add `ErrDatabaseHasRunningQueries`, `DatasetRef`, and `DatabaseInUseError`**

In `backend/internal/pkg/autherrors/errors.go`, after line 57 (`ErrUnknownDatabaseDriver`), add:

```go
	ErrDatabaseHasRunningQueries = errors.New("database has running queries")
```

After the database sentinel errors block (after line 57), add the structured error types in `backend/internal/domain/db/database.go` after `DatasetRef` doesn't exist yet. Add after `DatabaseColumn` (line 165):

```go
// DatasetRef is a lightweight reference to a dataset bound to a database.
type DatasetRef struct {
	ID        uint   `json:"id"`
	TableName string `json:"table_name"`
}

// DatabaseInUseError is returned when a database cannot be deleted because datasets depend on it.
type DatabaseInUseError struct {
	Datasets []DatasetRef `json:"datasets"`
}

func (e *DatabaseInUseError) Error() string {
	return fmt.Sprintf("database is in use by %d dataset(s)", len(e.Datasets))
}
```

In `backend/internal/domain/db/errors.go`, add re-exports:

```go
	ErrDatabaseHasRunningQueries = pkgerrors.ErrDatabaseHasRunningQueries
```

- [ ] **Step 4: Add new repository interface methods**

In `backend/internal/domain/db/repository.go`, add to `DatabaseRepository` interface (after line 29, before closing brace):

```go
	// CountRunningQueriesByDatabaseID returns the number of in-flight queries against a database.
	CountRunningQueriesByDatabaseID(ctx context.Context, databaseID uint) (int64, error)
	// ListDatasetsByDatabaseID returns dataset references bound to a database.
	ListDatasetsByDatabaseID(ctx context.Context, databaseID uint) ([]DatasetRef, error)
```

Add to `SchemaCacheRepository` interface:

```go
	// InvalidateByPrefix removes all cache entries whose keys start with the given prefix.
	InvalidateByPrefix(ctx context.Context, prefix string) error
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build ./...`
Expected: Compiles (only new exports/types added, no consumers yet).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/pkg/autherrors/errors.go backend/internal/domain/db/errors.go backend/internal/domain/db/database.go backend/internal/domain/db/repository.go
git commit -m "feat: add domain types and errors for DBC gap remediation phase 1"
```

---

### Task 2: Postgres Repo — New Methods (Phase 1 prep)

**Files:**
- Modify: `backend/internal/repository/postgres/database_repo.go`

- [ ] **Step 1: Implement `CountRunningQueriesByDatabaseID`**

After `CountDatasetsByDatabaseID` (line 212), add:

```go
func (r *databaseRepo) CountRunningQueriesByDatabaseID(ctx context.Context, databaseID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("query").
		Where("database_id = ? AND status IN ?", databaseID, []string{"running", "queued"}).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting running queries by database id: %w", err)
	}
	return count, nil
}
```

- [ ] **Step 2: Implement `ListDatasetsByDatabaseID`**

After that, add:

```go
func (r *databaseRepo) ListDatasetsByDatabaseID(ctx context.Context, databaseID uint) ([]domain.DatasetRef, error) {
	items := make([]domain.DatasetRef, 0)
	err := r.db.WithContext(ctx).
		Table("tables").
		Select("id, table_name").
		Where("database_id = ?", databaseID).
		Order("table_name ASC").
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("listing datasets by database id: %w", err)
	}
	return items, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build ./...`
Expected: Compiles. The `query` table may not exist — if it doesn't, the count just returns 0 (no error from GORM Select, but Table("query") would fail if the table truly doesn't exist).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/postgres/database_repo.go
git commit -m "feat: add running-queries count and dataset list repo methods"
```

---

### Task 3: Service Layer — RBAC Guards (Phase 1A)

**Files:**
- Modify: `backend/internal/app/db/database_service.go`
- Test: `backend/internal/app/db/database_service_test.go`

- [ ] **Step 1: Write failing tests for update/delete ownership**

In `backend/internal/app/db/database_service_test.go`, add after `TestDatabaseService_CreateDatabaseNonAdminReturnsForbidden` (line 370):

```go
func TestDatabaseService_UpdateDatabaseNonOwnerReturnsForbidden(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	// Database created by user 1 (CreatedByFK=1), actor is user 77 (non-admin, non-owner)
	repo := &fakeDatabaseRepo{
		isAdmin: false,
		getByIDResult: &domain.Database{ID: 7, DatabaseName: "analytics", SQLAlchemyURI: encryptedURI, CreatedByFK: 1},
	}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	name := "new-name"
	_, updateErr := svc.UpdateDatabase(context.Background(), 77, 7, domain.UpdateDatabaseRequest{
		DatabaseName: &name,
	})
	if !errors.Is(updateErr, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", updateErr)
	}
}

func TestDatabaseService_UpdateDatabaseOwnerCanUpdate(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	// Database created by user 77, actor is user 77 (non-admin, owner)
	repo := &fakeDatabaseRepo{
		isAdmin: false,
		getByIDResult: &domain.Database{ID: 7, DatabaseName: "analytics", SQLAlchemyURI: encryptedURI, CreatedByFK: 77},
	}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	name := "new-name"
	updated, updateErr := svc.UpdateDatabase(context.Background(), 77, 7, domain.UpdateDatabaseRequest{
		DatabaseName: &name,
	})
	if updateErr != nil {
		t.Fatalf("expected nil error, got %v", updateErr)
	}
	if updated.DatabaseName != "new-name" {
		t.Fatalf("expected updated name, got %s", updated.DatabaseName)
	}
}

func TestDatabaseService_DeleteDatabaseNonOwnerReturnsForbidden(t *testing.T) {
	// Database created by user 1 (CreatedByFK=1), actor is user 77 (non-admin, non-owner)
	repo := &fakeDatabaseRepo{
		isAdmin: false,
		getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: "postgresql://alice:enc@localhost:5432/analytics", CreatedByFK: 1},
		datasetCount: 0,
	}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	deleteErr := svc.DeleteDatabase(context.Background(), 77, 11)
	if !errors.Is(deleteErr, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", deleteErr)
	}
}

func TestDatabaseService_DeleteDatabaseOwnerCanDelete(t *testing.T) {
	// Database created by user 77, actor is user 77 (non-admin, owner)
	repo := &fakeDatabaseRepo{
		isAdmin: false,
		getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: "postgresql://alice:enc@localhost:5432/analytics", CreatedByFK: 77},
		datasetCount: 0,
	}
	pool := &fakeConnectionPool{}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}
	svc.SetConnectionPool(pool)

	deleteErr := svc.DeleteDatabase(context.Background(), 77, 11)
	if deleteErr != nil {
		t.Fatalf("expected nil error, got %v", deleteErr)
	}
	if repo.deleted != 11 {
		t.Fatalf("expected deleted id 11, got %d", repo.deleted)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/app/db/ -run "TestDatabaseService_UpdateDatabaseNonOwner|TestDatabaseService_UpdateDatabaseOwner|TestDatabaseService_DeleteDatabaseNonOwner|TestDatabaseService_DeleteDatabaseOwner" -v`
Expected: Non-owner tests FAIL (no guard yet). Non-admin create test also still FAILS.

- [ ] **Step 3: Add RBAC guards to service methods**

In `database_service.go`, in `CreateDatabase` method — after `normalizeCreateDatabaseRequest` success check, before duplicate name check (after line 280), add:

```go
	isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	if err != nil {
		return nil, fmt.Errorf("checking admin role: %w", err)
	}
	if !isAdmin {
		return nil, domain.ErrForbidden
	}
```

In `UpdateDatabase` method — after fetching existing record (after line 427), add:

```go
	isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	if err != nil {
		return nil, fmt.Errorf("checking admin role: %w", err)
	}
	if !isAdmin && existing.CreatedByFK != actorUserID {
		return nil, domain.ErrForbidden
	}
```

In `DeleteDatabase` method — after fetching existing record (after line 512), add:

```go
	isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	if err != nil {
		return fmt.Errorf("checking admin role: %w", err)
	}
	if !isAdmin && existing.CreatedByFK != actorUserID {
		return domain.ErrForbidden
	}
```

Note: the `existing` variable name needs to be used. In `DeleteDatabase`, line 510-512 currently does `if _, err := s.repo.GetDatabaseByID(...)`. Change it to:

```go
	existing, err := s.repo.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return err
	}
```

- [ ] **Step 4: Run all service tests**

Run: `cd backend && go test ./internal/app/db/ -v`
Expected: All tests pass, including `TestDatabaseService_CreateDatabaseNonAdminReturnsForbidden`, `TestDatabaseService_UpdateDatabaseNonOwnerReturnsForbidden`, etc.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/app/db/database_service.go backend/internal/app/db/database_service_test.go
git commit -m "feat: add RBAC ownership guards to Create, Update, Delete database"
```

---

### Task 4: Service Layer — Running-Queries Guard + Structured 409 (Phase 1B + 1C)

**Files:**
- Modify: `backend/internal/app/db/database_service.go`
- Modify: `backend/internal/app/db/database_service_test.go`

- [ ] **Step 1: Write failing tests**

In `database_service_test.go`, add:

```go
func TestDatabaseService_DeleteDatabaseHasRunningQueriesReturns409(t *testing.T) {
	repo := &fakeDatabaseRepo{
		isAdmin: true,
		getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: "postgresql://alice:enc@localhost:5432/analytics", CreatedByFK: 1},
		datasetCount:       0,
		runningQueryCount:  3,
	}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	deleteErr := svc.DeleteDatabase(context.Background(), 1, 11)
	if !errors.Is(deleteErr, domain.ErrDatabaseHasRunningQueries) {
		t.Fatalf("expected ErrDatabaseHasRunningQueries, got %v", deleteErr)
	}
}

func TestDatabaseService_DeleteDatabaseHasDatasetsReturns409WithList(t *testing.T) {
	repo := &fakeDatabaseRepo{
		isAdmin: true,
		getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: "postgresql://alice:enc@localhost:5432/analytics", CreatedByFK: 1},
		datasetCount: 2,
		datasets: []domain.DatasetRef{
			{ID: 1, TableName: "orders"},
			{ID: 2, TableName: "users"},
		},
	}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}

	deleteErr := svc.DeleteDatabase(context.Background(), 1, 11)
	var inUseErr *domain.DatabaseInUseError
	if !errors.As(deleteErr, &inUseErr) {
		t.Fatalf("expected DatabaseInUseError, got %v", deleteErr)
	}
	if len(inUseErr.Datasets) != 2 {
		t.Fatalf("expected 2 datasets in error, got %d", len(inUseErr.Datasets))
	}
	if inUseErr.Datasets[0].TableName != "orders" {
		t.Fatalf("expected first dataset 'orders', got '%s'", inUseErr.Datasets[0].TableName)
	}
}
```

- [ ] **Step 2: Add `runningQueryCount` and `datasets` fields to `fakeDatabaseRepo`**

In `database_service_test.go`, add to `fakeDatabaseRepo` struct (after `datasetCount` line):

```go
	runningQueryCount int64
	datasets          []domain.DatasetRef
```

Add methods:

```go
func (f *fakeDatabaseRepo) CountRunningQueriesByDatabaseID(_ context.Context, _ uint) (int64, error) {
	return f.runningQueryCount, nil
}

func (f *fakeDatabaseRepo) ListDatasetsByDatabaseID(_ context.Context, _ uint) ([]domain.DatasetRef, error) {
	return append([]domain.DatasetRef(nil), f.datasets...), nil
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd backend && go test ./internal/app/db/ -run "TestDatabaseService_DeleteDatabaseHasRunningQueries|TestDatabaseService_DeleteDatabaseHasDatasetsReturns409WithList" -v`
Expected: FAIL — guards not yet in service.

- [ ] **Step 4: Update `DeleteDatabase` with running-queries guard + structured error**

Replace the `DeleteDatabase` method body in `database_service.go` (lines 505-534) with:

```go
func (s *DatabaseService) DeleteDatabase(ctx context.Context, actorUserID uint, databaseID uint) error {
	if databaseID == 0 {
		return domain.ErrInvalidDatabase
	}

	existing, err := s.repo.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return err
	}

	isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	if err != nil {
		return fmt.Errorf("checking admin role: %w", err)
	}
	if !isAdmin && existing.CreatedByFK != actorUserID {
		return domain.ErrForbidden
	}

	queryCount, err := s.repo.CountRunningQueriesByDatabaseID(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("checking running queries: %w", err)
	}
	if queryCount > 0 {
		return domain.ErrDatabaseHasRunningQueries
	}

	datasets, err := s.repo.ListDatasetsByDatabaseID(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("listing database datasets: %w", err)
	}
	if len(datasets) > 0 {
		return &domain.DatabaseInUseError{Datasets: datasets}
	}

	if s.poolManager != nil {
		if err := s.poolManager.Close(ctx, databaseID); err != nil {
			return fmt.Errorf("closing database pool: %w", err)
		}
	}

	if err := s.repo.DeleteDatabase(ctx, databaseID); err != nil {
		return err
	}

	return nil
}
```

- [ ] **Step 5: Run all service tests**

Run: `cd backend && go test ./internal/app/db/ -v`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/app/db/database_service.go backend/internal/app/db/database_service_test.go
git commit -m "feat: add running-queries guard and structured 409 with dataset list on delete"
```

---

### Task 5: Handler Layer — Error Mappings (Phase 1 handler)

**Files:**
- Modify: `backend/internal/delivery/http/db/database_handler.go`
- Test: `backend/internal/delivery/http/db/database_handler_test.go`

- [ ] **Step 1: Add missing repo methods to `handlerDatabaseRepo`**

In `database_handler_test.go`, add to `handlerDatabaseRepo` struct:

```go
	runningQueryCount int64
	datasets          []domain.DatasetRef
```

Add methods:

```go
func (h *handlerDatabaseRepo) CountRunningQueriesByDatabaseID(_ context.Context, _ uint) (int64, error) {
	return h.runningQueryCount, nil
}

func (h *handlerDatabaseRepo) ListDatasetsByDatabaseID(_ context.Context, _ uint) ([]domain.DatasetRef, error) {
	return append([]domain.DatasetRef(nil), h.datasets...), nil
}
```

Also add `LogDatabaseDeleted` to `handlerDatabaseAuditLogger`:

```go
func (h *handlerDatabaseAuditLogger) LogDatabaseDeleted(_ context.Context, _ uint) {}
```

- [ ] **Step 2: Add `ErrDatabaseHasRunningQueries` and `*DatabaseInUseError` to `handleError`**

In `database_handler.go`, in `handleError` (line 331), add after the `ErrDatabaseInUse` case (after line 346):

```go
	case errors.Is(err, domain.ErrDatabaseHasRunningQueries):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
```

Change the `ErrDatabaseInUse` case to handle `*DatabaseInUseError`:

```go
	case errors.Is(err, domain.ErrDatabaseInUse):
		var inUseErr *domain.DatabaseInUseError
		if errors.As(err, &inUseErr) {
			c.JSON(http.StatusConflict, gin.H{"error": inUseErr.Error(), "datasets": inUseErr.Datasets})
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		}
```

Note: Since `*DatabaseInUseError.Error()` returns a different message than the bare sentinel `ErrDatabaseInUse`, `errors.Is(err, domain.ErrDatabaseInUse)` won't match the struct error. The `errors.As` case handles the struct; the bare sentinel fallback handles the old path if any code still returns the flat error. But actually, the struct error won't match `errors.Is(err, domain.ErrDatabaseInUse)` since it's a different error. So the `ErrDatabaseInUse` case becomes dead code once we switch to structured errors. Keep both — the struct check first via `errors.As`, then the bare sentinel fallback:

```go
	var inUseErr *domain.DatabaseInUseError
	if errors.As(err, &inUseErr) {
		c.JSON(http.StatusConflict, gin.H{"error": inUseErr.Error(), "datasets": inUseErr.Datasets})
	}
```

Insert this BEFORE the existing `case errors.Is(err, domain.ErrDatabaseInUse)` block. Keep the old block as a fallback.

- [ ] **Step 3: Write handler test for structured 409**

In `database_handler_test.go`, add:

```go
func TestDatabaseHandler_DeleteReturns409WithDatasetList(t *testing.T) {
	r := newDatabaseRouter(
		&handlerDatabaseRepo{
			isAdmin:       true,
			getByIDResult: &domain.Database{ID: 2, SQLAlchemyURI: "postgresql://superset:enc@localhost:5432/analytics", CreatedByFK: 1},
			datasets:      []domain.DatasetRef{{ID: 1, TableName: "orders"}, {ID: 2, TableName: "users"}},
		},
		&handlerDatabaseTester{allowRate: true},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/databases/2", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("\"datasets\"")) {
		t.Fatalf("expected datasets field in response, got: %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("\"orders\"")) {
		t.Fatalf("expected orders dataset in response, got: %s", w.Body.String())
	}
}
```

- [ ] **Step 4: Run handler tests**

Run: `cd backend && go test ./internal/delivery/http/db/ -v`
Expected: `TestDatabaseHandler_PostNonAdminReturns403` now passes. `TestDatabaseHandler_DeleteReturns409WithDatasetList` passes. All existing tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/db/database_handler.go backend/internal/delivery/http/db/database_handler_test.go
git commit -m "feat: add structured 409 error mapping and running-queries 409 to handler"
```

---

### Task 6: Remove Plaintext URI Log (Phase 2A)

**Files:**
- Modify: `backend/internal/app/db/database_service.go`

- [ ] **Step 1: Delete the log line**

In `database_service.go`, delete line 118:
```go
log.Println("[database_prober] sqlalchemyURI:" + sqlalchemyURI)
```

- [ ] **Step 2: Remove unused `log` import**

Check if `log` is used elsewhere in the file. If not, remove `"log"` from imports (line 8).

- [ ] **Step 3: Run tests**

Run: `cd backend && go test ./internal/app/db/ -v`
Expected: All pass.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/app/db/database_service.go
git commit -m "fix: remove plaintext URI log from Probe method"
```

---

### Task 7: Multi-Driver Support (Phase 2B — MySQL, BigQuery, Snowflake)

**Files:**
- Modify: `backend/internal/app/db/database_service.go`
- Modify: `backend/go.mod`

- [ ] **Step 1: Extend `resolveSQLDriver`**

Replace the function (lines 167-175) with:

```go
func resolveSQLDriver(scheme string) (string, string, error) {
	value := strings.ToLower(strings.TrimSpace(scheme))
	switch value {
	case "postgres", "postgresql":
		return "pgx", "postgresql", nil
	case "mysql":
		return "mysql", "mysql", nil
	case "bigquery":
		return "bigquery", "bigquery", nil
	case "snowflake":
		return "snowflake", "snowflake", nil
	default:
		return "", "", domain.ErrUnknownDatabaseDriver
	}
}
```

- [ ] **Step 2: Add version query mapping**

Add a helper function after `resolveSQLDriver`:

```go
func driverVersionQuery(driverName string) string {
	switch driverName {
	case "pgx":
		return "SELECT version()"
	case "mysql":
		return "SELECT VERSION()"
	case "bigquery":
		return "SELECT CURRENT_VERSION()"
	case "snowflake":
		return "SELECT CURRENT_VERSION()"
	default:
		return "SELECT version()"
	}
}
```

Update the `Probe` method to use it. Change line 150 from:
```go
if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&dbVersion); err != nil {
```
To:
```go
if err := db.QueryRowContext(ctx, driverVersionQuery(driverName)).Scan(&dbVersion); err != nil {
```

- [ ] **Step 3: Add driver imports and go.mod dependencies**

Add driver imports at the top of `database_service.go` (after pgx import, line 18):

```go
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/snowflakedb/gosnowflake"
```

Run: `cd backend && go get github.com/go-sql-driver/mysql@latest && go get github.com/snowflakedb/gosnowflake@latest && go mod tidy`

(Note: BigQuery uses the standard `database/sql` interface via the existing BigQuery storage driver — no separate SQL driver import needed; or if using a specific BigQuery SQL driver, add `_ "github.com/xxx/bigquery/driver"`)

- [ ] **Step 4: Write test**

In `database_service_test.go`, add:

```go
func TestResolveSQLDriver_SupportsMySQL(t *testing.T) {
	driverName, driverLabel, err := svcauth.ResolveSQLDriverForTest("mysql://user:pass@localhost:3306/db")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if driverName != "mysql" || driverLabel != "mysql" {
		t.Fatalf("expected mysql/mysql, got %s/%s", driverName, driverLabel)
	}
}
```

Add test helper in `database_service.go`:

```go
// ResolveSQLDriverForTest exposes resolveSQLDriver for tests.
func ResolveSQLDriverForTest(scheme string) (string, string, error) {
	return resolveSQLDriver(scheme)
}
```

Wait — `resolveSQLDriver` takes a scheme, not a full URI. Use the test with a scheme:

```go
func TestResolveSQLDriver_SupportsMySQL(t *testing.T) {
	driverName, driverLabel, err := svcauth.ResolveSQLDriverForTest("mysql")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if driverName != "mysql" || driverLabel != "mysql" {
		t.Fatalf("expected mysql/mysql, got %s/%s", driverName, driverLabel)
	}
}

func TestResolveSQLDriver_SupportsBigQuery(t *testing.T) {
	_, _, err := svcauth.ResolveSQLDriverForTest("bigquery")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestResolveSQLDriver_SupportsSnowflake(t *testing.T) {
	_, _, err := svcauth.ResolveSQLDriverForTest("snowflake")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
```

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/app/db/ -v -run "TestResolveSQLDriver"`
Expected: All 3 new tests pass. Full suite: `cd backend && go test ./internal/app/db/ -v` — all pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/app/db/database_service.go backend/internal/app/db/database_service_test.go backend/go.mod backend/go.sum
git commit -m "feat: add MySQL, BigQuery, Snowflake driver support with per-driver version queries"
```

---

### Task 8: Multi-Driver Schema Inspectors (Phase 2C)

**Files:**
- Modify: `backend/internal/app/db/schema_inspector.go`
- Test: `backend/internal/app/db/database_service_test.go`

- [ ] **Step 1: Add `mysqlSchemaInspector`**

In `schema_inspector.go`, after `postgresSchemaInspector` methods, add:

```go
type mysqlSchemaInspector struct{}

func (mysqlSchemaInspector) ListSchemas(ctx context.Context, conn SQLConnection) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing schemas: %w", err)
	}
	defer rows.Close()

	schemas := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scanning schema row: %w", scanErr)
		}
		schemas = append(schemas, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating schema rows: %w", rowsErr)
	}
	return schemas, nil
}

func (mysqlSchemaInspector) ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int) ([]domain.DatabaseTable, int64, error) {
	normalizedSchema := strings.TrimSpace(schema)
	if normalizedSchema == "" {
		return nil, 0, domain.ErrInvalidDatabase
	}

	normalizedPage, normalizedPageSize := normalizeTablesPagination(page, pageSize)
	offset := (normalizedPage - 1) * normalizedPageSize

	countRows, err := conn.QueryContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
	`, normalizedSchema)
	if err != nil {
		return nil, 0, fmt.Errorf("counting tables: %w", err)
	}
	defer countRows.Close()

	var total int64
	if countRows.Next() {
		if scanErr := countRows.Scan(&total); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table count: %w", scanErr)
		}
	}
	if countRowsErr := countRows.Err(); countRowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table count rows: %w", countRowsErr)
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
		LIMIT ? OFFSET ?
	`, normalizedSchema, normalizedPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	tables := make([]domain.DatabaseTable, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
		}
		tables = append(tables, domain.DatabaseTable{Name: name})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table rows: %w", rowsErr)
	}
	return tables, total, nil
}

func (mysqlSchemaInspector) ListColumns(ctx context.Context, conn SQLConnection, schema string, table string) ([]domain.DatabaseColumn, error) {
	normalizedSchema := strings.TrimSpace(schema)
	normalizedTable := strings.TrimSpace(table)
	if normalizedSchema == "" || normalizedTable == "" {
		return nil, domain.ErrInvalidDatabase
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT
			column_name,
			data_type,
			is_nullable,
			COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = ?
		  AND table_name = ?
		ORDER BY ordinal_position
	`, normalizedSchema, normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()

	columns := make([]domain.DatabaseColumn, 0)
	for rows.Next() {
		var (
			name         string
			dataType     string
			nullable     string
			defaultValue string
		)
		if scanErr := rows.Scan(&name, &dataType, &nullable, &defaultValue); scanErr != nil {
			return nil, fmt.Errorf("scanning column row: %w", scanErr)
		}
		columns = append(columns, domain.DatabaseColumn{
			Name:         name,
			DataType:     dataType,
			IsNullable:   strings.EqualFold(nullable, "YES"),
			DefaultValue: defaultValue,
			IsDttm:       isMySQLDateTimeColumnType(dataType),
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating column rows: %w", rowsErr)
	}
	return columns, nil
}
```

- [ ] **Step 2: Add `bigquerySchemaInspector` and `snowflakeSchemaInspector` stubs**

```go
type bigquerySchemaInspector struct{}

func (bigquerySchemaInspector) ListSchemas(ctx context.Context, conn SQLConnection) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT schema_name
		FROM INFORMATION_SCHEMA.SCHEMATA
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing schemas: %w", err)
	}
	defer rows.Close()

	schemas := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scanning schema row: %w", scanErr)
		}
		schemas = append(schemas, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating schema rows: %w", rowsErr)
	}
	return schemas, nil
}

func (bigquerySchemaInspector) ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int) ([]domain.DatabaseTable, int64, error) {
	normalizedSchema := strings.TrimSpace(schema)
	if normalizedSchema == "" {
		return nil, 0, domain.ErrInvalidDatabase
	}
	normalizedPage, normalizedPageSize := normalizeTablesPagination(page, pageSize)
	offset := (normalizedPage - 1) * normalizedPageSize

	countRows, err := conn.QueryContext(ctx,
		"SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE table_schema = @schema AND table_type = 'BASE TABLE'",
		normalizedSchema)
	if err != nil {
		return nil, 0, fmt.Errorf("counting tables: %w", err)
	}
	defer countRows.Close()
	var total int64
	if countRows.Next() {
		if scanErr := countRows.Scan(&total); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table count: %w", scanErr)
		}
	}
	if countRowsErr := countRows.Err(); countRowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table count rows: %w", countRowsErr)
	}

	rows, err := conn.QueryContext(ctx,
		"SELECT table_name FROM INFORMATION_SCHEMA.TABLES WHERE table_schema = @schema AND table_type = 'BASE TABLE' ORDER BY table_name LIMIT @limit OFFSET @offset",
		normalizedSchema, normalizedPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	tables := make([]domain.DatabaseTable, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
		}
		tables = append(tables, domain.DatabaseTable{Name: name})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table rows: %w", rowsErr)
	}

	return tables, total, nil
}

func (bigquerySchemaInspector) ListColumns(ctx context.Context, conn SQLConnection, schema string, table string) ([]domain.DatabaseColumn, error) {
	normalizedSchema := strings.TrimSpace(schema)
	normalizedTable := strings.TrimSpace(table)
	if normalizedSchema == "" || normalizedTable == "" {
		return nil, domain.ErrInvalidDatabase
	}

	rows, err := conn.QueryContext(ctx,
		"SELECT column_name, data_type, is_nullable, COALESCE(column_default, '') FROM INFORMATION_SCHEMA.COLUMNS WHERE table_schema = @schema AND table_name = @table ORDER BY ordinal_position",
		normalizedSchema, normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()

	columns := make([]domain.DatabaseColumn, 0)
	for rows.Next() {
		var name, dataType, nullable, defaultValue string
		if scanErr := rows.Scan(&name, &dataType, &nullable, &defaultValue); scanErr != nil {
			return nil, fmt.Errorf("scanning column row: %w", scanErr)
		}
		columns = append(columns, domain.DatabaseColumn{
			Name: name, DataType: dataType,
			IsNullable: strings.EqualFold(nullable, "YES"),
			DefaultValue: defaultValue, IsDttm: isDateTimeColumnType(dataType),
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating column rows: %w", rowsErr)
	}
	return columns, nil
}

type snowflakeSchemaInspector struct{}

func (snowflakeSchemaInspector) ListSchemas(ctx context.Context, conn SQLConnection) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('INFORMATION_SCHEMA')
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing schemas: %w", err)
	}
	defer rows.Close()
	schemas := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scanning schema row: %w", scanErr)
		}
		schemas = append(schemas, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating schema rows: %w", rowsErr)
	}
	return schemas, nil
}

func (snowflakeSchemaInspector) ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int) ([]domain.DatabaseTable, int64, error) {
	normalizedSchema := strings.TrimSpace(schema)
	if normalizedSchema == "" {
		return nil, 0, domain.ErrInvalidDatabase
	}
	normalizedPage, normalizedPageSize := normalizeTablesPagination(page, pageSize)
	offset := (normalizedPage - 1) * normalizedPageSize

	countRows, err := conn.QueryContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
	`, normalizedSchema)
	if err != nil {
		return nil, 0, fmt.Errorf("counting tables: %w", err)
	}
	defer countRows.Close()
	var total int64
	if countRows.Next() {
		if scanErr := countRows.Scan(&total); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table count: %w", scanErr)
		}
	}
	if countRowsErr := countRows.Err(); countRowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table count rows: %w", countRowsErr)
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
		LIMIT ? OFFSET ?
	`, normalizedSchema, normalizedPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()
	tables := make([]domain.DatabaseTable, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
		}
		tables = append(tables, domain.DatabaseTable{Name: name})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table rows: %w", rowsErr)
	}
	return tables, total, nil
}

func (snowflakeSchemaInspector) ListColumns(ctx context.Context, conn SQLConnection, schema string, table string) ([]domain.DatabaseColumn, error) {
	normalizedSchema := strings.TrimSpace(schema)
	normalizedTable := strings.TrimSpace(table)
	if normalizedSchema == "" || normalizedTable == "" {
		return nil, domain.ErrInvalidDatabase
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`, normalizedSchema, normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()
	columns := make([]domain.DatabaseColumn, 0)
	for rows.Next() {
		var name, dataType, nullable, defaultValue string
		if scanErr := rows.Scan(&name, &dataType, &nullable, &defaultValue); scanErr != nil {
			return nil, fmt.Errorf("scanning column row: %w", scanErr)
		}
		columns = append(columns, domain.DatabaseColumn{
			Name: name, DataType: dataType,
			IsNullable:   strings.EqualFold(nullable, "YES"),
			DefaultValue: defaultValue,
			IsDttm:       isSnowflakeDateTimeColumnType(dataType),
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating column rows: %w", rowsErr)
	}
	return columns, nil
}
```

- [ ] **Step 3: Add helper functions**

After `isDateTimeColumnType`:

```go
func isMySQLDateTimeColumnType(dataType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(dataType))
	switch normalized {
	case "date", "datetime", "time", "timestamp", "year":
		return true
	default:
		return false
	}
}

func isSnowflakeDateTimeColumnType(dataType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(dataType))
	switch normalized {
	case "date", "time", "datetime", "timestamp", "timestamp_ltz", "timestamp_ntz", "timestamp_tz":
		return true
	default:
		return false
	}
}

```

- [ ] **Step 4: Add `selectSchemaInspector` + factory**

At the top of `schema_inspector.go` (after the interface), add:

```go
func selectSchemaInspector(driverName string) SchemaInspector {
	switch driverName {
	case "pgx":
		return postgresSchemaInspector{}
	case "mysql":
		return mysqlSchemaInspector{}
	case "bigquery":
		return bigquerySchemaInspector{}
	case "snowflake":
		return snowflakeSchemaInspector{}
	default:
		return postgresSchemaInspector{}
	}
}
```

Update `NewDefaultSchemaInspector()` and `newDefaultSchemaInspector()` to still return postgres for backward compat (they are used in tests and constructor). The dynamic selection happens at call sites.

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/app/db/ -v`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/app/db/schema_inspector.go
git commit -m "feat: add MySQL, BigQuery, Snowflake schema inspector implementations"
```

---

### Task 9: Schema Cache Invalidation + Redis Wiring (Phase 2D, 2E, 2F)

**Files:**
- Modify: `backend/internal/app/db/schema_cache_memory.go`
- Modify: `backend/internal/repository/redis/database_schema_cache_repo.go`
- Modify: `backend/internal/app/db/database_service.go`
- Test: `backend/internal/app/db/database_service_test.go`

- [ ] **Step 1: Implement `InvalidateByPrefix` on in-memory cache**

In `schema_cache_memory.go`, add:

```go
func (c *inMemorySchemaCache) InvalidateByPrefix(_ context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
		}
	}
	return nil
}
```

- [ ] **Step 2: Implement `InvalidateByPrefix` on Redis cache**

In `repository/redis/database_schema_cache_repo.go`, add:

```go
func (r *databaseSchemaCacheRepo) InvalidateByPrefix(ctx context.Context, prefix string) error {
	iter := r.client.Scan(ctx, 0, prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("deleting schema cache key %s: %w", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("scanning schema cache keys: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Add cache invalidation to `UpdateDatabase` and `DeleteDatabase`**

In `database_service.go`, add after `s.poolManager.Close` in `UpdateDatabase` (after line 486):

```go
	if s.schemaCache != nil {
		dbIDStr := fmt.Sprintf("%d", databaseID)
		_ = s.schemaCache.InvalidateByPrefix(ctx, "schema:"+dbIDStr+":")
	}
```

Import `"fmt"` is already imported. Add the same code in `DeleteDatabase` after `s.poolManager.Close` (before the `DeleteDatabase` call).

- [ ] **Step 4: Write tests for cache invalidation**

In `database_service_test.go`, update `fakeSchemaCache` to track invalidated prefixes:

```go
type fakeSchemaCache struct {
	store             map[string]string
	invalidatedPrefix []string
}

func (f *fakeSchemaCache) Get(_ context.Context, key string) (string, bool, error) {
	if f.store == nil {
		return "", false, nil
	}
	value, ok := f.store[key]
	if !ok {
		return "", false, nil
	}
	return value, true, nil
}

func (f *fakeSchemaCache) Set(_ context.Context, key string, value string, _ time.Duration) error {
	if f.store == nil {
		f.store = map[string]string{}
	}
	f.store[key] = value
	return nil
}

func (f *fakeSchemaCache) InvalidateByPrefix(_ context.Context, prefix string) error {
	f.invalidatedPrefix = append(f.invalidatedPrefix, prefix)
	return nil
}
```

Add tests:

```go
func TestDatabaseService_UpdateDatabaseFlushesSchemaCache(t *testing.T) {
	encryptedURI, err := svcauth.EncryptSQLAlchemyURIPasswordForTest("postgresql://alice:secret@localhost:5432/analytics", "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil encrypt error, got %v", err)
	}

	repo := &fakeDatabaseRepo{isAdmin: true, getByIDResult: &domain.Database{ID: 7, DatabaseName: "analytics", SQLAlchemyURI: encryptedURI}}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}
	svc.SetConnectionPool(&fakeConnectionPool{})

	cache := &fakeSchemaCache{store: map[string]string{}}
	svc.SetSchemaCache(cache)

	name := "new-name"
	_, updateErr := svc.UpdateDatabase(context.Background(), 1, 7, domain.UpdateDatabaseRequest{
		DatabaseName: &name,
	})
	if updateErr != nil {
		t.Fatalf("expected nil error, got %v", updateErr)
	}
	if len(cache.invalidatedPrefix) != 1 || cache.invalidatedPrefix[0] != "schema:7:" {
		t.Fatalf("expected schema cache invalidated with prefix 'schema:7:', got %v", cache.invalidatedPrefix)
	}
}

func TestDatabaseService_DeleteDatabaseFlushesSchemaCache(t *testing.T) {
	repo := &fakeDatabaseRepo{
		isAdmin:       true,
		getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: "postgresql://alice:enc@localhost:5432/analytics"},
		datasetCount:  0,
	}
	pool := &fakeConnectionPool{}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, &fakeDatabaseAuditLogger{}, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}
	svc.SetConnectionPool(pool)

	cache := &fakeSchemaCache{store: map[string]string{}}
	svc.SetSchemaCache(cache)

	deleteErr := svc.DeleteDatabase(context.Background(), 1, 11)
	if deleteErr != nil {
		t.Fatalf("expected nil error, got %v", deleteErr)
	}
	if len(cache.invalidatedPrefix) != 1 || cache.invalidatedPrefix[0] != "schema:11:" {
		t.Fatalf("expected schema cache invalidated with prefix 'schema:11:', got %v", cache.invalidatedPrefix)
	}
}
```

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/app/db/ -v`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/app/db/schema_cache_memory.go backend/internal/repository/redis/database_schema_cache_repo.go backend/internal/app/db/database_service.go backend/internal/app/db/database_service_test.go
git commit -m "feat: add schema cache invalidation on update/delete and Redis InvalidateByPrefix"
```

---

### Task 10: Audit on Delete (Phase 3A)

**Files:**
- Modify: `backend/internal/app/db/database_service.go`
- Test: `backend/internal/app/db/database_service_test.go`

- [ ] **Step 1: Extend `DatabaseAuditLogger` interface**

In `database_service.go`, add to interface (line 52):

```go
	LogDatabaseDeleted(ctx context.Context, databaseID uint)
```

- [ ] **Step 2: Add noop implementation**

In `database_service.go`, after line 90:

```go
func (noopDatabaseAuditLogger) LogDatabaseDeleted(_ context.Context, _ uint) {}
```

- [ ] **Step 3: Call audit in `DeleteDatabase`**

In `DeleteDatabase`, after successful `s.repo.DeleteDatabase` call, add:

```go
	go s.auditLogger.LogDatabaseDeleted(context.Background(), databaseID)
```

- [ ] **Step 4: Update `fakeDatabaseAuditLogger` and write test**

In `database_service_test.go`, add to `fakeDatabaseAuditLogger`:

```go
	deleteCalled int
	deleteLastID uint
```

Add method:

```go
func (f *fakeDatabaseAuditLogger) LogDatabaseDeleted(_ context.Context, databaseID uint) {
	f.deleteCalled++
	f.deleteLastID = databaseID
}
```

Add test:

```go
func TestDatabaseService_DeleteDatabaseAuditLogCalled(t *testing.T) {
	repo := &fakeDatabaseRepo{
		isAdmin:       true,
		getByIDResult: &domain.Database{ID: 11, SQLAlchemyURI: "postgresql://alice:enc@localhost:5432/analytics"},
		datasetCount:  0,
	}
	pool := &fakeConnectionPool{}
	audit := &fakeDatabaseAuditLogger{}
	svc, err := svcauth.NewDatabaseService(repo, &fakeDatabaseTester{}, audit, "12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected nil constructor error, got %v", err)
	}
	svc.SetConnectionPool(pool)

	deleteErr := svc.DeleteDatabase(context.Background(), 1, 11)
	if deleteErr != nil {
		t.Fatalf("expected nil error, got %v", deleteErr)
	}

	// Wait briefly for async goroutine
	time.Sleep(10 * time.Millisecond)
	if audit.deleteCalled != 1 {
		t.Fatalf("expected delete audit called once, got %d", audit.deleteCalled)
	}
	if audit.deleteLastID != 11 {
		t.Fatalf("expected audit last id 11, got %d", audit.deleteLastID)
	}
}
```

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/app/db/ -v -run "TestDatabaseService_DeleteDatabaseAuditLogCalled"`
Expected: Pass. Full suite: all pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/app/db/database_service.go backend/internal/app/db/database_service_test.go
git commit -m "feat: add audit log on database delete"
```

---

### Task 11: Views Support in ListTables (Phase 3B)

**Files:**
- Modify: `backend/internal/domain/db/database.go`
- Modify: `backend/internal/app/db/schema_inspector.go`
- Modify: `backend/internal/app/db/database_service_introspection.go`
- Modify: `backend/internal/delivery/http/db/database_handler.go`

- [ ] **Step 1: Add `TableType` to domain request**

In `database.go`, add to `ListDatabaseTablesRequest` (line 132):

```go
	TableType string // "BASE TABLE", "VIEW", or "" for all
```

- [ ] **Step 2: Update `SchemaInspector` interface to accept `tableType`**

In `schema_inspector.go`, update the `SchemaInspector` interface `ListTables` signature:

```go
	ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int, tableType string) ([]domain.DatabaseTable, int64, error)
```

Update all implementations (`postgresSchemaInspector`, `mysqlSchemaInspector`, `bigquerySchemaInspector`, `snowflakeSchemaInspector`) to accept and use the `tableType` parameter:

For postgres, change the SQL from hardcoded `AND table_type = 'BASE TABLE'` to:

```go
func (postgresSchemaInspector) ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int, tableType string) ([]domain.DatabaseTable, int64, error) {
	normalizedSchema := strings.TrimSpace(schema)
	if normalizedSchema == "" {
		return nil, 0, domain.ErrInvalidDatabase
	}

	normalizedPage, normalizedPageSize := normalizeTablesPagination(page, pageSize)
	offset := (normalizedPage - 1) * normalizedPageSize

	typeFilter := ""
	typeArgs := []any{normalizedSchema}
	if tableType != "" {
		typeFilter = "  AND table_type = $2"
		typeArgs = append(typeArgs, tableType)
	}

	countQuery := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = $1" + typeFilter
	countRows, err := conn.QueryContext(ctx, countQuery, typeArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("counting tables: %w", err)
	}
	defer countRows.Close()

	var total int64
	if countRows.Next() {
		if scanErr := countRows.Scan(&total); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table count: %w", scanErr)
		}
	}
	if countRowsErr := countRows.Err(); countRowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table count rows: %w", countRowsErr)
	}

	dataQuery := "SELECT table_name FROM information_schema.tables WHERE table_schema = $1" + typeFilter + " ORDER BY table_name LIMIT $2 OFFSET $3"
	dataArgs := append(typeArgs, normalizedPageSize, offset)
	rows, err := conn.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	tables := make([]domain.DatabaseTable, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
		}
		tables = append(tables, domain.DatabaseTable{Name: name})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table rows: %w", rowsErr)
	}

	return tables, total, nil
}
```

Apply the same parameterized `tableType` pattern to `mysqlSchemaInspector` (with `?` placeholders), `bigquerySchemaInspector` (with `@` named params), and `snowflakeSchemaInspector` (with `?` placeholders).

- [ ] **Step 3: Pass `tableType` through service introspection**

In `database_service_introspection.go`, pass through from request:

Change line 75 from:
```go
tables, total, err := s.schemaInspector.ListTables(timeoutCtx, connection, normalized.Schema, normalized.Page, normalized.PageSize)
```
To:
```go
tables, total, err := s.schemaInspector.ListTables(timeoutCtx, connection, normalized.Schema, normalized.Page, normalized.PageSize, normalized.TableType)
```

Update cache key in `ListTables` to include table type (line 59):
```go
cacheKey := fmt.Sprintf("schema:%d:%s:tables:%d:%d:%s", databaseID, normalized.Schema, normalized.Page, normalized.PageSize, normalized.TableType)
```

Update `normalizeListTablesRequest` to include `TableType`:

```go
func normalizeListTablesRequest(req domain.ListDatabaseTablesRequest) domain.ListDatabaseTablesRequest {
	page, pageSize := normalizeTablesPagination(req.Page, req.PageSize)
	return domain.ListDatabaseTablesRequest{
		Schema:    strings.TrimSpace(req.Schema),
		Page:      page,
		PageSize:  pageSize,
		TableType: strings.TrimSpace(req.TableType),
	}
}
```

- [ ] **Step 4: Parse `table_type` query param in handler**

In `database_handler.go`, in `ListTables` handler, add parsing:

```go
	tableType := strings.ToUpper(strings.TrimSpace(c.Query("table_type")))
	if tableType == "" {
		tableType = "BASE TABLE"
	}
```

Add to request:

```go
	req := domain.ListDatabaseTablesRequest{
		Schema:    strings.TrimSpace(c.Query("schema")),
		Page:      page,
		PageSize:  pageSize,
		TableType: tableType,
	}
```

- [ ] **Step 5: Update fakes to match new signature**

Update `fakeSchemaInspector.ListTables` and `handlerDatabaseTester.ListTables` to accept `tableType string` parameter.

- [ ] **Step 6: Run tests**

Run: `cd backend && go test ./internal/app/db/ -v && go test ./internal/delivery/http/db/ -v`
Expected: All pass.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/domain/db/database.go backend/internal/app/db/schema_inspector.go backend/internal/app/db/database_service_introspection.go backend/internal/delivery/http/db/database_handler.go backend/internal/app/db/database_service_test.go backend/internal/delivery/http/db/database_handler_test.go
git commit -m "feat: add table_type filter for views support in ListTables"
```

---

### Task 12: Handler Test Compile Fix (Phase 3C)

**Files:**
- Modify: `backend/internal/delivery/http/db/database_handler_test.go`

- [ ] **Step 1: Add `GetPinned` stub to `handlerConnectionPool`**

In `database_handler_test.go`, after line 184 (`Shutdown` method), add:

```go
func (handlerConnectionPool) GetPinned(_ context.Context, _ uint, _ string) (*sql.Conn, error) {
	return nil, nil
}
```

Add import: `"database/sql"` at top of imports.

- [ ] **Step 2: Run handler tests**

Run: `cd backend && go test ./internal/delivery/http/db/ -v`
Expected: All compile and pass.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/db/database_handler_test.go
git commit -m "fix: add GetPinned stub to handlerConnectionPool for test compilation"
```

---

### Task 13: Full Test Suite Verification

- [ ] **Step 1: Run all service tests**

Run: `cd backend && go test ./internal/app/db/ -v -count=1`
Expected: All pass.

- [ ] **Step 2: Run all handler tests**

Run: `cd backend && go test ./internal/delivery/http/db/ -v -count=1`
Expected: All pass.

- [ ] **Step 3: Build the project**

Run: `cd backend && go build ./...`
Expected: No errors.

- [ ] **Step 4: Commit if any cleanups needed**

```bash
git status
```

---

## Success Criteria Verification

| # | Criteria | How to verify |
|---|----------|---------------|
| 1 | Non-admin 403 on create | `TestDatabaseService_CreateDatabaseNonAdminReturnsForbidden` passes |
| 2 | Non-owner 403 on update/delete | `TestDatabaseService_UpdateDatabaseNonOwnerReturnsForbidden` + delete test pass |
| 3 | Running queries → 409 | `TestDatabaseService_DeleteDatabaseHasRunningQueriesReturns409` passes |
| 4 | Datasets → `{"datasets": [...]}` | `TestDatabaseHandler_DeleteReturns409WithDatasetList` passes |
| 5 | MySQL/BQ/Snowflake drivers | `TestResolveSQLDriver_SupportsMySQL` etc. pass |
| 6 | Multi-driver inspectors | Schema inspector types exist for all drivers |
| 7 | Redis cache invalidation | `TestDatabaseService_UpdateDatabaseFlushesSchemaCache` passes |
| 8 | All tests pass | `go test ./...` in backend |
| 9 | No plaintext in logs | log line removed |
| 10 | Handler tests compile | `go test ./internal/delivery/http/db/` compiles |
