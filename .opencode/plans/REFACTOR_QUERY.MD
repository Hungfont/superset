Implementation Plan: Query Model Alignment (QE-001 to QE-006)
High-Level Summary
3 major areas of work: (1) DB model alignment, (2) Backend refactoring, (3) Frontend type updates.
---
Phase 1: Database Model Alignment
1.1 Update Query Entity (backend/internal/domain/query/entity.go)
ID type change: string (varchar) → uint (auto-increment int, matching ER diagram). This is breaking — the async infrastructure currently uses string IDs. ClientID (UUID varchar) will become the external query identifier used in API URLs.
16 new fields to add:
Field	Go Type	GORM Tag
TabName	*string	varchar(255)
SqlEditorID	*string	varchar(36)
Catalog	*string	varchar(255)
SelectSQL	*string	text
Limit	int	—
LimitingFactor	int	—
SelectAsCTA	*int	—
SelectAsCTAUsed	bool	—
Progress	*string	varchar(100)
StartRunningTime	*time.Time	—
EndResultBackendTime	int	—
TmpTableName	*string	varchar(255)
TrackingURL	*string	varchar(500)
TmpSchemaName	bool	—
CachedData	*string	text
IsSaved	bool	—
ExtraJSON	*string	text
ChangedOn	*time.Time	—
Type corrections vs ER diagram typos:
- error_message: already string/text in Go (correct; ER diagram says int — likely diagram error, keep as text)
- tmp_table_name: use *string (ER says datetime — likely diagram error)
- tracking_url: use *string (ER says int — likely diagram error)
- end_result_backend_time: use int as shown
1.2 Create New Entities (Models Only)
SavedQuery — backend/internal/domain/query/saved_query.go:
Table: saved_query
Fields: id (uint PK), db_id, user_id, label, schema, catalog, sql, description,
        sql_tables, extra_json, published, created_on, changed_on,
        created_by_fk, changed_by_fk, tags
TabState — backend/internal/domain/query/tab_state.go:
Table: tab_state
Fields: id (uint PK), user_id, db_id, schema, catalog, label, active, sql,
        query_limit, latest_query_id (FK→query.id), hide_left_bar,
        saved_query_id (FK→saved_query.id), extra_json,
        created_on, changed_on, created_by_fk, changed_by_fk
TableSchema — backend/internal/domain/query/table_schema.go:
Table: table_schema
Fields: id (uint PK), tab_state_id (FK→tab_state.id), db_id, schema,
        catalog, table, description, expanded, created_on, changed_on
1.3 Update main.go Auto-Migrate
Add &domainquery.SavedQuery{}, &domainquery.TabState{}, &domainquery.TableSchema{} to the AutoMigrate call.
---
Phase 2: Backend Refactoring for QE-001 to QE-006
2.1 Query ID Change Propagation
Since Query.ID changes from string to uint, these files need updating:
File	Change
domain/query/entity.go	ID uint with auto-increment
domain/query/repository.go	Change GetByID(id string) → GetByClientID(clientID string), add GetByID(id uint)
repository/postgres/query_repo.go	Update GORM queries: filter by client_id varchar
app/query/executor.go	Use client_id (UUID) as the external identifier
app/query/async_executor.go	Change task payload and lookups from query_id to client_id over string
delivery/http/query/handler.go	Route param :id → lookup by client_id
worker/query_worker.go	Update task processing
domain/query/entity.go DTOs	QueryID fields become string (they hold client_id)
API route paths that use query ID stay as-is (still /:id), but the handler internally looks up by client_id.
2.2 Update Request/Response DTOs
ExecuteRequest — add new requestable fields:
type ExecuteRequest struct {
    DatabaseID    uint   `json:"database_id" binding:"required"`
    SQL           string `json:"sql" binding:"required"`
    Limit         *int   `json:"limit"`
    Schema        string `json:"schema"`
    Catalog       string `json:"catalog"`        // NEW
    TabName       string `json:"tab_name"`       // NEW
    SqlEditorID   string `json:"sql_editor_id"`  // NEW
    ClientID      string `json:"client_id"`
    ForceRefresh  bool   `json:"force_refresh"`
    SelectAsCTA   bool   `json:"select_as_cta"`  // NEW
}
ExecuteResponse.Query (ExecuteMeta) — add new metadata fields:
type ExecuteMeta struct {
    ID                uint      `json:"id"`                 // CHANGED from omitted to uint
    ClientID          string    `json:"client_id"`          // NEW
    ExecutedSQL       string    `json:"executed_sql"`
    RLSApplied        bool      `json:"rls_applied"`
    Rows              int       `json:"rows"`
    Limit             int       `json:"limit"`              // NEW
    LimitingFactor    int       `json:"limiting_factor"`    // NEW
    StartTime         time.Time `json:"start_time"`
    EndTime           time.Time `json:"end_time"`
    StartRunningTime  *time.Time `json:"start_running_time,omitempty"` // NEW
    Progress          *string   `json:"progress,omitempty"` // NEW
    Status            string    `json:"status"`
    ResultsKey        string    `json:"results_key,omitempty"`
    SelectAsCTAUsed   bool      `json:"select_as_cta_used"`// NEW
}
2.3 Update Executor Logic (app/query/executor.go)
- Populate new fields when creating a Query record:
  - TabName, SqlEditorID, Catalog from ExecuteRequest
  - Limit from effective limit calculation
  - StartRunningTime when SQL execution begins in db.QueryContext
  - SelectAsCTA / SelectAsCTAUsed from request
  - Progress initially "", updated by async worker
  - ChangedOn set on status updates
2.4 Update Async Executor (app/query/async_executor.go)
- Progress field: set to "queued" on submit, "running" in worker, "done"/"failed" on completion
- Task payload: include new fields (catalog, tab_name, sql_editor_id, select_as_cta)
- Worker: update status, progress, start_running_time during execution
2.5 Verify QE-001 to QE-006 Acceptance Criteria
Requirement	Key AC to Verify
QE-001	executed_sql differs from sql when RLS active; start_running_time populated; limit/limiting_factor stored; client_id dedup works with new model
QE-002	Internal function — InjectRLS unchanged, still injects correctly; RLSApplied flag in response
QE-003	Cache key generation unchanged; results_key still works; from_cache flag
QE-004	3 priority queues; progress field indicates status; retry ×3 with backoff
QE-005	WebSocket endpoint forwards Redis pub/sub; inline data ≤1MB, download URL >1MB
QE-006	Cancel sets progress to "stopped"; idempotent; DB-level cancel
2.6 Database-Specific Requests
Add to ExecuteRequest:
type ExecuteRequest struct {
    ...
    // CTA (Create Table As) support
    SelectAsCTA     bool   `json:"select_as_cta"`
    TmpSchemaName   string `json:"tmp_schema_name"`
    TmpTableName    string `json:"tmp_table_name"`
}
---
Phase 3: Frontend Updates
3.1 Update TypeScript Types (frontend/src/api/queries.ts)
ExecuteQueryRequest — add new fields:
export interface ExecuteQueryRequest {
  database_id: number;
  sql: string;
  limit?: number;
  schema?: string;
  catalog?: string;         // NEW
  tab_name?: string;        // NEW
  sql_editor_id?: string;   // NEW
  client_id?: string;
  force_refresh?: boolean;
  select_as_cta?: boolean;  // NEW
}
QueryMeta — add new fields:
export interface QueryMeta {
  id: number;                // CHANGED from implicit to explicit
  client_id: string;         // NEW
  sql: string;
  executed_sql: string;
  start_time: string;
  end_time: string;
  start_running_time?: string; // NEW
  rows: number;
  limit: number;             // NEW
  limiting_factor: number;   // NEW
  status: string;
  progress?: string;         // NEW
  results_key?: string;
  select_as_cta_used: boolean; // NEW
}
3.2 Update SQL Lab Page (pages/sqllab/SQLLabPage.tsx)
- Send tab_name, sql_editor_id, catalog in execute/submit requests
- Display progress from async status in badges
- Display limit / limiting_factor in results metadata area (warning when truncated)
- Handle select_as_cta checkbox option in toolbar
3.3 Update QueryBadges Component
- AsyncStatusBadge: read progress field for finer-grained status display
- QueryStatusBadge: support new status transitions with progress data
3.4 Update Zustand Store (stores/sqlLabStore.ts)
- SqlLabTab: add catalog, sqlEditorId fields for per-tab metadata
- QueryResult: include new QueryMeta fields
---
Phase 4: Route Fix
4.1 Fix Query Route Registration (router.go)
Current (wrong):
admin := protected.Group("/admin")
{
    // ...
    protected.POST("/query/execute", queryHandler.Execute)  // → /api/v1/admin/query/execute
}
Fixed:
// Query endpoints — accessible to all authenticated users (not admin-restricted)
queryGroup := protected.Group("/query")
{
    queryGroup.POST("/execute", queryHandler.Execute)
    queryGroup.POST("/submit", queryHandler.Submit)
    queryGroup.GET("/:id/status", queryHandler.GetStatus)
    queryGroup.GET("/:id/result", queryHandler.GetResult)
    queryGroup.DELETE("/:id", queryHandler.Cancel)
}
> User decision: Keep under admin — will note this as a decision but the requirement explicitly says /api/v1/query/*. I'll flag this.
---
Detailed File Change List
Go Backend (17 files to touch)
File	Action
internal/domain/query/entity.go	Rewrite — new Query fields, new ExecuteMeta, update all DTOs
internal/domain/query/saved_query.go	CREATE — SavedQuery entity
internal/domain/query/tab_state.go	CREATE — TabState entity
internal/domain/query/table_schema.go	CREATE — TableSchema entity
internal/domain/query/repository.go	Update interface — Change GetByID(string) → GetByClientID(string), add GetByID(uint)
internal/repository/postgres/query_repo.go	Update — New field mappings, new lookup methods
internal/app/query/executor.go	Update — Populate new query fields, new response fields
internal/app/query/async_executor.go	Update — Progress tracking, new task payload fields
internal/app/query/rls_injector.go	Verify — No change expected (RLS logic unchanged)
internal/app/query/cache.go	Verify — No change expected
internal/delivery/http/query/handler.go	Update — Parse new request fields, return new response fields
internal/delivery/http/query/handler_test.go	Update tests
internal/delivery/http/router.go	Fix route paths
internal/worker/query_worker.go	Update — New task payload fields
cmd/api/main.go	Add new entities to AutoMigrate
internal/app/query/integration_test.go	Update tests for new model
internal/app/query/async_executor_test.go	Update tests
Frontend (5 files to touch)
File	Action
src/api/queries.ts	Update types: ExecuteQueryRequest, QueryMeta, ExecuteQueryResponse
src/pages/sqllab/SQLLabPage.tsx	Pass new fields, handle new response data
src/components/query/QueryBadges.tsx	Support progress-based status badges
src/stores/sqlLabStore.ts	Add catalog, sqlEditorId to tab state
src/components/query/QueryBadges.test.tsx	Update tests
Documentation (1 file)
File	Action
docs/db/db_query_saveQuery.md	No change — this is the source of truth
---
## Risks & Considerations
1. **ID type change (string → uint)** is breaking for the async queue — existing enqueued tasks will have string IDs. Mitigation: deploy with fresh Redis queues after migration.
2. **Route path change** may break frontend API calls if not coordinated. Mitigation: update both backend router AND frontend API URLs simultaneously.
3. **New `client_id`-based lookup** means all handler code must change from `GetByID(id)` to `GetByClientID(clientID)`. Mitigation: add helper that tries both for backward compatibility.
4. **GORM auto-migrate** will add nullable columns — existing rows will have NULL for new fields. Ensure nullable types (`*string`, `*time.Time`) for non-critical fields.