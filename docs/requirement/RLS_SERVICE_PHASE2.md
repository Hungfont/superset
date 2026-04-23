**🔒 Row Level Security (RLS) Service**

Rank #05 · Phase 2 - Core · 7 Requirements · 1 Independent · 6 Dependent

## **Service Overview**

The Row Level Security Service is the authorization layer that restricts data access at the row level, independent of table-level permissions. It allows admins to define named filter rules — raw SQL WHERE-clause fragments — that are automatically injected into every query executed against a secured dataset, scoped to the requesting user's roles.

Every chart render, SQL Lab execution, alert metric evaluation, and report data fetch that targets an RLS-secured table passes through the resolution engine before execution. Admins create filter rules, assign them to datasets and roles, and the Query Engine (QE-002 / RLS-003) enforces them transparently at execution time. RLS supports two injection modes: **Regular** (AND-appended to the existing WHERE clause, additive restriction) and **Base** (replaces the WHERE clause entirely, used when the base dataset must itself be scoped). Template variables `{{current_user_id}}` and `{{current_username}}` are resolved at query time using the authenticated user's JWT context.

The management UI lives at `/security/rls` (list, create, edit, delete filters). The enforcement logic is an internal backend function (`InjectRLS`) invoked by QE-001 (sync), QE-004 (async worker), and CHT-006 (chart render) — invisible to end users except via the "RLS Active" badge in SQL Lab when executed_sql differs from the original sql.

Filter grouping via `group_key`: filters sharing the same group_key are OR'd together (same-group filters = user belongs to any of these groups). Filters in different group_keys are AND'd together (all group restrictions must hold). This enables patterns like: `(org_id = 5 OR org_id = 12) AND region = 'APAC'`.

## **Tech Stack**

| **Layer**         | **Technology / Package**                                    | **Purpose**                                           |
| ----------------- | ----------------------------------------------------------- | ----------------------------------------------------- |
| UI Framework      | React 18 + TypeScript                                       | Type-safe component tree                              |
| Bundler           | Vite 5                                                      | Fast HMR and build                                    |
| Routing           | React Router v6                                             | SPA navigation + nested routes                        |
| Server State      | TanStack Query v5                                           | API cache, mutations, background refetch              |
| Client State      | Zustand                                                     | Global UI state (sidebar, user, theme)                |
| Component Library | shadcn/ui (Radix UI primitives)                             | Accessible - ALL components from here, no custom      |
| Forms             | React Hook Form + Zod                                       | Schema validation, field-level errors                 |
| Data Tables       | TanStack Table v8                                           | Sort, filter, paginate, row selection, virtualization |
| Styling           | Tailwind CSS v3                                             | Utility-first, no custom CSS                          |
| Icons             | Lucide React                                                | Consistent icon set                                   |
| API Client        | TanStack Query (fetch)                                      | No raw fetch/axios in components                      |
| Notifications     | shadcn Toaster + useToast                                   | Success/error/info toasts                             |
| Backend           | Gin + GORM + go-redis + xwb1989/sqlparser + OpenTelemetry  | RLS resolution, injection pipeline, audit             |
| Cache             | go-redis (MessagePack serialization)                        | RLS resolution cache keyed by roles_hash + datasetID  |
| SQL Parsing       | xwb1989/sqlparser                                           | AST-based WHERE injection — no string concatenation   |
| Queue             | Asynq (Redis-backed)                                        | Async audit log writes for apply events (low queue)   |
| Tracing           | OpenTelemetry → Jaeger                                      | Per-injection span tracing with clause metadata       |

| **Attribute**      | **Detail**                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------------ |
| Service Name       | Row Level Security Service                                                                 |
| Rank               | #05                                                                                        |
| Phase              | Phase 2 - Core                                                                             |
| Backend API Prefix | /api/v1/rls                                                                                |
| Frontend Routes    | /security/rls (filter list + management + audit log)                                       |
| Primary DB Tables  | row_level_security_filters, rls_filter_roles, rls_filter_tables, rls_audit_log             |
| Total Requirements | 7                                                                                          |
| Independent        | 1                                                                                          |
| Dependent          | 6                                                                                          |

## **DB Schema**

```sql
-- Core filter definition
CREATE TABLE row_level_security_filters (
  id              SERIAL PRIMARY KEY,
  name            VARCHAR(255) NOT NULL UNIQUE,
  filter_type     VARCHAR(10)  NOT NULL CHECK (filter_type IN ('Regular','Base')),
  clause          TEXT         NOT NULL,           -- raw SQL WHERE fragment (max 5000 chars)
  group_key       VARCHAR(255) DEFAULT '',         -- same group_key = OR'd; different = AND'd
  description     TEXT,
  created_by_fk   INT REFERENCES ab_user(id),
  changed_by_fk   INT REFERENCES ab_user(id),
  created_on      TIMESTAMP    NOT NULL DEFAULT NOW(),
  changed_on      TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- Many-to-many: filter <-> roles
CREATE TABLE rls_filter_roles (
  id      SERIAL PRIMARY KEY,
  rls_id  INT NOT NULL REFERENCES row_level_security_filters(id) ON DELETE CASCADE,
  role_id INT NOT NULL REFERENCES ab_role(id) ON DELETE CASCADE,
  UNIQUE (rls_id, role_id)
);

-- Many-to-many: filter <-> datasets
CREATE TABLE rls_filter_tables (
  id               SERIAL PRIMARY KEY,
  rls_id           INT NOT NULL REFERENCES row_level_security_filters(id) ON DELETE CASCADE,
  datasource_id    INT NOT NULL,
  datasource_type  VARCHAR(50) NOT NULL DEFAULT 'table',
  UNIQUE (rls_id, datasource_id, datasource_type)
);

-- Append-only audit log
CREATE TABLE rls_audit_log (
  id                  SERIAL PRIMARY KEY,
  event_type          VARCHAR(50)  NOT NULL,
  filter_id           INT          REFERENCES row_level_security_filters(id) ON DELETE SET NULL,
  filter_name         VARCHAR(255),
  changed_by_id       INT          REFERENCES ab_user(id),
  changed_by_username VARCHAR(255),
  old_value           JSONB,
  new_value           JSONB,
  query_id            VARCHAR(255),
  datasource_id       INT,
  ip_address          INET,
  created_at          TIMESTAMP    NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_rls_audit_filter_id  ON rls_audit_log(filter_id);
CREATE INDEX idx_rls_audit_created_at ON rls_audit_log(created_at DESC);
CREATE INDEX idx_rls_audit_event_type ON rls_audit_log(event_type);
```

## **Redis Key Schema**

| **Key Pattern**                  | **TTL** | **Content**                     | **Written By** | **Busted By**                        |
| -------------------------------- | ------- | ------------------------------- | -------------- | ------------------------------------ |
| `rls:{rolesHash}:{datasourceID}` | 5 min   | MessagePack []ResolvedClause    | RLS-003        | RLS-001, RLS-004, RLS-005, RLS-007   |
| `rls:rate:validate:{userID}`     | 60 s    | int (call count)                | RLS-002        | TTL expiry                           |
| `rls:cache:stats`                | 30 s    | JSON stats snapshot             | RLS-007        | TTL expiry + flush                   |

## **Frontend Stack Notes**

Frontend stack mirrors Apache Superset: React 18 + TypeScript, Vite 5, TanStack Query v5 for all server state, Zustand for global client state, React Router v6.

Component library: shadcn/ui ONLY - no custom components. Use: Button, Input, Form, Select, Dialog, Sheet, Tabs, Table, Toast, DropdownMenu, Command, Popover, Badge, Card, Skeleton, Alert, AlertDialog, Tooltip, ScrollArea, Separator, Textarea, Switch.

Forms: React Hook Form + Zod schema. All inputs via shadcn FormField/FormControl/FormMessage for consistent error display.

Data tables: shadcn DataTable + TanStack Table v8. Never raw HTML tables.

Toasts: shadcn Toaster + useToast. Success=green, error=destructive, info=default.

Loading: shadcn Skeleton for initial load. Button loading via disabled + Lucide Loader2 animate-spin. No full-page blocking spinners.

Styling: Tailwind CSS utility classes only. No inline styles, no CSS modules.

Icons: Lucide React exclusively. Semantic: Plus=create, Pencil=edit, Trash2=delete, Shield=rls-section, ShieldAlert=rls-active, ShieldCheck=rls-valid, ShieldOff=rls-inactive, Eye=preview, RefreshCw=flush/sync, DatabaseZap=cache, Users=roles, Table=datasets, History=audit, Lock=admin-only.

API: all calls via TanStack Query hooks (useQuery/useMutation). Never raw fetch in components. Hooks in /hooks/rls directory.

Error handling: React Error Boundary at page level. API errors via toast onError in useMutation.

## **Requirements**

**INDEPENDENT (1) - no cross-service calls required**

**DEPENDENT (6) - requires prior services/requirements**

---

**RLS-001** - **RLS Filter CRUD**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**                                                   | **API / Route**                                                                                                                   |
| ----------------- | ------------ | --------- | --------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| **INDEPENDENT**   | **P0**       | Phase 2   | row_level_security_filters, rls_filter_roles, rls_filter_tables | GET /api/v1/rls · POST /api/v1/rls · GET /api/v1/rls/:id · PUT /api/v1/rls/:id · DELETE /api/v1/rls/:id                         |

**Depends on:** none

| **Backend - Description**
- Full CRUD for RLS filter definitions. Each filter record: `name` (unique, max 255 chars), `filter_type` ("Regular" OR "Base"), `clause` (raw SQL WHERE-fragment, max 5000 chars), `group_key` (string, default "", groups clauses for OR logic), `description` (optional). Timestamps: `created_on`, `changed_on`. Ownership: `created_by_fk`, `changed_by_fk`.
- Many-to-many associations managed via junction tables: `rls_filter_roles` (filter <-> roles) and `rls_filter_tables` (filter <-> datasets). Both cascade-deleted when filter is deleted.
- Clause validation on every create/update: `sqlparser.ParseExpr(clause)` — return 400 if not a valid SQL boolean expression. This prevents broken clauses from silently corrupting queries at execution time.
- Admin-only write access (`role=Admin`). Alpha/Gamma users -> 403 on write. GET list is Admin-only too (security surface).
- On any create/update/delete: bust all Redis RLS cache keys for affected datasource_ids. Pattern: scan `rls:*:{dsID}` for each dsID linked to this filter -> DEL.
- GET list supports: pagination (`page`, `page_size`), search by name (`q`), filter by `filter_type`, filter by `role_id`, filter by `datasource_id`. Response includes hydrated `roles` and `tables` associations.
- Soft delete NOT supported. DELETE is hard delete — junction rows cascade via FK ON DELETE CASCADE.
**Request Flow**
1. Auth middleware: extract JWT -> verify Admin role -> 403 if not Admin (all write operations).
2. POST/PUT: bind + validate request body.
3. `_, err := sqlparser.ParseExpr(req.Clause)` -> if err != nil: return 400 `{ error:"Invalid SQL clause", detail:err.Error() }`.
4. `GORM.Session(&gorm.Session{FullSaveAssociations:true}).Save(&filter)` — upserts filter row + associations atomically in a transaction.
5. Cache bust: `GORM.Table("rls_filter_tables").Where("rls_id=?",filter.ID).Pluck("datasource_id",&dsIDs)` -> for each dsID: `rdb.Keys(ctx, fmt.Sprintf("rls:*:%d",dsID))` -> `rdb.Del(ctx,keys...)`.
6. Write audit log row synchronously (inside same transaction): `event_type="filter_created"|"filter_updated"|"filter_deleted"`, old/new diff as JSONB.
7. Return 201/200 with filter struct: `{ id, name, filter_type, clause, group_key, description, roles:[{id,name}], tables:[{datasource_id,datasource_type,table_name,database_name}], created_on, changed_on, created_by }`.
**Go Implementation**
1. `if _, err := sqlparser.ParseExpr(req.Clause); err != nil { c.JSON(400, gin.H{"error":"Invalid SQL clause","detail":err.Error()}); return }`
2. `tx := db.Begin()`
3. `tx.Session(&gorm.Session{FullSaveAssociations:true}).Save(&filter)` — roles + tables set via `filter.Roles = roles; filter.Tables = tables`
4. Cache bust: `var dsIDs []int; tx.Table("rls_filter_tables").Where("rls_id=?",filter.ID).Pluck("datasource_id",&dsIDs)` -> `for _,id:=range dsIDs { keys,_:=rdb.Keys(ctx,fmt.Sprintf("rls:*:%d",id)).Result(); rdb.Del(ctx,keys...) }`
5. `tx.Create(&RLSAuditLog{EventType:"filter_created", FilterID:filter.ID, FilterName:filter.Name, ChangedByID:uc.UserID, NewValue:toJSON(filter), IPAddress:c.ClientIP()})`
6. `tx.Commit()`
7. `db.Preload("Roles").Preload("Tables").First(&filter,filter.ID)` -> `c.JSON(201,filter)`
**Security**
- Admin-only middleware applied at router group: `rls.Use(RequireRole("Admin"))`.
- `clause` stored raw, rendered only at query time with typed int/string values — never executed on save (only sqlparser dry-parse).
- Hard delete with FK CASCADE ensures no orphaned junction rows that could re-activate a deleted filter. | **Acceptance Criteria**
- `POST {name:"tenant_isolation", filter_type:"Regular", clause:"tenant_id = {{current_user_id}}", roles:[1], tables:[5]}` -> 201 `{id:3, name:"tenant_isolation", filter_type:"Regular", clause:"tenant_id = {{current_user_id}}", roles:[{id:1,name:"Gamma"}], tables:[{datasource_id:5,table_name:"orders"}]}`.
- `POST {clause:"tenant_id = AND 1"}` -> 400 `{error:"Invalid SQL clause", detail:"syntax error at position 14"}`.
- `PUT /rls/3 {clause:"tenant_id = {{current_user_id}} AND active = true"}` -> 200 + Redis keys for datasource 5 DEL'd + audit row inserted with old/new clause diff.
- `DELETE /rls/3` -> 204. Verify: `SELECT * FROM rls_filter_roles WHERE rls_id=3` returns 0 rows.
- `GET /rls?filter_type=Base&role_id=2&page=1&page_size=20` -> paginated list, only Base filters assigned to role 2.
- `GET /rls?q=tenant` -> only filters whose name ILIKE '%tenant%'.
- Alpha user POST -> 403.
- Duplicate name POST -> 409 `{error:"Filter name already exists"}`.
**Error Responses**
- 400 - Invalid SQL clause (sqlparser rejected) or malformed request body.
- 403 - Non-Admin write attempt.
- 404 - Filter ID not found.
- 409 - Duplicate `name` unique constraint violation.
- 500 - DB or transaction error. | **Frontend Specification**
**Route & Page**
/security/rls — full-page RLS filter management. Layout: page header ("Row Level Security" title + subtitle "Restrict data access by role using SQL filter clauses" + Admin badge), Tabs ["Filters" | "Audit Log"], Filters tab contains search/filter controls row + DataTable.
**shadcn/ui Components**
- `DataTable` (TanStack Table v8) — columns: Name (sortable, click to open edit Dialog), Filter Type (Badge, sortable), Clause (truncated 60 chars + Tooltip full clause, monospace), Group Key (Badge gray monospace, hidden column if all empty), Roles (Badge list, max 3 + "+N more" Tooltip), Tables (Badge list, max 3 + "+N more" Tooltip), Created By, Modified (relative time, absolute in Tooltip), Actions (DropdownMenu)
- `Badge` filter_type: blue outlined="Regular", amber outlined="Base"
- `Badge` group_key: gray, `font-mono text-xs`, only rendered when non-empty
- `DropdownMenu` Actions: "Edit" (Pencil icon), "Manage Roles" (Users icon) -> opens RLS-004 Sheet, "Manage Datasets" (Table icon) -> opens RLS-005 Sheet, `DropdownMenuSeparator`, "Delete" (Trash2 icon, destructive red)
- `Dialog` create/edit — triggered by "+ Add Filter" button or "Edit" action or clicking Name cell. Title: "Create RLS Filter" / "Edit RLS Filter: {name}".
- Inside Dialog: `ScrollArea` wrapping form (for short screens). Form fields:
  - `FormField` name -> `Input` placeholder="e.g. tenant_isolation"
  - `FormField` filter_type -> `Select` options: ["Regular — AND appended to WHERE", "Base — Replaces WHERE entirely"]. On select "Base": show amber `Alert` "Base filters replace the entire WHERE clause. Use only when defining the base dataset scope for all users."
  - `FormField` clause -> `Textarea` (`font-mono text-sm min-h-[120px]`) placeholder="e.g. org_id = {{current_user_id}}". Below Textarea: chip row with gray `Badge` buttons "{{current_user_id}}" and "{{current_username}}" — click inserts at cursor. Below chips: "Validate Clause" button (see RLS-002).
  - `FormField` group_key -> `Input` placeholder="e.g. org_group (optional)". Helper: "Filters with the same group key are OR'd together. Leave empty to AND with all others."
  - `FormField` description -> `Textarea` optional, `min-h-[60px]` placeholder="Optional description..."
  - `FormField` roles -> `Command` inside `Popover` (multi-select). Each item: checkbox + role name + role type Badge. Selected shown as Badge chips.
  - `FormField` tables -> `Command` inside `Popover` (multi-select). Each item: "table_name · database_name" two-line. Grouped by database in Command groups. Selected shown as Badge chips.
- Dialog footer: `Button` ("Cancel", variant=outline) + `Button` ("Create Filter"/"Save Changes", variant=default, shows Loader2 during submit)
- `AlertDialog` delete: title="Delete RLS Filter?", description="Deleting '{name}' will immediately remove data restrictions for all users assigned to this filter's roles. This cannot be undone." Cancel + "Delete Filter" (destructive). Confirm button disabled 1.5s after Dialog open.
- `Input` (search, Search icon left) — placeholder "Search filters..."
- `Select` filter_type filter — "All Types" | "Regular" | "Base"
- `Select` role filter — "All Roles" | role options from `GET /api/v1/roles`
- `Skeleton` — 5 rows x 8 cols skeleton cells while loading
- Empty state: centered Shield icon + "No RLS filters configured." + subtitle + "+ Add Filter" Button
**State & TanStack Query**
- `useQuery({queryKey:["rls-filters",{page,pageSize,q,filter_type,role_id}], queryFn:()=>fetch("/api/v1/rls?"+qs).then(r=>r.json()), staleTime:30_000})`
- `useMutation({mutationFn:api.createRLSFilter, onSuccess:()=>{queryClient.invalidateQueries(["rls-filters"]); toast({title:"Filter created"}); setDialogOpen(false)}, onError:(e)=>toast({title:e.message,variant:"destructive"})})`
- `useMutation({mutationFn:({id,...b})=>api.updateRLSFilter(id,b), onSuccess:()=>{queryClient.invalidateQueries(["rls-filters"]); toast({title:"Filter updated"}); setDialogOpen(false)}})`
- `useMutation({mutationFn:api.deleteRLSFilter, onSuccess:()=>{queryClient.invalidateQueries(["rls-filters"]); toast({title:"Filter deleted"})}})`
- `useRLSStore` (Zustand): `{editingFilter:RLSFilter|null, setEditingFilter, dialogOpen, setDialogOpen}`
- `useForm<RLSFilterFormValues>({resolver:zodResolver(rlsFilterSchema)})` where schema: name (min 1), filter_type (enum), clause (min 1 max 5000), roles (array min 1), tables (array min 1)
**UX Behaviors**
- "+ Add Filter" opens Dialog with empty form. "Edit" opens Dialog with `reset(filter)` pre-filled.
- Clause Textarea: template chip click -> inserts text at cursor via `textarea.setSelectionRange` + `document.execCommand("insertText")` -> triggers debounced validation.
- filter_type Select "Base": amber Alert fades in inside Dialog body. Switching back to "Regular": Alert fades out.
- Roles Command: each item shows role type (Admin=purple/Alpha=blue/Gamma=green Badge). Shows "3 of 5 roles selected" counter in Popover trigger.
- Tables Command: grouped by database_name using `Command.Group`. Shows "2 tables selected" counter.
- Dialog submit: button Loader2 + disabled during mutation. Success: Dialog closes, DataTable row animates in (new) or highlight-flashes (updated, amber -> normal transition 600ms).
- Delete AlertDialog: "Delete Filter" button disabled for 1.5s (anti-accidental-click). On confirm: row fades out.
**Accessibility**
- Dialog: `aria-labelledby`=Dialog title id. Focus trap inside Dialog. ESC closes.
- DataTable: `role="grid"`. Actions DropdownMenu: `aria-label="Filter actions for {name}"`.
- AlertDialog: focus moves to Cancel on open.
**API Calls**
1. `useQuery({queryKey:["rls-filters",params], queryFn:()=>fetch("/api/v1/rls?"+new URLSearchParams(params)).then(r=>r.json())})`
2. `useMutation({mutationFn:(body)=>fetch("/api/v1/rls",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)}).then(r=>r.json())})`
3. `useMutation({mutationFn:({id,...b})=>fetch("/api/v1/rls/"+id,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(b)}).then(r=>r.json())})`
4. `useMutation({mutationFn:(id)=>fetch("/api/v1/rls/"+id,{method:"DELETE"}).then(r=>r.ok?null:r.json().then(e=>Promise.reject(e)))})` |
| --- | --- | --- |

---

**RLS-002** - **RLS Clause Validation & Live Preview**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                                          | **API / Route**           |
| --------------- | ------------ | --------- | ------------------------------------------------------ | ------------------------- |
| **DEPENDENT**   | **P0**       | Phase 2   | row_level_security_filters, dbs, datasources_reporting | POST /api/v1/rls/validate |

**Depends on:** RLS-001 (filter entity for edit-mode context), DBC-006 (connection pool for runtime probe), AUTH-011 (user context for template rendering preview)

| **Backend - Description**
- Two-phase clause validation callable independently from save, used by the admin form for real-time feedback before a filter is committed.
- **Phase 1 — Syntax (always executed):** `sqlparser.ParseExpr(clause)` — validates the clause is a syntactically valid SQL boolean expression. Returns immediately if syntax is invalid; no DB connection needed.
- **Phase 2 — Runtime (optional):** activated when `database_id` + `table_name` + `schema` + `test_user_id` are all provided. Steps: (a) Render template vars: `{{current_user_id}}` -> test_user_id (typed int via `strconv.Itoa`), `{{current_username}}` -> test_username (string). (b) Build probe SQL: `SELECT 1 FROM {schema}.{table_name} WHERE ({rendered_clause}) LIMIT 0`. (c) Execute via connection pool with 5s timeout. LIMIT 0 ensures zero rows returned regardless of clause — no data exposure. If column referenced in clause does not exist -> DB returns error -> surface as runtime validation failure.
- Returns: `{is_valid:bool, phase:"syntax"|"runtime", rendered_clause:string, error:string|null, error_position:int|null}`.
- Rate limit: 60 req/min per user (`rls:rate:validate:{userID}` Redis counter, 60s TTL). Prevents abuse of live DB probing endpoint.
**Request Flow**
1. Rate limit: `rdb.Incr("rls:rate:validate:"+uid)` + `rdb.Expire(60s)` -> if count > 60: return 429.
2. Phase 1: `_, err:=sqlparser.ParseExpr(req.Clause)` -> if err: return 200 `{is_valid:false, phase:"syntax", error:err.Error(), error_position:extractPos(err)}`.
3. If `database_id` + `table_name` not provided: return 200 `{is_valid:true, phase:"syntax", rendered_clause:req.Clause}`.
4. Phase 2: `rendered:=renderTemplateVars(req.Clause, req.TestUserID, req.TestUsername)`.
5. `probeSQL:=fmt.Sprintf("SELECT 1 FROM %s.%s WHERE (%s) LIMIT 0", pq.QuoteIdentifier(req.Schema), pq.QuoteIdentifier(req.TableName), rendered)`.
6. `conn:=pool.Get(req.DatabaseID)` -> `ctx5s,cancel:=context.WithTimeout(ctx,5*time.Second)` -> `conn.ExecContext(ctx5s,probeSQL)`.
7. Return 200 `{is_valid:true, phase:"runtime", rendered_clause:rendered}` or `{is_valid:false, phase:"runtime", error:dbErr.Error()}`.
**Go Implementation**
1. `cnt,_:=rdb.Incr(ctx,"rls:rate:validate:"+strconv.Itoa(uc.UserID)).Result(); rdb.Expire(ctx,key,60*time.Second); if cnt>60 { c.JSON(429,gin.H{"error":"Rate limit exceeded"}); return }`
2. `if _,err:=sqlparser.ParseExpr(req.Clause); err!=nil { c.JSON(200,ValidateResult{IsValid:false,Phase:"syntax",Error:err.Error()}); return }`
3. `rendered:=strings.NewReplacer("{{current_user_id}}",strconv.Itoa(req.TestUserID),"{{current_username}}",req.TestUsername).Replace(req.Clause)`
4. `probeSQL:=fmt.Sprintf("SELECT 1 FROM %s.%s WHERE (%s) LIMIT 0",pq.QuoteIdentifier(req.Schema),pq.QuoteIdentifier(req.TableName),rendered)`
5. `_,err=conn.ExecContext(ctx5s,probeSQL); if err!=nil { c.JSON(200,ValidateResult{IsValid:false,Phase:"runtime",Error:err.Error()}); return }`
6. `c.JSON(200,ValidateResult{IsValid:true,Phase:"runtime",RenderedClause:rendered})`
**Security**
- `pq.QuoteIdentifier` on schema + table_name prevents table reference injection in probe SQL.
- `strings.NewReplacer` with `strconv.Itoa(uc.UserID)` — user_id always rendered as integer string, never accepting arbitrary string value. Prevents `{{current_user_id}}` being set to "1 OR 1=1".
- `LIMIT 0` on probe: zero rows returned regardless of clause — validation never exposes real data rows.
- Rate limit (60/min) prevents using validate endpoint as a timing-oracle for row existence inference. | **Acceptance Criteria**
- `POST {clause:"org_id = {{current_user_id}}", test_user_id:42, test_username:"alice"}` -> 200 `{is_valid:true, phase:"syntax", rendered_clause:"org_id = 42"}`.
- `POST {clause:"org_id = AND"}` -> 200 `{is_valid:false, phase:"syntax", error:"syntax error near 'AND'", error_position:10}`.
- `POST {clause:"nonexistent_col = 1", database_id:1, table_name:"orders", schema:"public", test_user_id:42}` -> 200 `{is_valid:false, phase:"runtime", error:"column \"nonexistent_col\" does not exist"}`.
- `POST {clause:"org_id = {{current_user_id}}", database_id:1, table_name:"orders", schema:"public", test_user_id:42}` -> probe executed on real DB -> 200 `{is_valid:true, phase:"runtime", rendered_clause:"org_id = 42"}`.
- Injection: request with `test_user_id` as string "42 OR 1=1" -> 400 (int field validation rejects non-integer before rendering).
- 61st call within 60s -> 429 `{error:"Rate limit exceeded"}`.
**Error Responses**
- 200 always for validation result (valid or invalid encoded in body).
- 400 - Malformed request body (wrong field types, missing required fields).
- 429 - Rate limited (60/min per user).
- 500 - Connection pool unavailable for runtime phase. | **Frontend Specification**
**Route & Page**
/security/rls — validation is inline inside the create/edit Dialog. No separate route. Attached to the clause Textarea FormField.
**shadcn/ui Components**
- `Textarea` (clause field, `font-mono text-sm min-h-[120px]`) — border state reflects validation: default gray, valid green ring-2, invalid destructive ring-2
- `FormMessage` — shows syntax error text below Textarea (e.g. "Syntax error near 'AND' at position 10")
- `Button` ("Validate Clause", ShieldCheck icon, `variant="outline" size="sm"`) — right-aligned below Textarea. Disabled until test_user selected (for Phase 2) or always enabled for Phase 1 (syntax only).
- `Alert` (green border, ShieldCheck icon) — success: "Clause is valid · Rendered as: org_id = 42". Fades in on success, fades out on clause change.
- `Alert` (destructive, ShieldOff icon) — error: phase label ("Syntax error" / "Runtime error") + error message
- `Select` ("Test as user") — optional, enables Phase 2. Populated via `GET /api/v1/users?page_size=50`. Shows user display_name. Placeholder "Select user to test template vars..."
- `Select` ("Test against table") — optional, populated from tables already selected in the Tables FormField. Placeholder "Select table for runtime probe..."
- `Badge` (inline status next to Validate button): "Syntax OK" (green, ShieldCheck) / "Runtime OK" (green, ShieldCheck) / "Syntax Error" (red, ShieldOff) / "Runtime Error" (red, ShieldOff) — shown only after validation attempted
- `Skeleton` (1-line, 80px wide) — shown inside validate button area during mutation (replaces badge)
- `Tooltip` on Validate button (when disabled): "Select a test user and target table to enable runtime validation."
**State & TanStack Query**
- `useMutation({mutationFn:(body)=>fetch("/api/v1/rls/validate",{method:"POST",body:JSON.stringify(body)}).then(r=>r.json()), onSuccess:(r)=>setValidationResult(r)})`
- `validationResult: ValidateResult|null` — local component state in Dialog. Reset to null on every clause keystroke.
- `debouncedClause` — `useDebounce(clauseValue, 1500)` — triggers Phase 1 syntax-only validation automatically (no database_id in payload -> syntax phase only, minimal rate limit pressure)
- `useEffect(()=>{ if(debouncedClause) validateMutation.mutate({clause:debouncedClause}) },[debouncedClause])`
- `testUserID`, `testTableName`, `testSchema` — local state for Phase 2 selectors
**UX Behaviors**
- Clause field: debounced auto-validate every 1500ms on keystroke -> syntax phase -> immediate border color + FormMessage feedback.
- "Validate Clause" button: runs full Phase 2 runtime validation when test user + table selected. Shows Loader2 animate-spin + disabled during mutation (~200ms average probe round-trip).
- Template chip buttons above Textarea: click inserts at cursor position -> triggers immediate debounced re-validation.
- Success Alert: shows `rendered_clause` with actual substituted value ("org_id = 42") so admin can confirm template substitution is correct before saving.
- Error Alert distinguishes phase: syntax errors show position indicator; runtime errors show DB error verbatim (e.g. "column 'bad_col' does not exist").
- If "Validate Clause" clicked without test user selected: Tooltip on button explains requirement. Button stays disabled.
- Clause change after validation: Alert fades out (opacity-0 transition 300ms), border resets to default, FormMessage clears.
**Accessibility**
- Clause Textarea: `aria-label="SQL WHERE clause"`, `aria-invalid={validationResult && !validationResult.is_valid}`, `aria-describedby="clause-validation-message"`.
- Validation result Alert: `role="alert"` -> screen readers announce result on change.
- Validate button: `aria-label="Validate SQL clause syntax and runtime"`.
**API Calls**
1. `useMutation({mutationFn:(body:ValidateRequest)=>fetch("/api/v1/rls/validate",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)}).then(r=>r.json())})` |
| --- | --- | --- |

---

**RLS-003** - **RLS Resolution Engine (Internal)**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                                                           | **API / Route**                                          |
| --------------- | ------------ | --------- | ----------------------------------------------------------------------- | -------------------------------------------------------- |
| **DEPENDENT**   | **P0**       | Phase 2   | row_level_security_filters, rls_filter_roles, rls_filter_tables         | Internal function — called by QE-001, QE-004, CHT-006   |

**Depends on:** RLS-001 (filter records in DB), AUTH-011 (resolved roles []string from JWT), DS-010 (datasourceID from dataset context)

| **Backend - Description**
- Core internal function signature: `InjectRLS(ctx context.Context, sql string, datasourceID int, roles []string, uc UserContext) (injectedSQL string, rlsHash string, err error)`. Called before every SQL execution. Never exposed as an HTTP endpoint.
- **Admin shortcut:** `isAdmin(roles)` -> return original sql unchanged, rlsHash="admin". Optionally enqueue sampled audit event (10% sample).
- **Resolution steps (non-admin):**
  1. `rolesHash := sha256hex(strings.Join(sorted(roles),":"))`
  2. cacheKey = `rls:{rolesHash}:{datasourceID}` -> Redis GET -> if HIT: unmarshal MessagePack `[]ResolvedClause`, skip DB
  3. if MISS: join query across all three RLS tables scoped to role names IN (userRoles) AND datasource_id = datasourceID -> scan `[]ResolvedClause{Clause, FilterType, GroupKey}`
  4. Marshal to MessagePack -> `rdb.Set(cacheKey, packed, 5*time.Minute)`
  5. if `len(clauses)==0`: datasource is not RLS-secured -> return `(sql, "", nil)`
- **Template rendering:** `strings.NewReplacer("{{current_user_id}}",strconv.Itoa(uc.UserID),"{{current_username}}",uc.Username).Replace(clause)` per clause. Typed int for user_id — injection-safe.
- **Group-key OR/AND logic:** group clauses by `group_key`. Within a group: rendered clauses joined as `(clause1 OR clause2)` using `sqlparser.ParseExpr`. Across groups: all group expressions AND-chained as `(groupA) AND (groupB)`. Empty group_key = unique singleton group (never OR'd with others).
- **Filter type injection:**
  - `Regular`: compound expression AND-appended to existing WHERE via `addWhereAnd(stmt, compoundExpr)`. Existing WHERE preserved.
  - `Base`: compound expression REPLACES existing WHERE via `replaceWhere(stmt, compoundExpr)`. If multiple Base filters exist for same datasource (admin error): last one in query result order wins.
- **rlsHash returned:** `sha256hex(strings.Join(sortedRenderedClauses,"+"))` — included in QE-003 cache key so users with different RLS states never share query result cache entries. Admin rlsHash="admin" always produces a separate cache entry from non-admin.
- **OpenTelemetry span:** `"rls.inject"` with attributes: `rls.datasource_id`, `rls.roles_hash`, `rls.cache_hit` (bool), `rls.clauses_count`, `rls.filter_types` ([]string), `rls.injection_ms`.
**Request Flow**
1. `isAdmin(roles)` -> return `(sql,"admin",nil)`. Optionally enqueue sampled audit.
2. `sort.Strings(roles); rolesHash:=sha256hex(strings.Join(roles,":"))`.
3. `cacheKey:=fmt.Sprintf("rls:%s:%d",rolesHash,datasourceID)`.
4. `val,err:=rdb.Get(ctx,cacheKey).Bytes()` -> if err==nil: `msgpack.Unmarshal(val,&clauses)` -> skip to step 7.
5. DB: `SELECT f.clause,f.filter_type,f.group_key FROM row_level_security_filters f JOIN rls_filter_roles rfr ON rfr.rls_id=f.id JOIN rls_filter_tables rft ON rft.rls_id=f.id JOIN ab_role r ON rfr.role_id=r.id WHERE r.name IN (?) AND rft.datasource_id=?`.
6. `packed,_:=msgpack.Marshal(clauses); rdb.Set(ctx,cacheKey,packed,5*time.Minute)`.
7. `if len(clauses)==0 { return sql,"",nil }`.
8. For each clause: render template vars.
9. Group by group_key -> per group: `sqlparser.ParseExpr(strings.Join(renderedGroup," OR "))` -> wrap `&sqlparser.ParenExpr{Expr:orExpr}`.
10. AND all group exprs: `buildAndChain(groupExprs)` -> recursive `&sqlparser.AndExpr{Left:..,Right:..}`.
11. `stmt,_:=sqlparser.Parse(sql)` -> `addWhereAnd(stmt,compound)` OR `replaceWhere(stmt,compound)` based on filter_type.
12. `injectedSQL:=sqlparser.String(stmt)`.
13. `rlsHash:=sha256hex(strings.Join(sortedRenderedClauses,"+"))`.
14. Return `(injectedSQL, rlsHash, nil)`.
**Go Implementation**
1. `func InjectRLS(ctx context.Context, sql string, datasourceID int, roles []string, uc UserContext) (string,string,error) {`
2. `if isAdmin(roles) { return sql,"admin",nil }`
3. `sort.Strings(roles); rolesHash:=sha256hex(strings.Join(roles,":"))`
4. `cacheKey:=fmt.Sprintf("rls:%s:%d",rolesHash,datasourceID)`
5. `if val,err:=rdb.Get(ctx,cacheKey).Bytes(); err==nil { msgpack.Unmarshal(val,&clauses) } else { /* DB join query */ packed,_:=msgpack.Marshal(clauses); rdb.Set(ctx,cacheKey,packed,5*time.Minute) }`
6. `replacer:=strings.NewReplacer("{{current_user_id}}",strconv.Itoa(uc.UserID),"{{current_username}}",uc.Username)`
7. `groups:=map[string][]string{}; for _,c:=range clauses { groups[c.GroupKey]=append(groups[c.GroupKey],replacer.Replace(c.Clause)) }`
8. `var andExprs []sqlparser.Expr; for _,g:=range groups { orExpr,_:=sqlparser.ParseExpr(strings.Join(g," OR ")); andExprs=append(andExprs,&sqlparser.ParenExpr{Expr:orExpr}) }`
9. `compound:=buildAndChain(andExprs)`
10. `stmt,_:=sqlparser.Parse(sql); addWhereAnd(stmt,compound)` OR `replaceWhere(stmt,compound)`
11. `return sqlparser.String(stmt),sha256hex(strings.Join(sortedRendered,"+")),nil }`
**Security**
- `strings.NewReplacer` with `strconv.Itoa(uc.UserID)` — user_id always an integer string, never accepting user-controlled string value. `{{current_user_id}}` cannot be exploited even if clause value were attacker-influenced.
- sqlparser AST injection: WHERE modified at AST node level, never via string concatenation. Prevents UNION injection, comment escape, subquery bypass via crafted SQL input.
- rlsHash in QE-003 cache key: different users (even identical SQL) always get separate cache entries. No cross-user data leakage via query result cache.
- RLS failure is hard-fail: error returned to QE-001/QE-004 which surface 500. RLS failure NEVER silently passes original SQL through. | **Acceptance Criteria**
- Gamma user (user_id=42) + Regular filter `"org_id = {{current_user_id}}"` on datasource 5 -> `executed_sql` contains `AND (org_id = 42)`.
- Admin user -> SQL unchanged, rlsHash="admin".
- Base filter `"org_id = 42"` -> existing WHERE clause fully replaced: `WHERE (org_id = 42)`.
- Two filters same group_key "grp1": `org_id=5` + `org_id=12` -> injected as `AND (org_id=5 OR org_id=12)`.
- Two filters different group_key: `org_id=5` (grp1) + `region='APAC'` (grp2) -> injected as `AND (org_id=5) AND (region='APAC')`.
- Cache HIT (second call same roles + datasourceID): DB query NOT executed. Redis returns in <1ms.
- Cache MISS: DB join executed, result cached at TTL 5min, latency ~5-10ms.
- Datasource not in rls_filter_tables -> SQL unchanged, rlsHash="".
- Injection: `uc.UserID=99` -> rendered as `id = 99` (int), never `id = 99 OR 1=1`.
**Error Responses**
- Internal — all errors propagate to QE-001/QE-004/CHT-006 which return 500 to caller. RLS failure never silently skipped. | **Frontend Specification**
**Route & Page**
Transparent to frontend — surfaced in SQL Lab via query response metadata (`executed_sql` vs `sql` fields)
**shadcn/ui Components**
- `Badge` ("RLS Active", ShieldAlert icon, `bg-orange-100 text-orange-800 border border-orange-300 text-xs`) — in SQL Lab query metadata row below results DataTable, shown when `executed_sql !== sql`
- `Tooltip` on Badge — "Row-level security filters were applied to this query. Your results are filtered based on your role. Contact an admin to review filter rules."
- `Tabs` ["Results" | "Query Info"] — in SQL Lab results panel. "Query Info" tab shows query execution metadata.
- `Card` (Query Info tab) — two sections: "Your SQL" (gray `bg-muted` rounded, `font-mono text-sm`, original sql, full ScrollArea) + "Executed SQL" (blue `bg-blue-50` rounded when RLS active, `font-mono text-sm`, executed_sql with injected clause).
- `Alert` (info, blue, ShieldAlert icon) — inside Query Info tab when RLS active: "RLS Active: The WHERE clause shown above was automatically modified by your administrator to restrict your data access."
- `Info` Lucide icon (14px, muted) — next to "Executed SQL" section label, Tooltip "Modified by Row Level Security"
- `Badge` ("Admin View", Lock icon, gray) — shown in Query Info tab for Admin users, unlocks diff highlight mode
**State & TanStack Query**
- `queryResult.query.executed_sql` — from QE-001/QE-004 response
- `queryResult.query.sql` — original user SQL
- `rlsActive:boolean` — `queryResult?.query?.executed_sql !== queryResult?.query?.sql`
- No separate API call — all data from existing query execution response
**UX Behaviors**
- SQL Lab metadata row: if `rlsActive`, orange "RLS Active" Badge appears alongside from_cache Badge, duration Badge, row count Badge.
- "Query Info" tab: only shows RLS diff when `rlsActive`. When RLS not active: "Executed SQL" section matches original sql, no Alert shown.
- Admin users: Query Info tab shows diff highlight — injected clause fragment highlighted in `bg-amber-100` within the executed_sql block (simple string diff against original sql).
- Dashboard / Explore chart renders: RLS fully transparent. Users see filtered data with no UI indicator.
**API Calls**
1. N/A — internal. Frontend consumes from existing QE-001/QE-004 response: `{data, columns, query:{sql, executed_sql,...}, from_cache}` |
| --- | --- | --- |

---

**RLS-004** - **RLS Filter Role Assignment**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                                    | **API / Route**                                                            |
| --------------- | ------------ | --------- | ------------------------------------------------ | -------------------------------------------------------------------------- |
| **DEPENDENT**   | **P1**       | Phase 2   | rls_filter_roles, row_level_security_filters     | GET /api/v1/rls/:id/roles · PUT /api/v1/rls/:id/roles                     |

**Depends on:** RLS-001 (filter must exist), AUTH-006 (role list for Admin validation)

| **Backend - Description**
- Dedicated endpoints to manage the filter <-> role many-to-many independently of full filter CRUD. Allows admins to update role assignments without re-saving the full filter definition (clause, group_key etc). Useful for bulk role-set changes or onboarding a new role to an existing filter.
- **GET:** returns current assigned roles with metadata: `{roles:[{id, name, type, user_count}]}`. `user_count` derived from `COUNT(ab_user_role WHERE role_id=r.id)` — gives admin visibility into scope of impact.
- **PUT:** body `{role_ids:[1,3,7]}` — **full replacement** (not merge/append). Old roles not in `role_ids` are removed; new roles inserted. Idempotent. `role_ids=[]` -> all roles removed, filter becomes inactive (applies to no one). Validate every role_id exists in `ab_role` before applying -> 400 on any invalid id with specific error.
- After PUT: compute affected datasource_ids from `rls_filter_tables WHERE rls_id=:id` -> bust Redis `rls:*:{dsID}` for each. Write audit log: `event_type="roles_changed"`, `old_value={role_ids:[...]}, new_value={role_ids:[...]}`.
**Request Flow**
1. GET: `GORM.Preload("Roles",func(db *gorm.DB)*gorm.DB{return db.Select("ab_role.id,ab_role.name,ab_role.type").Joins("LEFT JOIN ab_user_role aur ON aur.role_id=ab_role.id").Group("ab_role.id").Select("ab_role.*,COUNT(aur.user_id) AS user_count")}).First(&filter,id)`.
2. PUT: snapshot old roles `oldRoleIDs`. Validate: `GORM.Where("id IN ?",req.RoleIDs).Find(&roles)` -> if `len(roles)!=len(req.RoleIDs)`: 400.
3. `GORM.Model(&filter).Association("Roles").Replace(&roles)` — atomic full replacement.
4. `GORM.Table("rls_filter_tables").Where("rls_id=?",id).Pluck("datasource_id",&dsIDs)`.
5. Cache bust each dsID.
6. `db.Create(&RLSAuditLog{EventType:"roles_changed",...})`.
7. Return updated roles list.
**Go Implementation**
1. GET: `db.Preload("Roles").First(&filter,id); c.JSON(200,gin.H{"roles":filter.Roles})`
2. PUT validation: `db.Where("id IN ?",req.RoleIDs).Find(&roles); if len(roles)!=len(req.RoleIDs) { c.JSON(400,gin.H{"error":"One or more role_ids not found"}); return }`
3. `db.Model(&filter).Association("Roles").Replace(&roles)`
4. Cache bust (same pattern as RLS-001)
5. `db.Create(&RLSAuditLog{EventType:"roles_changed",OldValue:toJSON(oldRoleIDs),NewValue:toJSON(req.RoleIDs),ChangedByID:uc.UserID,FilterID:id})`
6. `db.Preload("Roles").First(&filter,id); c.JSON(200,gin.H{"roles":filter.Roles})`
**Security**
- Admin-only via router group middleware.
- Full replacement semantics prevent privilege escalation via incremental append — entire role set always explicitly stated in each PUT. | **Acceptance Criteria**
- `GET /api/v1/rls/5/roles` -> 200 `{roles:[{id:1,name:"Gamma",type:"Gamma",user_count:142}]}`.
- `PUT /api/v1/rls/5/roles {role_ids:[1,3]}` -> 200 updated roles list. Redis keys for linked datasources DEL'd. Audit row: old `[1]`, new `[1,3]`.
- `PUT {role_ids:[]}` -> 200 `{roles:[]}` — filter inactive, no users affected.
- `PUT {role_ids:[1,9999]}` where 9999 missing -> 400 `{error:"One or more role_ids not found"}`. DB NOT partially modified (validate-before-replace).
- Non-admin -> 403.
- Filter id=999 -> 404.
**Error Responses**
- 400 - Invalid role_id(s).
- 403 - Non-admin.
- 404 - Filter not found. | **Frontend Specification**
**Route & Page**
/security/rls — `Sheet` (right side panel, 480px) opened from DataTable row Actions DropdownMenu "Manage Roles". No separate route.
**shadcn/ui Components**
- `Sheet` (side=right, `className="w-[480px]"`) — slides in from right. Overlay backdrop.
- `SheetHeader` — `SheetTitle` "Manage Roles", `SheetDescription` "Assign roles to control which users this filter applies to. At least one role required for the filter to be active."
- `SheetContent` — two sections:
  - Section 1: "Currently Assigned" `Card` — lists current roles as `Badge` chips ({name} · {user_count} users). Empty state: `Alert` (info, ShieldOff icon) "No roles assigned — this filter is currently inactive."
  - Section 2: `Separator`, "Add / Remove Roles" label, `Command` inside `Popover` (multi-select, all roles)
- `Command` — searchable by role name. Each item: `Checkbox` (checked=selected) + role name + role type `Badge` (Admin=purple, Alpha=blue, Gamma=green). Groups: `CommandGroup` "Admin Roles" / "Standard Roles".
- `Badge` chips above `Command` trigger — selected roles: "{RoleName} ×" (click × to remove). Shows "3 roles selected" counter in `Popover` trigger when chips overflow.
- `Separator` between sections.
- `SheetFooter` — `Button` ("Save Role Assignment", full-width, default) + `Button` ("Cancel", variant=outline, full-width).
- `Alert` (amber, AlertTriangle icon, inside Sheet) — appears when `localRoles.length===0`: "Warning: No roles assigned. Saving will deactivate this filter."
- `Skeleton` — 3-row skeleton in Command while roles list loads.
- `Badge` impact indicator in SheetHeader: "Currently applies to ~{totalUserCount} users" (sum of user_count across current roles).
**State & TanStack Query**
- `useQuery({queryKey:["rls-filter-roles",filterId], queryFn:()=>fetch("/api/v1/rls/"+filterId+"/roles").then(r=>r.json()), enabled:sheetOpen&&!!filterId})`
- `useQuery({queryKey:["auth-roles"], queryFn:()=>fetch("/api/v1/roles").then(r=>r.json()), staleTime:300_000})` — all roles for Command selector
- `useMutation({mutationFn:({id,role_ids})=>fetch("/api/v1/rls/"+id+"/roles",{method:"PUT",body:JSON.stringify({role_ids})}).then(r=>r.json()), onSuccess:()=>{ queryClient.invalidateQueries(["rls-filters"]); queryClient.invalidateQueries(["rls-filter-roles",filterId]); toast({title:"Roles updated"}); setSheetOpen(false) }})`
- `localRoleIDs:number[]` — local state, initialized from fetched roles on Sheet open. Tracks pending changes.
- `isDirty:boolean` — `JSON.stringify(localRoleIDs.sort()) !== JSON.stringify(fetchedRoleIDs.sort())` -> enables Save button.
**UX Behaviors**
- "Manage Roles" from DropdownMenu -> Sheet slides in from right. Fetches current roles on open (enabled flag). Shows Skeleton while loading.
- Command: each item checkbox reflects localRoleIDs. Click role -> toggles in localRoleIDs. Search filters list in real time.
- Remove role: click × on Badge chip OR uncheck in Command.
- Save button: disabled when `!isDirty || mutation.isPending`. Shows Loader2 during mutation.
- Empty roles warning Alert: fade in/out as localRoleIDs transitions to/from empty (CSS transition opacity 300ms).
- Impact badge in header: "Currently applies to ~{N} users" updates as localRoleIDs changes (sum of user_count from fetched roles data).
- ESC or clicking outside Sheet: if isDirty -> Tooltip "Unsaved changes. Are you sure?" on SheetOverlay click. Otherwise closes immediately.
- On save success: Sheet closes, DataTable Roles cell for that row reanimates with new badges.
**Accessibility**
- Sheet: `aria-label="Manage roles for filter {filterName}"`. Focus trapped inside Sheet (Radix FocusTrap). ESC closes.
- Command items: `role="option"`, `aria-selected={isSelected}`.
- Warning Alert: `role="alert"` when localRoles becomes empty.
**API Calls**
1. `useQuery({queryKey:["rls-filter-roles",id], queryFn:()=>fetch("/api/v1/rls/"+id+"/roles").then(r=>r.json()), enabled:sheetOpen&&!!id})`
2. `useMutation({mutationFn:({id,role_ids})=>fetch("/api/v1/rls/"+id+"/roles",{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({role_ids})}).then(r=>r.json())})` |
| --- | --- | --- |

---

**RLS-005** - **RLS Filter Dataset Assignment**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                                       | **API / Route**                                                           |
| --------------- | ------------ | --------- | --------------------------------------------------- | ------------------------------------------------------------------------- |
| **DEPENDENT**   | **P1**       | Phase 2   | rls_filter_tables, row_level_security_filters       | GET /api/v1/rls/:id/tables · PUT /api/v1/rls/:id/tables                   |

**Depends on:** RLS-001 (filter must exist), DS-001 (dataset list for selector population)

| **Backend - Description**
- Dedicated endpoints to manage the filter <-> dataset many-to-many independently of full CRUD. Mirrors RLS-004 pattern for the datasource axis. `datasource_type` always "table" for physical dataset assignments.
- **GET:** returns assigned datasets with full metadata: `{tables:[{datasource_id, datasource_type, table_name, schema, database_name, database_id}]}` — joined with `datasources_reporting` to hydrate human-readable names.
- **PUT:** body `{datasource_ids:[3,7,12]}` — full replacement. Compute diff (added, removed). Cache bust = union(added, removed) — stale "this filter applies" entries for removed datasets must be cleared; added datasets need fresh resolution on next query.
- Validate each datasource_id in `datasources_reporting`. Write audit log `event_type="tables_changed"`.
**Request Flow**
1. GET: `GORM.Table("rls_filter_tables rft").Select("rft.*,dr.table_name,dr.schema,dr.database_name,dr.database_id").Joins("JOIN datasources_reporting dr ON dr.id=rft.datasource_id").Where("rft.rls_id=?",id).Scan(&tables)`.
2. PUT: snapshot oldIDs. Validate: `GORM.Where("id IN ?",req.DatasourceIDs).Find(&ds)` -> length check -> 400.
3. Diff: `added:=setDiff(newIDs,oldIDs); removed:=setDiff(oldIDs,newIDs)`.
4. `GORM.Model(&filter).Association("Tables").Replace(&tableRefs)`.
5. Cache bust union(added,removed): `for _,id:=range append(added,removed...) { keys,_:=rdb.Keys(ctx,fmt.Sprintf("rls:*:%d",id)).Result(); rdb.Del(ctx,keys...) }`.
6. Audit log `event_type="tables_changed"`. Return updated tables list.
**Go Implementation**
1. GET: `db.Table("rls_filter_tables rft").Select("rft.datasource_id,rft.datasource_type,dr.table_name,dr.schema,dr.database_name,dr.database_id").Joins("JOIN datasources_reporting dr ON dr.id=rft.datasource_id").Where("rft.rls_id=?",id).Scan(&tables)`
2. PUT: `oldIDs:=getOldIDs(id); newIDs:=req.DatasourceIDs; added:=setDiff(newIDs,oldIDs); removed:=setDiff(oldIDs,newIDs)`
3. `db.Model(&filter).Association("Tables").Replace(&tableRefs)`
4. Cache bust diff: `for _,dsID:=range union(added,removed) { keys,_:=rdb.Keys(ctx,fmt.Sprintf("rls:*:%d",dsID)).Result(); rdb.Del(ctx,keys...) }`
5. Audit + return
**Security**
- Admin-only middleware.
- Diff-based cache bust: removing a dataset immediately invalidates RLS cache so that filter no longer injects for that dataset; adding a dataset immediately forces fresh resolution (no stale "no filters" entry). | **Acceptance Criteria**
- `GET /api/v1/rls/5/tables` -> 200 `{tables:[{datasource_id:3,table_name:"orders",schema:"public",database_name:"prod_pg",database_id:1}]}`.
- `PUT {datasource_ids:[3,7]}` -> replaces table list. Redis keys for added IDs (7) and removed IDs DEL'd. Audit row `tables_changed`.
- `PUT {datasource_ids:[]}` -> 200 `{tables:[]}` — filter inactive, applies to no tables.
- `PUT {datasource_ids:[3,99999]}` 99999 invalid -> 400 `{error:"One or more datasource_ids not found"}`. No partial update.
- Non-admin -> 403. Filter not found -> 404.
**Error Responses**
- 400 - datasource_id(s) not found.
- 403 - Non-admin.
- 404 - Filter not found. | **Frontend Specification**
**Route & Page**
/security/rls — `Sheet` (right side, 520px) from DataTable Actions DropdownMenu "Manage Datasets".
**shadcn/ui Components**
- `Sheet` (side=right, `className="w-[520px]"`) — "Manage Datasets" panel.
- `SheetHeader` — title "Manage Datasets", description "Assign datasets (tables) to control which queries this filter intercepts."
- `SheetContent` — two sections:
  - Section 1: "Currently Assigned" `Card` — chips: `Badge` per dataset "table_name (db_name)". Empty state: `Alert` (info) "No datasets assigned — filter is inactive."
  - Section 2: `Separator`, "Add / Remove Datasets" label, `Command` inside `Popover` (multi-select, grouped by database)
- `Command` — searchable by table_name OR database_name. Groups: `CommandGroup` per database_name (sorted alphabetically). Each item: `Checkbox` + primary text `table_name` + secondary text `database_name` (muted smaller). Virtualized list (cmdk scroll) when > 100 items.
- `Badge` chips above trigger — "{table_name} ×".
- `SheetFooter` — "Save Dataset Assignment" + "Cancel" buttons.
- `Alert` (amber) — when `localDatasetIDs.length===0`: "Warning: No datasets assigned. Saving will deactivate this filter."
- `Skeleton` — 3-row in Command while loading.
**State & TanStack Query**
- `useQuery({queryKey:["rls-filter-tables",filterId], queryFn:()=>fetch("/api/v1/rls/"+filterId+"/tables").then(r=>r.json()), enabled:sheetOpen&&!!filterId})`
- `useQuery({queryKey:["datasets-list"], queryFn:()=>fetch("/api/v1/datasets?page_size=500").then(r=>r.json()), staleTime:60_000})`
- `useMutation({mutationFn:({id,datasource_ids})=>fetch("/api/v1/rls/"+id+"/tables",{method:"PUT",body:JSON.stringify({datasource_ids})}).then(r=>r.json()), onSuccess:()=>{queryClient.invalidateQueries(["rls-filters"]); toast({title:"Datasets updated"}); setSheetOpen(false)}})`
- `localDatasetIDs:number[]` — local pending state.
- `isDirty:boolean` — enables Save button.
**UX Behaviors**
- Command grouped by database_name (CommandGroup per DB). Searchable across both table_name and database_name fields simultaneously.
- Chip click (×): removes dataset from localDatasetIDs, unchecks in Command.
- Pagination in Command: virtualized scroll (cmdk) for dataset lists > 100 items. Loading indicator at bottom if still fetching.
- Save: Loader2 during mutation. Success: Sheet closes, DataTable Tables cell updates.
- Unsaved changes: same ESC/close-outside UX as RLS-004 Sheet.
**Accessibility**
- Sheet: `aria-label="Manage datasets for filter {filterName}"`.
- Command groups: `role="group"`, `aria-label={databaseName}`.
**API Calls**
1. `useQuery({queryKey:["rls-filter-tables",id], queryFn:()=>fetch("/api/v1/rls/"+id+"/tables").then(r=>r.json()), enabled:sheetOpen&&!!id})`
2. `useMutation({mutationFn:({id,datasource_ids})=>fetch("/api/v1/rls/"+id+"/tables",{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({datasource_ids})}).then(r=>r.json())})` |
| --- | --- | --- |

---

**RLS-006** - **RLS Audit Log**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                                       | **API / Route**         |
| --------------- | ------------ | --------- | --------------------------------------------------- | ----------------------- |
| **DEPENDENT**   | **P1**       | Phase 2   | rls_audit_log, row_level_security_filters           | GET /api/v1/rls/audit   |

**Depends on:** RLS-001/004/005 (management events), RLS-003 (apply events), AUTH-004 (user identity), Asynq workers running (async apply event writes)

| **Backend - Description**
- Append-only audit log capturing all security-relevant RLS events across two write paths.
- **Synchronous writes (management events):** inside the same DB transaction as the operation: `filter_created`, `filter_updated` (diff only — changed fields in old/new JSONB, not full object), `filter_deleted`, `roles_changed`, `tables_changed`, `cache_flushed`.
- **Async writes via Asynq low queue (per-query events, 10% sample):** `filter_applied` (query_id, datasource_id, filter_name, clauses_applied_count), `admin_bypass` (query_id, user_id). Sampled to prevent log explosion at high query rates (e.g. 1000 QPS would generate 100 audit rows/sec — sampled to 10).
- **Event types and their JSONB content:**
  - `filter_created`: `old_value=null`, `new_value={name,clause,filter_type,group_key,roles,tables}`
  - `filter_updated`: `old_value={changed_fields_only}`, `new_value={changed_fields_only}` (diff computed via reflect)
  - `filter_deleted`: `old_value={name,clause,filter_type}`, `new_value=null`, `filter_id=null` (SET NULL on cascade), `filter_name` preserved
  - `roles_changed`: `old_value={role_ids:[...]}`, `new_value={role_ids:[...]}`
  - `tables_changed`: `old_value={datasource_ids:[...]}`, `new_value={datasource_ids:[...]}`
  - `filter_applied`: `query_id`, `datasource_id`, `new_value={filter_name, clauses_count}`
  - `admin_bypass`: `query_id`, `new_value={user_id, username}`
  - `cache_flushed`: `new_value={flushed_count, scope, filter_id?, datasource_id?}`
- **GET /audit** (Admin only): paginated (max page_size=100), newest first. Filters: `event_type` (comma-separated multi-value), `filter_id`, `changed_by_id`, `date_from` (ISO8601), `date_to` (ISO8601), `q` (ILIKE on filter_name + changed_by_username).
- **Retention:** Asynq scheduled task (Unique 24h) runs daily 02:00 UTC: `DELETE FROM rls_audit_log WHERE created_at < NOW() - INTERVAL '90 days'`. No API endpoint for manual deletion.
**Request Flow (write — synchronous)**
1. After successful CRUD op, inside same `tx`: `tx.Create(&RLSAuditLog{EventType:"filter_created", FilterID:&filter.ID, FilterName:filter.Name, ChangedByID:uc.UserID, ChangedByUsername:uc.Username, NewValue:toJSON(filter), IPAddress:c.ClientIP()})`. Transaction rolls back audit row if main op fails.
2. For `filter_updated`: compute diff: `oldMap:=structToMap(oldFilter); newMap:=structToMap(newFilter)` -> store only keys where values differ in old/new JSONB.

**Request Flow (write — async apply)**
1. `InjectRLS` completes -> `if rand.Float32() < 0.10 { payload,_:=json.Marshal(applyEvent); asynqClient.Enqueue(asynq.NewTask("rls:audit:apply",payload), asynq.Queue("low"), asynq.MaxRetry(2)) }`.
2. Worker: `db.Create(&RLSAuditLog{EventType:"filter_applied",...})`.

**Request Flow (read)**
1. Admin check. `db.Where(buildFilters(params)).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs); db.Model(&RLSAuditLog{}).Where(buildFilters(params)).Count(&total)`.
2. Return `{data:[AuditLog], total, page, page_size}`.

**Go Implementation**
1. Sync write: `tx.Create(&RLSAuditLog{EventType:evt, FilterID:fid, FilterName:fname, ChangedByID:uc.UserID, ChangedByUsername:uc.Username, OldValue:oldJSON, NewValue:newJSON, IPAddress:c.ClientIP(), CreatedAt:time.Now()})`
2. Async enqueue: `if rand.Float32()<0.10 { p,_:=json.Marshal(ApplyEvent{QueryID:queryID,DatasourceID:dsID,FilterName:name,UserID:uc.UserID}); asynqClient.Enqueue(asynq.NewTask("rls:audit:apply",p),asynq.Queue("low")) }`
3. Diff: `func diffFields(old,new interface{}) (json.RawMessage,json.RawMessage) { /* reflect field-by-field, return only differing fields */ }`
4. GET query: `db.Where("(? OR filter_id=?)",q==""," filter_name ILIKE ? OR changed_by_username ILIKE ?","%"+q+"%","%"+q+"%").Where("created_at BETWEEN ? AND ?",dateFrom,dateTo).Order("created_at DESC").Offset(off).Limit(sz).Find(&logs)`
5. Retention: `asynqClient.Enqueue(asynq.NewTask("rls:audit:prune",nil),asynq.ProcessAt(nextMidnightUTC()),asynq.Unique(24*time.Hour),asynq.Queue("low"))`
**Security**
- Admin-only GET. No write API endpoint — all writes internal.
- `ip_address` captured from `c.ClientIP()` (Gin, behind trusted proxy header) for forensics.
- Append-only enforced by absence of UPDATE/DELETE endpoints on rls_audit_log. Only age-based cron deletion.
- Async apply events use MaxRetry(2) — acceptable to lose some sampled events under DB pressure (not critical path). | **Acceptance Criteria**
- Create filter -> `rls_audit_log` row: `event_type="filter_created"`, `new_value={"name":"...","clause":"..."}`, `old_value=null`, `filter_id=new_id`, `changed_by_username="admin_user"`, `ip_address` set.
- Update clause only -> `event_type="filter_updated"`, `old_value={"clause":"old"}`, `new_value={"clause":"new"}`. Name, filter_type NOT in diff (unchanged).
- Delete filter -> `event_type="filter_deleted"`, `filter_id=null` (SET NULL), `filter_name="deleted_name"` preserved.
- Role change -> `event_type="roles_changed"`, `old_value={"role_ids":[1]}`, `new_value={"role_ids":[1,3]}`.
- RLS-003 apply (10% sample) -> `event_type="filter_applied"`, `query_id="q-abc"`, `datasource_id=5` in DB within 2s.
- `GET /audit?event_type=filter_updated,roles_changed&filter_id=3` -> only update + roles_changed events for filter 3.
- `GET /audit?date_from=2025-01-01T00:00:00Z&date_to=2025-01-31T23:59:59Z` -> Jan 2025 events only.
- `GET /audit?q=tenant` -> events where `filter_name ILIKE '%tenant%' OR changed_by_username ILIKE '%tenant%'`.
- `GET /audit?page_size=101` -> 400 `{error:"page_size max is 100"}`.
- Non-admin -> 403. No write endpoint -> 404 on any POST/PUT/DELETE to /audit.
**Error Responses**
- 400 - `date_from > date_to`, or `page_size > 100`.
- 403 - Non-admin.
- 404 - Any non-GET method on /audit. | **Frontend Specification**
**Route & Page**
/security/rls — "Audit Log" tab (second tab). Visible and accessible to Admin users only — tab not rendered in DOM for non-Admin (not just visually hidden). Full-width DataTable with filter controls row above.
**shadcn/ui Components**
- `Tabs` ["Filters" | "Audit Log"] — "Audit Log" tab label includes `Badge` ("Admin Only", Lock icon, `text-xs bg-muted`) inline after text
- `DataTable` (Audit Log tab, TanStack Table v8) — columns:
  - Timestamp (sortable desc default): relative time "2 min ago" + absolute in Tooltip "2025-01-15 14:32:07 UTC". `font-mono text-xs muted`.
  - Event (Badge, color-coded by event_type — see badge spec below)
  - Filter Name (clickable: click -> adds filter_id filter above table for drill-down)
  - Changed By (user display name, muted if system/cron event)
  - Details (`Button` "View", Eye icon, `variant="ghost" size="xs"`) -> opens `Popover` with diff content
  - IP Address (`font-mono text-xs muted`)
- `Badge` event_type colors:
  - `filter_created` -> `bg-green-100 text-green-800 border-green-200`
  - `filter_updated` -> `bg-blue-100 text-blue-800 border-blue-200`
  - `filter_deleted` -> `bg-red-100 text-red-800 border-red-200`
  - `roles_changed` -> `bg-purple-100 text-purple-800 border-purple-200`
  - `tables_changed` -> `bg-indigo-100 text-indigo-800 border-indigo-200`
  - `filter_applied` -> `bg-gray-100 text-gray-600 border-gray-200`
  - `admin_bypass` -> `bg-amber-100 text-amber-800 border-amber-200`
  - `cache_flushed` -> `bg-orange-100 text-orange-800 border-orange-200`
- `Popover` (Details) — opens on "View" click. `w-[480px]`, `ScrollArea max-h-[400px]`. Content:
  - Header: event_type Badge + timestamp
  - If `old_value` + `new_value` both present: two-column diff layout. Left "Before" (keys in `old_value`, red `bg-red-50` row per key). Right "After" (keys in `new_value`, green `bg-green-50` row per key). Keys shown in `font-mono text-sm`.
  - If only `new_value` (create): single column "Created with values".
  - If `query_id` set: "Query ID: {id}" row with `Button` ("View in History", ExternalLink icon) -> navigates to `/sqllab` with query pre-loaded from QE-007.
  - If `admin_bypass`: "Admin bypass recorded. No RLS filters applied for this query."
- `Select` + `Command` (event_type filter, multi-select) — "All Events" or multiple event types. Shows color Badge previews in Command items.
- `Input` (search, Search icon) — placeholder "Search by filter name or user..."
- `DateRangePicker` (`Calendar` + `Popover`, custom shadcn composed component) — presets: "Last 7 days", "Last 30 days", "Last 90 days", "Custom range"
- `Select` (filter_id filter) — "All Filters" + list of filter names from `GET /api/v1/rls?page_size=200` (names only)
- `Button` ("Export CSV", Download icon, `variant="outline" size="sm"`) — Admin only, top right of Audit tab. Triggers `GET /api/v1/rls/audit?format=csv&{currentFilters}` -> file download named `rls-audit-{date_from}-{date_to}.csv`.
- `Skeleton` — 8-row x 6-col skeleton while first load.
- Empty state: History icon + "No audit events match your filters." + `Button` "Clear Filters".
**State & TanStack Query**
- `useQuery({queryKey:["rls-audit",{page,pageSize,event_types,filter_id,changed_by_id,date_from,date_to,q}], queryFn:()=>fetch("/api/v1/rls/audit?"+qs).then(r=>r.json()), enabled:isAdmin&&activeTab==="audit", staleTime:10_000, refetchInterval:30_000})`
- `auditFilters` state: `{event_types:string[], filter_id:number|null, date_from:string|null, date_to:string|null, q:string, page:number}` — local state with `useReducer` for multi-field updates
- `isAdmin:boolean` from `useAuthStore(s=>s.user.roles.includes("Admin"))`
- Pagination: `page`, `pageSize=50` in query params. `total` from response -> `Math.ceil(total/pageSize)` pages.
**UX Behaviors**
- Audit Log tab not rendered in DOM for non-Admin (conditional render, not CSS hide).
- Auto-refresh every 30s (`refetchInterval`): subtle RefreshCw icon pulses in tab label during background refetch (opacity animation 0.5 -> 1 -> 0.5).
- Timestamp: relative ("2 minutes ago") with full ISO timestamp in Tooltip. Updates live as time passes (moment.js or date-fns `formatDistanceToNow` refreshed every 60s).
- Details Popover: diff keys are rendered in their display order. For `clause` updates: old/new clause shown full in monospace `ScrollArea`.
- `filter_applied` Popover: "View in History" button navigates to `/sqllab` with query_id passed as URL param -> SQL Lab opens History tab filtered to that query_id.
- Clicking "Filter Name" cell: adds `filter_id=X` filter to audit controls. Badge appears in active filters row above table showing "Filter: {name} ×".
- "Clear Filters" in empty state: resets all filters to defaults.
- Export CSV: only exports currently-filtered result set (not all time). Disabled (`Loader2`) while request in flight.
**Accessibility**
- Audit tab: `aria-label="Audit Log (Admin only)"`. Hidden from non-Admin users.
- Event Badge: supplemented with full text labels (not color-only). Screen reader reads "filter_created".
- Details Popover: `role="dialog"`, `aria-label="Event details for {event_type} at {timestamp}"`.
**API Calls**
1. `useQuery({queryKey:["rls-audit",auditFilters], queryFn:()=>fetch("/api/v1/rls/audit?"+new URLSearchParams(buildParams(auditFilters))).then(r=>r.json()), enabled:isAdmin&&activeTab==="audit", refetchInterval:30_000})` |
| --- | --- | --- |

---

**RLS-007** - **RLS Cache Management**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**              | **API / Route**                                                    |
| --------------- | ------------ | --------- | -------------------------- | ------------------------------------------------------------------ |
| **DEPENDENT**   | **P2**       | Phase 2   | row_level_security_filters | GET /api/v1/rls/cache/stats · POST /api/v1/rls/cache/flush         |

**Depends on:** RLS-003 (cache written by resolution engine), AUTH-004 (Admin role check), go-redis (KEYS + DEL + TTL + INFO)

| **Backend - Description**
- Manual RLS resolution cache management for admins. Two endpoints: read stats and trigger flush.
- **GET /cache/stats:** Returns live state of RLS cache in Redis. Metrics: `entry_count` (KEYS "rls:*" count, excludes rate-limit and stats keys), `oldest_ttl_seconds` (minimum TTL across sampled rls:* keys — indicates least-recently-populated entry), `newest_ttl_seconds` (maximum TTL — indicates most-recently-written entry), `memory_bytes` (estimated: `entry_count * avg_entry_size_bytes` where avg estimated from OBJECT ENCODING sample of 10 keys), `hit_rate_approx` (from `Redis INFO stats`: `keyspace_hits/(keyspace_hits+keyspace_misses)`, global Redis metric — approximate, not RLS-scoped). Stats response itself cached at `rls:cache:stats` (JSON, 30s TTL) to avoid repeated KEYS scan on busy cluster.
- **POST /cache/flush:** Three scopes:
  - `scope:"all"` — delete all `rls:*` keys (excludes `rls:rate:*` and `rls:cache:stats`). Full cold-cache impact.
  - `scope:"filter"` — resolve datasource_ids for given `filter_id` from `rls_filter_tables` -> delete `rls:*:{dsID}` for each. Surgical.
  - `scope:"datasource"` — delete only `rls:*:{datasource_id}`. Most surgical.
  - `confirm:true` required in all cases. Safety gate against accidental API calls.
  - Batched DEL (500 keys/batch) — avoids single giant DEL blocking Redis event loop.
  - Returns `{flushed_count:int, scope:string, duration_ms:int, warning:string?}`. Warning only on scope="all": "All RLS resolution cache cleared. Queries will be slower until cache re-warms (~5 min)."
  - Writes audit log `event_type="cache_flushed"`.
  - DEL's `rls:cache:stats` immediately after flush (forces stats refresh on next GET).
**Request Flow (stats)**
1. `val,err:=rdb.Get(ctx,"rls:cache:stats").Result()` -> if err==nil: return cached JSON stats.
2. `allKeys,_:=rdb.Keys(ctx,"rls:*").Result()` -> filter out `rls:rate:*` and `rls:cache:*` -> `entryKeys`.
3. Sample up to 100 keys: `rdb.TTL(ctx,key)` for each -> `minTTL`, `maxTTL`, `avgSize` (OBJECT ENCODING then estimate).
4. `info,_:=rdb.Info(ctx,"stats").Result()` -> parse `keyspace_hits` + `keyspace_misses`.
5. Build stats struct -> `rdb.Set("rls:cache:stats",toJSON(stats),30*time.Second)`.
6. Return stats.

**Request Flow (flush)**
1. Validate `confirm==true` -> 400 if not.
2. Validate `scope` in ["all","filter","datasource"] -> 400 if not.
3. If scope=filter: require `filter_id` -> resolve dsIDs -> build key patterns.
4. If scope=datasource: require `datasource_id` -> single key pattern.
5. If scope=all: `rdb.Keys(ctx,"rls:*")` -> exclude rate+stats keys.
6. Batch DEL 500/batch: `for i:=0; i<len(keys); i+=500 { rdb.Del(ctx,keys[i:min(i+500,len(keys))]...) }`.
7. `rdb.Del(ctx,"rls:cache:stats")` — force stats refresh.
8. `start` tracked -> `durationMs:=time.Since(start).Milliseconds()`.
9. Write audit log `event_type="cache_flushed"`.
10. Return `{flushed_count:len(keys), scope, duration_ms:durationMs, warning:warningIfAll}`.

**Go Implementation**
1. Stats: `keys,_:=rdb.Keys(ctx,"rls:*").Result(); entryKeys:=filterNot(keys,[]string{"rls:rate:","rls:cache:"}); sample:=entryKeys[:min(100,len(entryKeys))]; ttls:=[]int{}; for _,k:=range sample { t,_:=rdb.TTL(ctx,k).Result(); ttls=append(ttls,int(t.Seconds())) }; minTTL,maxTTL:=minMax(ttls)`
2. Flush batch: `for i:=0;i<len(keys);i+=500 { batch:=keys[i:min(i+500,len(keys))]; rdb.Del(ctx,batch...) }`
3. Scope=filter: `db.Table("rls_filter_tables").Where("rls_id=?",req.FilterID).Pluck("datasource_id",&dsIDs)` -> `for _,id:=range dsIDs { patterns,_:=rdb.Keys(ctx,fmt.Sprintf("rls:*:%d",id)).Result(); /* batch DEL */ }`
**Security**
- Admin-only endpoint pair.
- `confirm:true` required in all flush requests — structural guard against automated tooling or API explorers accidentally hitting endpoint.
- Batched DEL (500/batch): prevents single Redis DEL with thousands of keys causing latency spike on the Redis event loop. Production-safe.
- Rate limit: no separate rate limit (Admin-only, low-frequency operation by nature). | **Acceptance Criteria**
- `GET /cache/stats` -> 200 `{entry_count:42, oldest_ttl_seconds:18, newest_ttl_seconds:298, memory_bytes:86016, hit_rate_approx:0.94}`.
- Second `GET /cache/stats` within 30s -> served from `rls:cache:stats` Redis key (no KEYS scan — same latency as regular Redis GET ~1ms).
- `POST {confirm:true, scope:"all"}` -> 200 `{flushed_count:42, scope:"all", duration_ms:12, warning:"All RLS resolution cache cleared..."}`. All `rls:{hash}:{id}` keys gone. `rls:rate:*` keys preserved.
- `POST {confirm:true, scope:"datasource", datasource_id:5}` -> only `rls:*:5` pattern keys flushed. Other `rls:*` keys untouched.
- `POST {confirm:true, scope:"filter", filter_id:3}` -> resolve dsIDs for filter 3 (e.g. [5,7]) -> flush `rls:*:5` and `rls:*:7` only.
- `POST {scope:"all"}` (missing `confirm`) -> 400 `{error:"confirm:true required"}`.
- `POST {confirm:true, scope:"filter"}` (missing `filter_id`) -> 400 `{error:"filter_id required for scope=filter"}`.
- `POST {confirm:true, scope:"all"}` -> audit row `event_type="cache_flushed"`, `new_value={flushed_count:42, scope:"all"}`.
- Non-admin GET or POST -> 403.
**Error Responses**
- 400 - Missing `confirm:true`.
- 400 - Invalid scope value.
- 400 - `scope="filter"` without `filter_id`, or `scope="datasource"` without `datasource_id`.
- 403 - Non-admin.
- 500 - Redis connection error. | **Frontend Specification**
**Route & Page**
/security/rls — Admin-only toolbar above Filters DataTable. Cache management via `DropdownMenu` ("Cache" button) + stats `Popover` + flush `AlertDialog`s.
**shadcn/ui Components**
- `Button` ("Cache", DatabaseZap icon, `variant="outline" size="sm"`) — Admin-only, top toolbar right of "+ Add Filter". Opens `DropdownMenu`.
- `DropdownMenu` — items:
  - `DropdownMenuItem` ("View Cache Stats", BarChart2 icon) — opens stats `Popover` (anchored to button)
  - `DropdownMenuSeparator`
  - `DropdownMenuItem` ("Flush Cache for Dataset...", RefreshCw icon) — opens datasource-scoped flush `AlertDialog`
  - `DropdownMenuItem` ("Flush Cache for Filter...", RefreshCw icon) — opens filter-scoped flush `AlertDialog`
  - `DropdownMenuSeparator`
  - `DropdownMenuItem` ("Flush ALL RLS Cache", RefreshCw icon, `className="text-destructive"`) — opens full-flush `AlertDialog`
- `Popover` (stats, `w-[320px]`) — triggered by "View Cache Stats". Content: `Card`:
  - `CardHeader`: "RLS Cache Statistics" + `Badge` ("Live", pulsing green dot animation) + manual `Button` (RefreshCw icon, `variant="ghost" size="icon" className="h-6 w-6"`) to force refetch
  - Stats grid (2-col): "Cache Entries" / `entry_count`, "Oldest Entry" / `{oldest_ttl_seconds}s remaining`, "Hit Rate" / `{(hit_rate_approx*100).toFixed(1)}%` (color: green >80%, amber 50-80%, red <50%), "Memory" / formatted bytes (KB/MB)
  - `Badge` "Cache Warm" (green, ShieldCheck icon) if entry_count>0; `Badge` "Cache Cold" (red, ShieldOff icon) if entry_count===0
  - Footer text: "Auto-refreshes every 30s"
- `AlertDialog` (full flush) — title "Flush All RLS Cache?", description "This will clear all {entry_count} cached RLS resolutions. Queries may be slower for ~5 minutes while the cache re-warms." Body: `Input` (`font-mono placeholder="Type FLUSH to confirm"`, `aria-label="Confirmation input"`). Buttons: "Cancel" (outline) + "Flush All Cache" (destructive, `disabled={confirmText!=="FLUSH"||isPending}`).
- `AlertDialog` (scoped flush — filter) — title "Flush Cache for Filter?", description "Clear cached RLS resolutions for all datasets assigned to '{filterName}'." Body: `Select` to pick which filter (pre-filled if opened from DataTable row action). `Switch` ("I understand queries may be temporarily slower") must be toggled. Buttons: "Cancel" + "Flush" (default, `disabled={!switchOn||isPending}`).
- `AlertDialog` (scoped flush — datasource) — same pattern as filter-scoped but with `Select` to pick datasource.
- `Toast` — flush success: `{title:"Cache flushed", description:"Flushed {N} entries. Cache warming up..."}` (info variant). Redis error: destructive.
- `Skeleton` — 4-row skeleton inside stats Popover while first load (before cache warms).
**State & TanStack Query**
- `useQuery({queryKey:["rls-cache-stats"], queryFn:()=>fetch("/api/v1/rls/cache/stats").then(r=>r.json()), refetchInterval:30_000, enabled:isAdmin})`
- `useMutation({mutationFn:(body)=>fetch("/api/v1/rls/cache/flush",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)}).then(r=>r.json()), onSuccess:(r)=>{ queryClient.invalidateQueries(["rls-cache-stats"]); toast({title:`Flushed ${r.flushed_count} cache entries`,description:r.warning||"Cache cleared successfully."}) }, onError:(e)=>toast({title:"Flush failed",description:e.message,variant:"destructive"})})`
- `confirmText:string` — local state in full-flush AlertDialog. Cleared on close.
- `scopedFlushSwitchOn:boolean` — local state for scoped flush Switch. Reset on close.
- `selectedFlushFilterID:number|null` — from DataTable row action or AlertDialog Select.
**UX Behaviors**
- Stats Popover: opens inline, shows cached stats immediately (no flicker within 30s window). Manual RefreshCw button triggers `queryClient.refetchQueries(["rls-cache-stats"])`. Hit rate `%` color transitions smoothly as value changes.
- Full-flush AlertDialog: "FLUSH" Input — Confirm button enabled in real-time as user types exactly "FLUSH". `font-mono`. Mistake in typing: button stays disabled (character-by-character check `confirmText === "FLUSH"`).
- Scoped flush AlertDialog: Switch label "I understand queries for this scope may be temporarily slower." Confirm enabled only when Switch toggled on. Much lower friction than full flush (scoped impact is smaller).
- After flush: stats Popover immediately shows `entry_count:0` (query invalidated by mutation onSuccess). Badge switches "Cache Warm" -> "Cache Cold". Toast with `r.warning` if scope=all.
- "Flush Cache for Filter..." can also be triggered from DataTable Actions DropdownMenu per-row ("Flush Cache", RefreshCw icon) -> pre-fills filter_id in scoped flush AlertDialog.
- Cache button: hidden from Alpha/Gamma (not just disabled — conditional render).
**Accessibility**
- Flush AlertDialog: `aria-labelledby` = dialog title. Confirmation Input: `aria-label="Type FLUSH to confirm cache flush"`. Confirm button: `aria-disabled` when `confirmText !== "FLUSH"`.
- Stats Popover: `role="dialog"`, `aria-label="RLS Cache Statistics"`. Hit rate colored value has `title` attribute with full text "Hit rate: 94%".
- Cache button: `aria-label="RLS cache management"`.
**API Calls**
1. `useQuery({queryKey:["rls-cache-stats"], queryFn:()=>fetch("/api/v1/rls/cache/stats").then(r=>r.json()), refetchInterval:30_000})`
2. `useMutation({mutationFn:(body)=>fetch("/api/v1/rls/cache/flush",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)}).then(r=>r.json())})` |
| --- | --- | --- |

---

## **Requirements Summary**

| **ID**  | **Name**                          | **Priority** | **Dep**       | **FE Route**                                                                                          | **Endpoint(s)**                                                                                                                               | **Phase** |
| ------- | --------------------------------- | ------------ | ------------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| RLS-001 | RLS Filter CRUD                   | P0           | INDEPENDENT   | /security/rls (Filters tab — DataTable + create/edit Dialog + delete AlertDialog)                     | GET /api/v1/rls · POST /api/v1/rls · GET /api/v1/rls/:id · PUT /api/v1/rls/:id · DELETE /api/v1/rls/:id                                      | Phase 2   |
| RLS-002 | RLS Clause Validation & Preview   | P0           | DEPENDENT     | /security/rls (inline in create/edit Dialog — Textarea debounced validation + "Validate Clause" button)| POST /api/v1/rls/validate                                                                                                                     | Phase 2   |
| RLS-003 | RLS Resolution Engine (Internal)  | P0           | DEPENDENT     | Transparent — surfaced as "RLS Active" Badge + Query Info tab diff in SQL Lab results panel           | Internal `InjectRLS()` — called by QE-001, QE-004, CHT-006. No HTTP endpoint.                                                                | Phase 2   |
| RLS-004 | RLS Filter Role Assignment        | P1           | DEPENDENT     | /security/rls (Sheet 480px right — "Manage Roles" from DataTable Actions DropdownMenu)                | GET /api/v1/rls/:id/roles · PUT /api/v1/rls/:id/roles                                                                                        | Phase 2   |
| RLS-005 | RLS Filter Dataset Assignment     | P1           | DEPENDENT     | /security/rls (Sheet 520px right — "Manage Datasets" from DataTable Actions DropdownMenu)             | GET /api/v1/rls/:id/tables · PUT /api/v1/rls/:id/tables                                                                                      | Phase 2   |
| RLS-006 | RLS Audit Log                     | P1           | DEPENDENT     | /security/rls (Audit Log tab, Admin-only — DataTable with event diff Popovers + CSV export)           | GET /api/v1/rls/audit                                                                                                                         | Phase 2   |
| RLS-007 | RLS Cache Management              | P2           | DEPENDENT     | /security/rls (Admin toolbar "Cache" DropdownMenu + stats Popover + scoped/full flush AlertDialogs)   | GET /api/v1/rls/cache/stats · POST /api/v1/rls/cache/flush                                                                                   | Phase 2   |
