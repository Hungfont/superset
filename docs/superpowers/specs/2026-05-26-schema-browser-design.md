# SQL-006 — Schema Browser Design

## Overview

Left panel schema browser for SQLLab. Lists tables with expand/collapse for columns. Merges DBC-007 introspection with per-user `table_schema` expanded state.

**Depends on:** DBC-007 (schema cached in Redis), SQL-001 (tab DB/schema context)

---

## Routes

| Method | Route | Handler | Purpose |
|--------|-------|---------|---------|
| GET | `/api/v1/sqllab/tabs/:id/schema` | `GetSchema` | Schemas list + tables with expanded state |
| POST | `/api/v1/sqllab/tabs/:id/schema` | `ExpandTable` | Columns + upsert table_schema |
| DELETE | `/api/v1/sqllab/tabs/:id/schema/:table` | `CollapseTable` | Set expanded=false |
| DELETE | `/api/v1/sqllab/tabs/:id/schema` | `ClearSchema` | Delete all schema state for tab (schema switch) |

---

## Backend

### Domain changes

**`domain/db/database.go`** — Add `TableType` to `DatabaseTable`:
```go
type DatabaseTable struct {
    Name      string `json:"name"`
    TableType string `json:"table_type"`
}
```

**`domain/query/sqllab_types.go`** — New request/response types:
```go
type ExpandTableRequest struct {
    TableName string `json:"table_name" binding:"required"`
}
type SchemaTableItem struct {
    TableName string                `json:"table_name"`
    TableType string                `json:"table_type"`
    Expanded  bool                  `json:"expanded"`
    Columns   []domdb.DatabaseColumn `json:"columns,omitempty"`
}
type ExpandTableResponse struct {
    TableName string                `json:"table_name"`
    Columns   []domdb.DatabaseColumn `json:"columns"`
}
type GetSchemaResponse struct {
    Schemas []string          `json:"schemas"`
    Tables  []SchemaTableItem `json:"tables"`
}
```

### Repository interface (`sqllab_repository.go`)

New methods:
```go
FindSchemaState(ctx, tabStateID uint) ([]TableSchema, error)
UpsertSchemaState(ctx, ts *TableSchema) error
UpdateSchemaStateCollapsed(ctx, tabStateID uint, table string) error
DeleteSchemaStateByTab(ctx, tabStateID uint) error
```

### DB migration

New unique constraint for upsert to work:

```sql
ALTER TABLE table_schema
ADD CONSTRAINT uq_table_schema_tab_db_schema_table
UNIQUE (tab_state_id, db_id, schema, "table");
```

### Postgres implementation (`sqllab_repo.go`)

- `FindSchemaState`: `WHERE tab_state_id = ?`
- `UpsertSchemaState`: `Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tab_state_id"}, {Name: "db_id"}, {Name: "schema"}, {Name: "table"}}, DoUpdates: clause.AssignmentColumns([]string{"expanded", "changed_on"})}).Create(ts)`
- `UpdateSchemaStateCollapsed`: `Model(&TableSchema{}).Where("tab_state_id = ? AND \"table\" = ?", id, table).Updates(map[string]any{"expanded": false, "changed_on": time.Now()})`
- `DeleteSchemaStateByTab`: `WHERE tab_state_id = ? DELETE`

### Schema inspector (`schema_inspector.go`)

All 4 drivers (postgres, mysql, bigquery, snowflake): change `ListTables` query from `SELECT table_name` to `SELECT table_name, table_type`, update Scan to capture both, populate `DatabaseTable.TableType`.

### Handler (`handler.go`)

Handler struct gains `*DatabaseService`:
```go
type Handler struct {
    sqllabRepo   domainquery.SQLLabRepository
    databaseRepo domdb.DatabaseRepository
    dbSvc        *svcdb.DatabaseService
}
```

**GetSchema** — `GET /api/v1/sqllab/tabs/:id/schema`:
1. Parse `id`, resolve tab via `GetByID(id, userID)` → nil = 403
2. `DatabaseService.ListSchemas(dbID, forceRefresh)` → schemas list
3. `DatabaseService.ListTables(dbID, schema, page=1, pageSize=500)` → tables
4. `repo.FindSchemaState(tabID)` → map of expanded states
5. Merge: `stateMap := map[string]bool{}` from schemaState rows
6. Response: `{schemas: [...], tables: [{table_name, table_type, expanded, columns: null}]}`
7. Query param `?force_refresh=true` bypasses Redis cache

**ExpandTable** — `POST /api/v1/sqllab/tabs/:id/schema`:
1. Resolve tab → 403 if not owner
2. Bind `{table_name}` from body
3. `DatabaseService.ListColumns(dbID, schema, table)` → columns
4. Upsert: `repo.UpsertSchemaState({TabStateID, DbID, Schema, Table, Expanded:true})`
5. Response: `{table_name, columns: [...]}`

**CollapseTable** — `DELETE /api/v1/sqllab/tabs/:id/schema/:table`:
1. Resolve tab → 403 if not owner
2. `repo.UpdateSchemaStateCollapsed(tabID, table)`
3. 204

**ClearSchema** — `DELETE /api/v1/sqllab/tabs/:id/schema`:
1. Resolve tab → 403 if not owner
2. `repo.DeleteSchemaStateByTab(tabID)`
3. 204

### Constructor & wiring

`cmd/api/main.go`:
```go
sqllabHandler := httpsqllab.NewHandler(sqllabRepo, databaseRepo, databaseSvc)
```

`router.go` — register under existing `/sqllab` group:
```go
sqlLab.GET("/tabs/:id/schema", sqllabHandler.GetSchema)
sqlLab.POST("/tabs/:id/schema", sqllabHandler.ExpandTable)
sqlLab.DELETE("/tabs/:id/schema/:table", sqllabHandler.CollapseTable)
sqlLab.DELETE("/tabs/:id/schema", sqllabHandler.ClearSchema)
```

### Error responses

| Scenario | Code | Body |
|----------|------|------|
| Not tab owner | 403 | `{"error":"forbidden"}` |
| DB unreachable (from DatabaseService) | 502 | `{"error":"db_unreachable"}` |
| Invalid tab ID | 400 | `{"error":"invalid_request"}` |

---

## Frontend

### API (`api/sqllab.ts`)

```typescript
export interface SchemaTableItem {
  table_name: string;
  table_type: string;
  expanded: boolean;
  columns?: DatabaseColumn[] | null;
}
export interface GetSchemaResponse {
  schemas: string[];
  tables: SchemaTableItem[];
}
export interface ExpandTableResponse {
  table_name: string;
  columns: DatabaseColumn[];
}

export async function getSchemaTables(tabId: number, forceRefresh?: boolean): Promise<GetSchemaResponse>
export async function expandTable(tabId: number, tableName: string): Promise<ExpandTableResponse>
export async function collapseTable(tabId: number, tableName: string): Promise<void>
export async function clearSchemaState(tabId: number): Promise<void>
```

### SchemaBrowser component (`components/sqllab/SchemaBrowser.tsx`)

Props:
```typescript
interface SchemaBrowserProps {
  tabId: number;
  currentSchema: string;
  onColumnClick: (columnName: string) => void;
}
```

**Components used (shadcn/ui):**
- `Select` — schema picker at top
- `Input` + `Search` icon — client-side table filter
- `Button` (`RefreshCw` icon) — force refresh
- `ScrollArea` — table list container
- `Collapsible` + `CollapsibleTrigger` + `CollapsibleContent` — per table
- `CollapsibleTrigger` content: `Table`/`Eye` icon + table name + column count `Badge`
- `CollapsibleContent`: column list with type `Badge` labels
- `Tooltip` — full `data_type` on column hover
- `Button` (`Copy` icon, `size="xs"`) — copy column name
- `Skeleton` × 5 — loading state
- `Badge` (`VIEW`) — for view-type tables

**Data fetching:**
```typescript
const { data, isLoading } = useQuery({
  queryKey: ["schema-tables", tabId],
  queryFn: () => api.getSchemaTables(tabId),
});
const expandMutation = useMutation({
  mutationFn: (tableName: string) => api.expandTable(tabId, tableName),
});
const collapseMutation = useMutation({
  mutationFn: (tableName: string) => api.collapseTable(tabId, tableName),
});
```

**State:**
- `filterText` (local) — filters table list client-side
- `columnCache: Map<string, DatabaseColumn[]>` — cached columns per table (avoid re-fetching)
- `expandedTables: Set<string>` — local tracking mirrors server state

**UX behaviors:**
- Table Collapsible click → if not yet expanded, POST expand → cache columns → show; if already expanded → toggle show/hide; if collapsing → DELETE collapse
- Already expanded (from previous session): `expanded:true` in GET response → columns loaded immediately via expand on first open
- Column click → `onColumnClick(columnName)` → inserts at Monaco cursor
- Column type: small `Badge` with color coding (INT=blue, VARCHAR=green, TIMESTAMP=orange, etc.)
- Copy icon: `navigator.clipboard.writeText(columnName)` → Tooltip shows "Copied!"
- Refresh Button: refetch with `forceRefresh:true` → Skeleton during reload
- Schema Select change: clear state → DELETE clearSchema → refetch tables
- "No tables found" empty state: "Select a schema to browse tables"
- Loading: `Skeleton` × 5 rows

### SQLLabPage integration

Replace the placeholder `Skeleton` block (around lines 815-826) with:
```tsx
<SchemaBrowser
  tabId={activeTabId}
  currentSchema={activeTab?.schema ?? ""}
  onColumnClick={(col) => {
    const editor = editorRef.current;
    if (!editor) return;
    const position = editor.getPosition();
    if (!position) return;
    editor.executeEdits("schema-browser", [{
      range: new monaco.Range(position.lineNumber, position.column, position.lineNumber, position.column),
      text: col,
    }]);
    editor.focus();
  }}
/>
```

---

## Acceptance Criteria

- [ ] GET → returns tables with merged `expanded` state + schemas list
- [ ] POST expand → returns columns + upserts `table_schema`
- [ ] DELETE collapse → sets `expanded:false`
- [ ] Schema change (DELETE clear) → fresh table list, no stale state
- [ ] `force_refresh=true` → bypasses Redis cache
- [ ] 403 for non-tab-owner on all endpoints
- [ ] Table type (BASE TABLE / VIEW) shown in frontend Badge
- [ ] Column type badges with color coding
- [ ] Copy column name to clipboard
- [ ] Click column → insert at Monaco cursor
- [ ] Search filters tables client-side
- [ ] Refresh button forces cache refresh
- [ ] Empty state when no tables
- [ ] Loading skeletons during fetch
