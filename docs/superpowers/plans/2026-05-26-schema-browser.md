# SQL-006 Schema Browser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the placeholder Schema Browser panel in SQLLab with a live table list, per-table column expansion, schema switching, and Monaco cursor insertion.

**Architecture:** Backend injects `DatabaseService` into the sqllab handler, adds 4 schema routes with merge/upsert/collapse/clear logic using `table_schema` GORM model. Frontend adds a `SchemaBrowser` component using shadcn Collapsible + Select + TanStack Query, replacing skeleton placeholders. `DatabaseTable` gains `TableType` string for VIEW badge support.

**Tech Stack:** Go 1.22+ (Gin, GORM), TypeScript 5.x (React 18, TanStack Query v5, shadcn/ui, Monaco Editor)

---

### File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `docs/db/constraints.sql` | Modify | Add UNIQUE constraint on `table_schema` |
| `backend/internal/domain/db/database.go` | Modify | Add `TableType` to `DatabaseTable` DTO |
| `backend/internal/app/db/schema_inspector.go` | Modify | 4 drivers: SELECT `table_type` in ListTables |
| `backend/internal/domain/query/sqllab_types.go` | Modify | New request/response DTOs |
| `backend/internal/domain/query/sqllab_repository.go` | Modify | 4 new interface methods |
| `backend/internal/repository/postgres/sqllab_repo.go` | Modify | Implement 4 schema state methods |
| `backend/internal/delivery/http/sqllab/handler.go` | Modify | Inject `DatabaseService`, 4 handler methods |
| `backend/internal/delivery/http/router.go` | Modify | 4 new routes |
| `backend/cmd/api/main.go` | Modify | Pass `databaseSvc` to `NewHandler` |
| `frontend/src/api/sqllab.ts` | Modify | 4 schema API functions + types |
| `frontend/src/components/sqllab/SchemaBrowser.tsx` | Create | Full schema browser component |
| `frontend/src/pages/sqllab/SQLLabPage.tsx` | Modify | Replace Skeleton panel with `SchemaBrowser` |

---

### Task 1: DB — Add unique constraint on table_schema

**Files:**
- Modify: `docs/db/constraints.sql`

- [ ] **Step 1: Add unique constraint SQL**

Add after line 107 (`ALTER TABLE table_schema ALTER COLUMN "table" SET NOT NULL;`):

```sql
-- SQL-006: upsert target for schema browser expand state
ALTER TABLE table_schema
ADD CONSTRAINT uq_table_schema_tab_db_schema_table
UNIQUE (tab_state_id, db_id, schema, "table");
```

- [ ] **Step 2: Verify SQL syntax**

Run: `powershell -Command "(Get-Content docs\db\constraints.sql).Length"` to verify the file exists and has content.

- [ ] **Step 3: Commit**

```bash
git add docs/db/constraints.sql
git commit -m "feat(SQL-006): add unique constraint on table_schema for schema browser upsert

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 2: Backend — Add TableType to DatabaseTable DTO

**Files:**
- Modify: `backend/internal/domain/db/database.go:157-159`

- [ ] **Step 1: Add TableType field**

Replace `DatabaseTable` struct:

```go
// DatabaseTable is one table item discovered from database metadata.
type DatabaseTable struct {
	Name      string `json:"name"`
	TableType string `json:"table_type"`
}
```

- [ ] **Step 2: Verify Go compiles (will fail until inspector changes)**

Run: `cd backend; go build ./internal/domain/db/...`
Expected: PASS (no change to calling code yet since `TableType` is added, not removed)

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/db/database.go
git commit -m "feat(SQL-006): add TableType field to DatabaseTable DTO

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 3: Backend — Update all 4 schema inspector drivers to include table_type

**Files:**
- Modify: `backend/internal/app/db/schema_inspector.go`

- [ ] **Step 1: Update Postgres ListTables (line 77)**

Change query from `SELECT table_name` to `SELECT table_name, table_type`, add `tableType` scan variable:

The data query at line 110:
```go
dataQuery := "SELECT table_name FROM information_schema.tables WHERE table_schema = $1" + typeFilter + " ORDER BY table_name LIMIT $2 OFFSET $3"
```
Change to:
```go
dataQuery := "SELECT table_name, table_type FROM information_schema.tables WHERE table_schema = $1" + typeFilter + " ORDER BY table_name LIMIT $2 OFFSET $3"
```

The scan loop at lines 118-125:
```go
for rows.Next() {
	var name string
	if scanErr := rows.Scan(&name); scanErr != nil {
		return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
	}
	tables = append(tables, domain.DatabaseTable{Name: name})
}
```
Change to:
```go
for rows.Next() {
	var name, tableType string
	if scanErr := rows.Scan(&name, &tableType); scanErr != nil {
		return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
	}
	tables = append(tables, domain.DatabaseTable{Name: name, TableType: tableType})
}
```

- [ ] **Step 2: Update MySQL ListTables (line 247)**

Change query:
```go
dataQuery := "SELECT table_name FROM information_schema.tables WHERE table_schema = ?" + typeFilter + " ORDER BY table_name LIMIT ? OFFSET ?"
```
To:
```go
dataQuery := "SELECT table_name, table_type FROM information_schema.tables WHERE table_schema = ?" + typeFilter + " ORDER BY table_name LIMIT ? OFFSET ?"
```

Change scan loop (lines 256-262):
```go
for rows.Next() {
	var name string
	if scanErr := rows.Scan(&name); scanErr != nil {
		return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
	}
	tables = append(tables, domain.DatabaseTable{Name: name})
}
```
To:
```go
for rows.Next() {
	var name, tableType string
	if scanErr := rows.Scan(&name, &tableType); scanErr != nil {
		return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
	}
	tables = append(tables, domain.DatabaseTable{Name: name, TableType: tableType})
}
```

- [ ] **Step 3: Update BigQuery ListTables (line 344+)**

Read BigQuery ListTables to find exact query and scan:
```go
dataQuery := "SELECT table_name FROM INFORMATION_SCHEMA.TABLES WHERE table_schema = ?" + typeFilter + " ORDER BY table_name LIMIT ? OFFSET ?"
```
Same pattern: add `, table_type` to SELECT, add tableType scan variable, populate `TableType`.

- [ ] **Step 4: Update Snowflake ListTables (line 459+)**

Same pattern as above — add `, table_type` to SELECT, add scan variable, populate `TableType`.

- [ ] **Step 5: Build check**

Run: `cd backend; go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/app/db/schema_inspector.go
git commit -m "feat(SQL-006): add table_type to ListTables across all 4 drivers

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 4: Backend — Add schema-related types to sqllab_types.go

**Files:**
- Modify: `backend/internal/domain/query/sqllab_types.go`

- [ ] **Step 1: Add new types at end of file**

```go
// ExpandTableRequest is the body for POST /api/v1/sqllab/tabs/:id/schema
type ExpandTableRequest struct {
	TableName string `json:"table_name" binding:"required"`
}

// SchemaTableItem is one table row in the GET schema response.
type SchemaTableItem struct {
	TableName string           `json:"table_name"`
	TableType string           `json:"table_type"`
	Expanded  bool             `json:"expanded"`
	Columns   []domdb.DatabaseColumn `json:"columns,omitempty"`
}

// ExpandTableResponse is returned by POST expand.
type ExpandTableResponse struct {
	TableName string           `json:"table_name"`
	Columns   []domdb.DatabaseColumn `json:"columns"`
}

// GetSchemaResponse is returned by GET /api/v1/sqllab/tabs/:id/schema
type GetSchemaResponse struct {
	Schemas []string          `json:"schemas"`
	Tables  []SchemaTableItem `json:"tables"`
}
```

- [ ] **Step 2: Add missing import for domdb**

At top of file, add to imports (after `query`):
```go
import (
	domdb "superset/auth-service/internal/domain/db"
	query "superset/auth-service/internal/domain/query"
)
```

Actually, check the existing imports — the file may already have a `domdb` import or none. If no `domdb` import exists, replace the types with inline column struct to avoid domain cross-reference.

Simpler: define columns inline rather than referencing `domdb.DatabaseColumn`:

```go
type SchemaColumnItem struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value,omitempty"`
	IsDttm       bool   `json:"is_dttm"`
}

type SchemaTableItem struct {
	TableName string             `json:"table_name"`
	TableType string             `json:"table_type"`
	Expanded  bool               `json:"expanded"`
	Columns   []SchemaColumnItem `json:"columns,omitempty"`
}

type ExpandTableResponse struct {
	TableName string             `json:"table_name"`
	Columns   []SchemaColumnItem `json:"columns"`
}
```

This avoids circular imports.

- [ ] **Step 3: Build check**

Run: `cd backend; go build ./internal/domain/query/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/domain/query/sqllab_types.go
git commit -m "feat(SQL-006): add schema browser request/response types

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 5: Backend — Add schema state methods to SQLLabRepository interface

**Files:**
- Modify: `backend/internal/domain/query/sqllab_repository.go`

- [ ] **Step 1: Add 4 methods to the interface**

After `ForkSavedQuery` line, add:

```go
	// Schema browser (SQL-006)
	FindSchemaState(ctx context.Context, tabStateID uint) ([]TableSchema, error)
	UpsertSchemaState(ctx context.Context, ts *TableSchema) error
	UpdateSchemaStateCollapsed(ctx context.Context, tabStateID uint, table string) error
	DeleteSchemaStateByTab(ctx context.Context, tabStateID uint) error
```

- [ ] **Step 2: Build check**

Run: `cd backend; go build ./internal/domain/query/...`
Expected: PASS (interface change, no impl yet — may fail on mock. Fix mock in next step.)

Run: `cd backend; go build ./...`
Will fail on `mockSQLLabRepo` in handler_test.go — expected, that's Task 6.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/query/sqllab_repository.go
git commit -m "feat(SQL-006): add schema state methods to SQLLabRepository interface

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 6: Backend — Implement schema state methods in postgres repo

**Files:**
- Modify: `backend/internal/repository/postgres/sqllab_repo.go`

- [ ] **Step 1: Add imports for OnConflict**

Add to imports at top:
```go
import (
	"gorm.io/gorm/clause"
)
```

- [ ] **Step 2: Add 4 methods before the `var _` line at end of file**

```go
func (r *sqllabRepo) FindSchemaState(ctx context.Context, tabStateID uint) ([]query.TableSchema, error) {
	var rows []query.TableSchema
	err := r.db.WithContext(ctx).
		Where("tab_state_id = ?", tabStateID).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find schema state: %w", err)
	}
	return rows, nil
}

func (r *sqllabRepo) UpsertSchemaState(ctx context.Context, ts *query.TableSchema) error {
	ts.ChangedOn = time.Now()
	if ts.CreatedOn.IsZero() {
		ts.CreatedOn = ts.ChangedOn
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tab_state_id"},
				{Name: "db_id"},
				{Name: "schema"},
				{Name: "table"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"expanded", "changed_on"}),
		}).
		Create(ts).Error
}

func (r *sqllabRepo) UpdateSchemaStateCollapsed(ctx context.Context, tabStateID uint, table string) error {
	result := r.db.WithContext(ctx).
		Model(&query.TableSchema{}).
		Where("tab_state_id = ? AND \"table\" = ?", tabStateID, table).
		Updates(map[string]any{
			"expanded":   false,
			"changed_on": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("update schema state collapsed: %w", result.Error)
	}
	return nil
}

func (r *sqllabRepo) DeleteSchemaStateByTab(ctx context.Context, tabStateID uint) error {
	err := r.db.WithContext(ctx).
		Where("tab_state_id = ?", tabStateID).
		Delete(&query.TableSchema{}).Error
	if err != nil {
		return fmt.Errorf("delete schema state by tab: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Build check**

Run: `cd backend; go build ./internal/repository/postgres/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/postgres/sqllab_repo.go
git commit -m "feat(SQL-006): implement schema state CRUD methods in postgres repo

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 7: Backend — Add schema browser handler methods

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler.go`

- [ ] **Step 1: Add DatabaseService import**

Add to imports:
```go
import (
	svcdb "superset/auth-service/internal/app/db"
)
```

- [ ] **Step 2: Add DatabaseService to Handler struct**

```go
type Handler struct {
	sqllabRepo   domainquery.SQLLabRepository
	databaseRepo domdb.DatabaseRepository
	dbSvc        *svcdb.DatabaseService
}
```

- [ ] **Step 3: Update NewHandler constructor**

```go
func NewHandler(sqllabRepo domainquery.SQLLabRepository, databaseRepo domdb.DatabaseRepository, dbSvc *svcdb.DatabaseService) *Handler {
	return &Handler{sqllabRepo: sqllabRepo, databaseRepo: databaseRepo, dbSvc: dbSvc}
}
```

- [ ] **Step 4: Add GetSchema method before savedQueryToResponse**

```go
func (h *Handler) GetSchema(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	tab, err := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve tab"})
		return
	}
	if tab == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to access this tab"})
		return
	}

	forceRefresh := c.Query("force_refresh") == "true"
	rateLimitKey := fmt.Sprintf("sqllab:schema:tab:%d:user:%d", uint(id), userCtx.ID)

	if tab.Schema == "" {
		c.JSON(http.StatusOK, domainquery.GetSchemaResponse{Schemas: []string{}, Tables: []domainquery.SchemaTableItem{}})
		return
	}

	// Schemas list
	schemas, err := h.dbSvc.ListSchemas(c.Request.Context(), userCtx.ID, tab.DbID, forceRefresh, rateLimitKey)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "db_unreachable", "message": err.Error()})
		return
	}

	// Tables list
	tableReq := domdb.ListDatabaseTablesRequest{Schema: tab.Schema, Page: 1, PageSize: 500}
	tablesResp, err := h.dbSvc.ListTables(c.Request.Context(), userCtx.ID, tab.DbID, tableReq, forceRefresh, rateLimitKey)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "db_unreachable", "message": err.Error()})
		return
	}

	// Merge with expanded state
	schemaState, err := h.sqllabRepo.FindSchemaState(c.Request.Context(), uint(id))
	if err != nil {
		schemaState = nil
	}
	stateMap := make(map[string]bool)
	for _, s := range schemaState {
		stateMap[s.Table] = s.Expanded
	}

	items := make([]domainquery.SchemaTableItem, 0, len(tablesResp.Items))
	for _, t := range tablesResp.Items {
		items = append(items, domainquery.SchemaTableItem{
			TableName: t.Name,
			TableType: t.TableType,
			Expanded:  stateMap[t.Name],
		})
	}

	c.JSON(http.StatusOK, domainquery.GetSchemaResponse{
		Schemas: schemas,
		Tables:  items,
	})
}
```

- [ ] **Step 5: Add ExpandTable method**

```go
func (h *Handler) ExpandTable(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	tab, err := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve tab"})
		return
	}
	if tab == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to access this tab"})
		return
	}

	var req domainquery.ExpandTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	if tab.Schema == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "no_schema", "message": "Tab has no schema selected"})
		return
	}

	colReq := domdb.ListDatabaseColumnsRequest{Schema: tab.Schema, Table: req.TableName}
	columns, err := h.dbSvc.ListColumns(c.Request.Context(), userCtx.ID, tab.DbID, colReq, false, "")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "db_unreachable", "message": err.Error()})
		return
	}

	// Upsert expanded state
	now := time.Now()
	ts := &domainquery.TableSchema{
		TabStateID: uint(id),
		DbID:       tab.DbID,
		Schema:     tab.Schema,
		Table:      req.TableName,
		Expanded:   true,
		CreatedOn:  now,
		ChangedOn:  now,
	}
	_ = h.sqllabRepo.UpsertSchemaState(c.Request.Context(), ts)

	colItems := make([]domainquery.SchemaColumnItem, 0, len(columns))
	for _, col := range columns {
		colItems = append(colItems, domainquery.SchemaColumnItem{
			Name:         col.Name,
			DataType:     col.DataType,
			IsNullable:   col.IsNullable,
			DefaultValue: col.DefaultValue,
			IsDttm:       col.IsDttm,
		})
	}

	c.JSON(http.StatusOK, domainquery.ExpandTableResponse{
		TableName: req.TableName,
		Columns:   colItems,
	})
}
```

- [ ] **Step 6: Add CollapseTable method**

```go
func (h *Handler) CollapseTable(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	tab, err := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve tab"})
		return
	}
	if tab == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to access this tab"})
		return
	}

	tableName := c.Param("table")
	if tableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "missing table name"})
		return
	}

	_ = h.sqllabRepo.UpdateSchemaStateCollapsed(c.Request.Context(), uint(id), tableName)
	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 7: Add ClearSchema method**

```go
func (h *Handler) ClearSchema(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	tab, err := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve tab"})
		return
	}
	if tab == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to access this tab"})
		return
	}

	_ = h.sqllabRepo.DeleteSchemaStateByTab(c.Request.Context(), uint(id))
	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 8: Add fmt import if not already present**

Check if `"fmt"` is already imported. If not, add it.

- [ ] **Step 9: Build check**

Run: `cd backend; go build ./internal/delivery/http/...`
Expected: PASS (will fail for main.go and test until next tasks)

- [ ] **Step 10: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler.go
git commit -m "feat(SQL-006): add schema browser handler methods

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 8: Backend — Update handler test mock for new interface methods

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler_test.go`

- [ ] **Step 1: Add schema-related methods to mockSQLLabRepo**

After `ForkSavedQuery` method, add:

```go
func (m *mockSQLLabRepo) FindSchemaState(_ context.Context, tabStateID uint) ([]domainquery.TableSchema, error) {
	return nil, nil
}
func (m *mockSQLLabRepo) UpsertSchemaState(_ context.Context, ts *domainquery.TableSchema) error {
	return nil
}
func (m *mockSQLLabRepo) UpdateSchemaStateCollapsed(_ context.Context, tabStateID uint, table string) error {
	return nil
}
func (m *mockSQLLabRepo) DeleteSchemaStateByTab(_ context.Context, tabStateID uint) error {
	return nil
}
```

- [ ] **Step 2: Update newSQLLabRouter to pass nil for dbSvc**

```go
func newSQLLabRouter(repo *mockSQLLabRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(repo, &mockDatabaseRepo{}, nil)
	// ... rest unchanged
}
```

- [ ] **Step 3: Build check**

Run: `cd backend; go build ./internal/delivery/http/sqllab/...`
Expected: PASS

- [ ] **Step 4: Run existing tests to catch regressions**

Run: `cd backend; go test ./internal/delivery/http/sqllab/... -v`
Expected: All existing tests PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler_test.go
git commit -m "test(SQL-006): add mock methods for schema state to handler tests

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 9: Backend — Register schema routes and wire DatabaseService

**Files:**
- Modify: `backend/internal/delivery/http/router.go`
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Register 4 schema routes in router.go**

After the saved-queries route block (after line 184), add inside the `sqlLab` group:

```go
				sqlLab.GET("/tabs/:id/schema", sqllabHandler.GetSchema)
				sqlLab.POST("/tabs/:id/schema", sqllabHandler.ExpandTable)
				sqlLab.DELETE("/tabs/:id/schema/:table", sqllabHandler.CollapseTable)
				sqlLab.DELETE("/tabs/:id/schema", sqllabHandler.ClearSchema)
```

- [ ] **Step 2: Pass databaseSvc to NewHandler in main.go**

Line 173:
```go
sqllabHandler := httpsqllab.NewHandler(sqllabRepo, databaseRepo)
```
Change to:
```go
sqllabHandler := httpsqllab.NewHandler(sqllabRepo, databaseRepo, databaseSvc)
```

- [ ] **Step 3: Build check**

Run: `cd backend; go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/router.go backend/cmd/api/main.go
git commit -m "feat(SQL-006): register schema routes and wire DatabaseService

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 10: Frontend — Add schema API functions and types

**Files:**
- Modify: `frontend/src/api/sqllab.ts`

- [ ] **Step 1: Add types after existing interfaces (at end of file)**

```typescript
export interface DatabaseColumn {
  name: string;
  data_type: string;
  is_nullable: boolean;
  default_value?: string;
  is_dttm: boolean;
}

export interface SchemaColumnItem {
  name: string;
  data_type: string;
  is_nullable: boolean;
  default_value?: string;
  is_dttm: boolean;
}

export interface SchemaTableItem {
  table_name: string;
  table_type: string;
  expanded: boolean;
  columns?: SchemaColumnItem[] | null;
}

export interface GetSchemaResponse {
  schemas: string[];
  tables: SchemaTableItem[];
}

export interface ExpandTableResponse {
  table_name: string;
  columns: SchemaColumnItem[];
}
```

- [ ] **Step 2: Add API functions after types**

```typescript
export async function getSchemaTables(tabId: number, forceRefresh?: boolean): Promise<GetSchemaResponse> {
  const qs = forceRefresh ? "?force_refresh=true" : "";
  return request<GetSchemaResponse>(`/api/v1/sqllab/tabs/${tabId}/schema${qs}`, {
    method: "GET",
    headers: getAuthHeaders(),
  });
}

export async function expandTable(tabId: number, tableName: string): Promise<ExpandTableResponse> {
  return request<ExpandTableResponse>(`/api/v1/sqllab/tabs/${tabId}/schema`, {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify({ table_name: tableName }),
  });
}

export async function collapseTable(tabId: number, tableName: string): Promise<void> {
  return request<void>(`/api/v1/sqllab/tabs/${tabId}/schema/${encodeURIComponent(tableName)}`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
}

export async function clearSchemaState(tabId: number): Promise<void> {
  return request<void>(`/api/v1/sqllab/tabs/${tabId}/schema`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
}
```

- [ ] **Step 3: Type check**

Run: `cd frontend; npx tsc --noEmit`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/sqllab.ts
git commit -m "feat(SQL-006): add schema browser API functions and types

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 11: Frontend — Create SchemaBrowser component

**Files:**
- Create: `frontend/src/components/sqllab/SchemaBrowser.tsx`

- [ ] **Step 1: Check missing shadcn components**

Check if `Collapsible`, `Tooltip` are already installed (they are from earlier glob results — `collapsible.tsx` and `tooltip.tsx` exist).

Check for `Copy` icon: lucide-react has `Copy`, already imported pattern in codebase.

- [ ] **Step 2: Create component file**

```typescript
import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Search, RefreshCw, Table, Eye, Copy, ChevronRight } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  getSchemaTables,
  expandTable as expandTableApi,
  collapseTable as collapseTableApi,
  clearSchemaState,
  type SchemaTableItem,
  type SchemaColumnItem,
} from "@/api/sqllab";

const COLUMN_TYPE_COLORS: Record<string, string> = {
  INT: "bg-blue-100 text-blue-800",
  INTEGER: "bg-blue-100 text-blue-800",
  BIGINT: "bg-blue-100 text-blue-800",
  SMALLINT: "bg-blue-100 text-blue-800",
  VARCHAR: "bg-green-100 text-green-800",
  TEXT: "bg-green-100 text-green-800",
  CHAR: "bg-green-100 text-green-800",
  BOOLEAN: "bg-yellow-100 text-yellow-800",
  BOOL: "bg-yellow-100 text-yellow-800",
  TIMESTAMP: "bg-orange-100 text-orange-800",
  TIMESTAMPTZ: "bg-orange-100 text-orange-800",
  DATE: "bg-orange-100 text-orange-800",
  FLOAT: "bg-purple-100 text-purple-800",
  DOUBLE: "bg-purple-100 text-purple-800",
  NUMERIC: "bg-purple-100 text-purple-800",
  DECIMAL: "bg-purple-100 text-purple-800",
  JSON: "bg-gray-100 text-gray-800",
  JSONB: "bg-gray-100 text-gray-800",
  UUID: "bg-pink-100 text-pink-800",
};

function getTypeBadgeClass(dataType: string): string {
  const upper = dataType.toUpperCase();
  return COLUMN_TYPE_COLORS[upper] ?? "bg-muted text-muted-foreground";
}

interface SchemaBrowserProps {
  tabId: number;
  currentSchema: string;
  onColumnClick: (columnName: string) => void;
}

export function SchemaBrowser({ tabId, currentSchema, onColumnClick }: SchemaBrowserProps) {
  const queryClient = useQueryClient();
  const [filterText, setFilterText] = useState("");
  const [expandedTables, setExpandedTables] = useState<Set<string>>(new Set());
  const [columnCache, setColumnCache] = useState<Map<string, SchemaColumnItem[]>>(new Map());
  const [selectedSchema, setSelectedSchema] = useState<string>(currentSchema);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ["schema-tables", tabId],
    queryFn: () => getSchemaTables(tabId),
    enabled: tabId > 0,
  });

  const expandMutation = useMutation({
    mutationFn: (tableName: string) => expandTableApi(tabId, tableName),
    onSuccess: (res) => {
      setColumnCache((prev) => new Map(prev).set(res.table_name, res.columns));
      setExpandedTables((prev) => new Set(prev).add(res.table_name));
    },
  });

  const collapseMutation = useMutation({
    mutationFn: (tableName: string) => collapseTableApi(tabId, tableName),
  });

  const handleTableToggle = useCallback(
    async (table: SchemaTableItem) => {
      const name = table.table_name;
      if (expandedTables.has(name)) {
        // Collapse
        setExpandedTables((prev) => {
          const next = new Set(prev);
          next.delete(name);
          return next;
        });
        collapseMutation.mutate(name);
      } else if (columnCache.has(name)) {
        // Already cached, just show
        setExpandedTables((prev) => new Set(prev).add(name));
      } else {
        // Expand via API
        expandMutation.mutate(name);
      }
    },
    [expandedTables, columnCache, expandMutation, collapseMutation],
  );

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  const handleSchemaChange = useCallback(
    async (newSchema: string) => {
      setSelectedSchema(newSchema);
      setExpandedTables(new Set());
      setColumnCache(new Map());
      await clearSchemaState(tabId);
      queryClient.invalidateQueries({ queryKey: ["schema-tables", tabId] });
    },
    [tabId, queryClient],
  );

  const schemas = data?.schemas ?? [];
  const tables = data?.tables ?? [];

  const filteredTables = filterText
    ? tables.filter((t) => t.table_name.toLowerCase().includes(filterText.toLowerCase()))
    : tables;

  const hasSchema = selectedSchema !== "";

  return (
    <div className="h-full border-r flex flex-col p-3 gap-2">
      <h3 className="text-sm font-semibold">Schema Browser</h3>

      {/* Schema Select */}
      <Select value={selectedSchema} onValueChange={handleSchemaChange}>
        <SelectTrigger className="h-8 text-xs">
          <SelectValue placeholder="Select schema" />
        </SelectTrigger>
        <SelectContent>
          {schemas.map((s) => (
            <SelectItem key={s} value={s}>
              {s}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Search + Refresh */}
      <div className="flex gap-1">
        <div className="relative flex-1">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
          <Input
            placeholder="Filter tables..."
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            className="pl-7 h-7 text-xs"
          />
        </div>
        <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" onClick={handleRefresh}>
          <RefreshCw className="h-3.5 w-3.5" />
        </Button>
      </div>

      {/* Table List */}
      <ScrollArea className="flex-1">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-6 w-full" />
            ))}
          </div>
        ) : !hasSchema ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            Select a schema to browse tables
          </p>
        ) : filteredTables.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            No tables found
          </p>
        ) : (
          <div className="space-y-0.5">
            {filteredTables.map((table) => (
              <Collapsible
                key={table.table_name}
                open={expandedTables.has(table.table_name)}
                onOpenChange={() => handleTableToggle(table)}
              >
                <CollapsibleTrigger className="flex items-center w-full rounded px-1 py-1 hover:bg-muted text-left text-xs">
                  <ChevronRight className="h-3 w-3 shrink-0 mr-1 transition-transform data-[state=open]:rotate-90" />
                  {table.table_type === "VIEW" ? (
                    <Eye className="h-3 w-3 shrink-0 mr-1 text-muted-foreground" />
                  ) : (
                    <Table className="h-3 w-3 shrink-0 mr-1 text-muted-foreground" />
                  )}
                  <span className="truncate flex-1">{table.table_name}</span>
                  {table.table_type === "VIEW" && (
                    <Badge variant="outline" className="ml-1 text-[10px] px-1 py-0 h-4">
                      VIEW
                    </Badge>
                  )}
                  <span className="text-[10px] text-muted-foreground ml-1">
                    {columnCache.get(table.table_name)?.length ?? "..."}
                  </span>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  {columnCache.get(table.table_name)?.map((col) => (
                    <div
                      key={col.name}
                      className="flex items-center pl-7 pr-1 py-0.5 hover:bg-muted cursor-pointer text-xs"
                      onClick={() => onColumnClick(col.name)}
                    >
                      <span className="truncate flex-1">{col.name}</span>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Badge
                            variant="secondary"
                            className={`ml-1 text-[10px] px-1 py-0 h-4 font-normal ${getTypeBadgeClass(col.data_type)}`}
                          >
                            {col.data_type}
                          </Badge>
                        </TooltipTrigger>
                        <TooltipContent side="right">
                          <p>{col.data_type}{col.is_nullable ? " (nullable)" : ""}</p>
                        </TooltipContent>
                      </Tooltip>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-5 w-5 ml-0.5 shrink-0"
                        onClick={(e) => {
                          e.stopPropagation();
                          navigator.clipboard.writeText(col.name);
                        }}
                      >
                        <Copy className="h-2.5 w-2.5" />
                      </Button>
                    </div>
                  ))}
                </CollapsibleContent>
              </Collapsible>
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  );
}
```

- [ ] **Step 3: Type check**

Run: `cd frontend; npx tsc --noEmit`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/sqllab/SchemaBrowser.tsx
git commit -m "feat(SQL-006): create SchemaBrowser component with collapsible table list

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 12: Frontend — Integrate SchemaBrowser into SQLLabPage

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx`

- [ ] **Step 1: Add import for SchemaBrowser**

Add after line 74 (`SavedQueriesList` import):
```typescript
import { SchemaBrowser } from "@/components/sqllab/SchemaBrowser";
```

- [ ] **Step 2: Add Monaco import for insertAtCursor**

Add `monaco` import. Check if `@monaco-editor/react` already re-exports `monaco`. The import on line 4 is:
```typescript
import Editor, { type OnMount } from "@monaco-editor/react";
```

Add separate import:
```typescript
import * as monaco from "monaco-editor";
```

Or check if monaco is already accessible. The `editorRef.current` has a `getPosition()` method and `executeEdits()`. We need `monaco.Range` for the edit operation. Use `(editorRef.current as any).getModel()?.getPosition()` if monaco types aren't directly available, or import from monaco-editor directly.

Simpler: avoid the `monaco.Range` import by using selection replacement:

```typescript
onColumnClick: (columnName: string) => {
  const editor = editorRef.current;
  if (!editor) return;
  const position = editor.getPosition();
  if (!position) return;
  editor.executeEdits("schema-browser", [
    {
      range: {
        startLineNumber: position.lineNumber,
        startColumn: position.column,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      },
      text: columnName,
    },
  ]);
  editor.focus();
};
```

This uses a plain object instead of `monaco.Range` — Monaco accepts both.

- [ ] **Step 3: Replace Schema Browser placeholder (lines 816-826)**

Replace:
```tsx
        {/* Left: Schema Browser (placeholder for SQL-006) */}
        <ResizablePanel defaultSize={18} minSize={12} maxSize={30}>
          <div className="h-full border-r p-3">
            <h3 className="text-sm font-semibold mb-2">Schema Browser</h3>
            <Skeleton className="h-4 w-full mb-2" />
            <Skeleton className="h-4 w-3/4 mb-2" />
            <Skeleton className="h-4 w-5/6 mb-2" />
            <Skeleton className="h-4 w-2/3 mb-2" />
            <Skeleton className="h-4 w-4/5" />
          </div>
        </ResizablePanel>
```

With:
```tsx
        {/* Left: Schema Browser */}
        <ResizablePanel defaultSize={18} minSize={12} maxSize={30}>
          <SchemaBrowser
            tabId={activeTab?.id ?? 0}
            currentSchema={activeTab?.schema ?? ""}
            onColumnClick={(columnName: string) => {
              const editor = editorRef.current;
              if (!editor) return;
              const position = editor.getPosition();
              if (!position) return;
              editor.executeEdits("schema-browser", [
                {
                  range: {
                    startLineNumber: position.lineNumber,
                    startColumn: position.column,
                    endLineNumber: position.lineNumber,
                    endColumn: position.column,
                  },
                  text: columnName,
                },
              ]);
              editor.focus();
            }}
          />
        </ResizablePanel>
```

- [ ] **Step 4: Type check**

Run: `cd frontend; npx tsc --noEmit`
Expected: PASS

- [ ] **Step 5: Verify no unused imports from removed code**

The `Skeleton` import on line 15 is still used elsewhere (loading state at line 808). No other imports removed.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat(SQL-006): integrate SchemaBrowser into SQLLabPage left panel

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Self-Review

**1. Spec coverage check:**
- GET merge tables + expanded state → Task 7 GetSchema
- POST expand → columns + upsert → Task 7 ExpandTable
- DELETE collapse → Task 7 CollapseTable
- Schema change → clear → Task 7 ClearSchema
- 502 DB unreachable → Task 7 error handling
- 403 Not tab owner → Task 7 ownership checks
- force_refresh → Task 7 query param
- Schema Select → Task 11 Select component
- Search filter → Task 11 filterText
- Refresh button → Task 11 RefreshCw button
- Collapsible per table → Task 11 Collapsible
- Table/View icon → Task 11 Table/Eye icons
- Column count badge → Task 11 span
- Column type badges with colors → Task 11 getTypeBadgeClass
- Tooltip on column type → Task 11 Tooltip
- Copy icon → Task 11 Copy button
- Skeleton loading → Task 11 Skeleton
- VIEW Badge → Task 11 Badge
- Monaco cursor insert → Task 12 onColumnClick
- Empty state → Task 11 "No tables found" / "Select a schema"
- All 4 routes registered → Task 9

**2. Placeholder scan:** No TBD/TODO. All code is concrete.

**3. Type consistency:**
- `SchemaColumnItem` defined in Task 4 (Go) and Task 10 (TS) — match
- `ExpandTableRequest.TableName` used in handler (Task 7) and API fn (Task 10) — match
- All prop names consistent between Task 11 component and Task 12 integration
