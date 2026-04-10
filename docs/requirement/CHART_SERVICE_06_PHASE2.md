**📈 Chart (Slice) Service**

Rank #06 · Phase 2 - Core · 7 Requirements · 0 Independent · 7 Dependent

## **Service Overview**

The Chart Service manages chart (slice) definitions - the saved visualization configs that combine a dataset, chart type, metric/dimension/filter settings, and rendering preferences. Charts are the atomic building blocks of dashboards.

The frontend provides two main surfaces: (1) the Chart List page for browse/manage operations, and (2) the Explore view - a full-featured chart builder where users select metrics, dimensions, filters, and see live query previews. The Explore view is the most complex UI in the platform.

Chart data is never stored here - configs only. Actual query execution at render time is delegated to the Query Engine (QE-001). The backend translates chart params JSON into SQL, passes it to QE, and returns structured data for the frontend chart library to render.

## **Tech Stack**

| **Layer**         | **Technology / Package**                     | **Purpose**                                           |
| ----------------- | -------------------------------------------- | ----------------------------------------------------- |
| UI Framework      | React 18 + TypeScript                        | Type-safe component tree                              |
| Bundler           | Vite 5                                       | Fast HMR and build                                    |
| Routing           | React Router v6                              | SPA navigation + nested routes                        |
| Server State      | TanStack Query v5                            | API cache, mutations, background refetch              |
| Client State      | Zustand                                      | Global UI state (sidebar, user, theme)                |
| Component Library | shadcn/ui (Radix UI primitives)              | Accessible - ALL components from here, no custom      |
| Forms             | React Hook Form + Zod                        | Schema validation, field-level errors                 |
| Data Tables       | TanStack Table v8                            | Sort, filter, paginate, row selection, virtualization |
| Styling           | Tailwind CSS v3                              | Utility-first, no custom CSS                          |
| Icons             | Lucide React                                 | Consistent icon set                                   |
| API Client        | TanStack Query (fetch)                       | No raw fetch/axios in components                      |
| Notifications     | shadcn Toaster + useToast                    | Success/error/info toasts                             |
| Charts            | Apache ECharts / Recharts (same as Superset) | Chart rendering in Explore view                       |
| DnD               | @dnd-kit/core + @dnd-kit/sortable            | Dashboard grid drag-and-drop                          |
| Layout            | shadcn ResizablePanel                        | Resizable pane layouts (SQL Lab, Explore)             |
| Backend           | Gin + GORM + encoding/json                   | Chart CRUD + config storage                           |
| Chart Rendering   | Apache ECharts (@apache/echarts-react)       | Superset-compatible chart renders                     |
| Explore State     | Zustand exploreStore                         | Tracks metrics/dims/filters live                      |
| Query Exec        | QE-001 (internal call)                       | Translates params → SQL → execute                     |
| Cache             | go-redis (SCAN+DEL pattern)                  | Chart query result cache invalidation                 |

| **Attribute**      | **Detail**                                               |
| ------------------ | -------------------------------------------------------- |
| Service Name       | Chart (Slice) Service                                    |
| Rank               | #06                                                      |
| Phase              | Phase 2 - Core                                           |
| Backend API Prefix | /api/v1/charts                                           |
| Frontend Routes    | /explore · /explore?slice_id=:id · /charts · /charts/:id |
| Primary DB Tables  | slices, slice_user                                       |
| Total Requirements | 7                                                        |
| Independent        | 0                                                        |
| Dependent          | 7                                                        |

## **Frontend Stack Notes**

Frontend stack mirrors Apache Superset: React 18 + TypeScript, Vite 5, TanStack Query v5 for all server state, Zustand for global client state, React Router v6.

Component library: shadcn/ui ONLY - no custom components. Use: Button, Input, Form, Select, Dialog, Sheet, Tabs, Table, Toast, DropdownMenu, Command, Popover, Badge, Card, Skeleton, Alert, AlertDialog, Tooltip, ScrollArea, Separator, ResizablePanel, Slider.

Forms: React Hook Form + Zod schema. All inputs via shadcn FormField/FormControl/FormMessage for consistent error display.

Data tables: shadcn DataTable + TanStack Table v8. Never raw HTML tables.

Toasts: shadcn Toaster + useToast. Success=green, error=destructive, info=default.

Loading: shadcn Skeleton for initial load. Button loading via disabled + Lucide Loader2 animate-spin. No full-page blocking spinners.

Styling: Tailwind CSS utility classes only. No inline styles, no CSS modules.

Icons: Lucide React exclusively. Semantic: Plus=create, Pencil=edit, Trash2=delete, RefreshCw=sync, LayoutDashboard=dashboard, BarChart2=chart, Play=run.

API: all calls via TanStack Query hooks (useQuery/useMutation). Never raw fetch in components. Hooks in /hooks directory.

Error handling: React Error Boundary at page level. API errors via toast onError in useMutation.

## **Requirements**

**✓ INDEPENDENT (0) - no cross-service calls required**

**⚠ DEPENDENT (7) - requires prior services/requirements**

**CHT-001** - **Create Chart**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**      | **API / Route**     |
| --------------- | ------------ | --------- | ------------------ | ------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | slices, slice_user | POST /api/v1/charts |

**⚑ Depends on:** DS-001/DS-002 (datasource_id must exist), AUTH-004 (user context)

| **⚙️ Backend - Description**
- Create a new chart definition. Validate datasource_id exists and the calling user has read access to the dataset (perm check). Required: slice_name, viz_type (e.g. "bar","line","pie","table","big_number","scatter","heatmap"), datasource_id, datasource_type ("table"). Optional: params (JSON config: metrics, groupby, filters, color_scheme, row_limit), query_context, description, cache_timeout, certified_by, certification_details.
- Set last_saved_at=NOW(), last_saved_by_fk=uid, perm and schema_perm derived from the datasource perm string. Create slice_user record linking the creator as owner.
- The params JSON is stored verbatim - no schema validation at create time. The Explore view builds params incrementally. Allow any valid JSON.
**🔄 Request Flow**
1. Validate datasource exists + user has perm → 403.
2. Derive perm from datasource.perm string.
3. GORM.Create(&slices{...LastSavedAt:now()}).
4. GORM.Create(&slice_user{SliceID:id,UserID:uid}).
5. Return 201 with chart record.
**⚙️ Go Implementation**
1. GORM.First(&ds,datasourceID) → 422 if not found
2. RequirePermission check on ds.Perm → 403
3. perm:=ds.Perm; schemaPerm:=ds.SchemaPerm
4. GORM.Create(&slices{...})
5. GORM.Create(&slice_user{SliceID:chartID,UserID:uid}) | **✅ Acceptance Criteria**
- POST /api/v1/charts { slice_name:"Revenue by Month", viz_type:"bar", datasource_id:3 } → 201 { id, slice_name, viz_type, last_saved_at }.
- datasource_id not found → 422.
- User without dataset read perm → 403.
- slice_user record created linking creator.
- params stored as-is (any valid JSON accepted).
**⚠️ Error Responses**
- 403 - No dataset access.
- 422 - Invalid datasource_id.
- 400 - Invalid params JSON. | **🖥️ Frontend Specification**
**📍 Route & Page**
/explore (new chart starts here, save triggers CHT-001)
**🧩 shadcn/ui Components**
- Dialog ("Save Chart") - triggered by "Save" Button in Explore toolbar
- Form + FormField + Input - chart name (slice_name)
- Textarea - description (optional)
- Button ("Save") - submits POST /api/v1/charts
- Toast - "Chart saved successfully" on success
- Badge (viz_type) - shown in save dialog header confirming chart type
**📦 State & TanStack Query**
- Zustand exploreStore: { datasourceId, vizType, params, queryContext }
- useMutation({ mutationFn: api.createChart, onSuccess: (c)=>{ navigate("/charts/"+c.id); toast.success("Chart saved") } })
- React Hook Form: { slice_name: z.string().min(1).max(255), description: z.string().optional() }
**✨ UX Behaviors**
- First save from Explore: opens Dialog asking for chart name.
- Subsequent saves (chart already exists): direct PUT /api/v1/charts/:id - no Dialog.
- After save: URL updates to /explore?slice_id= so the chart is bookmarkable.
- Unsaved indicator: "● Unsaved changes" Badge in Explore toolbar when params change.
- Keyboard shortcut: Ctrl+S triggers save flow.
**🛡️ Client-Side Validation**
- slice_name: z.string().min(1,"Chart name is required").max(255)
**🌐 API Calls**
1. useMutation({ mutationFn: (data)=>fetch("/api/v1/charts",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**CHT-002** - **List and Get Charts**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**      | **API / Route**                             |
| --------------- | ------------ | --------- | ------------------ | ------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | slices, slice_user | GET /api/v1/charts · GET /api/v1/charts/:id |

**⚑ Depends on:** AUTH-011 (visibility filtering by perm)

| **⚙️ Backend - Description**
- Paginated chart list with role-based visibility: Admin sees all, Alpha sees own + RBAC-permitted, Gamma sees only charts whose perm matches their grants. List includes: id, slice_name, viz_type, datasource_name (join), last_saved_at, last_saved_by_name, certified_by, dashboard_count (subquery on dashboard_slices).
- Filters: datasource_id, viz_type, owner (user_id), certified (bool), q (ILIKE on slice_name + description). Sort: last_saved_at DESC (default). Paginated default 20, max 100.
- Detail additionally includes: full params JSON, query_context, description, cache_timeout, perm, certification_details, thumbnail_url (if generated), created_by name.
**🔄 Request Flow**
1. Visibility scope from JWT roles → GORM.Where chain.
2. Apply filters + sort + paginate.
3. Subquery: dashboard_count per chart.
4. Detail: full record with joins.
**⚙️ Go Implementation**
1. GORM.Joins("LEFT JOIN slice_user su ON su.slice_id=slices.id").Where("su.user_id=? OR slices.perm IN (?)",uid,userPerms)
2. subquery dashboard_count: SELECT COUNT(*) FROM dashboard_slices WHERE slice_id=slices.id
3. Filter chain: .Where(vizType).Where(certified).Where("slice_name ILIKE ?","%"+q+"%") | **✅ Acceptance Criteria**
- GET /api/v1/charts → { items:[...], total, page }.
- GET /api/v1/charts?viz_type=bar → only bar charts.
- GET /api/v1/charts?certified=true → only certified.
- GET /api/v1/charts/:id → full detail including params JSON.
- Gamma without perm → chart excluded (not 403).
**⚠️ Error Responses**
- 404 - Not found or no access. | **🖥️ Frontend Specification**
**📍 Route & Page**
/charts
**🧩 shadcn/ui Components**
- DataTable - cols: Thumbnail, Name, Type (Badge), Dataset, Dashboards (count), Modified, Certified, Actions
- Button ("+ Chart") - opens /explore (new chart wizard)
- Input + Search icon - search by name
- Select (viz_type filter) - "All Types" &#124; Bar &#124; Line &#124; Pie &#124; Table &#124; ...
- Select (owner filter) - "All" &#124; "Mine"
- Switch ("Certified only") - filter toggle
- DropdownMenu (Actions) - Edit (→Explore), Duplicate, Add to Dashboard, Delete
- Badge (viz_type, color-coded by chart family)
- Avatar (Certified, ShieldCheck icon) - certified_by tooltip
- Tooltip on Dashboards count - hover shows dashboard names
- Skeleton - 6 loading rows
- Empty state - BarChart2 icon + "No charts yet" + "Create your first chart" Button
- Sheet (quick preview) - right-click or hover → chart thumbnail + metadata
**📦 State & TanStack Query**
- useQuery({ queryKey:["charts",filters], queryFn: ()=>api.getCharts(filters) })
- useState: { searchQ, vizType, owner, certified, page }
- useMutation for delete: onSuccess→invalidate+toast
- useMutation for duplicate: onSuccess→navigate to new chart Explore
**✨ UX Behaviors**
- Thumbnail column: 60×40px chart image (if generated), else viz_type icon placeholder.
- Type Badge: color-coded families - Line/Area=blue, Bar/Column=green, Pie/Donut=orange, Table=gray, Big Number=purple, Map=teal.
- Dashboard count Badge: click → Popover listing dashboard names with links.
- Duplicate: useMutation → onSuccess navigates to /explore?slice_id=.
- "Add to Dashboard" action: opens Command dialog to search+select dashboard.
- Bulk select (checkbox column): Delete selected, Add to Dashboard bulk actions in DataTable toolbar.
**♿ Accessibility**
- DataTable: aria-label="Charts list". Sortable column headers: aria-sort.
- Thumbnail img: alt="{slice_name} chart thumbnail".
**🌐 API Calls**
1. useQuery({ queryKey:["charts",{q,viz_type,owner,certified,page}], queryFn: ()=>fetch("/api/v1/charts?"+new URLSearchParams(filters)).then(r=>r.json()) }) |
| --- | --- | --- |


**CHT-003** - **Update Chart Configuration (Explore Save)**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**        |
| --------------- | ------------ | --------- | ------------- | ---------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | slices        | PUT /api/v1/charts/:id |

**⚑ Depends on:** CHT-001 (chart must exist), AUTH-011 (owner/Admin check)

| **⚙️ Backend - Description**
- Update chart config. Updatable: slice_name, viz_type, datasource_id (re-derive perm), params (full replace), query_context, description, cache_timeout, certified_by, certification_details. Resets last_saved_at=NOW(), last_saved_by_fk=uid.
- Post-update: invalidate Redis chart query cache (SCAN+DEL pattern on perm). If datasource changed: re-derive perm+schema_perm from new dataset. Only owner or Admin can update.
**🔄 Request Flow**
1. GORM.First(&slice,id) → ownership check.
2. If datasource changed: re-derive perm.
3. GORM.Save(&slice) with updated fields.
4. redis SCAN+DEL "qcache:"+perm+":*".
**⚙️ Go Implementation**
1. ownership: GORM.Joins("JOIN slice_user su ON su.slice_id=slices.id AND su.user_id=?",uid).First → 403
2. GORM.Save(&slice) // full struct save
3. rdb.Scan(0,"qcache:"+perm+":*",100) → pipeline DEL | **✅ Acceptance Criteria**
- PUT /api/v1/charts/:id { params:{...updated} } → 200 updated chart.
- last_saved_at/by updated to current user/time.
- Cache invalidated (next render returns from_cache:false).
- Non-owner → 403.
- Datasource change → perm recomputed.
**⚠️ Error Responses**
- 403 - Not owner.
- 422 - Invalid datasource_id. | **🖥️ Frontend Specification**
**📍 Route & Page**
/explore?slice_id=:id (Explore view - save existing)
**🧩 shadcn/ui Components**
- Button ("Save") in Explore toolbar - direct PUT (no dialog since chart already named)
- Button ("Save as") - clone flow (CHT-004) opening name Dialog
- Badge ("Saved" / "● Unsaved") - real-time dirty state indicator in toolbar
- Tooltip on "Save" - shows last_saved_at relative time ("Saved 3 min ago")
**📦 State & TanStack Query**
- exploreStore.isDirty: true when any param changes since last save
- useMutation({ mutationFn: api.updateChart, onSuccess: ()=>{ exploreStore.clearDirty(); toast.success("Chart saved") } })
- Auto-save option (configurable): useDebouncedEffect 3000ms → PUT on param change
**✨ UX Behaviors**
- Explore toolbar: "Save" Button (solid) + "Save as" Button (outline) side by side.
- Dirty state: "Save" Button shows orange dot badge "Unsaved changes".
- Ctrl+S shortcut: triggers PUT immediately.
- On success: Toast "Chart saved" + Badge changes back to "Saved".
- On failure (403): Toast "You don't have permission to save this chart. Use Save As to create a copy."
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,...data})=>fetch("/api/v1/charts/"+id,{method:"PUT",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**CHT-004** - **Duplicate Chart**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**      | **API / Route**                   |
| --------------- | ------------ | --------- | ------------------ | --------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | slices, slice_user | POST /api/v1/charts/:id/duplicate |

**⚑ Depends on:** CHT-001 (original must exist), AUTH-004 (user context)

| **⚙️ Backend - Description**
- Clone a chart with all params. New slice_name="Copy of {original_name}" (auto-increment if already taken: "Copy of X (2)"). New owner = current user. Original unchanged. Any user with read access can duplicate.
**🔄 Request Flow**
1. GORM.First(&original,id) → visibility check.
2. Resolve copy name: check conflicts → "Copy of X (N)".
3. TX: GORM.Create(&newSlice) + GORM.Create(&slice_user).
**⚙️ Go Implementation**
1. newName := resolveCopyName(original.SliceName,uid,db)
2. newSlice := original; newSlice.ID=0; newSlice.SliceName=newName; newSlice.LastSavedByFK=uid; newSlice.CreatedByFK=uid
3. TX: GORM.Create(&newSlice); GORM.Create(&slice_user{SliceID:newSlice.ID,UserID:uid}) | **✅ Acceptance Criteria**
- POST → 201 { id:, slice_name:"Copy of Revenue by Month" }.
- "Copy of X" already exists → "Copy of X (2)".
- Original unchanged.
- New chart owned by current user.
- User without read access → 403.
**⚠️ Error Responses**
- 403 - No read access.
- 404 - Original not found. | **🖥️ Frontend Specification**
**📍 Route & Page**
Triggered from /charts list or /explore toolbar ("Save as")
**🧩 shadcn/ui Components**
- Dialog ("Duplicate Chart") - name Input + description
- Input - pre-filled "Copy of {original name}", editable
- Button ("Duplicate") - submits
- Toast - "Chart duplicated. Opening in Explore..."
**📦 State & TanStack Query**
- useMutation({ mutationFn: (id)=>api.duplicateChart(id), onSuccess: (c)=>navigate("/explore?slice_id="+c.id) })
**✨ UX Behaviors**
- From chart list DropdownMenu: "Duplicate" → Dialog with auto-filled name.
- From Explore "Save as" Button: same Dialog.
- On success: navigate to the new chart's Explore page.
- User can edit the name in the Dialog before duplicating.
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,name})=>fetch("/api/v1/charts/"+id+"/duplicate",{method:"POST",body:JSON.stringify({slice_name:name})}).then(r=>r.json()) }) |
| --- | --- | --- |


**CHT-005** - **Delete Chart**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                        | **API / Route**           |
| --------------- | ------------ | --------- | ------------------------------------ | ------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | slices, slice_user, dashboard_slices | DELETE /api/v1/charts/:id |

**⚑ Depends on:** AUTH-011 (owner check), needs dashboard_slices check

| **⚙️ Backend - Description**
- Delete chart and slice_user records. Guard: if dashboard_slices.slice_id references exist → 409 with dashboard list. Guard: active report schedules targeting chart → 409. Admin force=true: remove from dashboards first. Emit audit log.
**🔄 Request Flow**
1. Ownership → count dashboard_slices → 409 if exists (unless force&&Admin).
2. If force: TX delete dashboard_slices first.
3. TX: delete slice_user, slices.
4. Redis cache invalidation.
5. Audit log.
**⚙️ Go Implementation**
1. GORM.Where("slice_id=?",id).Count on dashboard_slices → 409
2. if force && isAdmin: GORM.Where("slice_id=?",id).Delete(&dashboard_slices{})
3. TX: GORM.Where("slice_id=?",id).Delete(&slice_user{}); GORM.Delete(&slices{},id) | **✅ Acceptance Criteria**
- DELETE → 204.
- On 2 dashboards → 409 { dashboards:[{id,title}] }.
- force=true (Admin) → 204 (removed from dashboards first).
- Active report → 409.
- Non-owner → 403.
**⚠️ Error Responses**
- 403 - Not owner.
- 409 - On dashboards or active reports. | **🖥️ Frontend Specification**
**📍 Route & Page**
AlertDialog from /charts list or /explore header
**🧩 shadcn/ui Components**
- AlertDialog - delete confirmation with chart name bold
- AlertDialogDescription - lists dashboards if on any: "This chart appears on: Dashboard A, Dashboard B"
- AlertDialogAction (destructive, disabled if on dashboards) - "Delete Chart"
- Checkbox ("Also remove from all dashboards") - Admin only, enables force delete
**📦 State & TanStack Query**
- useMutation({ mutationFn: ({id,force})=>api.deleteChart(id,force), onSuccess: ()=>{ navigate("/charts"); toast.success("Chart deleted") } })
**✨ UX Behaviors**
- Standard: AlertDialog "Delete {name}? This cannot be undone."
- On dashboards: dialog expands showing dashboard list. Action button disabled.
- Admin checkbox "Remove from all dashboards" → enables Action button with warning text.
- From Explore: after delete navigate /charts.
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,force})=>fetch("/api/v1/charts/"+id+(force?"?force=true":""),{method:"DELETE"}) }) |
| --- | --- | --- |


**CHT-006** - **Chart Query / Explore Preview**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                  | **API / Route**                                                             |
| --------------- | ------------ | --------- | ------------------------------ | --------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | slices (params, query_context) | POST /api/v1/charts/:id/query · POST /api/v1/charts/query (ad-hoc, no save) |

**⚑ Depends on:** QE-001 (Query Engine execution), DS-004 (dataset metadata for SQL gen), AUTH-011 (dataset perm)

| **⚙️ Backend - Description**
- Translate chart params → SQL → execute via Query Engine → return structured data for chart rendering. This is the core "Run" action in the Explore view.
- SQL generation: read dataset columns (for calculated column expressions) and metrics (for named aggregations) → build SELECT with: metrics → GROUP BY dimensions → WHERE filters → ORDER BY → LIMIT. Inject time filter using main_dttm_col and params.time_range ("Last 7 days" → NOW()-7d etc.).
- Response: { data:[{col:val,...}], columns:[{name,type}], query:{sql,executed_sql,from_cache,start_time,end_time,duration_ms}, applied_filters:[...], rejected_filters:[...] }.
- Ad-hoc endpoint (no chart ID): accepts full params in request body - used by Explore before a chart is saved.
- force_refresh=true: bypass QE-003 Redis cache.
**🔄 Request Flow**
1. Load dataset (columns + metrics) from DB.
2. buildSQL(params, dataset) → SQL string.
3. QE-001.Execute(ctx, {DatabaseID, SQL, ForceRefresh}) → QueryResult.
4. Return data + query metadata + filter analysis.
**⚙️ Go Implementation**
1. GORM.Preload("TableColumns").Preload("SqlMetrics").First(&dataset,datasourceID)
2. buildSQL(params,dataset) - resolves metric exprs, column refs, time range
3. timeRange: parseSupersetTimeRange(params.time_range) → start,end → WHERE {mainDttmCol} BETWEEN ? AND ?
4. QE.Execute(ctx,ExecRequest{DatabaseID,SQL,ForceRefresh}) | **✅ Acceptance Criteria**
- POST → 200 { data:[...], columns:[...], query:{sql,from_cache}, applied_filters }.
- force_refresh=true → from_cache:false.
- time_range "Last 30 days" → correct WHERE clause.
- Unknown metric in params → in rejected_filters with reason.
- User without dataset access → 403.
**⚠️ Error Responses**
- 403 - No dataset access.
- 422 - Unparseable params.
- 408 - Query timeout.
- 502 - DB unreachable. | **🖥️ Frontend Specification**
**📍 Route & Page**
/explore (core Explore view - "Run" action)
**🧩 shadcn/ui Components**
- - Explore Page Layout -
- ResizablePanelGroup (horizontal) - left config panel + right chart+results panel
- - Left Config Panel -
- Card ("Datasource & Chart Type") - at top, shows current dataset + viz_type
- Button (change datasource) - opens Command dialog to search datasets
- Select (viz_type) - chart type picker with icons, grouped by family
- - Metric / Dimension Pickers -
- Accordion [Metrics &#124; Dimensions &#124; Filters &#124; Options] - collapsible config sections
- Command + Popover (metric picker) - search dataset metrics, add to list
- Badge × N (draggable, removable) - selected metrics list
- Command + Popover (dimension picker) - search groupby columns
- Badge × N (draggable) - selected dimensions
- - Filters -
- Button ("+ Add Filter") - opens Popover with column+operator+value
- Select (column) → Select (operator: =, !=, IN, BETWEEN, ILIKE) → Input (value)
- Badge × N (removable) - active filters
- DateRangePicker (shadcn Calendar+Popover) - time range filter
- - Chart Render Area (right panel) -
- Card - chart container with Apache ECharts canvas
- Button ("Run", variant=default, Play icon) - fires CHT-006
- Button ("Force Refresh", RefreshCw) - fires with force_refresh=true
- Skeleton (chart-shaped) - loading state
- Alert (destructive) - query error with SQL snippet
- - Results / Query Panel (below chart) -
- Tabs [Chart &#124; Data &#124; Query] - toggle between viz, raw data table, raw SQL
- DataTable (Data tab) - raw query results (TanStack Table)
- Textarea (Query tab) - displays executed_sql (read-only, Monaco)
- Badge (from_cache, duration_ms, rows) - metadata row
- Button (DownloadIcon) - export results (→ SQL-008)
**📦 State & TanStack Query**
- Zustand exploreStore: { datasourceId, vizType, metrics[], groupby[], filters[], timeRange, rowLimit, orderDesc, params, queryResult, queryStatus }
- useMutation({ mutationFn: api.runChartQuery, onSuccess: (r)=>exploreStore.setQueryResult(r), onError: (e)=>exploreStore.setQueryError(e) })
- useQuery({ queryKey:["chart",sliceId], enabled:!!sliceId }) - load existing chart config
- exploreStore.isDirty - tracks unsaved param changes
- exploreStore derived: paramsJSON = computed from metrics/groupby/filters for API call
**✨ UX Behaviors**
- Auto-run: when key params change (metrics, groupby added/removed) → auto-run after 500ms debounce.
- Manual Run Button: always available for explicit trigger.
- Chart render: Apache ECharts (same as Superset) via @apache/echarts-react component. Config generated from query result + viz_type.
- Data tab: shows raw result rows. "Showing 1-100 of 5,000 rows" with pagination.
- Query tab: read-only Monaco Editor (SQL mode) showing executed_sql with syntax highlighting.
- Rejected filters Alert: yellow Alert listing filters that could not be applied with reasons.
- from_cache Badge: green "Cached (3ms)" or gray "Live (234ms)".
- Resize: left panel collapsible with ResizablePanelGroup for more chart space.
**♿ Accessibility**
- Run Button: aria-label="Run chart query".
- Chart canvas: role="img" aria-label="{slice_name} chart".
**🌐 API Calls**
1. useMutation({ mutationFn: ({sliceId,params,forceRefresh})=>fetch("/api/v1/charts/"+(sliceId&#124;&#124;"")+"/query",{method:"POST",body:JSON.stringify({params,force_refresh:forceRefresh})}).then(r=>r.json()) }) |
| --- | --- | --- |


**CHT-007** - **Chart Cache Invalidation**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                | **API / Route**                          |
| --------------- | ------------ | --------- | ---------------------------- | ---------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | slices (perm, cache_timeout) | POST /api/v1/charts/:id/cache/invalidate |

**⚑ Depends on:** QE-003 (cache keyed by perm), CHT-003 (auto-called on chart update)

| **⚙️ Backend - Description**
- Scan Redis MATCH "qcache:{chart_perm}:*" → pipeline DEL all found keys → return count. Rate-limited 10/hr per chart per user. Also called internally by CHT-003 on any chart config update.
**🔄 Request Flow**
1. Ownership check → rate limit → SCAN+DEL → return {keys_deleted:N}
**⚙️ Go Implementation**
1. redis.Incr("rate:cache_inv:"+uid+":"+id) Expire(3600s) → 429 if >10
2. iter:=rdb.Scan(ctx,0,"qcache:"+perm+":*",100).Iterator()
3. pipe:=rdb.Pipeline(); for iter.Next(ctx){ pipe.Del(iter.Val()); count++ }; pipe.Exec(ctx) | **✅ Acceptance Criteria**
- POST → 200 { keys_deleted:7 }.
- Next chart render → from_cache:false.
- Rate limit → 429.
- Non-owner → 403.
**⚠️ Error Responses**
- 403 - Not owner.
- 429 - Rate limit. | **🖥️ Frontend Specification**
**📍 Route & Page**
/explore (toolbar) and /charts list (Actions menu)
**🧩 shadcn/ui Components**
- Button ("Refresh Data", RefreshCw icon) - in Explore toolbar
- DropdownMenuItem ("Clear Cache") - in chart list Actions menu
- Toast - "Cache cleared (7 keys deleted)"
- Tooltip on Refresh Button - "Force refresh from database, bypassing cache"
**📦 State & TanStack Query**
- useMutation({ mutationFn: api.invalidateChartCache, onSuccess: (r)=>toast.success("Cache cleared - "+r.keys_deleted+" entries removed") })
**✨ UX Behaviors**
- Refresh Button in Explore = invalidate cache + immediately re-run query.
- "Clear Cache" in list Actions = invalidate only (no re-run).
- Tooltip explains the difference between normal run (may use cache) and force refresh.
**🌐 API Calls**
1. useMutation({ mutationFn: (id)=>fetch("/api/v1/charts/"+id+"/cache/invalidate",{method:"POST"}).then(r=>r.json()) }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID**  | **Name**                                  | **Priority** | **Dep**     | **FE Route**                                                | **Endpoint(s)**                                                             | **Phase** |
| ------- | ----------------------------------------- | ------------ | ----------- | ----------------------------------------------------------- | --------------------------------------------------------------------------- | --------- |
| CHT-001 | Create Chart                              | P0           | ⚠ DEPENDENT | /explore (new chart starts here, save triggers CHT-001)     | POST /api/v1/charts                                                         | Phase 2   |
| CHT-002 | List and Get Charts                       | P0           | ⚠ DEPENDENT | /charts                                                     | GET /api/v1/charts · GET /api/v1/charts/:id                                 | Phase 2   |
| CHT-003 | Update Chart Configuration (Explore Save) | P0           | ⚠ DEPENDENT | /explore?slice_id=:id (Explore view - save existing)        | PUT /api/v1/charts/:id                                                      | Phase 2   |
| CHT-004 | Duplicate Chart                           | P1           | ⚠ DEPENDENT | Triggered from /charts list or /explore toolbar ("Save as") | POST /api/v1/charts/:id/duplicate                                           | Phase 2   |
| CHT-005 | Delete Chart                              | P0           | ⚠ DEPENDENT | AlertDialog from /charts list or /explore header            | DELETE /api/v1/charts/:id                                                   | Phase 2   |
| CHT-006 | Chart Query / Explore Preview             | P0           | ⚠ DEPENDENT | /explore (core Explore view - "Run" action)                 | POST /api/v1/charts/:id/query · POST /api/v1/charts/query (ad-hoc, no save) | Phase 2   |
| CHT-007 | Chart Cache Invalidation                  | P1           | ⚠ DEPENDENT | /explore (toolbar) and /charts list (Actions menu)          | POST /api/v1/charts/:id/cache/invalidate                                    | Phase 2   |