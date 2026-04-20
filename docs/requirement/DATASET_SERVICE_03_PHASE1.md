**?? Dataset Service**

Rank #03 · Phase 1 - Foundation · 10 Requirements · 7 Independent · 3 Dependent

## **Service Overview**

Manages the logical dataset layer: physical table registrations, virtual SQL datasets, column metadata, and metric definitions. Powers the Explore chart builder's datasource selection and column/metric pickers.

Frontend includes a Dataset Management list page, a two-mode creation wizard (physical vs. virtual), and a Dataset Editor with column and metric configuration tabs.

## **Tech Stack**

| **Layer**         | **Technology / Package**          | **Purpose**                                     |
| ----------------- | --------------------------------- | ----------------------------------------------- |
| UI Framework      | React 18 + TypeScript             | Type-safe component tree                        |
| Bundler           | Vite 5                            | Fast HMR and build                              |
| Routing           | React Router v6                   | SPA navigation and nested routes                |
| Server State      | TanStack Query v5                 | API cache, background refetch, mutations        |
| Client State      | Zustand                           | Global UI state (sidebar, user prefs)           |
| Component Library | shadcn/ui (Radix UI primitives)   | Accessible, unstyled - ALL components from here |
| Forms             | React Hook Form + Zod             | Validation schema, field-level errors           |
| Data Tables       | TanStack Table v8                 | Sort, filter, paginate, row selection           |
| Styling           | Tailwind CSS v3                   | Utility-first, no custom CSS                    |
| Icons             | Lucide React                      | Consistent icon set                             |
| HTTP Client       | TanStack Query (fetch under hood) | No raw fetch/axios in components                |
| Toasts            | shadcn Toaster + useToast         | Success/error/info notifications                |
| Date Picker       | shadcn Calendar + Popover         | Date/time inputs                                |
| Code Editor       | Monaco Editor (for SQL)           | SQL Lab and expression editors                  |
| Backend           | Gin + GORM + sqlparser + Asynq    | Dataset + column sync                           |

| **Attribute**      | **Detail**                                                                                     |
| ------------------ | ---------------------------------------------------------------------------------------------- |
| Service Name       | Dataset Service                                                                                |
| Rank / Build Order | #03                                                                                            |
| Phase              | Phase 1 - Foundation                                                                           |
| Backend API Prefix | /api/v1/datasets                                                                               |
| Frontend Routes    | /datasets · /datasets/new · /datasets/:id/edit · /datasets/:id/columns · /datasets/:id/metrics |
| Primary DB Tables  | tables, table_columns, sql_metrics                                                             |
| Total Requirements | 10                                                                                             |
| Independent        | 7                                                                                              |
| Dependent          | 3                                                                                              |

## **Frontend Stack Notes**

Frontend stack mirrors Apache Superset: React 18 + TypeScript, Vite bundler, TanStack Query (React Query) v5 for all server state and API calls, Zustand for global client state, React Router v6 for routing.

Component library: shadcn/ui ONLY - no custom component implementations. Use shadcn primitives: Button, Input, Form, Select, Dialog, Sheet, Table, Tabs, Toast, DropdownMenu, Command, Popover, Badge, Card, Skeleton, Alert, AlertDialog, Tooltip, ScrollArea, Separator, Avatar.

Forms: React Hook Form + Zod schema validation. All form fields must use shadcn Form wrapper with FormField, FormItem, FormLabel, FormControl, FormDescription, FormMessage for consistent error display.

Data tables: shadcn DataTable pattern with TanStack Table v8 (column defs, sorting, pagination, row selection). Never build raw HTML tables.

Notifications: shadcn Toaster + useToast hook. Success toasts = green, error toasts = red, info = default. Never use alert() or custom notification systems.

Loading states: shadcn Skeleton for initial loads. Button loading state via disabled + spinner icon (Lucide Loader2 with animate-spin). Never block UI with full-page spinners.

Styling: Tailwind CSS utility classes only. No inline styles, no CSS modules, no styled-components. Use shadcn CSS variables for theming consistency.

Icons: Lucide React exclusively. Match icon semantics: Plus for create, Pencil for edit, Trash2 for delete, RefreshCw for sync, Download for export, Eye for view, Lock for security.

API integration: all server calls via TanStack Query. useQuery for GET, useMutation for POST/PUT/DELETE. Never use fetch or axios directly in components - always through query hooks in /hooks directory.

Error handling: wrap all page-level components with React Error Boundary. API errors surfaced via toast notifications using onError callback in useMutation.

## **Requirements**

**? INDEPENDENT (7) - no cross-service calls required**

**DS-001** - **Create Physical Dataset**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**       |
| ----------------- | ------------ | --------- | ------------- | --------------------- |
| **? INDEPENDENT** | **P0**       | Phase 1   | tables        | POST /api/v1/datasets |

| **?? Backend - Description**
- Select DB + table_name + schema. Check uniqueness. Compute perm strings. Background Asynq column sync. Return 201 immediately with background_sync:true.
**?? Request Flow**
1. Validate role ? DB check ? unique check ? compute perm ? GORM.Create ? Asynq.Enqueue("dataset:sync_columns") ? 201
**?? Go Implementation**
1. perm:=fmt.Sprintf("[can_read].[%s].[%s]",db.DatabaseName,tableName)
2. GORM.Create(&tables{...})
3. asynq.Enqueue("dataset:sync_columns",{DatasetID:id}) | 
**? Acceptance Criteria**
- 201 {id,table_name,background_sync:true}.
- Duplicate ? 409.
- Gamma ? 403.
- Column sync job enqueued.
**?? Error Responses**
- 403 - Gamma role.
- 409 - Duplicate.
- 422 - Invalid database_id. | 
**??? Frontend Specification**
**?? Route & Page**
/admin/settings/datasets/new (wizard)
**?? shadcn/ui Components**
- Dialog or Page wizard with Tabs [Physical Table &#124; Virtual SQL]
- Step 1 of physical flow: Select (database) ? populated from GET /api/v1/databases
- Select (schema) ? populated from GET /databases/:id/schemas after DB selection
- Command + CommandInput - searchable table list from GET /databases/:id/tables
- CommandItem per table with TableIcon Lucide icon
- Input - dataset display name (optional override of table_name)
- Button ("Create Dataset") - submits
- Checkbox - filter: "Show views only" / "Show tables only"
- Skeleton - table list loading state
- Toast - "Dataset created. Columns are being synced..." on success
**?? State & Data Fetching**
- useState: { selectedDbId, selectedSchema, selectedTable }
- useQuery({ queryKey:["databases"] }) - DB selector options
- useQuery({ queryKey:["schemas",selectedDbId], enabled:!!selectedDbId })
- useQuery({ queryKey:["tables",selectedDbId,selectedSchema], enabled:!!selectedSchema })
- useMutation({ mutationFn: api.createDataset, onSuccess: (d)=>{ navigate("/datasets/"+d.id+"/edit"); toast("Syncing columns...") } })
**? UX Behaviors**
- Wizard flow: DB dropdown ? Schema dropdown ? Table command search.
- Table list: Command with CommandInput for real-time filter. Each item shows table name + type badge (table/view).
- After create: navigate to /admin/settings/datasets/:id/edit with Toast "Columns are being synced. Refresh to see them."
- background_sync polling: useQuery on dataset columns with refetchInterval:3000 until columns.length>0.
**??? Client-Side Validation**
- database_id required.
- table_name required.
**?? API Calls (TanStack Query)**
1. useQuery({ queryKey:["db-tables",dbId,schema] })
2. useMutation({ mutationFn: (data)=>fetch("/api/v1/datasets",{method:"POST",body:JSON.stringify(data)}) }) |
| --- | --- | --- |


**DS-002** - **Create Virtual Dataset (Custom SQL)**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**                   |
| ----------------- | ------------ | --------- | ------------- | --------------------------------- |
| **? INDEPENDENT** | **P0**       | Phase 1   | tables        | POST /api/v1/datasets (sql field) |

| **?? Backend - Description**
- Custom SQL SELECT stored as virtual dataset. sqlparser validates SELECT only + no semicolons. Optional validate_sql=true runs LIMIT 0 for semantic check + immediate column population. Jinja template syntax stored verbatim.
**?? Request Flow**
1. sqlparser.Parse ? SELECT check ? semicolon check ? optional LIMIT 0 ? GORM.Create ? (if validated: create columns inline, else: Asynq sync)
**?? Go Implementation**
1. sqlparser.Parse(sql) ? check *sqlparser.Select type
2. db.QueryContext("SELECT * FROM ("+sql+") AS t LIMIT 0") ? rows.ColumnTypes() | 
**? Acceptance Criteria**
- 201.
- Non-SELECT ? 422.
- Semicolon ? 422.
- validate_sql=true ? columns populated immediately.
**?? Error Responses**
- 422 - Non-SELECT, semicolon, semantic error. | **??? Frontend Specification**
**?? Route & Page**
/datasets/new (Virtual SQL tab)
**?? shadcn/ui Components**
- Tabs - "Physical Table" &#124; "Virtual SQL" toggle at top of wizard
- Select (database) - same as DS-001
- Input - dataset_name (required, user-defined label)
- Monaco Editor - SQL editor for custom SELECT query (read-only highlighting, autocomplete off)
- Button ("Validate SQL") - fires validate_sql=true request
- Alert - validation result (success or error with line/column info)
- Switch ("Validate before saving") - controls whether to validate_sql
- Button ("Create Dataset") - enabled after validation or if switch off
- CodeBlock (monospace) - shows expected SQL format hint
**?? State & Data Fetching**
- useState: { sql, validated, validationResult }
- useMutation({ mutationFn: api.validateSQL }) - "Validate SQL" button
- useMutation({ mutationFn: api.createDataset }) - "Create Dataset" button
- Validate first ? if success: enable Create button
**? UX Behaviors**
- Monaco Editor: dark theme, SQL syntax highlighting, line numbers, basic SQL keywords autocomplete.
- "Validate SQL" Button ? shows Loader2 ? result Alert: green "SQL is valid (42ms)" or red with error details.
- Error Alert: includes line number if parser provides it: "Error on line 3: unexpected token FROM".
- "Create Dataset" enabled only after successful validation (if switch on).
**??? Client-Side Validation**
- Client-side: detect obvious non-SELECT (starts with INSERT/UPDATE/DELETE) ? show inline warning before API call.
- Semicolon detection: warn "SQL should not contain semicolons" as user types.
- dataset_name: required, non-empty.
**?? API Calls (TanStack Query)**
1. useMutation({ mutationFn: (sql)=>fetch("/api/v1/datasets",{method:"POST",body:JSON.stringify({sql,database_id,table_name,validate_sql:true})}) }) |
| --- | --- | --- |


**DS-003** - **List and Get Datasets**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**                      | **API / Route**                                 |
| ----------------- | ------------ | --------- | ---------------------------------- | ----------------------------------------------- |
| **? INDEPENDENT** | **P0**       | Phase 1   | tables, table_columns, sql_metrics | GET /api/v1/datasets · GET /api/v1/datasets/:id |

| **?? Backend - Description**
- Paginated list with RBAC visibility. Filters: database_id, schema, type, owner, q. Detail includes columns + metrics. Column sort: order from params JSON then alpha.
**?? Request Flow**
1. Visibility scope ? filter chain ? GORM.Paginate ? detail: Preload columns+metrics
**?? Go Implementation**
1. GORM.Where("perm IN ? OR created_by_fk=? OR ?",userPerms,uid,isAdmin)
2. GORM.Preload("TableColumns","is_active=true").Preload("SqlMetrics") | **? Acceptance Criteria**
- 200 paginated list with column_count + metric_count.
- Detail includes full columns array.
- Gamma without perm ? excluded (not 403).
**?? Error Responses**
- 404 - Not found or no access. | **??? Frontend Specification**
**?? Route & Page**
/admin/settings/datasets
**?? shadcn/ui Components**
- DataTable - columns: Name, Type (Badge), Database, Schema, Owner, Columns, Metrics, Modified, Actions
- Button ("+ Dataset") - opens /datasets/new wizard
- Input + Search icon - search by name
- Select (Database filter, Schema filter, Type filter: Physical/Virtual)
- Select (Owner filter)
- DropdownMenu (Actions) - Edit, Explore, Sync Columns, Delete
- Badge (Physical/Virtual, color-coded)
- Tooltip on "Columns" count - hover shows first 5 column names
- Skeleton - loading rows
- Sheet - quick-view panel on row click (shows columns + metrics without navigating)
**?? State & Data Fetching**
- useQuery({ queryKey:["datasets",filters], queryFn: ()=>api.getDatasets(filters) })
- useState: { searchQ, dbFilter, schemaFilter, typeFilter, page }
- useMutation for delete: onSuccess: invalidate + toast
**? UX Behaviors**
- Type Badge: "Physical" (blue) / "Virtual" (purple) with different icons (Table vs Code).
- Row click ? Sheet opens with dataset summary: DB name, perm string, column list preview.
- "Explore" action ? navigates to /explore?datasource_id=X to build charts.
- "Sync Columns" action ? triggers POST /refresh, shows inline Badge on row.
- Empty state per filter: "No datasets match your filters" + clear filters link.
**?? API Calls (TanStack Query)**
1. useQuery({ queryKey:["datasets",{q,database_id,type,owner,page}], queryFn: ()=>fetch("/api/v1/datasets?"+new URLSearchParams(filters)).then(r=>r.json()) }) |
| --- | --- | --- |


**DS-004** - **Update Dataset Metadata**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**          |
| ----------------- | ------------ | --------- | ------------- | ------------------------ |
| **? INDEPENDENT** | **P0**       | Phase 1   | tables        | PUT /api/v1/datasets/:id |

| **?? Backend - Description**
- Partial update: description, main_dttm_col (must be is_dttm=true), cache_timeout (-1/0/positive), normalize_columns, filter_select_enabled, is_featured (Admin only), virtual SQL update (re-validates + re-syncs).
**?? Request Flow**
1. Ownership ? validate main_dttm_col ? validate SQL if changed ? GORM.Updates ? if SQL/table changed: Asynq.Enqueue sync
**?? Go Implementation**
1. GORM.Where("table_id=? AND column_name=? AND is_dttm=true",id,col).First ? 422
2. GORM.Model(&ds).Updates(allowedFields) | **? Acceptance Criteria**
- 200 updated.
- Invalid main_dttm_col ? 422.
- SQL update ? sync enqueued.
- is_featured by non-Admin ? 403.
**?? Error Responses**
- 403 - Not owner or is_featured by non-admin.
- 422 - Invalid main_dttm_col. | **??? Frontend Specification**
**?? Route & Page**
/admin/datasets/:id/edit (Overview tab)
**?? shadcn/ui Components**
- Tabs [Overview &#124; Columns &#124; Metrics &#124; Settings] - dataset editor page structure
- Form - Overview tab fields
- Input - dataset display name
- Textarea - description (markdown supported)
- Select - main_dttm_col (options: only is_dttm=true columns from column list)
- Input (type=number) - cache_timeout with helper text "Seconds. 0=default, -1=disabled"
- Switch - filter_select_enabled, normalize_columns
- Switch - is_featured (Admin only, hidden for non-admins)
- Monaco Editor (if virtual) - SQL edit with same validate flow as DS-002
- Button ("Save") - disabled until isDirty
- Breadcrumb - "Datasets / {dataset_name} / Edit"
**?? State & Data Fetching**
- useQuery({ queryKey:["dataset",id] }) - pre-populate form
- useQuery({ queryKey:["dataset-columns",id] }) - for main_dttm_col Select options (filtered is_dttm=true)
- useMutation({ mutationFn: api.updateDataset, onSuccess: ()=>toast.success("Dataset saved") })
- React Hook Form + Zod, isDirty for Save Button enable
**? UX Behaviors**
- main_dttm_col Select: only shows columns with is_dttm=true as options. If none exist: "No datetime columns - mark a column as datetime first" disabled Select with Alert.
- cache_timeout: Input with NumberFormat, suffix "seconds". Show computed "Expires every {Xh Ym}" helper text below.
- Virtual SQL edit: Monaco Editor in edit mode - same validation flow as DS-002 on change.
**?? API Calls (TanStack Query)**
1. useQuery(["dataset",id])
2. useMutation({ mutationFn: (data)=>fetch("/api/v1/datasets/"+id,{method:"PUT",body:JSON.stringify(data)}) }) |
| --- | --- | --- |


// TODO: unkown tasks
**DS-005** - **Update Column Metadata (Single + Bulk)**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                             |
| ----------------- | ------------ | --------- | ------------- | --------------------------------------------------------------------------- |
| **? INDEPENDENT** | **P0**       | Phase 1   | table_columns | PUT /api/v1/datasets/:id/columns/:col_id · PUT /api/v1/datasets/:id/columns |

| **?? Backend - Description**
- Per-column: verbose_name, description, filterable, groupby, is_dttm override, python_date_format, expression (validated via sqlparser.ParseExpr), type override, exported. Bulk PUT TX: all or nothing.
**?? Request Flow**
1. Ownership ? validate expression (sqlparser) ? validate date format ? GORM.Updates or TX bulk update
**?? Go Implementation**
1. sqlparser.ParseExpr(expression) ? 422 if err
2. db.Transaction(func(tx){ for each col: tx.Model.Where.Updates }) | **? Acceptance Criteria**
- 200 single.
- Bulk: all updated or none (TX).
- Invalid expression ? 422.
- Non-owner ? 403.
**?? Error Responses**
- 403 - Not owner.
- 422 - Invalid expression. | **??? Frontend Specification**
**?? Route & Page**
/admin/datasets/:id/edit (Columns tab)
**?? shadcn/ui Components**
- DataTable - columns: Column Name, Verbose Name, Type, Is DateTime, Filterable, Group By, Expression, Actions
- Inline editable cells (click to edit) using shadcn Popover + Input/Switch pattern
- Popover + Input - verbose_name inline edit on cell click
- Switch (inline) - filterable, groupby, is_dttm toggles per row
- Popover + Monaco Editor (mini) - expression inline editor
- Sheet (row expand) - full column detail edit (all fields at once)
- Button ("Save All Changes") - bulk PUT trigger
- Badge ("N unsaved changes") - count of modified rows
- Badge (Inactive, muted) - for is_active=false columns
- Tooltip - "This is a calculated column" for rows with expression set
- Button ("Add Calculated Column") - adds new row with expression editor
**?? State & Data Fetching**
- useQuery({ queryKey:["dataset-columns",id] }) - column list
- useState: localEdits (Map>) - track unsaved changes
- isDirty: localEdits.size > 0
- useMutation({ mutationFn: ()=>api.bulkUpdateColumns(id, Array.from(localEdits)) })
- Single save: useMutation({ mutationFn: (col)=>api.updateColumn(id,col.id,col) })
**? UX Behaviors**
- Inline editing: click a cell ? Popover opens with Input prefilled ? blur/Enter saves to localEdits (not API).
- "Save All Changes" Button persists all localEdits via bulk PUT.
- Unsaved rows: highlighted with amber left border.
- "N unsaved changes" Badge in tab header.
- Discard: "Reset" Button reverts localEdits to server state.
- Expression column: Popover with Monaco Editor mini (100px height). Real-time syntax validation.
**??? Client-Side Validation**
- expression: client-side basic SQL syntax check (warn, don't block).
- python_date_format: Zod regex for valid strftime pattern.
**?? API Calls (TanStack Query)**
1. useMutation({ mutationFn: (cols)=>fetch("/api/v1/datasets/"+id+"/columns",{method:"PUT",body:JSON.stringify(cols)}) }) |
| --- | --- | --- |


**DS-006** - **Create & Manage Dataset Metrics**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                                 |
| ----------------- | ------------ | --------- | ------------- | ------------------------------------------------------------------------------- |
| **? INDEPENDENT** | **P0**       | Phase 1   | sql_metrics   | POST/PUT/DELETE /api/v1/datasets/:id/metrics · GET /api/v1/datasets/:id/metrics |

| **?? Backend - Description**
- Named SQL aggregate metrics. expression must contain aggregate function (AST walker check: SUM/COUNT/MAX/MIN/AVG etc). Bulk replace-all PUT. Delete warns if referenced in charts.
**?? Request Flow**
1. Parse expression ? walk AST for AggregateFunc ? unique check ? GORM.Create. Bulk: TX delete+insert. Delete: scan slices.params for reference ? warn
**?? Go Implementation**
1. type aggWalker struct{found bool}; sqlparser.Walk(&walker,expr)
2. aggFuncs:=map[string]bool{"sum":true,"count":true,...} | **? Acceptance Criteria**
- 201.
- No-aggregate expression ? 422.
- Duplicate name ? 409.
- Delete referenced ? 200 with warnings.
**?? Error Responses**
- 409 - Duplicate name.
- 422 - No aggregate function. | **??? Frontend Specification**
**?? Route & Page**
/admin/settings/datasets/:id/edit (Metrics tab)
**?? shadcn/ui Components**
- DataTable - columns: Metric Name, Verbose Name, Expression, Type, Format, Certified, Actions
- Button ("+ Add Metric") - opens Dialog
- Dialog + DialogContent - create/edit metric form
- Form + FormField + Input - metric_name, verbose_name
- Select - metric_type (Sum, Count, Average, Max, Min, Count Distinct, Custom)
- Monaco Editor (mini, SQL mode) - expression field
- Input - d3format with preview ("$1,234.56" live format preview)
- Textarea - warning_text (optional)
- Switch - is_restricted
- AlertDialog - delete confirmation with chart references list if any
- Badge (Certified) - certifed_by display with ShieldCheck icon
**?? State & Data Fetching**
- useQuery({ queryKey:["dataset-metrics",id] })
- useMutation({ mutationFn: api.createMetric, onSuccess: ()=>{ queryClient.invalidateQueries(["dataset-metrics",id]); dialog.close(); toast.success("Metric created") } })
- useMutation({ mutationFn: api.deleteMetric, onSuccess: (r)=>{ if(r.warnings) toast.warning("Metric deleted. "+r.warnings.length+" charts may be affected") } })
**? UX Behaviors**
- d3format preview: type ",.2f" in format Input ? live preview shows "1,234.56" below field.
- Metric type Select ? auto-suggests expression: "Sum" ? "SUM()", "Count" ? "COUNT(*)".
- Expression Monaco Editor: validates aggregate function client-side (warn if no SUM/COUNT/etc.).
- Delete: if warnings in response ? Toast (warning, not error): "Metric deleted. 3 charts may show errors."
- Certified badge: click opens Popover with certified_by + certification_details.
**??? Client-Side Validation**
- metric_name: /^[a-z][a-z0-9_]*$/ - snake_case, min 3 chars.
- expression: warn if no aggregate keyword detected (client-side heuristic, server does full check).
- d3format: validate against known d3-format spec patterns.
**?? API Calls (TanStack Query)**
1. useMutation({ mutationFn: (m)=>fetch("/api/v1/datasets/"+id+"/metrics",{method:"POST",body:JSON.stringify(m)}) })
2. useMutation({ mutationFn: (mId)=>fetch("/api/v1/datasets/"+id+"/metrics/"+mId,{method:"DELETE"}) }) |
| --- | --- | --- |


**DS-008** - **Delete Dataset**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**                      | **API / Route**             |
| ----------------- | ------------ | --------- | ---------------------------------- | --------------------------- |
| **? INDEPENDENT** | **P0**       | Phase 1   | tables, table_columns, sql_metrics | DELETE /api/v1/datasets/:id |

| **?? Backend - Description**
- Guard: count charts using dataset. Optional force=true (Admin) deletes charts first. TX cascade: table_columns, sql_metrics, rls_filter_tables, tagged_object, tables. Redis qcache cleanup.
**?? Request Flow**
1. Ownership ? count charts ? if force&&Admin: delete slices first ? TX cascade ? redis cleanup ? audit
**?? Go Implementation**
1. GORM.Where("datasource_id=?",id).Count ? 409
2. TX: Delete table_columns,sql_metrics,tables
3. redis SCAN+DEL "qcache:"+perm+":*" | **? Acceptance Criteria**
- 204.
- Has charts ? 409 with chart list.
- force=true (Admin) ? 204 with charts deleted.
**?? Error Responses**
- 403 - Not owner.
- 409 - Referenced by charts. | **??? Frontend Specification**
**?? Route & Page**
AlertDialog from /datasets list or /datasets/:id/edit page
**?? shadcn/ui Components**
- AlertDialog - full delete confirmation
- AlertDialogDescription - shows chart count if any ("This dataset is used by 5 charts")
- DataTable (mini, inside dialog) - list of dependent charts with links
- AlertDialogAction (destructive, disabled if charts exist and no force) - "Delete Dataset"
- Checkbox ("Also delete all dependent charts") - Admin only, enables force=true
**?? State & Data Fetching**
- useMutation({ mutationFn: ({id,force})=>fetch("/api/v1/datasets/"+id+"?force="+force,{method:"DELETE"}) })
- Pre-fetch chart_count to configure dialog
**? UX Behaviors**
- If no charts: simple AlertDialog "Delete {name}? This cannot be undone.".
- If has charts: expanded AlertDialog with chart list. Action button disabled.
- Admin only: "Also delete dependent charts" Checkbox ? enables force delete. Action button becomes active.
- On success: navigate /datasets + Toast "Dataset deleted".
**?? API Calls (TanStack Query)**
1. useMutation({ mutationFn: ({id,force})=>fetch("/api/v1/datasets/"+id+(force?"?force=true":""),{method:"DELETE"}) }) |
| --- | --- | --- |


**? DEPENDENT (3) - requires prior services/requirements**

**DS-003** - **Column Auto-Sync from Remote Schema**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                        |
| --------------- | ------------ | --------- | ------------- | ---------------------------------------------------------------------- |
| **? DEPENDENT** | **P0**       | Phase 1   | table_columns | POST /api/v1/datasets/:id/refresh · Asynq worker: dataset:sync_columns |

**? Depends on:** DBC-007 (Schema Introspection) or DBC-006 (pool for virtual LIMIT 0)

| **?? Backend - Description**
- Asynq worker: physical?DBC-007, virtual?LIMIT 0. Upsert table_columns (ON CONFLICT update type+is_active only). Mark missing columns is_active=false. Preserve user customizations (verbose_name, description, etc.).
**?? Request Flow**
1. Worker: GORM.First(dataset) ? inspector.ListColumns or LIMIT 0 ? build ColumnMeta ? GORM.Clauses(OnConflict).CreateInBatches ? mark missing inactive
**?? Go Implementation**
1. OnConflict: DoUpdates(["type","is_active","changed_on"]) - only these fields updated on conflict
2. GORM.Where("table_id=? AND column_name NOT IN ?",id,remoteNames).Update("is_active",false) | **? Acceptance Criteria**
- Refresh ? 202 {job_id}.
- New column added after sync.
- Removed column is_active=false.
- User verbose_name preserved.
- Timestamp column is_dttm=true.
**?? Error Responses**
- 404 - Dataset not found.
- 502 - DB unreachable. | **??? Frontend Specification**
**?? Route & Page**
/datasets/:id/edit (Columns tab - shows sync status)
**?? shadcn/ui Components**
- Button ("Sync Columns") with RefreshCw Lucide icon - triggers POST /datasets/:id/refresh
- Badge ("Syncing..." with Loader2) - shown while job in progress
- Badge (green "Synced") or (red "Sync failed") - final state
- Tooltip on sync Badge - "Last synced: 3 minutes ago"
- Alert (info) - "X new columns added, Y columns deactivated" after sync
- DataTable - column list (updated after sync)
**?? State & Data Fetching**
- useMutation({ mutationFn: ()=>api.refreshDataset(id) }) - returns {job_id}
- useQuery({ queryKey:["job",jobId], refetchInterval:2000 }) - poll job status
- On job success: queryClient.invalidateQueries(["dataset-columns",id]) ? table updates
**? UX Behaviors**
- "Sync Columns" Button ? shows Loader2 + "Syncing..." Badge while polling.
- On completion: success Alert "Sync complete: 3 new columns, 1 deactivated".
- Deactivated columns shown in table with strikethrough + muted color + "Inactive" Badge.
**?? API Calls (TanStack Query)**
1. useMutation({ mutationFn: ()=>fetch("/api/v1/datasets/"+id+"/refresh",{method:"POST"}).then(r=>r.json()) })
2. useQuery({ queryKey:["job",jobId], enabled:!!jobId, refetchInterval:2000 }) |
| --- | --- | --- |


**DS-009** - **Dataset Cache Policy**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**          | **API / Route**                                                                                           |
| --------------- | ------------ | --------- | ---------------------- | --------------------------------------------------------------------------------------------------------- |
| **? DEPENDENT** | **P1**       | Phase 2   | tables (cache_timeout) | PUT /api/v1/datasets/:id · POST /api/v1/datasets/:id/cache/flush · POST /api/v1/datasets/:id/cache/warmup |

**? Depends on:** DS-001/DS-002 (dataset exists), QE-003 (Query Engine reads cache_timeout)

| **?? Backend - Description**
- cache_timeout: -1=disabled, 0=global default, positive=seconds. Flush: redis SCAN+DEL qcache pattern. Warmup: Asynq jobs per chart.
**?? Request Flow**
1. PUT cache_timeout ? GORM.Update. Flush: SCAN+DEL. Warmup: find charts ? Asynq.Enqueue per chart
**?? Go Implementation**
1. redis SCAN "qcache:"+perm+":*" ? pipeline DEL ? count
2. GORM.Where("datasource_id=?",id).Find(&charts) ? for each: asynq.Enqueue | **? Acceptance Criteria**
- cache_timeout=-1 ? no cache.
- Flush ? {keys_deleted:N}.
- Warmup ? {jobs_enqueued:N}.
**?? Error Responses**
- 422 - Invalid cache_timeout value.
- 429 - Flush rate limit. | **??? Frontend Specification**
**?? Route & Page**
/datasets/:id/edit (Settings tab)
**?? shadcn/ui Components**
- Card ("Cache Settings") - section in Settings tab
- RadioGroup - cache_timeout mode: "Use global default (0)" &#124; "Disable (-1)" &#124; "Custom"
- Input (type=number, conditional) - shown only when "Custom" selected, suffix "seconds"
- Button ("Flush Cache") with Trash2 icon - POST /cache/flush
- Button ("Warm Up Cache") with Zap icon - POST /cache/warmup
- Alert (info) - shows keys_deleted count after flush
- Progress - warmup job progress (polls N jobs)
- Badge - "Last flushed: 2 min ago" timestamp
**?? State & Data Fetching**
- useMutation({ mutationFn: ()=>api.flushCache(id), onSuccess: (r)=>setFlushResult(r) })
- useMutation({ mutationFn: ()=>api.warmupCache(id), onSuccess: (r)=>setWarmupJobs(r.chart_ids) })
- useQuery polling for each warmup job
**? UX Behaviors**
- RadioGroup: "Use global default" selected by default.
- Custom: Input enabled, placeholder "e.g. 3600".
- Flush: Button ? Loader2 ? Alert "42 cache keys cleared".
- Warmup: Button ? Progress tracks N chart jobs ? "7/7 charts warmed up".
**?? API Calls (TanStack Query)**
1. useMutation({ mutationFn: ()=>fetch("/api/v1/datasets/"+id+"/cache/flush",{method:"POST"}) })
2. useMutation({ mutationFn: ()=>fetch("/api/v1/datasets/"+id+"/cache/warmup",{method:"POST"}) }) |
| --- | --- | --- |


**DS-010** - **RLS Dataset Assignment**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**     | **API / Route**                                             |
| --------------- | ------------ | --------- | ----------------- | ----------------------------------------------------------- |
| **? DEPENDENT** | **P1**       | Phase 2   | rls_filter_tables | PUT /api/v1/datasets/:id/rls · GET /api/v1/datasets/:id/rls |

**? Depends on:** AUTH-011 (Admin RBAC), RLS-001 (RLS filters exist)

| **?? Backend - Description**
- Admin: replace-all RLS filter associations. Validate filter IDs. TX delete+insert. Invalidate Redis RLS cache for affected users. Publish rls:invalidated pub/sub.
**?? Request Flow**
1. Admin check ? validate filter IDs ? TX delete+insert ? bust RLS cache ? pub/sub
**?? Go Implementation**
1. TX: GORM.Where("table_id=?",id).Delete; GORM.CreateInBatches(newRows)
2. rdb.Publish("rls:invalidated",datasetID) | **? Acceptance Criteria**
- 200 {assigned:N}.
- GET /rls ? filters with clauses+roles.
- Non-admin ? 403.
- Invalid filter_id ? 422.
**?? Error Responses**
- 403 - Non-admin.
- 422 - Invalid filter ID. | **??? Frontend Specification**
**?? Route & Page**
/datasets/:id/edit (Settings tab - "Row Level Security" section)
**?? shadcn/ui Components**
- Card ("Row Level Security") - in Settings tab, Admin-only visible
- MultiSelect (Command+Popover pattern) - select from available RLS filters
- Badge × N (removable) - currently assigned filters shown as Tags with X button
- Tooltip on each Badge - shows filter clause and type
- Button ("Save RLS") - calls PUT /rls
- Alert (warning) - "Saving will immediately affect all queries for affected user roles"
- Alert (info, shown if empty) - "No RLS filters assigned - all users see all data"
**?? State & Data Fetching**
- useQuery({ queryKey:["dataset-rls",id] }) - current assignments
- useQuery({ queryKey:["rls-filters"] }) - all available filters for multi-select
- useMutation({ mutationFn: (ids)=>api.setDatasetRLS(id,ids) })
**? UX Behaviors**
- MultiSelect: Command component, type to search filters by name.
- Each selected filter shown as Badge with filter name + clause snippet in Tooltip.
- Warning Alert before save: "Changing RLS will affect live queries immediately."
- Confirmation Dialog before save if filters are removed (reducing restrictions).
**?? API Calls (TanStack Query)**
1. useMutation({ mutationFn: (ids)=>fetch("/api/v1/datasets/"+id+"/rls",{method:"PUT",body:JSON.stringify({rls_filter_ids:ids})}) }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID** | **Name**                               | **Priority** | **Dep**       | **FE Route**                                                     | **Endpoint(s)**                                                                                           | **Phase** |
| ------ | -------------------------------------- | ------------ | ------------- | ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | --------- |
| DS-001 | Create Physical Dataset                | P0           | ? INDEPENDENT | /datasets/new (wizard)                                           | POST /api/v1/datasets                                                                                     | Phase 1   |
| DS-002 | Create Virtual Dataset (Custom SQL)    | P0           | ? INDEPENDENT | /datasets/new (Virtual SQL tab)                                  | POST /api/v1/datasets (sql field)                                                                         | Phase 1   |
| DS-004 | List and Get Datasets                  | P0           | ? INDEPENDENT | /datasets                                                        | GET /api/v1/datasets · GET /api/v1/datasets/:id                                                           | Phase 1   |
| DS-005 | Update Dataset Metadata                | P0           | ? INDEPENDENT | /datasets/:id/edit (Overview tab)                                | PUT /api/v1/datasets/:id                                                                                  | Phase 1   |
| DS-006 | Update Column Metadata (Single + Bulk) | P0           | ? INDEPENDENT | /datasets/:id/edit (Columns tab)                                 | PUT /api/v1/datasets/:id/columns/:col_id · PUT /api/v1/datasets/:id/columns                               | Phase 1   |
| DS-007 | Create & Manage Dataset Metrics        | P0           | ? INDEPENDENT | /datasets/:id/edit (Metrics tab)                                 | POST/PUT/DELETE /api/v1/datasets/:id/metrics · GET /api/v1/datasets/:id/metrics                           | Phase 1   |
| DS-008 | Delete Dataset                         | P0           | ? INDEPENDENT | AlertDialog from /datasets list or /datasets/:id/edit page       | DELETE /api/v1/datasets/:id                                                                               | Phase 1   |
| DS-003 | Column Auto-Sync from Remote Schema    | P0           | ? DEPENDENT   | /datasets/:id/edit (Columns tab - shows sync status)             | POST /api/v1/datasets/:id/refresh · Asynq worker: dataset:sync_columns                                    | Phase 1   |
| DS-009 | Dataset Cache Policy                   | P1           | ? DEPENDENT   | /datasets/:id/edit (Settings tab)                                | PUT /api/v1/datasets/:id · POST /api/v1/datasets/:id/cache/flush · POST /api/v1/datasets/:id/cache/warmup | Phase 2   |
| DS-010 | RLS Dataset Assignment                 | P1           | ? DEPENDENT   | /datasets/:id/edit (Settings tab - "Row Level Security" section) | PUT /api/v1/datasets/:id/rls · GET /api/v1/datasets/:id/rls                                               | Phase 2   |