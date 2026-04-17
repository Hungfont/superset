**🗄️ Database Connection Service**

Rank #02 · Phase 1 - Foundation · 9 Requirements · 7 Independent · 2 Dependent
## **Service Overview**

Manages the lifecycle of external database connections. Provides encrypted credential storage, connection pooling, schema introspection, and SSH tunnel support.

On the frontend, this service powers the Database Management settings page, a multi-step "Add Database" wizard, and a connection test panel used by multiple other setup flows.

## **Tech Stack**

| **Layer**         | **Technology / Package**                                            | **Purpose**                                     |
| ----------------- | ------------------------------------------------------------------- | ----------------------------------------------- |
| UI Framework      | React 18 + TypeScript                                               | Type-safe component tree                        |
| Bundler           | Vite 5                                                              | Fast HMR and build                              |
| Routing           | React Router v6                                                     | SPA navigation and nested routes                |
| Server State      | TanStack Query v5                                                   | API cache, background refetch, mutations        |
| Client State      | Zustand                                                             | Global UI state (sidebar, user prefs)           |
| Component Library | shadcn/ui (Radix UI primitives)                                     | Accessible, unstyled - ALL components from here |
| Forms             | React Hook Form + Zod                                               | Validation schema, field-level errors           |
| Data Tables       | TanStack Table v8                                                   | Sort, filter, paginate, row selection           |
| Styling           | Tailwind CSS v3                                                     | Utility-first, no custom CSS                    |
| Icons             | Lucide React                                                        | Consistent icon set                             |
| HTTP Client       | TanStack Query (fetch under hood)                                   | No raw fetch/axios in components                |
| Toasts            | shadcn Toaster + useToast                                           | Success/error/info notifications                |
| Date Picker       | shadcn Calendar + Popover                                           | Date/time inputs                                |
| Code Editor       | Monaco Editor (for SQL)                                             | SQL Lab and expression editors                  |
| Backend           | Gin + GORM + AES-256-GCM + sync.Map + pgx/go-sql-driver/gosnowflake | Connection management                           |

| **Attribute**      | **Detail**                                                              |
| ------------------ | ----------------------------------------------------------------------- |
| Service Name       | Database Connection Service                                             |
| Rank / Build Order | #02                                                                     |
| Phase              | Phase 1 - Foundation                                                    |
| Backend API Prefix | /api/v1/admin/databases                                                 |
| Frontend Routes    | /admin/settings/databases · /admin/settings/databases/new ·             |
|                    | /admin/settings/databases/:id                                           |
| Primary DB Tables  | dbs                                                                     |
| Total Requirements | 9                                                                       |
| Independent        | 7                                                                       |
| Dependent          | 2                                                                       |

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

**✓ INDEPENDENT (7) - no cross-service calls required**

**DBC-001** - **Create Database Connection with Encrypted Credentials**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**        |
| ----------------- | ------------ | --------- | ------------- | ---------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | dbs           | POST /api/v1/admin/databases |

| **⚙️ Backend - Description**
- Accept database_name, sqlalchemy_uri, capability flags. Extract + AES-256-GCM encrypt password. Run connection test (strict_test=true default). Persist dbs record. Async audit log.
**🔄 Request Flow**
1. Validate → unique check → encrypt password → test connection → GORM.Create → audit log
**⚙️ Go Implementation**
1. encryptField(plain,key) → AES-256-GCM + base64
2. GORM.Create(&dbs{...})
3. go auditLog("database_created",id) | **✅ Acceptance Criteria**
- 201 with masked URI.
- Duplicate name → 409.
- Test failure + strict=true → 422.
- Raw DB stores ciphertext, not plaintext password.
**⚠️ Error Responses**
- 409 - Duplicate name.
- 422 - Test failed or invalid URI.
- 500 - Encryption error. | **🖥️ Frontend Specification**
**📍 Route & Page**
admin/settings/databases/new (multi-step Dialog or full page wizard)
**🧩 shadcn/ui Components**
- Dialog or full-page wizard shell - 3 steps: "Select DB Type" → "Configure Connection" → "Test & Save"
- Tabs or stepper: shadcn Tabs with TabsList + TabsTrigger per step (visual step indicator)
- RadioGroup + RadioGroupItem - DB type selector (PostgreSQL, MySQL, BigQuery, Snowflake, etc.) with logos
- Form + FormField + Input - database_name, host, port, database, username, password fields (auto-populated from type template)
- Input (type=password) - password with show/hide toggle
- Switch - capability toggles (Allow DML, Expose in SQL Lab, Allow Async, Allow File Upload)
- Textarea - Advanced: extra JSON for engine_params
- Button ("Test Connection") - manual test trigger
- Alert (variant=default&#124;destructive) - connection test result inline
- Badge (green/red) - test status indicator
- Button ("Save" + "Back") - wizard navigation
- Accordion - "Advanced Settings" section (SSH Tunnel, SSL cert)
**📦 State & Data Fetching**
- useMutation({ mutationFn: api.testConnection }) - "Test Connection" button
- useMutation({ mutationFn: api.createDatabase, onSuccess: ()=>{ navigate("/admin/settings/databases"); toast.success("Database connected successfully") } })
- React Hook Form with Zod - field validation per DB type template
- useState: { step: 0&#124;1&#124;2, testResult: null&#124;{success,latency,version} }
- DB type selection → auto-populate default host/port/driver template
**✨ UX Behaviors**
- Step 1: grid of DB type cards (RadioGroup). Each card has DB logo + name. Click selects and enables "Next".
- Step 2: form auto-fills default port (5432 for PG, 3306 for MySQL, etc.) based on DB type.
- Step 3: "Test Connection" Button → POST test endpoint → shows latency, DB version, green/red Badge.
- "Save" disabled until test passes (or user explicitly checks "Save without testing").
- Connection string preview: live-updated masked URI shown below form as user types.
- Error feedback: inline Alert per field (host unreachable, auth failure) from test result.
- Password field: never pre-filled on edit (shows placeholder "Unchanged - leave blank to keep current").
**🛡️ Client-Side Validation**
- database_name: z.string().min(3).max(128)
- port: z.number().int().min(1).max(65535)
- host: z.string().min(1,"Host is required")
- At least database_name + sqlalchemy_uri (or structured fields) required
**♿ Accessibility (a11y)**
- DB type RadioGroup: aria-label="Select database type".
- Step indicator: aria-current="step" on active step.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (data)=>fetch("/api/v1/admin/databases/test",{method:"POST",body:JSON.stringify(data)}) })
2. useMutation({ mutationFn: (data)=>fetch("/api/v1/admin/databases",{method:"POST",body:JSON.stringify(data)}) }) |
| --- | --- | --- |


**DBC-002** - **Test Database Connection**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**           | **API / Route**                                               |
| ----------------- | ------------ | --------- | ----------------------- | ------------------------------------------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | dbs (read for existing) | POST /api/v1/admin/databases/test · POST /api/v1/admin/databases/:id/test |

| **⚙️ Backend - Description**
- Two modes: pre-save (raw config) and existing (by ID). sql.Open → PingContext(5s) → SELECT version(). Return {success, latency_ms, db_version, driver, error}. Rate limit 10/min.
**🔄 Request Flow**
1. Decrypt creds (mode B) → sql.Open → PingContext(5s) → version query → return TestResult
**⚙️ Go Implementation**
1. context.WithTimeout(5s)
2. db.PingContext(ctx)
3. sanitizeError(err,dsn) → remove credentials from error string | **✅ Acceptance Criteria**
- 200 {success:true, latency_ms, db_version}.
- Bad creds → {success:false, error:"..."}.
- Timeout at 5s.
- Rate limit → 429.
**⚠️ Error Responses**
- 200 with success:false - connection failure.
- 422 - Unknown driver.
- 429 - Rate limited. | **🖥️ Frontend Specification**
**📍 Route & Page**
Inline within DBC-001 wizard (Step 3) + database detail page
**🧩 shadcn/ui Components**
- Button ("Test Connection") - primary action, shows Loader2 during test
- Alert - result display: success (green, CheckCircle icon) or failure (red, XCircle icon)
- Badge - latency_ms display ("42ms")
- Card (mini) - DB version string display on success
- Collapsible - "Error details" expansion on failure to show full driver error
**📦 State & Data Fetching**
- useState: { testStatus: "idle"&#124;"testing"&#124;"success"&#124;"error", testResult: null&#124;TestResult }
- useMutation({ mutationFn: api.testConnection, onSuccess: (r)=>setTestResult(r), onError: ()=>setTestStatus("error") })
**✨ UX Behaviors**
- Button shows Loader2 spinner during test (disabled, no double-submit).
- Success: green Alert "Connection successful - PostgreSQL 15.4 (42ms)".
- Failure: red Alert "Connection failed" + Collapsible "Show error details" → driver message.
- Test result persists until form changes (clears on any input change).
- Rate limit hit: Toast "Too many test attempts. Wait 60 seconds."
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (cfg)=>fetch("/api/v1/admin/databases/test",{method:"POST",body:JSON.stringify(cfg)}).then(r=>r.json()) }) |
| --- | --- | --- |


**DBC-003** - **List & Get Database Connections**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**                                   |
| ----------------- | ------------ | --------- | ------------- | ------------------------------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | dbs           | GET /api/v1/admin/databases · GET /api/v1/admin/databases/:id |

| **⚙️ Backend - Description**
- Paginated list with role-based visibility (Admin=all, Alpha=own+expose_in_sqllab, Gamma=expose_in_sqllab only). Masks password. Detail adds dataset_count.
**🔄 Request Flow**
1. Resolve visibility scope → apply filters → GORM.Paginate → mask URIs → return
**⚙️ Go Implementation**
1. GORM.Scopes(TenantScope,visibilityScope).Where(filters).Paginate
2. maskURI: regexp replace password with "***" | **✅ Acceptance Criteria**
- 200 paginated list.
- Admin sees all.
- Gamma sees only expose_in_sqllab=true.
- Password never returned.
**⚠️ Error Responses**
- 404 - Not found or not visible. | **🖥️ Frontend Specification**
**📍 Route & Page**
/admin/settings/databases
**🧩 shadcn/ui Components**
- DataTable - columns: Name, Backend (Badge), SQL Lab (Switch), Async (Switch), Status, Actions
- Button ("+ Connect a Database") - opens DBC-001 wizard
- DropdownMenu (Actions column) - Edit, Test Connection, Delete
- AlertDialog - delete confirmation
- Badge - backend type label (PostgreSQL, MySQL, BigQuery...)
- Switch (read-only) - expose_in_sqllab, allow_run_async display
- Tooltip - hover shows full sqlalchemy_uri with masked password
- Input + Search icon - search by database_name
- Select - filter by backend type
- Skeleton - loading state (3 skeleton rows)
- Empty state - "No databases connected yet" illustration + "Connect a Database" Button
**📦 State & Data Fetching**
- useQuery({ queryKey:["databases", filters], queryFn: ()=>api.getDatabases(filters) })
- useState: { searchQ, selectedBackend, page } - filter state
- useMutation for delete: onSuccess: invalidateQueries(["databases"]) + toast.success
**✨ UX Behaviors**
- Empty state: centered icon (Database Lucide) + "No databases yet" + CTA Button.
- Table row click → navigate to /settings/databases/:id (detail/edit page).
- Delete: AlertDialog with database name + "This will disconnect all datasets using this database."
- "Test" action in row DropdownMenu → runs DBC-002, shows result in Toast.
- Backend Badge: color-coded (blue=PostgreSQL, orange=MySQL, green=BigQuery, etc.).
**🌐 API Calls (TanStack Query)**
1. useQuery({ queryKey:["databases",{q,backend}], queryFn: ()=>fetch("/api/v1/admin/databases?q="+q+"&backend="+backend).then(r=>r.json()) }) |
| --- | --- | --- |


**DBC-004** - **Update Database Connection**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**           |
| ----------------- | ------------ | --------- | ------------- | ------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | dbs           | PUT /api/v1/admin/databases/:id |

| **⚙️ Backend - Description**
- Owner/Admin partial update. Smart password merge (*** = keep existing). Re-test credentials on change. Flush pool + Redis schema cache.
**🔄 Request Flow**
1. Ownership check → smart password merge → re-encrypt if changed → test → GORM.Save → pool.Close → redis SCAN+DEL
**⚙️ Go Implementation**
1. if !strings.Contains(newURI,"***"): re-encrypt
2. pool.Close(dbID); redis SCAN+DEL "schema:"+dbID+":*" | **✅ Acceptance Criteria**
- 200 updated record.
- *** password → existing unchanged.
- Test failure → 422, no update.
**⚠️ Error Responses**
- 403 - Not owner.
- 422 - Test failed. | 
**🖥️ Frontend Specification**
**📍 Route & Page**
/admin/settings/databases/:id
**🧩 shadcn/ui Components**
- Same Form as DBC-001 wizard but pre-populated (edit mode)
- Input (password) - placeholder "•••••••• Leave blank to keep current password" (never pre-filled)
- Button ("Save Changes") - disabled until isDirty
- Button ("Test Connection") - same inline test as DBC-002
- Alert - "Changes saved" success inline + Toast
- Breadcrumb - "Settings / Databases / {database_name}"
- Tabs [Connection, Advanced, Datasets] - organize edit sections
- "Datasets" tab: DataTable of datasets using this connection (read-only, links to /admin/datasets/:id)
**📦 State & Data Fetching**
- useQuery({ queryKey:["database",id] }) - pre-populate form
- useMutation({ mutationFn: api.updateDatabase, onSuccess: ()=>toast.success("Database updated") })
- React Hook Form + Zod - same schema as create, password not required on edit
- isDirty from React Hook Form formState.isDirty
**✨ UX Behaviors**
- Keep the same step/form visual language as DBC-001; update mode only changes descriptive copy (title, helper text, success message).
- Edit mode: form pre-filled from GET /api/v1/admin/databases/:id response (masked URI).
- "Save Changes" Button disabled until isDirty=true.
- Password: empty = keep existing, any value = update password.
- Datasets tab shows warning if updating connection used by N active datasets.
**🌐 API Calls (TanStack Query)**
1. useQuery(["database",id])
2. useMutation({ mutationFn: (data)=>fetch("/api/v1/admin/databases/"+id,{method:"PUT",body:JSON.stringify(data)}) }) |
| --- | --- | --- |


**DBC-005** - **Delete Database Connection**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**              |
| ----------------- | ------------ | --------- | ------------- | ---------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | dbs           | DELETE /api/v1/admin/databases/:id |

| **⚙️ Backend - Description**
- Guard: block if datasets exist or queries running. On approve: close pool, clear Redis cache, hard delete.
**🔄 Request Flow**
1. Ownership → count datasets → count running queries → pool.Close → redis cleanup → GORM.Delete → audit
**⚙️ Go Implementation**
1. GORM.Where("database_id=?",id).Count → 409
2. pool.Close(id); redis SCAN+DEL pattern | **✅ Acceptance Criteria**
- 204 on success.
- Has datasets → 409 with list.
- Has running queries → 409.
**⚠️ Error Responses**
- 403 - Not owner.
- 409 - In use. | **🖥️ Frontend Specification**
**📍 Route & Page**
AlertDialog triggered from DBC-003 table or DBC-004 edit page
**🧩 shadcn/ui Components**
- AlertDialog + AlertDialogContent + AlertDialogHeader + AlertDialogTitle + AlertDialogDescription
- AlertDialogFooter + AlertDialogCancel + AlertDialogAction (destructive variant)
- Alert (variant=destructive, shown inside dialog) - if dataset_count > 0: "This database has N datasets. Delete or reassign them first."
- Button (variant=destructive, disabled if has_datasets) - "Delete Database"
**📦 State & Data Fetching**
- useMutation({ mutationFn: api.deleteDatabase, onSuccess: ()=>{ navigate("/admin/settings/databases"); toast.success("Database deleted") } })
- Pre-fetch dataset_count before showing AlertDialog to configure disable state
**✨ UX Behaviors**
- AlertDialog: "Delete {database_name}? This cannot be undone. All connections to this database will be closed."
- If has_datasets: Action Button disabled + Alert inside dialog listing dataset names.
- If has_running_queries: same pattern with running query count.
- On success: navigate back to list + Toast "Database removed".
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (id)=>fetch("/api/v1/admin/databases/"+id,{method:"DELETE"}) }) |
| --- | --- | --- |


**DBC-006** - **Connection Pool Management**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**             |
| ----------------- | ------------ | --------- | ------------- | --------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | - in-memory   | Internal - no HTTP endpoint |

| **⚙️ Backend - Description**
- sync.Map[dbID→*sql.DB]. Lazy init with singleflight. MaxOpenConns=10, MaxIdleConns=3, ConnMaxLifetime=30min. Health monitor goroutine every 60s. Graceful shutdown on SIGTERM.
**🔄 Request Flow**
1. Get(dbID): sync.Map.Load → hit: return. Miss: singleflight.Do(initPool) → configure → ping → store
**⚙️ Go Implementation**
1. type PoolManager struct{ pools sync.Map; sf singleflight.Group }
2. db.SetMaxOpenConns(10); db.SetMaxIdleConns(3); db.SetConnMaxLifetime(30*time.Minute)
3. go healthMonitor(60s ticker) | **✅ Acceptance Criteria**
- 1000 concurrent queries → max 10 actual DB connections.
- Singleflight prevents thundering herd.
- Graceful shutdown closes all pools within 10s.
**⚠️ Error Responses**
- 503 - Pool exhausted (context deadline). | 
**🖥️ Frontend Specification**
**📍 Route & Page**
N/A - internal backend component
**🧩 shadcn/ui Components**
- No UI component
**📦 State & Data Fetching**
- No frontend state - transparent to UI
**✨ UX Behaviors**
- Pool errors surface as "Database unavailable" Toast when a query fails due to connection exhaustion.
**🌐 API Calls (TanStack Query)**
1. N/A |
| --- | --- | --- |


**DBC-007** - **Schema Introspection**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                                                          |
| ----------------- | ------------ | --------- | ------------- | -------------------------------------------------------------------------------------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | dbs (read)  | GET /api/v1/admin/databases/:id/schemas · GET /api/v1/databases/admin/:id/tables · GET /api/v1/admin/databases/:id/columns |

| **⚙️ Backend - Description**
- Driver-abstracted schema discovery. Redis cache 10min. Paginated table list. Per-driver INFORMATION_SCHEMA or native queries. force_refresh bypasses cache (rate-limited 5/min).
**🔄 Request Flow**
1. Pool.Get → cache check → if miss: inspector.ListX() → redis.Set(10min) → return
**⚙️ Go Implementation**
1. type SchemaInspector interface{ ListSchemas,ListTables,ListColumns }
2. redis.Get("schema:"+dbID+":"+schema+":tables") → if miss: inspector → redis.Set(10min)
3. isDttm map per driver | **
✅ Acceptance Criteria**
- GET /schemas → string array.
- GET /tables?schema=X → paginated.
- GET /columns?schema=X&table=Y → column metadata with is_dttm.
- Cache hit on second request.
- force_refresh=true bypasses cache.
**⚠️ Error Responses**
- 502 - DB unreachable.
- 504 - Timeout.
- 429 - force_refresh rate limit. | 
**🖥️ Frontend Specification**
**📍 Route & Page**
Used by SQL Lab schema browser (SQL-006) + Dataset create wizard (DS-001)
**🧩 shadcn/ui Components**
- No dedicated page - consumed as API by SQL Lab and Dataset wizard
- In Dataset create wizard: Select (schema dropdown) populated from GET /schemas
- In Dataset create wizard: Command + CommandList (searchable table list) from GET /tables
- ScrollArea + TreeView pattern - schemas → tables → columns hierarchy in SQL Lab sidebar
- Skeleton - loading state during introspection
- Tooltip - column type shown on hover
**📦 State & Data Fetching**
- useQuery({ queryKey:["db-schemas",dbId], queryFn: ()=>api.getSchemas(dbId), staleTime: 10*60*1000 }) - match server cache TTL
- useQuery({ queryKey:["db-tables",dbId,schema] }) - populated after schema select
- useQuery({ queryKey:["db-columns",dbId,schema,table] }) - on table expand
- All 3 queries use staleTime=600000 (10min) to match server cache
**✨ UX Behaviors**
- Schema select: populated on DB selection, Skeleton while loading.
- Table command: searchable Command component, lazy-loaded on schema selection.
- Columns: loaded on table expand (accordion/collapsible pattern in SQL Lab sidebar).
- force_refresh: "Refresh Schema" Button in SQL Lab sidebar header → fires request with ?force_refresh=true.
**🌐 API Calls (TanStack Query)**
1. useQuery({ queryKey:["schemas",dbId], queryFn: ()=>fetch("/api/v1/admin/databases/"+dbId+"/schemas").then(r=>r.json()) })
2. useQuery({ queryKey:["tables",dbId,schema], queryFn: ()=>fetch("/api/v1/admin/databases/"+dbId+"/tables?schema="+schema).then(r=>r.json()) }) |
| --- | --- | --- |


**⚠ DEPENDENT (2) - requires prior services/requirements**

**DBC-008** - **SSH Tunnel Support**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**    | **API / Route**                            |
| --------------- | ------------ | --------- | ---------------- | ------------------------------------------ |
| **⚠ DEPENDENT** | **P2**       | Phase 3   | dbs (extra JSON) | Transparent - part of DBC-001/DBC-002 flow |

**⚑ Depends on:** DBC-001 (SSH config in extra JSON), DBC-006 (tunnel lifecycle)

| **⚙️ Backend - Description**
- SSH PKCE flow: parse private key → ssh.Dial → local listener → forward to remote → sql.Open with local port. Private key encrypted. Health monitor verifies SSH every 60s.
**🔄 Request Flow**
1. initPool: detect ssh_tunnel → decrypt PEM → ssh.Dial → net.Listen(":0") → forwardTunnel goroutine → sql.Open(localPort)
**⚙️ Go Implementation**
1. ssh.ParsePrivateKeyWithPassphrase(pem,passphrase)
2. net.Listen("tcp","127.0.0.1:0") → localPort
3. go io.Copy tunnel | **✅ Acceptance Criteria**
- DB behind bastion → connects through tunnel.
- Wrong passphrase → test returns {success:false}.
- Tunnel drop → health monitor detects + pool cleared.
**⚠️ Error Responses**
- 502 - SSH unreachable.
- 401 - SSH auth failure. | **🖥️ Frontend Specification**
**📍 Route & Page**
Advanced section in DBC-001 wizard (Accordion "SSH Tunnel")
**🧩 shadcn/ui Components**
- Accordion + AccordionItem + AccordionTrigger + AccordionContent - "SSH Tunnel" section
- Switch - enable/disable SSH tunnel
- Form + FormField + Input - SSH Host, Port (default 22), Username
- Textarea - Private Key PEM paste area
- Input (type=password) - Private Key Passphrase (optional)
- Alert (info) - "SSH tunnel will be used to connect to the database through a bastion host"
**📦 State & Data Fetching**
- useState: sshEnabled (bool) - toggles SSH form visibility
- SSH fields added to React Hook Form when sshEnabled=true
**✨ UX Behaviors**
- Switch enables SSH section accordion expansion.
- Private key: Textarea with monospace font, placeholder "-----BEGIN RSA PRIVATE KEY-----".
- Test Connection (DBC-002) validates SSH tunnel too.
**🛡️ Client-Side Validation**
- If sshEnabled: ssh_host and ssh_username required.
- Port: integer 1-65535.
- private_key_pem: must start with "-----BEGIN" if provided.
**🌐 API Calls (TanStack Query)**
1. N/A - part of existing DBC-001/DBC-002 mutations, SSH fields included in payload |
| --- | --- | --- |


**DBC-009** - **File Upload to Database Table**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**           | **API / Route**                   |
| --------------- | ------------ | --------- | ----------------------- | --------------------------------- |
| **⚠ DEPENDENT** | **P2**       | Phase 3   | dbs (allow_file_upload) | POST /api/v1/databases/:id/upload |

**⚑ Depends on:** DBC-001 (allow_file_upload flag), DBC-006 (pool), AUTH-011 (RBAC)

| **⚙️ Backend - Description**
- Multipart CSV/XLSX upload. Guard allow_file_upload flag + RBAC. Validate max 100MB, MIME type, formula injection. Async Asynq job. Worker: parse → infer types → CREATE TABLE → batch INSERT.
**🔄 Request Flow**
1. Check flag → RBAC → validate file → write temp → Asynq.Enqueue → return 202 {job_id}
**⚙️ Go Implementation**
1. multipart parse → validate MIME → formula scan
2. asynq.Enqueue("csv:import",payload)
3. Worker: csv.Reader → type inference → batch INSERT | **✅ Acceptance Criteria**
- 202 + job_id.
- allow_file_upload=false → 403.
- >100MB → 413.
- Formula injection → 422.
**⚠️ Error Responses**
- 403 - Flag disabled.
- 413 - Too large.
- 422 - Invalid file or injection. | **🖥️ Frontend Specification**
**📍 Route & Page**
/settings/databases/:id (Upload tab)
**🧩 shadcn/ui Components**
- Tabs - add "Upload Data" tab to DBC-004 edit page (visible only if allow_file_upload=true)
- Card + drag-drop zone - file drop area using react-dropzone + shadcn styling
- Input (type=file, accept=".csv,.xlsx") - hidden, triggered by dropzone click
- Form + FormField + Input - table_name, schema select, if_exists RadioGroup
- RadioGroup + RadioGroupItem - if_exists: "Fail" &#124; "Replace" &#124; "Append"
- Switch - has_header_row (default on)
- Button ("Upload & Import") - submit with Progress bar below
- Progress - upload + processing progress (polls job status)
- Alert - success (rows_imported count) or error
- Table (preview) - first 5 rows of file shown before import (client-side parse)
**📦 State & Data Fetching**
- useMutation({ mutationFn: api.uploadFile }) → returns {job_id}
- useQuery({ queryKey:["job",jobId], refetchInterval: 2000, enabled: !!jobId }) - polls job status
- useState: { file, preview, jobId, importStatus }
- Client-side file parse (PapaParse for CSV, SheetJS for XLSX) for preview
**✨ UX Behaviors**
- Drag-drop zone: dashed border, CloudUpload icon, "Drop CSV or Excel file here or click to browse".
- After file selection: show file name + size Badge + 5-row preview DataTable.
- Upload Button → Progress bar 0% → polls job → Progress updates to 100% on completion.
- Success Alert: "✓ 50,000 rows imported into table 'my_data'".
- Error: red Alert with specific error (table exists + if_exists=fail, formula detected, etc.).
**🛡️ Client-Side Validation**
- File: max 100MB client-side (before upload) - show Alert before submitting.
- table_name: /^[a-zA-Z_][a-zA-Z0-9_]*$/ regex - real-time validation.
- File type: only .csv and .xlsx accepted.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (formData)=>fetch("/api/v1/databases/"+id+"/upload",{method:"POST",body:formData}) })
2. useQuery({ queryKey:["job",jobId], queryFn: ()=>fetch("/api/v1/jobs/"+jobId).then(r=>r.json()), refetchInterval:2000 }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID**  | **Name**                                              | **Priority** | **Dep**       | **FE Route**                                                              | **Endpoint(s)**                                                                                          | **Phase** |
| ------- | ----------------------------------------------------- | ------------ | ------------- | ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | --------- |
| DBC-001 | Create Database Connection with Encrypted Credentials | P0           | ✓ INDEPENDENT | /settings/databases/new (multi-step Dialog or full page wizard)           | POST /api/v1/databases                                                                                   | Phase 1   |
| DBC-002 | Test Database Connection                              | P0           | ✓ INDEPENDENT | Inline within DBC-001 wizard (Step 3) + database detail page              | POST /api/v1/databases/test · POST /api/v1/databases/:id/test                                            | Phase 1   |
| DBC-003 | List & Get Database Connections                       | P0           | ✓ INDEPENDENT | /settings/databases                                                       | GET /api/v1/databases · GET /api/v1/databases/:id                                                        | Phase 1   |
| DBC-004 | Update Database Connection                            | P0           | ✓ INDEPENDENT | /settings/databases/:id                                                   | PUT /api/v1/databases/:id                                                                                | Phase 1   |
| DBC-005 | Delete Database Connection                            | P0           | ✓ INDEPENDENT | AlertDialog triggered from DBC-003 table or DBC-004 edit page             | DELETE /api/v1/databases/:id                                                                             | Phase 1   |
| DBC-006 | Connection Pool Management                            | P0           | ✓ INDEPENDENT | N/A - internal backend component                                          | Internal - no HTTP endpoint                                                                              | Phase 1   |
| DBC-007 | Schema Introspection                                  | P0           | ✓ INDEPENDENT | Used by SQL Lab schema browser (SQL-006) + Dataset create wizard (DS-001) | GET /api/v1/databases/:id/schemas · GET /api/v1/databases/:id/tables · GET /api/v1/databases/:id/columns | Phase 1   |
| DBC-008 | SSH Tunnel Support                                    | P2           | ⚠ DEPENDENT   | Advanced section in DBC-001 wizard (Accordion "SSH Tunnel")               | Transparent - part of DBC-001/DBC-002 flow                                                               | Phase 3   |
| DBC-009 | File Upload to Database Table                         | P2           | ⚠ DEPENDENT   | /settings/databases/:id (Upload tab)                                      | POST /api/v1/databases/:id/upload                                                                        | Phase 3   |