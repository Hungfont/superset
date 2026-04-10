**💻 SQL Lab Service**

Rank #05 · Phase 2 - Core · 8 Requirements · 0 Independent · 8 Dependent

## **Service Overview**

SQL Lab is the interactive SQL IDE. The backend manages tab state, saved queries, and schema browser persistence. Execution is delegated to the Query Engine.

Frontend: a full-featured IDE page at /sqllab with Monaco Editor, schema browser sidebar, result table, multi-tab support, and saved query library. Mirrors Apache Superset's SQL Lab UX with shadcn components.

## **Tech Stack**

| **Layer**         | **Technology / Package**             | **Purpose**                                     |
| ----------------- | ------------------------------------ | ----------------------------------------------- |
| UI Framework      | React 18 + TypeScript                | Type-safe component tree                        |
| Bundler           | Vite 5                               | Fast HMR and build                              |
| Routing           | React Router v6                      | SPA navigation and nested routes                |
| Server State      | TanStack Query v5                    | API cache, background refetch, mutations        |
| Client State      | Zustand                              | Global UI state (sidebar, user prefs)           |
| Component Library | shadcn/ui (Radix UI primitives)      | Accessible, unstyled - ALL components from here |
| Forms             | React Hook Form + Zod                | Validation schema, field-level errors           |
| Data Tables       | TanStack Table v8                    | Sort, filter, paginate, row selection           |
| Styling           | Tailwind CSS v3                      | Utility-first, no custom CSS                    |
| Icons             | Lucide React                         | Consistent icon set                             |
| HTTP Client       | TanStack Query (fetch under hood)    | No raw fetch/axios in components                |
| Toasts            | shadcn Toaster + useToast            | Success/error/info notifications                |
| Date Picker       | shadcn Calendar + Popover            | Date/time inputs                                |
| Code Editor       | Monaco Editor (for SQL)              | SQL Lab and expression editors                  |
| SQL Editor        | Monaco Editor (@monaco-editor/react) | SQL syntax highlighting + autocomplete          |
| CSV Export        | encoding/csv (Go)                    | Streaming CSV download                          |
| Excel Export      | github.com/xuri/excelize (Go)        | XLSX generation                                 |

| **Attribute**      | **Detail**                           |
| ------------------ | ------------------------------------ |
| Service Name       | SQL Lab Service                      |
| Rank / Build Order | #05                                  |
| Phase              | Phase 2 - Core                       |
| Backend API Prefix | /api/v1/sqllab                       |
| Frontend Routes    | /sqllab · /sqllab/saved-queries      |
| Primary DB Tables  | saved_query, tab_state, table_schema |
| Total Requirements | 8                                    |
| Independent        | 0                                    |
| Dependent          | 8                                    |

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

**✓ INDEPENDENT (0) - no cross-service calls required**

**⚠ DEPENDENT (8) - requires prior services/requirements**

**SQL-001** - **Create and Restore SQL Lab Editor Tabs**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                                  |
| --------------- | ------------ | --------- | ------------- | -------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | tab_state     | POST /api/v1/sqllab/tabs · GET /api/v1/sqllab/tabs · GET /api/v1/sqllab/tabs/:id |

**⚑ Depends on:** AUTH-004 (user context), DBC-001 (db_id must be visible)

| **⚙️ Backend - Description**
- Create tab with db_id, schema, label (auto "Untitled Query N"), sql, query_limit. Restore: GET all active tabs ordered by created_on. latest_query status included for badge display.
**🔄 Request Flow**
1. Validate db_id visibility → auto-label → GORM.Create → return 201. GET: GORM.Where(user,active=true).Preload(LatestQuery).Order(created_on)
**⚙️ Go Implementation**
1. autoLabel: GORM.Where("user_id=? AND label LIKE ?",uid,"Untitled Query%").Find → maxN → "Untitled Query N+1"
2. GORM.Create(&tab_state{Active:true,...}) | **✅ Acceptance Criteria**
- 201 {id,label,active:true}.
- Auto-label increments correctly.
- GET returns cross-device tabs.
- db_id not visible → 422.
**⚠️ Error Responses**
- 422 - Not visible db_id. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab
**🧩 shadcn/ui Components**
- - SQL Lab Page Layout -
- ResizablePanel + ResizablePanelGroup + ResizableHandle (shadcn) - 3-pane layout: Schema Browser &#124; Editor+Results &#124; (optional detail)
- - Tab Bar (top) -
- Tabs + TabsList - horizontal tab strip above editor
- TabsTrigger per tab - label + close X button + status badge
- Button ("+", size=icon) - add new tab → POST /sqllab/tabs
- DropdownMenu on tab right-click - Close, Close All, Rename
- - Schema Browser (left panel) -
- ScrollArea - scrollable schema tree
- Collapsible + CollapsibleTrigger + CollapsibleContent - schema > tables > columns tree
- Input + Search icon - schema search
- Select (schema) - schema switcher at top of browser
- Skeleton × 5 - loading state for schema list
- - Editor Panel (center) -
- Monaco Editor (SQL mode, dark theme) - main SQL editor
- Button ("Run", size=sm) with Play icon - execute selected SQL or all
- Button ("Run All") - executes entire editor content
- Select (limit) - result row limit dropdown (100/1000/10000)
- Button ("Save") with Save icon - opens saved query Dialog
- Badge (db_name, schema_name) - current connection indicator
- - Results Panel (bottom, inside TabsContent) -
- Tabs [Results &#124; Query Details &#124; Saved Queries]
- DataTable (TanStack Table) - query results with sort + pagination
- Button ("Download CSV/Excel") - export dropdown
- DropdownMenu - format selector for download
- Alert (destructive) - query error display
- Skeleton - results loading state
- Badge (from_cache, latency_ms, rows_count) - query metadata
**📦 State & Data Fetching**
- Zustand sqlLabStore: { tabs: Tab[], activeTabId, addTab, closeTab, updateTab }
- Tab state: { id, label, dbId, schema, sql, queryLimit, latestQueryId, resultData, queryStatus }
- useQuery({ queryKey:["sqllab-tabs"], queryFn: api.getTabs, onSuccess: (tabs)=>sqlLabStore.initTabs(tabs) }) - on page mount
- useMutation({ mutationFn: api.createTab, onSuccess: (tab)=>sqlLabStore.addTab(tab) })
- Auto-save: useEffect with debounce(1000ms) → PUT /sqllab/tabs/:id on sql/schema change
**✨ UX Behaviors**
- Page load: GET /sqllab/tabs → restore all active tabs. First tab auto-selected.
- Tab strip: each TabsTrigger shows label + status icon (⏳ running / ✅ success / ❌ error) + × close.
- Tab double-click → inline label rename (Input replaces label, blur saves via PUT).
- Editor: Monaco with SQL syntax, line numbers, keyboard shortcut Ctrl+Enter = Run.
- Ctrl+Enter: executes selected text if selection exists, else full editor content.
- Schema browser: click column name → inserts column name at cursor position in Monaco.
- Results: DataTable with sticky header, virtual scrolling for 10k+ rows (TanStack Table virtualized).
- from_cache Badge: green "Cached (42ms)" or gray "Live (234ms)".
**♿ Accessibility (a11y)**
- Monaco Editor: aria-label="SQL Editor".
- Tab strip: role="tablist" with aria-label="SQL Editor Tabs".
- Run Button: aria-label="Run Query (Ctrl+Enter)".
**🌐 API Calls (TanStack Query)**
1. useQuery({ queryKey:["sqllab-tabs"], queryFn: ()=>fetch("/api/v1/sqllab/tabs").then(r=>r.json()) })
2. useMutation({ mutationFn: (data)=>fetch("/api/v1/sqllab/tabs",{method:"POST",body:JSON.stringify(data)}) })
3. Auto-save: debounced fetch("/api/v1/sqllab/tabs/"+id,{method:"PUT",body:JSON.stringify({sql,schema})}) |
| --- | --- | --- |


**SQL-002** - **Auto-Save Tab SQL and Editor State**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**             |
| --------------- | ------------ | --------- | ------------- | --------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | tab_state     | PUT /api/v1/sqllab/tabs/:id |

**⚑ Depends on:** SQL-001 (tab must exist)

| **⚙️ Backend - Description**
- Partial update: sql (max 64KB), schema, db_id, query_limit, label, latest_query_id, hide_left_bar, extra_json. Only non-null provided fields updated. Ownership enforced.
**🔄 Request Flow**
1. Ownership → validate sql size → GORM.Model.Updates(nonZeroFields)
**⚙️ Go Implementation**
1. nonZeroFields from JSON body (distinguish null vs absent)
2. GORM.Model(&tab_state{ID:id}).Updates(fields) | **✅ Acceptance Criteria**
- 200 updated.
- SQL > 64KB → 422.
- Other user → 403.
- Partial update: only provided fields change.
**⚠️ Error Responses**
- 403 - Not owner.
- 422 - SQL > 64KB. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (transparent auto-save, no UI)
**🧩 shadcn/ui Components**
- No dedicated component - auto-save is a background behavior
- Toast (brief, non-intrusive) - "Saving..." indicator (optional, debounced)
**📦 State & Data Fetching**
- useEffect: watch sqlLabStore.tabs[activeTabId].sql → debounce 1000ms → PUT /sqllab/tabs/:id
- useEffect: watch selected schema → immediate PUT (schema changes are intentional, not auto-saved)
- sqlLabStore.setTabDirty(id, bool) - track save state per tab
- useMutation({ mutationFn: api.updateTab }) - called by debounced effect
**✨ UX Behaviors**
- Auto-save: silent, no user action needed. Tab label shows unsaved dot (•) if dirty.
- latest_query_id: after QE query completes → PUT { latest_query_id: queryId } → tab linked to last query.
- Network error during auto-save: Toast (warning) "Failed to save tab. Check connection."
**🌐 API Calls (TanStack Query)**
1. debounced: fetch("/api/v1/sqllab/tabs/"+id,{method:"PUT",body:JSON.stringify(changes)}) |
| --- | --- | --- |


**SQL-003** - **Close and Delete Tabs**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**           | **API / Route**                                                                                 |
| --------------- | ------------ | --------- | ----------------------- | ----------------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | tab_state, table_schema | PUT /api/v1/sqllab/tabs/:id/close · DELETE /api/v1/sqllab/tabs/:id · DELETE /api/v1/sqllab/tabs |

**⚑ Depends on:** SQL-001 (tab must exist)

| **⚙️ Backend - Description**
- Soft close: active=false, retained 30d. Hard delete: permanent, cascades table_schema. Close all. include_closed=true restores recently closed tabs.
**🔄 Request Flow**
1. Soft: GORM.Update(active=false). Hard: TX(delete table_schema, delete tab_state). Close all: GORM.Where(user,active=true).Update(active,false)
**⚙️ Go Implementation**
1. TX: GORM.Where("tab_state_id=?",id).Delete(&table_schema{}); GORM.Delete(&tab_state{},id) | **✅ Acceptance Criteria**
- Soft close → tab gone from GET list.
- Hard delete → 204 + table_schema gone.
- Close all → {closed:N}.
**⚠️ Error Responses**
- 403 - Not owner.
- 404 - Not found. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (tab × button and context menu)
**🧩 shadcn/ui Components**
- TabsTrigger × button - close tab on click (soft close)
- DropdownMenu (right-click on tab) - "Close", "Close All", "Close Others", "Reopen Closed Tab"
- AlertDialog - only if tab has unsaved query running: "Close tab while query is running?"
- Sheet ("Recently Closed") - shows tabs closed in last 7 days for recovery
- Button (in Sheet) - "Reopen" per closed tab
**📦 State & Data Fetching**
- useMutation({ mutationFn: (id)=>api.closeTab(id), onSuccess: (id)=>sqlLabStore.removeTab(id) })
- useMutation({ mutationFn: ()=>api.closeAllTabs() })
- useQuery({ queryKey:["closed-tabs"], queryFn: ()=>api.getTabs({include_closed:true}) }) - for Sheet
**✨ UX Behaviors**
- Close ×: if tab has running query → AlertDialog confirmation. Else → immediate soft close.
- "Reopen Closed Tab" (Ctrl+Shift+T shortcut): opens Sheet with recently closed tabs list.
- Reopen: POST /sqllab/tabs with recovered sql+schema → adds back as new active tab.
- Close All: DropdownMenu item → AlertDialog "Close all N tabs?" → DELETE /sqllab/tabs.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (id)=>fetch("/api/v1/sqllab/tabs/"+id+"/close",{method:"PUT"}) })
2. useMutation({ mutationFn: ()=>fetch("/api/v1/sqllab/tabs",{method:"DELETE"}) }) |
| --- | --- | --- |


**SQL-004** - **Save Query**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                      |
| --------------- | ------------ | --------- | ------------- | -------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | saved_query   | POST /api/v1/sqllab/saved-queries · GET /api/v1/sqllab/saved-queries |

**⚑ Depends on:** AUTH-004 (user context), DBC-001 (db_id visible)

| **⚙️ Backend - Description**
- Create named labeled query. Case-insensitive unique label per user. sql_tables auto-extracted via sqlparser. Published = org-visible. Paginated list: own + published.
**🔄 Request Flow**
1. Validate db_id → label uniqueness (case-insensitive) → sqlparser extract tables → GORM.Create
**⚙️ Go Implementation**
1. sqlparser table extractor → strings.Join → sql_tables
2. GORM.Where("LOWER(label)=LOWER(?) AND created_by_fk=?",label,uid).First → 409 | **✅ Acceptance Criteria**
- 201 {id,label,sql_tables}.
- Duplicate label (case-insensitive) → 409.
- Published queries visible to org.
**⚠️ Error Responses**
- 409 - Duplicate label.
- 422 - Missing fields. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (Save dialog) + /sqllab/saved-queries (library page)
**🧩 shadcn/ui Components**
- - Save Dialog (from SQL Lab "Save" Button) -
- Dialog + DialogContent - save form
- Form + Input - label (query name)
- Textarea - description (optional, markdown)
- Switch - published (share with org)
- Button ("Save Query") - submit
- - Saved Queries Sidebar (Results tab → "Saved Queries" subtab) -
- ScrollArea - list of saved queries
- Input + Search - filter by label
- Button per query - click to load into current tab editor
- - /sqllab/saved-queries Page -
- DataTable - columns: Name, Database, Schema, Modified, Published, Actions
- Badge (Published/Private) - status
- DropdownMenu (Actions) - Load in SQL Lab, Edit, Fork, Delete
**📦 State & Data Fetching**
- useMutation({ mutationFn: api.saveQuery, onSuccess: ()=>{ dialog.close(); toast.success("Query saved"); queryClient.invalidateQueries(["saved-queries"]) } })
- useQuery({ queryKey:["saved-queries",{q,published}] }) - list
- Load query: onClick → sqlLabStore.setTabSQL(activeTabId, query.sql)
**✨ UX Behaviors**
- Save Button in editor toolbar → Dialog opens with current tab label pre-filled.
- "Published" Switch: info text "Visible to all team members in your organization".
- Saved Queries subtab in Results panel: compact list with label + DB badge. Click loads SQL into editor.
- "Fork" action: creates copy with "Copy of {label}" → navigate to edit page.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (q)=>fetch("/api/v1/sqllab/saved-queries",{method:"POST",body:JSON.stringify(q)}) })
2. useQuery({ queryKey:["saved-queries",filters], queryFn: ()=>fetch("/api/v1/sqllab/saved-queries?"+qs).then(r=>r.json()) }) |
| --- | --- | --- |


**SQL-005** - **Update and Delete Saved Query**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**          | **API / Route**                                                                                                             |
| --------------- | ------------ | --------- | ---------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | saved_query, tab_state | PUT /api/v1/sqllab/saved-queries/:id · DELETE /api/v1/sqllab/saved-queries/:id · POST /api/v1/sqllab/saved-queries/:id/fork |

**⚑ Depends on:** SQL-004 (query must exist)

| **⚙️ Backend - Description**
- Owner/Admin update: label (uniqueness re-check), sql (re-extract sql_tables), description, published, extra_json. Delete: null tab FK refs → hard delete. Fork: copy with "Copy of" prefix.
**🔄 Request Flow**
1. Ownership → label uniqueness → sql table re-extract → GORM.Updates. Delete: null tab_state.saved_query_id → GORM.Delete
**⚙️ Go Implementation**
1. GORM.Model(&tab_state{}).Where("saved_query_id=?",id).Update("saved_query_id",gorm.Expr("NULL"))
2. Fork: GORM.First → new struct → GORM.Create | **✅ Acceptance Criteria**
- 200 updated.
- Fork → 201 new query.
- Delete → tab FK nulled.
- Non-owner → 403.
**⚠️ Error Responses**
- 403 - Not owner.
- 409 - Duplicate label. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab/saved-queries (edit via Sheet or dedicated page /sqllab/saved-queries/:id/edit)
**🧩 shadcn/ui Components**
- Sheet - slide-in edit panel from list page
- Form + Input (label) + Textarea (description) + Switch (published) inside Sheet
- Monaco Editor (mini) - SQL edit inside Sheet
- Button ("Save Changes") - in Sheet footer
- Button ("Fork") - creates copy, opens in new SQL Lab tab
- AlertDialog - delete confirmation
- Badge ("In use by N tabs") - if saved_query_id referenced by tabs
**📦 State & Data Fetching**
- useMutation({ mutationFn: (data)=>api.updateSavedQuery(id,data) })
- useMutation({ mutationFn: (id)=>api.deleteSavedQuery(id) })
- useMutation({ mutationFn: (id)=>api.forkSavedQuery(id), onSuccess: (q)=>{ sqlLabStore.openNewTab({sql:q.sql}); toast.success("Forked to new tab") } })
**✨ UX Behaviors**
- Sheet: opens on row click from list. Pre-fills all fields.
- "Fork" Button: creates copy + immediately opens forked SQL in new SQL Lab tab.
- Delete: AlertDialog "Delete {label}? This cannot be undone." If used by tabs: info note.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: ({id,...data})=>fetch("/api/v1/sqllab/saved-queries/"+id,{method:"PUT",body:JSON.stringify(data)}) })
2. useMutation({ mutationFn: (id)=>fetch("/api/v1/sqllab/saved-queries/"+id+"/fork",{method:"POST"}) }) |
| --- | --- | --- |


**SQL-006** - **Schema Browser - Table List & Column Expansion**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                                                                         |
| --------------- | ------------ | --------- | ------------- | ----------------------------------------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | table_schema  | GET /api/v1/sqllab/tabs/:id/schema · POST /api/v1/sqllab/tabs/:id/schema · DELETE /api/v1/sqllab/tabs/:id/schema/:table |

**⚑ Depends on:** DBC-007 (schema cached in Redis), SQL-001 (tab DB/schema context)

| **⚙️ Backend - Description**
- Left panel schema browser: GET merges DBC-007 table list with table_schema expanded state. POST expand: DBC-007.ListColumns + upsert table_schema. DELETE collapse: expanded=false. Schema switch: delete all table_schema for tab.
**🔄 Request Flow**
1. GET: DBC-007.ListTables → merge with GORM table_schema expanded state. POST: DBC-007.ListColumns → upsert table_schema{expanded:true}. DELETE: GORM.Update(expanded,false)
**⚙️ Go Implementation**
1. Merge: stateMap:=map[string]bool{}; for _,s:=range schemaState{ stateMap[s.Table]=s.Expanded }
2. GORM.Clauses(OnConflict DoUpdates[expanded,changed_on]).Create(&table_schema{Expanded:true}) | **✅ Acceptance Criteria**
- GET → tables with expanded state.
- POST expand → columns returned + table_schema upserted.
- DELETE collapse → expanded:false.
- Schema change → fresh table list.
**⚠️ Error Responses**
- 502 - DB unreachable.
- 403 - Not tab owner. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (left Schema Browser panel)
**🧩 shadcn/ui Components**
- ResizablePanel (left) - schema browser container
- Select - schema picker at top of browser
- Input + Search icon - table filter (client-side)
- Button (RefreshCw icon) - force_refresh schema
- ScrollArea - table list container
- Collapsible + CollapsibleTrigger + CollapsibleContent - per table (expand = columns)
- CollapsibleTrigger: table name + type icon (Table/Eye for view) + column count Badge
- CollapsibleContent: column list with type labels
- Tooltip - full data_type on column hover
- Button (Copy icon, size=xs) on column - copy column name to clipboard
- Skeleton × N - table list loading state
- Badge (VIEW) - for view type tables
**📦 State & Data Fetching**
- useQuery({ queryKey:["schema-tables",tabId], queryFn: ()=>api.getSchemaTables(tabId) })
- useState: expandedTables (Set) - local expand tracking (mirrors server state)
- useMutation({ mutationFn: (tableName)=>api.expandTable(tabId,tableName), onSuccess: (r)=>setColumnCache(tableName,r.columns) })
- On schema Select change: clear expandedTables + refetch table list
**✨ UX Behaviors**
- Table Collapsible: click trigger → POST expand if not yet expanded → show columns in CollapsibleContent.
- Already expanded (from previous session): columns shown immediately from table_schema state.
- Column click → insert column name at Monaco Editor cursor position.
- Column type: shown as small Badge (INT, VARCHAR, TIMESTAMP, etc.) with color coding.
- Copy icon: click → navigator.clipboard.writeText(columnName) → Tooltip "Copied!".
- Refresh Button: fires GET with force_refresh=true → Skeleton during reload.
- "No tables found" empty state with info text "Select a schema to browse tables".
**🌐 API Calls (TanStack Query)**
1. useQuery({ queryKey:["schema-tables",tabId], queryFn: ()=>fetch("/api/v1/sqllab/tabs/"+tabId+"/schema").then(r=>r.json()) })
2. useMutation({ mutationFn: (table)=>fetch("/api/v1/sqllab/tabs/"+tabId+"/schema",{method:"POST",body:JSON.stringify({table_name:table})}).then(r=>r.json()) }) |
| --- | --- | --- |


**SQL-007** - **SQL Autocomplete Hints**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**        | **API / Route**                  |
| --------------- | ------------ | --------- | -------------------- | -------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | - Redis schema cache | POST /api/v1/sqllab/autocomplete |

**⚑ Depends on:** DBC-007 (schema in Redis), SQL-001 (tab provides DB/schema)

| **⚙️ Backend - Description**
- Return autocomplete suggestions: SQL keywords (score 300), schema names (250), table names (200), column names (100), DB functions (150). Fuzzy match (levenshtein ≤2 OR prefix). Context boost: FROM/JOIN → tables+100. Cache miss → keywords only + cache_miss:true. score>leven | **✅ Acceptance Criteria**
- <50ms p99.
- FROM context → tables scored higher.
- Cache miss → keywords only + cache_miss flag.
- Top 20 returned.
**⚠️ Error Responses**
- 200 with cache_miss:true - schema not cached. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (Monaco Editor autocomplete provider)
**🧩 shadcn/ui Components**
- Monaco Editor - built-in completion provider registration (no shadcn component needed)
- Alert (info, dismissible) - "Schema not loaded yet. Autocomplete showing SQL keywords only." when cache_miss
**📦 State & Data Fetching**
- Monaco completion provider registered via monaco.languages.registerCompletionItemProvider("sql", provider)
- provider.provideCompletionItems: fires POST /api/v1/sqllab/autocomplete with current word + prefix
- Map response to Monaco CompletionItem[] with kind (Keyword/Field/Module) + detail (meta)
**✨ UX Behaviors**
- Autocomplete: Ctrl+Space or triggered automatically after 200ms debounce.
- Suggestion list: Monaco native popup. Keyword suggestions shown with "keyword" detail. Table with "table" detail.
- cache_miss=true: Monaco shows keywords only. Alert below editor "Schema loading - full autocomplete will be available shortly.".
- Alert auto-hides when next autocomplete request returns cache_miss=false.
**🌐 API Calls (TanStack Query)**
1. fetch("/api/v1/sqllab/autocomplete",{method:"POST",body:JSON.stringify({word,prefix,db_id,schema})}).then(r=>r.json()) |
| --- | --- | --- |


**SQL-008** - **Export Query Results (CSV / XLSX / JSON)**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**       | **API / Route**                                       |
| --------------- | ------------ | --------- | ------------------- | ----------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | query (results_key) | GET /api/v1/query/:id/download?format=csv\|xlsx\|json |

**⚑ Depends on:** QE-001/QE-003 (result in Redis via results_key), AUTH-004 (ownership)

| **⚙️ Backend - Description**
- Stream result from Redis. CSV: UTF-8 BOM + encoding/csv streaming. XLSX: excelize with typed cells + bold headers. JSON: streaming array. Rate limit 10/hour.
**🔄 Request Flow**
1. Ownership → results_key check → set headers → stream format encoder to c.Writer → audit log
**⚙️ Go Implementation**
1. c.Writer.Write([]byte{0xEF,0xBB,0xBF}) // BOM
2. csv.NewWriter(c.Writer) → stream chunks
3. excelize: f.Write(c.Writer) | **✅ Acceptance Criteria**
- Streamed file download.
- BOM in CSV.
- XLSX bold headers + typed cells.
- Expired result → 410.
- Rate limit → 429.
**⚠️ Error Responses**
- 403 - Not owner.
- 410 - Expired.
- 422 - Invalid format.
- 429 - Rate limit. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (Results panel download actions)
**🧩 shadcn/ui Components**
- DropdownMenu ("Download" button in Results panel) - format selector
- DropdownMenuItem ("Download as CSV") with FileText Lucide icon
- DropdownMenuItem ("Download as Excel") with FileSpreadsheet icon
- DropdownMenuItem ("Download as JSON") with Braces icon
- Toast - "Preparing download..." during fetch + "Download complete" or error
- Button (variant=outline, size=sm) - "Download" trigger in results toolbar
**📦 State & Data Fetching**
- useMutation({ mutationFn: ({queryId,format})=>downloadFile("/api/v1/query/"+queryId+"/download?format="+format) })
- downloadFile: fetch → response.blob() → URL.createObjectURL → click  → URL.revokeObjectURL
**✨ UX Behaviors**
- Download Button in results toolbar → DropdownMenu with 3 format options.
- Click format → Toast "Preparing your download..." with Loader2.
- On blob ready: browser triggers file download with filename "query_{id}_{timestamp}.{ext}".
- Error (410 expired): Toast "Result expired. Re-run query to download.".
- Large downloads (>1MB): progress indication if server streams via ReadableStream.
**🌐 API Calls (TanStack Query)**
1. async function downloadFile(url){ const r=await fetch(url,{headers:{Authorization:"Bearer "+token}}); const blob=await r.blob(); const a=document.createElement("a"); a.href=URL.createObjectURL(blob); a.download=filename; a.click(); URL.revokeObjectURL(a.href) } |
| --- | --- | --- |


## **Requirements Summary**

| **ID**  | **Name**                                       | **Priority** | **Dep**     | **FE Route**                                                                            | **Endpoint(s)**                                                                                                             | **Phase** |
| ------- | ---------------------------------------------- | ------------ | ----------- | --------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | --------- |
| SQL-001 | Create and Restore SQL Lab Editor Tabs         | P0           | ⚠ DEPENDENT | /sqllab                                                                                 | POST /api/v1/sqllab/tabs · GET /api/v1/sqllab/tabs · GET /api/v1/sqllab/tabs/:id                                            | Phase 2   |
| SQL-002 | Auto-Save Tab SQL and Editor State             | P0           | ⚠ DEPENDENT | /sqllab (transparent auto-save, no UI)                                                  | PUT /api/v1/sqllab/tabs/:id                                                                                                 | Phase 2   |
| SQL-003 | Close and Delete Tabs                          | P0           | ⚠ DEPENDENT | /sqllab (tab × button and context menu)                                                 | PUT /api/v1/sqllab/tabs/:id/close · DELETE /api/v1/sqllab/tabs/:id · DELETE /api/v1/sqllab/tabs                             | Phase 2   |
| SQL-004 | Save Query                                     | P0           | ⚠ DEPENDENT | /sqllab (Save dialog) + /sqllab/saved-queries (library page)                            | POST /api/v1/sqllab/saved-queries · GET /api/v1/sqllab/saved-queries                                                        | Phase 2   |
| SQL-005 | Update and Delete Saved Query                  | P0           | ⚠ DEPENDENT | /sqllab/saved-queries (edit via Sheet or dedicated page /sqllab/saved-queries/:id/edit) | PUT /api/v1/sqllab/saved-queries/:id · DELETE /api/v1/sqllab/saved-queries/:id · POST /api/v1/sqllab/saved-queries/:id/fork | Phase 2   |
| SQL-006 | Schema Browser - Table List & Column Expansion | P0           | ⚠ DEPENDENT | /sqllab (left Schema Browser panel)                                                     | GET /api/v1/sqllab/tabs/:id/schema · POST /api/v1/sqllab/tabs/:id/schema · DELETE /api/v1/sqllab/tabs/:id/schema/:table     | Phase 2   |
| SQL-007 | SQL Autocomplete Hints                         | P1           | ⚠ DEPENDENT | /sqllab (Monaco Editor autocomplete provider)                                           | POST /api/v1/sqllab/autocomplete                                                                                            | Phase 2   |
| SQL-008 | Export Query Results (CSV / XLSX / JSON)       | P1           | ⚠ DEPENDENT | /sqllab (Results panel download actions)                                                | GET /api/v1/query/:id/download?format=csv\|xlsx\|json                                                                       | Phase 2   |