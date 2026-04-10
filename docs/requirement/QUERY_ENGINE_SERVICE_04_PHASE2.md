**⚡ Query Engine Service**

Rank #04 · Phase 2 - Core · 8 Requirements · 0 Independent · 8 Dependent

## **Service Overview**

The Query Engine is the central execution layer. It translates dataset + chart params into SQL, enforces Row Level Security by injecting WHERE clauses, executes queries via the connection pool, caches results in Redis, and delivers results synchronously or asynchronously.

Every chart render, SQL Lab execution, alert metric evaluation, and report data fetch flows through this service. It must handle 100+ concurrent queries, gracefully time out, support cancellation, and serve cached results with sub-20ms latency.

The frontend surfaces of the Query Engine are distributed: SQL Lab has its own dedicated UI (SQL-001 to SQL-008), chart rendering happens via CHT-006 in the Explore view and Dashboard view, and this service provides the status polling / WebSocket layer used by all three.

## **Tech Stack**

| **Layer**         | **Technology / Package**                           | **Purpose**                                           |
| ----------------- | -------------------------------------------------- | ----------------------------------------------------- |
| UI Framework      | React 18 + TypeScript                              | Type-safe component tree                              |
| Bundler           | Vite 5                                             | Fast HMR and build                                    |
| Routing           | React Router v6                                    | SPA navigation + nested routes                        |
| Server State      | TanStack Query v5                                  | API cache, mutations, background refetch              |
| Client State      | Zustand                                            | Global UI state (sidebar, user, theme)                |
| Component Library | shadcn/ui (Radix UI primitives)                    | Accessible - ALL components from here, no custom      |
| Forms             | React Hook Form + Zod                              | Schema validation, field-level errors                 |
| Data Tables       | TanStack Table v8                                  | Sort, filter, paginate, row selection, virtualization |
| Styling           | Tailwind CSS v3                                    | Utility-first, no custom CSS                          |
| Icons             | Lucide React                                       | Consistent icon set                                   |
| API Client        | TanStack Query (fetch)                             | No raw fetch/axios in components                      |
| Notifications     | shadcn Toaster + useToast                          | Success/error/info toasts                             |
| Charts            | Apache ECharts / Recharts (same as Superset)       | Chart rendering in Explore view                       |
| DnD               | @dnd-kit/core + @dnd-kit/sortable                  | Dashboard grid drag-and-drop                          |
| Layout            | shadcn ResizablePanel                              | Resizable pane layouts (SQL Lab, Explore)             |
| Backend           | Gin + Asynq + go-redis + sqlparser + OpenTelemetry | Core execution pipeline                               |
| WebSocket         | gorilla/websocket + Redis pub/sub                  | Real-time result push                                 |
| Queue             | Asynq (Redis-backed)                               | Async query worker queues: critical/default/low       |
| Cache             | go-redis (MessagePack serialization)               | Query result caching, RLS resolution cache            |
| Tracing           | OpenTelemetry → Jaeger                             | Per-query execution tracing                           |
| Frontend WS       | Native WebSocket API (browser)                     | WS client for async result streaming                  |

| **Attribute**      | **Detail**                                                                             |
| ------------------ | -------------------------------------------------------------------------------------- |
| Service Name       | Query Engine Service                                                                   |
| Rank               | #04                                                                                    |
| Phase              | Phase 2 - Core                                                                         |
| Backend API Prefix | /api/v1/query                                                                          |
| Frontend Routes    | /sqllab (SQL Lab execution) · /explore (chart preview) · /dashboards/:id (chart loads) |
| Primary DB Tables  | query, row_level_security_filters, rls_filter_roles, rls_filter_tables                 |
| Total Requirements | 8                                                                                      |
| Independent        | 0                                                                                      |
| Dependent          | 8                                                                                      |

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

**⚠ DEPENDENT (8) - requires prior services/requirements**

**QE-001** - **Synchronous Query Execution**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**            |
| --------------- | ------------ | --------- | ------------- | -------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | query         | POST /api/v1/query/execute |

**⚑ Depends on:** DBC-006 (Connection Pool), AUTH-004 (user context for limits), AUTH-012 (tenant scope)

| **⚙️ Backend - Description**
- Execute SQL synchronously (for small result sets ≤ SYNC_QUERY_MAX_ROWS=10,000). Validate database_id access (created_by or expose_in_sqllab). Apply effective row limit: min(request_limit, roleQueryLimit(roles)) where role limits are Admin=unlimited, Alpha=100k, Gamma=10k.
- RLS injection (QE-002): before execution inject applicable RLS WHERE clauses. Store original sql, RLS-modified executed_sql separately in query table for audit.
- Record to query table: client_id (UUID for deduplication), database_id, user_id, status, start_time, start_running_time, end_time, rows, error_message, results_key. Cache result in Redis (TTL from dataset.cache_timeout). Return { data, columns, query, from_cache }.
**🔄 Request Flow**
1. Validate DB access → apply row limit → RLS inject → cache check (QE-003).
2. Cache HIT: return from_cache:true immediately.
3. Cache MISS: GORM.Create(query{status:running}) → pool.Get(dbID) → db.QueryContext(30s) → scan rows.
4. Store result in Redis → GORM.Update(query{status:success,rows,results_key}) → return.
**⚙️ Go Implementation**
1. effectiveLimit:=min(req.Limit,roleLimit(uc.Roles)) // Admin=10M,Alpha=100k,Gamma=10k
2. executedSQL:=QE002.InjectRLS(ctx,sql,datasourceID,uc.Roles)
3. cacheKey:=sha256(normSQL+dbID+schema+rlsHash)
4. if hit: return fromCache=true
5. ctx30s,cancel:=context.WithTimeout(30*time.Second)
6. rows,err:=db.QueryContext(ctx30s,executedSQL)
7. scan → msgpack.Marshal → rdb.Set(cacheKey,result,datasetTTL) | **✅ Acceptance Criteria**
- POST { database_id:1, sql:"SELECT * FROM orders LIMIT 100" } → 200 { data:[...], columns:[...], query:{executed_sql,from_cache:false}, from_cache:false }.
- Cache hit on repeat call → from_cache:true, latency <20ms.
- Row limit exceeded → data truncated + warning in response.
- Query timeout (30s) → 408.
- RLS active: executed_sql differs from sql (injected WHERE clause).
**⚠️ Error Responses**
- 403 - No DB access.
- 408 - Query timeout.
- 400 - Invalid SQL.
- 500 - Execution error. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (SQL Lab Run Button) and /explore (chart preview)
**🧩 shadcn/ui Components**
- - SQL Lab "Run" (fires QE-001 for sync queries) -
- Button ("Run", Play icon) - in SQL Lab toolbar, triggers QE-001
- Badge (status: "Running..." Loader2 / "Done" / "Failed") - in tab header
- DataTable - results display (TanStack Table with virtual scroll)
- Alert (destructive) - error display with error_message from response
- Badge (from_cache, duration_ms, rows_count) - query metadata row below table
- - Explore "Run" (chart preview) -
- Button ("Run Chart") in Explore toolbar
- Apache ECharts canvas - renders chart from response.data
- Skeleton (chart-shaped) - while query in flight
**📦 State & TanStack Query**
- useMutation({ mutationFn: api.executeQuery, onSuccess: (r)=>{ setQueryResult(r); setQueryStatus("success") }, onError: (e)=>setQueryStatus("error") })
- queryStatus: "idle" &#124; "running" &#124; "success" &#124; "error"
- queryResult: { data, columns, query, from_cache }
- from_cache Badge: green "Cached (3ms)" &#124; gray "Live (234ms)" based on from_cache + duration_ms
**✨ UX Behaviors**
- SQL Lab: Run Button → Loader2 in tab badge → results appear in Results DataTable.
- DataTable: sticky column headers, virtual scroll for 10k rows, sort on column header click.
- from_cache Banner: subtle green bar "Results from cache - 3ms" on cached hits.
- Error: red Alert with error_message. If SQL error: show the problematic SQL snippet.
- Row limit warning: amber Alert "Results limited to {N} rows. Export for full data."
- Explore: chart re-renders automatically with new data. Skeleton during fetch.
**♿ Accessibility**
- Run Button: aria-label="Execute SQL query". aria-busy=true during execution.
- Results table: aria-label="Query results, {N} rows".
**🌐 API Calls**
1. useMutation({ mutationFn: (req)=>fetch("/api/v1/query/execute",{method:"POST",body:JSON.stringify(req)}).then(r=>r.json()) }) |
| --- | --- | --- |


**QE-002** - **Row Level Security (RLS) Injection**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                                                   | **API / Route**                                       |
| --------------- | ------------ | --------- | --------------------------------------------------------------- | ----------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | row_level_security_filters, rls_filter_roles, rls_filter_tables | Internal function - called by QE-001, QE-004, CHT-006 |

**⚑ Depends on:** AUTH-011 (role resolution from JWT), DS-010 (RLS assigned to dataset), RLS-001 (filters exist)

| **⚙️ Backend - Description**
- Before any SQL execution, inject applicable RLS WHERE clauses. Resolution: (1) fetch RLS filters WHERE role_id IN user_roles AND table_id = datasetID. (2) Regular type: AND each clause. (3) Base type: REPLACE existing WHERE. (4) Template rendering: {{current_user_id}} and {{current_username}} replaced with actual values. Cache result in Redis at rls:{roles_hash}:{datasetID} TTL 5min. Admin bypass (return original SQL unchanged).
- SQL AST injection using sqlparser: parse SQL, locate WHERE node, inject as additional AND conditions - never string concat.
**🔄 Request Flow**
1. Admin role → return original SQL unchanged.
2. cacheKey = "rls:"+hashRoles(roles)+":"+datasetID.
3. redis.Get → if miss: DB join query → redis.SAdd(TTL 5min).
4. For Regular: stmt.AddWhere(parsed_clause).
5. For Base: stmt.ReplaceWhere(parsed_clause).
6. return sqlparser.String(stmt).
**⚙️ Go Implementation**
1. if isAdmin(roles): return sql,nil
2. rendered:=renderTemplate(clause,uc) // replace {{current_user_id}} etc
3. expr,_:=sqlparser.ParseExpr(rendered)
4. switch filter.FilterType { case "Regular": addWhereAnd(stmt,expr); case "Base": replaceWhere(stmt,expr) }
5. return sqlparser.String(stmt),nil
**🔒 Security**
- Template rendering uses fmt.Sprintf with typed values (int for user_id) - never raw string concat.
- sqlparser AST injection prevents WHERE bypass via UNION/subquery tricks. | **✅ Acceptance Criteria**
- Gamma user + RLS "org_id={{current_user_id}}" → executed_sql has "AND (org_id = 42)".
- Admin → SQL unchanged.
- Base filter replaces WHERE entirely.
- Cache hit <1ms (Redis only).
- Reuse attack: {{current_user_id}} rendered as actual int - not injectable.
**⚠️ Error Responses**
- Internal - errors propagate to calling service (QE-001 etc.). | **🖥️ Frontend Specification**
**📍 Route & Page**
Transparent to frontend - surfaced only via executed_sql in query response
**🧩 shadcn/ui Components**
- No direct UI component
- Info icon (Info Lucide) next to executed_sql in SQL Lab Query tab - hover Tooltip "RLS filters applied"
**📦 State & TanStack Query**
- queryResult.query.executed_sql shown in Query tab of SQL Lab
- if executed_sql !== sql: show Badge "RLS Active" in query metadata row
**✨ UX Behaviors**
- SQL Lab Query tab: shows executed_sql (may differ from user's sql if RLS active).
- Badge "RLS Active" (orange, ShieldAlert icon) in query metadata row if executed_sql differs from sql.
- Tooltip: "Row-level security filters were applied to this query."
**🌐 API Calls**
1. N/A - internal backend function, no direct frontend call |
| --- | --- | --- |


**QE-003** - **Query Result Caching**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**             | **API / Route**                                                                       |
| --------------- | ------------ | --------- | ------------------------- | ------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | query (results_key field) | Internal - transparent to callers. Cache flush: POST /api/v1/datasets/:id/cache/flush |

**⚑ Depends on:** DS-009 (cache_timeout from dataset), AUTH-004 (roles affect cache key), QE-001 (caching during execution)

| **⚙️ Backend - Description**
- SHA-256 cache key from: normalize_sql(strip comments, lowercase keywords) + database_id + schema + hash(sorted_rls_clauses). Store result as MessagePack in Redis with TTL from dataset.cache_timeout (0=global default=86400s, -1=no cache). Results >10MB not cached. Return from_cache:true flag with original query start/end timestamps.
- Cache invalidation: dataset sync (DS-003), RLS update (RLS-003), manual flush (DS-009), TTL expiry.
**🔄 Request Flow**
1. normSQL := normalize(sql)
2. cacheKey := sha256(normSQL+dbID+schema+rlsHash)
3. if ttl==-1: skip cache
4. redis.Get("qcache:"+cacheKey) → if found: return fromCache:true
5. MISS: execute → if size<10MB: redis.Set(result,ttl)
6. Store results_key in query record
**⚙️ Go Implementation**
1. normalizeSQL: strings.ToLower + regexp strip comments + normalize whitespace
2. cacheKey:=hex.EncodeToString(sha256.Sum256([]byte(normSQL+strconv.Itoa(dbID)+schema+rlsHash))[:])
3. val,err:=rdb.Get(ctx,"qcache:"+cacheKey).Bytes() → if err==nil: unmarshal → fromCache:true
4. if len(resultBytes)<10*1024*1024: rdb.Set("qcache:"+cacheKey,resultBytes,ttl) | **✅ Acceptance Criteria**
- Same query twice → second call from_cache:true, DB not queried.
- Different RLS → different cache key, separate cache entries.
- >10MB result → cache skipped.
- cache_timeout=-1 → never cached.
- Cache hit latency <20ms p95.
**⚠️ Error Responses**
- Cache errors non-fatal - fall through to DB execution. | **🖥️ Frontend Specification**
**📍 Route & Page**
Visible in all query result UIs (SQL Lab + Explore + Dashboard)
**🧩 shadcn/ui Components**
- Badge (green "Cached {N}ms" or gray "Live {N}ms") - in SQL Lab results toolbar
- Tooltip on Badge - "Results served from cache. Force refresh to get latest data."
- Badge ("Cache Disabled") - shown when from_cache:false and dataset.cache_timeout=-1
**📦 State & TanStack Query**
- queryResult.from_cache: bool - from API response
- queryResult.query.start_time + end_time → compute duration_ms for Badge
**✨ UX Behaviors**
- Green "Cached (3ms)" Badge when from_cache:true - makes cache transparent and trustworthy.
- Tooltip: "Cached at {timestamp}. TTL: {dataset.cache_timeout}s." with RefreshCw Button link.
- Force Refresh Button: clears cache + re-runs (calls QE-001 with force_refresh:true).
**🌐 API Calls**
1. Part of QE-001 response - from_cache:bool in query metadata |
| --- | --- | --- |


**QE-004** - **Asynchronous Query Execution**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                          |
| --------------- | ------------ | --------- | ------------- | -------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | query         | POST /api/v1/query/submit · GET /api/v1/query/:id/status |

**⚑ Depends on:** QE-001 (shared execution logic), AUTH-004, Asynq workers running

| **⚙️ Backend - Description**
- Submit query to Asynq queue, return 202 + query_id immediately. Three priority queues: critical (Admin → reports/alerts), default (Alpha → charts, SQL Lab), low (Gamma → background). Payload includes full UserContext (for RLS in worker). Status transitions: pending → running → success&#124;failed&#124;timed_out&#124;stopped. Retry ×3 with exponential backoff (5s,25s,125s). Worker publishes Redis pub/sub events for WebSocket delivery (QE-005).
**🔄 Request Flow**
1. GORM.Create(query{status:pending}) → get query_id.
2. asynq.Enqueue("query:execute",payload,Queue(resolveQueue(roles)),MaxRetry(3)).
3. Return 202 { query_id, status:"pending", queue }.
4. Worker: execute → cache → GORM.Update(status) → rdb.Publish("query:status:"+id,event).
**⚙️ Go Implementation**
1. GORM.Create(&query{ClientID:clientID,Status:"pending"})
2. asynq.NewTask("query:execute",mustMarshal(payload))
3. asynqClient.Enqueue(task,asynq.Queue(resolveQueue(roles)),asynq.MaxRetry(3))
4. Worker: executeQueryHandler(ctx,task) → execute → publish status event | **✅ Acceptance Criteria**
- POST → 202 { query_id:"q-abc", status:"pending", queue:"default" }.
- GET /status → { status:"running"&#124;"success"&#124;"failed" }.
- Admin → critical queue.
- Failed ×3 retries → dead letter, status=failed.
- 20 concurrent default workers → 20 parallel DB queries.
**⚠️ Error Responses**
- 202 - Always on submit (async).
- 500 - Queue push failure. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (async queries from SQL Lab Run button when query is expected to be slow)
**🧩 shadcn/ui Components**
- Button ("Run Async") - in SQL Lab toolbar (shown for queries estimated >5s)
- Badge ("Queued" → "Running..." → "Done" / "Failed") - live status in tab
- Progress (indeterminate) - shown while status is "running"
- Toast - "Query submitted. Results will appear when complete."
- Button ("Cancel Query", StopCircle icon) - triggers QE-006
- Badge (queue name: "Priority" / "Default" / "Background") - Admin visibility
**📦 State & TanStack Query**
- useMutation({ mutationFn: api.submitQuery, onSuccess: (r)=>{ setQueryId(r.query_id); setStatus("pending"); subscribeWS(r.query_id) } })
- useQuery({ queryKey:["query-status",queryId], refetchInterval:2000, enabled:status==="pending"&#124;&#124;status==="running" }) - polling fallback if WS fails
- WebSocket: see QE-005 for WS state
**✨ UX Behaviors**
- SQL Lab detects slow queries: if previous similar query took >5s → auto-submit async.
- Async indicator: Progress bar below editor + "Query running in background..." Badge.
- Tab stays interactive while async query runs.
- On completion: results appear in Results panel + success Toast.
- Browser notification (if permission granted): "Query complete" system notification.
**🌐 API Calls**
1. useMutation({ mutationFn: (req)=>fetch("/api/v1/query/submit",{method:"POST",body:JSON.stringify({...req,async:true})}).then(r=>r.json()) })
2. useQuery({ queryKey:["query-status",id], queryFn: ()=>fetch("/api/v1/query/"+id+"/status").then(r=>r.json()), refetchInterval:2000 }) |
| --- | --- | --- |


**QE-005** - **WebSocket Result Streaming**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**        |
| --------------- | ------------ | --------- | ------------- | ---------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | query         | WS /ws/query/:query_id |

**⚑ Depends on:** QE-004 (async query must be submitted), AUTH-004 (token in WS handshake)

| **⚙️ Backend - Description**
- WebSocket endpoint: client subscribes to query status events. Worker publishes via Redis pub/sub channel "query:status:{query_id}". Server forwards events to all subscribed WS connections. Heartbeat ping every 30s. On "done" event: send result inline if ≤1MB, else send { type:"result_ready", download_url }. Multiple browser tabs can subscribe to same query_id.
**🔄 Request Flow**
1. WS upgrade → validate JWT → verify query ownership.
2. rdb.Subscribe("query:status:"+queryID).
3. goroutine: forward Redis messages to WS connection.
4. Heartbeat ticker 30s.
5. On disconnect: rdb.Unsubscribe + conn.Close.
**⚙️ Go Implementation**
1. upgrader:=websocket.Upgrader{CheckOrigin:originWhitelist}
2. conn,_:=upgrader.Upgrade(c.Writer,c.Request,nil)
3. redisSub:=rdb.Subscribe(ctx,"query:status:"+queryID)
4. go func(){ for msg:=range redisSub.Channel(){ conn.WriteMessage(TextMessage,msg.Payload) } }()
5. ticker:=time.NewTicker(30s); conn.WriteMessage(PingMessage,...)
6. defer redisSub.Close(); conn.Close() | **✅ Acceptance Criteria**
- WS connect → receives status events as query progresses.
- Invalid token → 401 (before upgrade).
- >1MB result → { type:"result_ready", download_url }.
- Heartbeat every 30s.
- Disconnect → goroutine cleaned up (no leak).
**⚠️ Error Responses**
- 101 - Upgrade OK.
- 401 - Invalid token.
- 403 - Not query owner.
- 1001 - WS close on heartbeat timeout. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (transparent - WebSocket managed in background)
**🧩 shadcn/ui Components**
- No visible component - WS is a background connection
- Badge ("Connected" / "Reconnecting...") - subtle WS status indicator in SQL Lab footer
- Toast - "Connection lost. Reconnecting..." on WS disconnect
**📦 State & TanStack Query**
- Zustand wsStore: { connections: Map, subscribe, unsubscribe }
- wsStore.subscribe(queryId): creates new WebSocket → ws.onmessage → update queryResult in sqlLabStore
- ws.onclose: attempt reconnect ×3 with exponential backoff → fall back to polling (QE-004)
- ws.onmessage: parse event type → "progress": update Badge → "done": setQueryResult → "error": setQueryError
**✨ UX Behaviors**
- WS connect on async query submit: transparent, no UI action needed.
- Progress event: Badge "Running (42%)..." with percentage if server sends progress.
- "Done" event with inline data ≤1MB: instantly show results in DataTable.
- "result_ready" event >1MB: show Button "Download Results" → triggers SQL-008 flow.
- WS disconnect: Toast "Connection dropped. Reconnecting..." + auto-reconnect silently.
- Reconnect fails: fall back to polling (useQuery refetchInterval:2000) - seamless degradation.
**🌐 API Calls**
1. new WebSocket("wss://"+location.host+"/ws/query/"+queryId+"?token="+accessToken)
2. ws.onmessage = (e)=>{ const event=JSON.parse(e.data); dispatch(event) } |
| --- | --- | --- |


**QE-006** - **Query Cancellation**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**          |
| --------------- | ------------ | --------- | ------------- | ------------------------ |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | query         | DELETE /api/v1/query/:id |

**⚑ Depends on:** QE-004 (async query running), DBC-006 (pool for DB-level cancel)

| **⚙️ Backend - Description**
- Cancel in-flight async query. Two layers: (1) Application: redis.Set("query:cancel:{id}","1",5min). Worker polls every 500ms → context.Cancel(). (2) DB-level: PostgreSQL pg_cancel_backend, MySQL KILL QUERY, BigQuery job.Cancel(). Update query.status="stopped". Only owner or Admin. Idempotent: already-done query → 200 with current status.
**🔄 Request Flow**
1. Ownership check → GORM.First(query) verify status is pending/running.
2. redis.Set("query:cancel:"+id,"1",5min).
3. DB-level cancel (PostgreSQL: pg_cancel_backend).
4. GORM.Update(query,{Status:"stopped",EndTime:now()}).
**⚙️ Go Implementation**
1. rdb.Set("query:cancel:"+id,"1",5*time.Minute)
2. Worker poll: ticker 500ms → redis.Exists("query:cancel:"+id) → cancelFunc()
3. PostgreSQL: db.ExecContext(ctx,"SELECT pg_cancel_backend($1)",backendPID)
4. GORM.Model(&query{ID:id}).Updates({Status:"stopped",EndTime:now()}) | **✅ Acceptance Criteria**
- DELETE → 202 { status:"stopping" }.
- Query transitions to "stopped" within 2s.
- pg_cancel_backend called for PostgreSQL.
- Non-owner → 403.
- Already completed → 200 { status:"success", message:"Query already completed" }.
**⚠️ Error Responses**
- 403 - Not owner.
- 404 - Query not found. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (Cancel button in toolbar)
**🧩 shadcn/ui Components**
- Button ("Cancel", StopCircle Lucide icon, variant=destructive) - shown only when query is running
- AlertDialog (for long-running queries >10s) - "Cancel this query? It may take a moment to stop."
- Badge → transitions from "Running" to "Cancelled" after cancel
- Toast - "Query cancelled"
**📦 State & TanStack Query**
- useMutation({ mutationFn: (id)=>api.cancelQuery(id), onSuccess: ()=>{ setQueryStatus("stopped"); toast.info("Query cancelled") } })
- Show Cancel Button only when queryStatus === "running" &#124;&#124; queryStatus === "pending"
**✨ UX Behaviors**
- Cancel Button replaces Run Button while query is in-flight.
- Immediate: Button becomes disabled + Loader2 while cancel request in flight.
- After cancel: Badge "Cancelled" + empty result DataTable.
- If DB-level cancel successful: Toast "Query cancelled" within 2s.
**🌐 API Calls**
1. useMutation({ mutationFn: (id)=>fetch("/api/v1/query/"+id,{method:"DELETE"}).then(r=>r.json()) }) |
| --- | --- | --- |


**QE-007** - **Query History & Result Retrieval**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                          |
| --------------- | ------------ | --------- | ------------- | -------------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | query         | GET /api/v1/query/history · GET /api/v1/query/:id/result |

**⚑ Depends on:** QE-001/QE-004 (queries must be recorded first)

| **⚙️ Backend - Description**
- Paginated query history: own queries (non-Admin) or all (Admin). Filters: status, database_id, sql_contains (GIN index on PG), start_time range. Response: id, client_id, database_id, database_name, status, sql (first 500 chars), rows, start_time, end_time, duration_ms, error_message, results_key.
- Result retrieval: GET /:id/result → fetch from Redis using results_key. If expired → 410. Admin bulk delete: DELETE /history?older_than=30d.
**🔄 Request Flow**
1. GET history: GORM.Where(user_id OR isAdmin).Where(filters).Order(start_time DESC).Paginate.
2. GET result: GORM.First(query) → redis.Get("qresult:"+results_key) → 410 if nil.
**⚙️ Go Implementation**
1. GORM.Where("user_id=? OR ?",uid,isAdmin).Where(statusFilter).Order("start_time DESC").Offset(off).Limit(sz)
2. GIN index on query.sql for contains search (PostgreSQL)
3. redis.Get("qresult:"+query.ResultsKey) → if nil: 410 | **✅ Acceptance Criteria**
- GET /history → paginated list, newest first.
- GET /history?status=failed → only failed.
- GET /history?sql_contains=orders → GIN index search.
- GET /:id/result → data if Redis key valid.
- /:id/result (expired) → 410.
- DELETE /history?older_than=30d (Admin) → 200 { deleted:N }.
**⚠️ Error Responses**
- 403 - Non-admin accessing others.
- 410 - Result expired. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (Results panel → "History" tab)
**🧩 shadcn/ui Components**
- Tabs [Results &#124; History &#124; Saved Queries] - in SQL Lab results panel
- DataTable (History tab) - cols: Status (Badge), SQL (truncated), DB, Duration, Rows, Actions
- Badge (status color-coded: green=success, red=failed, amber=running, gray=stopped)
- Tooltip on SQL cell - full SQL on hover (Popover)
- Button ("Run Again") per row - loads SQL into editor + runs
- Button ("Load SQL") per row - loads SQL into editor without running
- Button ("Download") per row - triggers SQL-008 if results_key exists
- Input + Search - filter history by SQL content
- Select (Status filter)
- Button ("Clear History") - Admin only, opens AlertDialog
**📦 State & TanStack Query**
- useQuery({ queryKey:["query-history",{status,q,page}], queryFn: ()=>api.getQueryHistory(filters), refetchInterval:5000 }) - auto-refreshes to show running queries
- useMutation({ mutationFn: (sql)=>{ sqlLabStore.setTabSQL(activeTabId,sql); executeQuery(sql) } }) - "Run Again"
**✨ UX Behaviors**
- History tab auto-refreshes every 5s to show running query progress.
- Status Badge: pulsing animation for "Running" status.
- "Run Again": loads SQL into editor + immediately executes.
- "Load SQL": loads SQL into editor without executing.
- Download icon: only shown if results_key exists (result still cached).
- Expired results: Download icon grayed out with Tooltip "Result expired - rerun query".
**🌐 API Calls**
1. useQuery({ queryKey:["query-history",filters], queryFn: ()=>fetch("/api/v1/query/history?"+qs).then(r=>r.json()), refetchInterval:5000 })
2. useQuery({ queryKey:["query-result",id], queryFn: ()=>fetch("/api/v1/query/"+id+"/result").then(r=>r.json()), enabled:false }) - triggered manually |
| --- | --- | --- |


**QE-008** - **Query Cost Estimation**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**             |
| --------------- | ------------ | --------- | ------------- | --------------------------- |
| **⚠ DEPENDENT** | **P2**       | Phase 3   | dbs (type)    | POST /api/v1/query/estimate |

**⚑ Depends on:** DBC-006 (pool), DBC-001 (DB type detection)

| **⚙️ Backend - Description**
- Estimate query cost before execution. PostgreSQL: EXPLAIN (FORMAT JSON) → parse planner cost + estimated_rows. BigQuery: dry-run API → bytes_processed + estimated_cost_usd. Snowflake: EXPLAIN → partitions + bytes. MySQL/ClickHouse: EXPLAIN. Unsupported: { supported:false }. Rate limit: 30/min.
**🔄 Request Flow**
1. Detect driver from dbs.sqlalchemy_uri scheme → route to driver-specific estimator.
2. PostgreSQL: db.QueryRow("EXPLAIN (FORMAT JSON) "+sql) → parse JSON.
3. BigQuery: bqClient.Jobs.Insert(dryRun:true).
4. Return EstimateResult.
**⚙️ Go Implementation**
1. switch driver { case "postgresql": explainQuery(); case "bigquery": dryRunBQ(); default: return Estimate{Supported:false} }
2. rdb.Incr("rate:estimate:"+uid) Expire(60s) → 429 if >30 | **✅ Acceptance Criteria**
- POST (PG DB) → { supported:true, total_cost:1250, estimated_rows:50000, driver:"postgresql" }.
- BigQuery → { supported:true, bytes_processed:1073741824, estimated_cost_usd:0.005 }.
- Unsupported → { supported:false }.
- Rate limit → 429.
**⚠️ Error Responses**
- 200 with supported:false.
- 429 - Rate limited.
- 422 - SQL error in EXPLAIN. | **🖥️ Frontend Specification**
**📍 Route & Page**
/sqllab (optional "Estimate Cost" button, shown for BigQuery/Snowflake connections)
**🧩 shadcn/ui Components**
- Button ("Estimate Cost", Zap icon, variant=outline) - in SQL Lab toolbar, shown only for supported DBs
- Popover - shows estimate result on button click
- Popover content: Card with metrics (rows, cost, bytes)
- Badge ("PostgreSQL: ~50k rows" or "BigQuery: ~$0.005") - summary in toolbar
- Skeleton - loading state inside Popover
- Alert (info, inside Popover) - "Estimate only. Actual execution may differ."
**📦 State & TanStack Query**
- useMutation({ mutationFn: api.estimateQuery, onSuccess: (r)=>setEstimate(r) })
- Show button only if: db.backend in ["postgresql","bigquery","snowflake","mysql"]
**✨ UX Behaviors**
- Button → Loader2 → Popover opens with estimate.
- PostgreSQL: "Estimated: ~50,000 rows, planner cost: 1,250".
- BigQuery: "Estimated: 1 GB processed (~$0.005 at $5/TB)".
- Estimate refreshes on SQL change (debounced 2s).
- Unsupported DB: Button hidden entirely.
**🌐 API Calls**
1. useMutation({ mutationFn: ({sql,db_id})=>fetch("/api/v1/query/estimate",{method:"POST",body:JSON.stringify({sql,database_id:db_id})}).then(r=>r.json()) }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID** | **Name**                           | **Priority** | **Dep**     | **FE Route**                                                                        | **Endpoint(s)**                                                                       | **Phase** |
| ------ | ---------------------------------- | ------------ | ----------- | ----------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- | --------- |
| QE-001 | Synchronous Query Execution        | P0           | ⚠ DEPENDENT | /sqllab (SQL Lab Run Button) and /explore (chart preview)                           | POST /api/v1/query/execute                                                            | Phase 2   |
| QE-002 | Row Level Security (RLS) Injection | P0           | ⚠ DEPENDENT | Transparent to frontend - surfaced only via executed_sql in query response          | Internal function - called by QE-001, QE-004, CHT-006                                 | Phase 2   |
| QE-003 | Query Result Caching               | P0           | ⚠ DEPENDENT | Visible in all query result UIs (SQL Lab + Explore + Dashboard)                     | Internal - transparent to callers. Cache flush: POST /api/v1/datasets/:id/cache/flush | Phase 2   |
| QE-004 | Asynchronous Query Execution       | P0           | ⚠ DEPENDENT | /sqllab (async queries from SQL Lab Run button when query is expected to be slow)   | POST /api/v1/query/submit · GET /api/v1/query/:id/status                              | Phase 2   |
| QE-005 | WebSocket Result Streaming         | P1           | ⚠ DEPENDENT | /sqllab (transparent - WebSocket managed in background)                             | WS /ws/query/:query_id                                                                | Phase 2   |
| QE-006 | Query Cancellation                 | P1           | ⚠ DEPENDENT | /sqllab (Cancel button in toolbar)                                                  | DELETE /api/v1/query/:id                                                              | Phase 2   |
| QE-007 | Query History & Result Retrieval   | P1           | ⚠ DEPENDENT | /sqllab (Results panel → "History" tab)                                             | GET /api/v1/query/history · GET /api/v1/query/:id/result                              | Phase 2   |
| QE-008 | Query Cost Estimation              | P2           | ⚠ DEPENDENT | /sqllab (optional "Estimate Cost" button, shown for BigQuery/Snowflake connections) | POST /api/v1/query/estimate                                                           | Phase 3   |