**⚡ QUERY ENGINE SERVICE**

Implementation Planning Document - Chi tiết đầy đủ

Rank #04 · Phase 2 Core · 8 Requirements · ~6 Tuần · 3 Sprints

| **Thuộc tính**        | **Chi tiết**                                                                   |
| --------------------- | ------------------------------------------------------------------------------ |
| **Tên service**       | Query Engine Service - Lớp thực thi truy vấn trung tâm                         |
| **API prefix**        | /api/v1/query                                                                  |
| **Frontend routes**   | /sqllab · /explore · /dashboards/:id                                           |
| **DB tables**         | query, row_level_security_filters, rls_filter_roles, rls_filter_tables         |
| **Tổng requirements** | 8 (tất cả đều dependent - phải implement theo đúng thứ tự)                     |
| **Backend stack**     | Go · Gin · Asynq · go-redis · sqlparser · gorilla/websocket · OpenTelemetry    |
| **Frontend stack**    | React 18 · TypeScript · TanStack Query v5 · Zustand · shadcn/ui · Tailwind CSS |

# **1\. Tổng quan Service**

### **Query Engine là gì?**

Query Engine Service là bộ não xử lý truy vấn của toàn hệ thống. Mọi lần người dùng nhấn "Run" trong SQL Lab, mọi lần một biểu đồ được render, mọi lần một alert được kiểm tra - đều đi qua service này.

Service làm 5 việc chính:

- Nhận SQL từ người dùng hoặc hệ thống, dịch thành câu query thực thi được
- Kiểm tra và chèn thêm điều kiện bảo mật vào câu SQL (Row Level Security) để đảm bảo mỗi người chỉ thấy dữ liệu của họ
- Lấy kết quả từ database qua Connection Pool, hoặc trả về từ cache Redis nếu đã có sẵn
- Xử lý cả hai mode: đồng bộ (trả kết quả ngay) và bất đồng bộ (gửi background, notify qua WebSocket khi xong)
- Lưu lịch sử truy vấn và cho phép user xem lại, chạy lại, tải về kết quả

### **Ai sử dụng Query Engine?**

| **Người dùng / System** | **Hành động**                                  | **Requirement liên quan**              |
| ----------------------- | ---------------------------------------------- | -------------------------------------- |
| Data Analyst / SQL User | Gõ SQL vào SQL Lab → nhấn Run → xem kết quả    | QE-001 (sync), QE-004 (async)          |
| Viewer / Dashboard User | Mở dashboard → các chart tự load dữ liệu       | QE-001 + QE-002 (RLS) + QE-003 (cache) |
| Admin                   | Xem query history của mọi user, xóa history cũ | QE-007 (history)                       |
| Alert/Report System     | Tự động chạy query để kiểm tra ngưỡng cảnh báo | QE-004 (async), QE-002 (RLS)           |

### **Non-functional Requirements (Hiệu năng bắt buộc)**

| **Yêu cầu**                | **Ngưỡng**                | **Lý do quan trọng**                           |
| -------------------------- | ------------------------- | ---------------------------------------------- |
| Concurrent queries         | **100+ đồng thời**        | Nhiều user cùng chạy query một lúc             |
| Cache hit latency          | **< 20ms p95**            | Dashboard phải load nhanh khi dữ liệu đã cache |
| Sync query timeout         | **30 giây hard limit**    | Tránh query chạy mãi, giữ tài nguyên server    |
| Async cancel response      | **< 2 giây**              | User cần dừng query ngay khi nhấn Cancel       |
| RLS cache TTL              | **5 phút / user+dataset** | Không query DB mỗi lần để kiểm tra RLS rule    |
| Max result size để cache   | **10 MB**                 | Không nhồi kết quả lớn vào Redis               |
| Rate limit cost estimation | **30 req/phút/user**      | Ngăn abuse EXPLAIN query                       |

# **2\. Thứ tự Implementation (Dependency Chain)**

Tất cả 8 requirements đều là DEPENDENT - tức là không có requirement nào có thể bắt đầu tùy tiện. Phải tuân thủ thứ tự dưới đây vì các requirement sau phụ thuộc vào kết quả của requirement trước.

### **Sơ đồ phụ thuộc (đọc theo chiều mũi tên)**

QE-002 (RLS Injection) ←── phải có trước nhất, được gọi bởi mọi thứ

↓

QE-003 (Query Caching) ←── build cache layer trước khi QE-001 cần dùng

↓

QE-001 (Sync Execution) ←── core pipeline, mọi thứ khác phụ thuộc vào đây

↓

QE-004 (Async Execution) ←── dùng lại logic QE-001, thêm Asynq queue

↓ ↓

QE-005 QE-006 ←── WS Streaming và Cancellation cần async query

↓

QE-007 (Query History) ←── cần QE-001 & QE-004 ghi dữ liệu vào DB trước

↓

QE-008 (Cost Estimation) ←── P2, Phase 3, không block ai

| **#**  | **ID**     | **Priority** | **Phụ thuộc vào**                                 | **Unblocks (sau khi xong thì mở khóa)** |
| ------ | ---------- | ------------ | ------------------------------------------------- | --------------------------------------- |
| **1**  | **QE-002** | P0 🔴        | AUTH-011, DS-010, RLS-001 (external services)     | QE-001, QE-004, CHT-006                 |
| **2**  | **QE-003** | P0 🔴        | DS-009, AUTH-004 (external), QE-001 (cùng sprint) | QE-001 (cache check step)               |
| **3**  | **QE-001** | P0 🔴        | QE-002 ✓, QE-003 ✓, DBC-006, AUTH-004, AUTH-012   | QE-004, QE-007, CHT-006                 |
| **4**  | **QE-004** | P0 🔴        | QE-001 ✓, AUTH-004, Asynq worker infra            | QE-005, QE-006                          |
| **5a** | **QE-005** | P1 🟡        | QE-004 ✓, AUTH-004                                | Real-time UX cho SQL Lab                |
| **5b** | **QE-006** | P1 🟡        | QE-004 ✓, DBC-006                                 | User control - dừng query               |
| **6**  | **QE-007** | P1 🟡        | QE-001 ✓, QE-004 ✓                                | History tab, result download            |
| **7**  | **QE-008** | P2 🟢        | DBC-006, DBC-001                                  | Không block ai - Phase 3                |

# **3\. Sprint Planning**

## **Sprint 1: Foundation - Core Sync Pipeline**

**Tuần 1-2 (10 ngày làm việc)**

**Mục tiêu Sprint**

Xây dựng nền móng của toàn bộ Query Engine: RLS injection, caching, và sync execution. Đây là foundation mà tất cả sprint sau phụ thuộc vào.

| **ID**        | **Task**                  | **Timeline** | **Who**  | **Ghi chú kỹ thuật**                                                 |
| ------------- | ------------------------- | ------------ | -------- | -------------------------------------------------------------------- |
| **QE-002**    | RLS Injection             | Day 1-3      | Backend  | Bắt buộc làm đầu tiên - zero internal deps, nhưng mọi thứ sau cần nó |
| **QE-003**    | Query Result Caching      | Day 3-5      | Backend  | Build cache key logic + Redis store; QE-001 cần dùng ngay            |
| **QE-001 BE** | Sync Execution (Backend)  | Day 4-8      | Backend  | Core pipeline: validate → RLS → cache check → execute → store        |
| **QE-001 FE** | Sync Execution (Frontend) | Day 7-10     | Frontend | SQL Lab Run button + DataTable + from_cache Badge + Explore chart    |

**✓ Sprint Deliverable: User có thể gõ SQL vào SQL Lab, nhấn Run, nhận kết quả. Kết quả được cache và RLS được áp dụng tự động.**

## **Sprint 2: Async Pipeline - Streaming & Cancellation**

**Tuần 3-4 (10 ngày làm việc)**

**Mục tiêu Sprint**

Cho phép chạy query lâu ở background, nhận kết quả real-time qua WebSocket, và dừng query khi cần.

| **ID**        | **Task**                      | **Timeline** | **Who**  | **Ghi chú kỹ thuật**                                                 |
| ------------- | ----------------------------- | ------------ | -------- | -------------------------------------------------------------------- |
| **QE-004 BE** | Async Execution (Backend)     | Day 11-14    | Backend  | Asynq queue setup: 3 priority queues, retry logic, worker pool       |
| **QE-005 BE** | WebSocket Streaming (Backend) | Day 14-16    | Backend  | gorilla/websocket + Redis pub/sub → forward status events to browser |
| **QE-006 BE** | Query Cancellation (Backend)  | Day 15-17    | Backend  | Redis cancel flag + DB-level pg_cancel_backend cho PostgreSQL        |
| **QE-004 FE** | Async Execution (Frontend)    | Day 14-17    | Frontend | Badge Queued→Running→Done, progress bar, polling fallback            |
| **QE-005 FE** | WebSocket Client (Frontend)   | Day 16-19    | Frontend | Zustand wsStore, reconnect ×3, degradation về polling                |
| **QE-006 FE** | Cancellation UI (Frontend)    | Day 17-19    | Frontend | Cancel button thay Run button, AlertDialog, Badge transition         |

**✓ Sprint Deliverable: Query lâu chạy ở background, kết quả stream về real-time, user có thể cancel bất kỳ lúc nào.**

## **Sprint 3: History, Observability & Cost**

**Tuần 5-6 (10 ngày làm việc)**

**Mục tiêu Sprint**

Surface query history để user xem lại và chạy lại, thêm cost estimation cho data engineers.

| **ID**        | **Task**                      | **Timeline** | **Who**  | **Ghi chú kỹ thuật**                                               |
| ------------- | ----------------------------- | ------------ | -------- | ------------------------------------------------------------------ |
| **QE-007 BE** | Query History (Backend)       | Day 21-23    | Backend  | Paginated history API + GIN index trên cột sql để full-text search |
| **QE-007 FE** | History UI (Frontend)         | Day 22-25    | Frontend | History tab: DataTable với Run Again / Load SQL / Download actions |
| **QE-008 BE** | Cost Estimation (Backend)     | Day 25-27    | Backend  | EXPLAIN routing theo DB driver + rate limiting 30req/min           |
| **QE-008 FE** | Cost Estimation UI (Frontend) | Day 26-28    | Frontend | Estimate Cost button, Popover metrics, debounced 2s refresh        |

**✓ Sprint Deliverable: User thấy toàn bộ lịch sử query, có thể chạy lại bất kỳ query nào, và data engineer biết query sẽ tốn bao nhiêu trước khi chạy.**

# **5\. Detailed Requirements - Mô tả Chi tiết Từng Requirement**

Phần này mô tả chi tiết từng requirement đủ để developer không cần hỏi thêm, và người không làm kỹ thuật vẫn hiểu được mục tiêu và cách hoạt động.

**QE-002 - Row Level Security (RLS) Injection**

| **Priority**        | **P0**                                                                                            | **Phase** | Phase 2 - phải làm ĐẦU TIÊN |
| ------------------- | ------------------------------------------------------------------------------------------------- | --------- | --------------------------- |
| **API / Endpoint**  | Internal function - không expose HTTP endpoint. Được gọi bởi: QE-001, QE-004, CHT-006             |
| **Frontend route**  | Transparent với frontend - user không biết có RLS, chỉ thấy kết quả đã lọc                        |
| **DB Tables**       | row_level_security_filters, rls_filter_roles, rls_filter_tables                                   |
| **Effort ước tính** | **3 ngày (backend only)**                                                                         |
| **Phụ thuộc vào**   | AUTH-011 (role resolution từ JWT), DS-010 (RLS gắn với dataset), RLS-001 (các filter đã được tạo) |

### **📋 Mô tả nghiệp vụ (Business Context)**

Row Level Security (RLS) là cơ chế bảo mật đảm bảo mỗi người dùng chỉ thấy đúng phần dữ liệu họ được phép. Ví dụ: Sales Rep của Region A không được thấy dữ liệu của Region B, dù cả hai cùng query bảng orders. RLS hoạt động bằng cách tự động chèn điều kiện WHERE vào câu SQL của user trước khi thực thi - user không biết điều này xảy ra, họ chỉ thấy kết quả đã được lọc.

### **🔄 Luồng hoạt động chi tiết (Request Flow)**

- Nhận vào: sql (câu query gốc), datasourceID (dataset đang query), roles (danh sách role của user từ JWT)
- Kiểm tra: nếu user là Admin → bỏ qua toàn bộ RLS, trả về sql gốc không thay đổi
- Tạo cache key: "rls:" + hash(sorted_roles) + ":" + datasetID. Thử lấy từ Redis (TTL 5 phút)
- Cache MISS: Query DB để lấy danh sách RLS filter: WHERE role_id IN (user_roles) AND table_id = datasetID
- Với mỗi RLS filter: render template - thay {{current_user_id}} bằng số nguyên thực, {{current_username}} bằng string thực
- Parse RLS clause thành SQL AST node bằng sqlparser (KHÔNG dùng string concatenation)
- Regular filter: thêm clause vào WHERE hiện tại bằng AND. Base filter: thay thế toàn bộ WHERE clause
- Ghi cache vào Redis với TTL 5 phút. Trả về câu SQL đã inject RLS

### **⚙️ Backend - Yêu cầu kỹ thuật**

- Hàm signature: func InjectRLS(ctx context.Context, sql string, datasourceID int, roles \[\]string) (string, error)
- Admin check: if isAdmin(roles) { return sql, nil } - admin không bị RLS
- Cache key format: "rls:" + hex(sha256(sort(roles).join(","))) + ":" + strconv.Itoa(datasourceID)
- Redis GET cache → nếu hit: unmarshal RLS clauses, skip DB query (< 1ms)
- Cache MISS: JOIN query - row_level_security_filters JOIN rls_filter_roles ON role_id JOIN rls_filter_tables ON table_id
- Template rendering dùng typed values: fmt.Sprintf với %d cho user_id (int), %s cho username - KHÔNG dùng raw string concat
- SQL AST injection: sqlparser.Parse(sql) → locate SELECT stmt → stmt.AddWhere(parsed_expr) cho Regular type
- Base filter: stmt.ReplaceWhere(parsed_expr) - thay thế hoàn toàn WHERE clause gốc
- Lưu cache: rdb.Set(ctx, cacheKey, marshaledClauses, 5\*time.Minute)
- Return: sqlparser.String(stmt) - convert AST về string SQL hoàn chỉnh

### **🖥️ Frontend - Yêu cầu UI/UX**

**Components sử dụng (shadcn/ui)**

- Không có UI component trực tiếp cho RLS - hoạt động hoàn toàn ở backend
- Lucide Info icon: xuất hiện bên cạnh trường "Executed SQL" trong SQL Lab Query tab khi hover
- shadcn Tooltip: hiển thị "Row-level security filters were applied to this query" khi hover Info icon
- shadcn Badge màu orange với ShieldAlert icon: "RLS Active" - xuất hiện trong query metadata row

**State Management**

- queryResult.query.executed_sql: string - SQL thực sự được chạy (có thể khác sql gốc nếu RLS được áp dụng)
- Điều kiện hiển thị Badge "RLS Active": if (queryResult.query.executed_sql !== queryResult.query.sql)
- Không có TanStack Query call riêng - RLS info trả về trong response của QE-001/QE-004

**UX Behaviors - Mô tả trải nghiệm người dùng**

- User gõ SQL: "SELECT \* FROM orders" → nhấn Run → thấy kết quả với 50 rows (chỉ của mình)
- Trong SQL Lab: tab "Query" hiển thị "Executed SQL" = "SELECT \* FROM orders WHERE org_id = 42"
- Badge "RLS Active" (màu cam, icon khiên) xuất hiện trong thanh metadata dưới bảng kết quả
- Hover vào badge: Tooltip giải thích "Row-level security filters were applied to this query"
- Admin không thấy badge này - executed_sql của admin giống với sql gốc

### **📡 API Contract**

// Không có HTTP endpoint - internal Go function

Input: sql="SELECT \* FROM orders", datasourceID=5, roles=\["Gamma","Sales-APAC"\]

Output: "SELECT \* FROM orders WHERE (org_id = 42)", nil

// Redis cache key

"rls:a3f2e1:5" → \[\]RLSClause{{"org_id={{current_user_id}}", "Regular"}}

### **✅ Acceptance Criteria - Điều kiện nghiệm thu**

Requirement này chỉ được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#** | **Điều kiện kiểm tra**                                           | **Kết quả mong đợi**                                                   |
| ----- | ---------------------------------------------------------------- | ---------------------------------------------------------------------- |
| **1** | Gamma user có RLS rule "org_id={{current_user_id}}" → chạy query | **executed_sql chứa "AND (org_id = 42)"**                              |
| **2** | Admin user chạy cùng query với cùng RLS rules tồn tại            | **SQL giữ nguyên, không inject gì**                                    |
| **3** | RLS type = Base → chạy query với WHERE clause gốc                | **Base clause THAY THẾ WHERE gốc hoàn toàn**                           |
| **4** | Gọi lần 2 với cùng roles + datasourceID                          | **Lấy từ Redis cache, response < 1ms**                                 |
| **5** | SQL có UNION: "SELECT a FROM t1 UNION SELECT b FROM t2"          | **RLS inject đúng vào từng SELECT, không bypass**                      |
| **6** | Template {{current_user_id}} với user_id=42                      | **Render thành integer 42, không phải string "42" - không injectable** |

### **⚠️ Error Responses**

| **HTTP Code** | **Tình huống**                          | **Response body / Hành động**                                     |
| ------------- | --------------------------------------- | ----------------------------------------------------------------- |
| **Internal**  | sqlparser không parse được SQL của user | Trả error về QE-001/QE-004 để xử lý → 400 Invalid SQL cho user    |
| **Internal**  | Redis không available                   | Fallback: query DB trực tiếp để lấy RLS clauses, không dùng cache |

### **🔒 Security Notes**

- TUYỆT ĐỐI không dùng string concatenation cho template rendering - nguy cơ SQL injection
- Dùng typed values: fmt.Sprintf("%d", userID) với int, không phải interface{}
- sqlparser AST injection ngăn chặn WHERE bypass qua UNION tricks, subquery injection
- Admin bypass phải check role từ verified JWT - không trust client-sent role

**QE-003 - Query Result Caching**

| **Priority**        | **P0**                                                                                                         | **Phase** | Phase 2 - làm song song với QE-002, trước QE-001 |
| ------------------- | -------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------------------ |
| **API / Endpoint**  | Internal - transparent với callers. Endpoint liên quan: POST /api/v1/datasets/:id/cache/flush                  |
| **Frontend route**  | Visible trong tất cả query result UI: SQL Lab + Explore + Dashboard                                            |
| **DB Tables**       | query (cột results_key lưu Redis key)                                                                          |
| **Effort ước tính** | **2 ngày (backend) + 0.5 ngày (frontend badge)**                                                               |
| **Phụ thuộc vào**   | DS-009 (cache_timeout từ dataset config), AUTH-004 (roles ảnh hưởng cache key), QE-002 (RLS hash cần có trước) |

### **📋 Mô tả nghiệp vụ (Business Context)**

Mỗi khi một query được chạy, kết quả được lưu vào Redis. Lần sau nếu cùng user (cùng role, cùng RLS) chạy cùng câu SQL, hệ thống trả kết quả từ Redis trong vài millisecond thay vì query database. Điều này đặc biệt quan trọng cho Dashboard: nhiều viewer mở cùng một dashboard, chỉ cần chạy query một lần, những người sau lấy từ cache. Cache key được thiết kế để đảm bảo người có RLS khác nhau không nhìn thấy cache của nhau.

### **🔄 Luồng hoạt động chi tiết (Request Flow)**

- Normalize SQL: lowercase keywords, strip comments (-- và /\* \*/), normalize whitespace → ra normSQL
- Tính RLS hash: sha256 của sorted list các RLS clauses áp dụng cho user này
- Tạo cache key: hex(sha256(normSQL + "|" + dbID + "|" + schema + "|" + rlsHash))
- Kiểm tra dataset.cache_timeout: nếu = -1 → bỏ qua cache hoàn toàn, đi thẳng vào execute
- Redis GET "qcache:" + cacheKey → nếu tìm thấy: unmarshal MessagePack → return {data, columns, from_cache:true}
- Cache MISS: thực thi query bình thường (QE-001/QE-004 lo phần này)
- Sau khi có kết quả: kiểm tra size. Nếu > 10MB → skip cache. Nếu ≤ 10MB → Redis SET với TTL từ dataset
- Lưu results_key vào bảng query để sau này dùng cho QE-007 (history result download)

### **⚙️ Backend - Yêu cầu kỹ thuật**

- normalizeSQL(): strings.ToLower + regexp.MustCompile(\`--\[^\\n\]\*\`).ReplaceAll + regexp.MustCompile(\`\\s+\`).ReplaceAll (normalize whitespace)
- Cache key: hex.EncodeToString(sha256.Sum256(\[\]byte(normSQL+"|"+strconv.Itoa(dbID)+"|"+schema+"|"+rlsHash))\[:\])
- TTL logic: if dataset.CacheTimeout == 0 → dùng global default 86400s (24h). if == -1 → no cache. else → dùng giá trị đó
- Redis GET: rdb.Get(ctx, "qcache:"+cacheKey).Bytes() - nếu err == redis.Nil → cache miss
- Deserialization: msgpack.Unmarshal(val, &result) → return QueryResult{FromCache: true}
- Size check trước khi cache: if len(resultBytes) > 10\*1024\*1024 → skip, log warning
- Redis SET: rdb.Set(ctx, "qcache:"+cacheKey, resultBytes, time.Duration(ttl)\*time.Second)
- Cache flush endpoint: DELETE tất cả keys match pattern "qcache:\*" cho dataset đó (dùng SCAN + DEL)
- Cache invalidation triggers: dataset sync (DS-003), RLS update (RLS-003), manual flush (DS-009), TTL expiry tự nhiên

### **🖥️ Frontend - Yêu cầu UI/UX**

**Components sử dụng (shadcn/ui)**

- shadcn Badge màu xanh lá "Cached (3ms)": xuất hiện trong toolbar kết quả SQL Lab khi from_cache = true
- shadcn Badge màu xám "Live (234ms)": xuất hiện khi from_cache = false
- shadcn Badge "Cache Disabled": xuất hiện khi dataset.cache_timeout = -1
- shadcn Tooltip trên Badge: "Results served from cache. Force refresh to get latest data."
- shadcn Button với RefreshCw icon: "Force Refresh" - xuất hiện trong Tooltip, trigger re-run với cache bypass

**State Management**

- queryResult.from_cache: boolean - từ API response
- duration_ms = (queryResult.query.end_time - queryResult.query.start_time) - tính từ timestamps trong response
- Hiển thị "Cached (3ms)" nếu from_cache === true, "Live (234ms)" nếu false

**UX Behaviors - Mô tả trải nghiệm người dùng**

- User chạy query lần 1 → thấy Badge "Live (850ms)" - query mới, chưa cache
- User chạy lại cùng query → thấy Badge "Cached (3ms)" màu xanh - từ cache, nhanh hơn rất nhiều
- Hover vào Badge: Tooltip "Cached at 14:30:22. TTL: 3600s." và button Force Refresh
- Nhấn Force Refresh: gọi lại QE-001 với tham số force_refresh:true, bỏ qua cache, cập nhật kết quả
- Nếu dataset có cache_timeout=-1: Badge "Cache Disabled" - không cache, mỗi lần chạy đều query DB

### **📡 API Contract**

// Response từ QE-001 khi cache hit

POST /api/v1/query/execute

Response: { "data": \[...\], "columns": \[...\], "from_cache": true,

"query": { "start_time": "2024-01-15T14:30:22Z",

"end_time": "2024-01-15T14:30:22.003Z" } }

// Force flush cache cho một dataset

POST /api/v1/datasets/5/cache/flush

Response: { "status": "ok", "keys_deleted": 12 }

### **✅ Acceptance Criteria - Điều kiện nghiệm thu**

Requirement này chỉ được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#** | **Điều kiện kiểm tra**                                       | **Kết quả mong đợi**                                     |
| ----- | ------------------------------------------------------------ | -------------------------------------------------------- |
| **1** | Chạy cùng query 2 lần liên tiếp với cùng user                | **Lần 2: from_cache=true, DB không bị query**            |
| **2** | User A (Gamma, org=1) và User B (Gamma, org=2) chạy cùng SQL | **Hai cache key khác nhau (khác rlsHash)**               |
| **3** | Query trả về 15MB dữ liệu                                    | **Không cache, log warning, response vẫn trả về đầy đủ** |
| **4** | Dataset có cache_timeout=-1                                  | **Không bao giờ cache, luôn query DB**                   |
| **5** | Cache hit response time                                      | **< 20ms p95 (measured tại API layer)**                  |
| **6** | POST /api/v1/datasets/5/cache/flush                          | **Tất cả cache keys của dataset 5 bị xóa**               |

### **⚠️ Error Responses**

| **HTTP Code** | **Tình huống**                               | **Response body / Hành động**                                                  |
| ------------- | -------------------------------------------- | ------------------------------------------------------------------------------ |
| **Non-fatal** | Redis connection lỗi khi GET cache           | Fallthrough: thực thi query bình thường, log warning, không trả error cho user |
| **Non-fatal** | Redis connection lỗi khi SET cache           | Bỏ qua việc cache, trả kết quả bình thường, log warning                        |
| **500**       | Cache flush endpoint: Redis lỗi khi xóa keys | { "error": "cache flush failed", "detail": "..." }                             |

**QE-001 - Synchronous Query Execution**

| **Priority**        | **P0**                                                                                                                                        | **Phase** | Phase 2 - làm sau QE-002 và QE-003 |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ---------------------------------- |
| **API / Endpoint**  | POST /api/v1/query/execute                                                                                                                    |
| **Frontend route**  | /sqllab (SQL Lab Run Button) và /explore (Explore chart preview)                                                                              |
| **DB Tables**       | query (lưu mỗi execution: status, sql, rows, timing, error)                                                                                   |
| **Effort ước tính** | **5 ngày (3 BE + 2 FE)**                                                                                                                      |
| **Phụ thuộc vào**   | QE-002 ✓ (RLS phải xong trước), QE-003 ✓ (Cache phải xong trước), DBC-006 (Connection Pool), AUTH-004 (user context), AUTH-012 (tenant scope) |

### **📋 Mô tả nghiệp vụ (Business Context)**

Đây là requirement cốt lõi: khi user nhấn nút "Run" trong SQL Lab hoặc chart preview trong Explore, system thực thi câu SQL và trả kết quả ngay (synchronous). Dùng cho query nhỏ ≤ 10,000 rows. System kiểm tra quyền truy cập database, giới hạn số row theo role của user, inject RLS, kiểm tra cache, nếu miss thì query DB thật với timeout 30 giây, lưu kết quả vào Redis và lịch sử vào DB.

### **🔄 Luồng hoạt động chi tiết (Request Flow)**

- Nhận request: POST body gồm { database_id, sql, limit?, schema?, force_refresh? }
- Auth check: lấy UserContext từ JWT (user_id, roles, tenant_id)
- DB access check: SELECT 1 FROM dbs WHERE id=database_id AND (created_by_fk=user_id OR expose_in_sqllab=true) - nếu miss → 403
- Tính effectiveLimit = min(request.limit, roleLimit) - Admin=10,000,000, Alpha=100,000, Gamma=10,000
- Gọi QE-002.InjectRLS(ctx, sql, datasourceID, roles) → nhận executedSQL (có thể khác sql gốc)
- Gọi QE-003.CheckCache(normSQL, dbID, schema, rlsHash) → nếu HIT: return cached result (from_cache:true)
- Cache MISS: GORM.Create(&Query{Status:"running", SQL:sql, ExecutedSQL:executedSQL, StartTime:now()})
- Lấy connection từ pool: pool.Get(database_id). Chạy với timeout: ctx, cancel = context.WithTimeout(30s)
- db.QueryContext(ctx, executedSQL) → scan rows → msgpack.Marshal → kiểm tra size
- Lưu cache: QE-003.Store(cacheKey, result, dataset.CacheTimeout)
- GORM.Update(query, {Status:"success", Rows:rowCount, EndTime:now(), ResultsKey:cacheKey})
- Return: { data: rows, columns: cols, query: {executed_sql, from_cache:false}, from_cache: false }

### **⚙️ Backend - Yêu cầu kỹ thuật**

- Endpoint: POST /api/v1/query/execute, Auth: JWT required
- effectiveLimit: min(req.Limit, roleLimit(uc.Roles)) - Admin=10M, Alpha=100k, Gamma=10k. Áp dụng LIMIT vào SQL trước khi execute
- DB access validation: query bảng dbs với điều kiện created_by_fk OR expose_in_sqllab=true
- Query record fields: client_id (UUID từ client, dùng để dedup request), database_id, user_id, schema, sql (gốc), executed_sql (sau RLS), status, start_time, start_running_time, end_time, rows, error_message, results_key
- Context timeout: ctx, cancel := context.WithTimeout(ctx, 30\*time.Second); defer cancel()
- Scan rows: dùng sql.Rows.Columns() để lấy column names, scan vào \[\]interface{}, convert types
- Row limit warning: nếu rowCount == effectiveLimit → thêm warning field vào response: "results_truncated": true
- Error handling: ctx.Err() == context.DeadlineExceeded → GORM.Update(status:"timed_out") → return 408
- force_refresh flag: nếu true → skip cache check ở bước 6, vẫn lưu cache mới sau khi execute
- client_id dedup: nếu query với cùng client_id đã tồn tại và status=success → trả kết quả cũ luôn

### **🖥️ Frontend - Yêu cầu UI/UX**

**Components sử dụng (shadcn/ui)**

- \--- SQL Lab ---
- shadcn Button ("Run", Play icon, variant=default): trong toolbar SQL Lab, trigger QE-001
- shadcn Badge trong tab header: "Running..." với Loader2 animate-spin | "Done" | "Failed"
- shadcn DataTable (TanStack Table v8): hiển thị kết quả, virtual scroll cho 10k rows, sticky column headers
- shadcn Alert (destructive): hiển thị error_message khi query fail
- shadcn Badge row metadata: from_cache badge + duration_ms + rows_count bên dưới DataTable
- shadcn Alert (warning/amber): "Results limited to N rows. Export for full data." nếu results_truncated:true
- \--- Explore ---
- shadcn Button ("Run Chart"): trong Explore toolbar
- Apache ECharts canvas: render chart từ response.data
- shadcn Skeleton (chart-shaped): hiển thị trong khi query đang chạy

**State Management**

- useMutation({ mutationFn: (req) => fetch("/api/v1/query/execute", {method:"POST", body:JSON.stringify(req)}).then(r=>r.json()) })
- queryStatus: "idle" | "running" | "success" | "error" - quản lý trong sqlLabStore (Zustand)
- queryResult: { data: Row\[\], columns: Column\[\], query: QueryMeta, from_cache: boolean }
- from_cache Badge logic: from_cache===true → green "Cached (3ms)" | false → gray "Live (234ms)"
- duration_ms: tính từ query.start_time và query.end_time trong response

**UX Behaviors - Mô tả trải nghiệm người dùng**

- User nhấn Run → Button disabled, tab badge đổi sang "Running..." với spinner
- Kết quả về → DataTable xuất hiện với dữ liệu, badge đổi sang "Done"
- Click vào column header → sort DataTable theo cột đó
- Scroll xuống trong DataTable → virtual scroll tự load thêm rows, không lag với 10k rows
- Kết quả từ cache → banner xanh nhạt "Results from cache - 3ms" xuất hiện trên DataTable
- Nếu có lỗi SQL → Alert đỏ hiển thị error_message, highlight đoạn SQL bị lỗi nếu có
- Nếu bị limit row → Alert vàng "Results limited to 10,000 rows. Export for full data."
- Explore: chart tự re-render khi có data mới, Skeleton chart-shaped trong lúc chờ

**Accessibility**

- Run Button: aria-label="Execute SQL query", aria-busy=true khi đang chạy
- Results table: aria-label="Query results, N rows" (cập nhật sau khi có data)
- Loading state: aria-live="polite" announce "Query running..." → "Query complete"

### **📡 API Contract**

POST /api/v1/query/execute

Authorization: Bearer &lt;jwt&gt;

Content-Type: application/json

Request body:

{ "database_id": 1,

"sql": "SELECT \* FROM orders WHERE status = 'pending'",

"limit": 1000,

"schema": "public",

"client_id": "uuid-v4",

"force_refresh": false }

Response 200:

{ "data": \[{"id":1,"order":"ABC","total":500}, ...\],

"columns": \[{"name":"id","type":"INTEGER"}, ...\],

"from_cache": false,

"results_truncated": false,

"query": {

"id": 42,

"sql": "SELECT \* FROM orders WHERE status = 'pending'",

"executed_sql": "SELECT \* FROM orders WHERE status = 'pending' AND org_id = 42",

"start_time": "2024-01-15T14:30:00Z",

"end_time": "2024-01-15T14:30:00.850Z",

"rows": 127,

"status": "success" } }

### **✅ Acceptance Criteria - Điều kiện nghiệm thu**

Requirement này chỉ được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#** | **Điều kiện kiểm tra**                                | **Kết quả mong đợi**                            |
| ----- | ----------------------------------------------------- | ----------------------------------------------- |
| **1** | POST với sql hợp lệ, database có quyền truy cập       | **200 với data + columns + query metadata**     |
| **2** | Chạy lại cùng query (cùng SQL + cùng role)            | **from_cache:true, latency < 20ms**             |
| **3** | User Gamma query bảng 1M rows, limit không đặt        | **Tối đa 10,000 rows, results_truncated:true**  |
| **4** | Query chạy quá 30 giây                                | **408 Timeout, status trong DB = "timed_out"**  |
| **5** | User không có quyền vào database_id đó                | **403 Forbidden**                               |
| **6** | SQL syntax sai                                        | **400 Bad Request với error_message từ DB**     |
| **7** | RLS đang active: executed_sql PHẢI khác sql           | **executed_sql có WHERE clause bổ sung**        |
| **8** | Gửi 2 request với cùng client_id, request 1 đang chạy | **Request 2 trả kết quả của request 1 (dedup)** |

### **⚠️ Error Responses**

| **HTTP Code** | **Tình huống**                      | **Response body / Hành động**                                              |
| ------------- | ----------------------------------- | -------------------------------------------------------------------------- |
| **400**       | SQL không parse được (syntax error) | { "error": "invalid_sql", "message": "syntax error at position 42" }       |
| **403**       | User không có quyền vào database_id | { "error": "forbidden", "message": "Access denied to database" }           |
| **408**       | Query chạy quá 30 giây              | { "error": "query_timeout", "message": "Query exceeded 30s timeout" }      |
| **500**       | Lỗi DB connection hoặc scan rows    | { "error": "execution_error", "message": "..." } - query.status = "failed" |

**QE-004 - Asynchronous Query Execution**

| **Priority**        | **P0**                                                                                           | **Phase** | Phase 2 - sau QE-001 |
| ------------------- | ------------------------------------------------------------------------------------------------ | --------- | -------------------- |
| **API / Endpoint**  | POST /api/v1/query/submit · GET /api/v1/query/:id/status                                         |
| **Frontend route**  | /sqllab - khi query dự kiến chạy lâu (> 5 giây)                                                  |
| **DB Tables**       | query                                                                                            |
| **Effort ước tính** | **4 ngày (2.5 BE + 1.5 FE)**                                                                     |
| **Phụ thuộc vào**   | QE-001 ✓ (dùng lại execution logic), AUTH-004, Asynq + Redis worker infra phải được deploy trước |

### **📋 Mô tả nghiệp vụ (Business Context)**

Khi user chạy query nặng (export data, join nhiều bảng, aggregate lớn), việc đợi 30 giây là trải nghiệm tệ. Async execution cho phép: (1) gửi query vào hàng đợi, nhận query_id ngay lập tức, (2) UI vẫn dùng được trong khi query chạy ở background, (3) kết quả được push về real-time qua WebSocket (QE-005) khi xong. Có 3 mức ưu tiên: Admin (critical queue - chạy ngay, dùng cho báo cáo/alert), Alpha (default queue), Gamma (low queue - chạy sau cùng).

### **🔄 Luồng hoạt động chi tiết (Request Flow)**

- User nhấn "Run Async" hoặc system tự detect query lâu (từ lần chạy trước > 5s) → submit async
- GORM.Create(&Query{ClientID:uuid, Status:"pending", SQL:sql, UserID:uid}) → lấy query.ID
- Enqueue vào Asynq: asynqClient.Enqueue(task, asynq.Queue(resolveQueue(roles)), asynq.MaxRetry(3))
- resolveQueue: Admin → "critical", Alpha → "default", Gamma → "low"
- Return ngay: 202 Accepted { query_id: "q-abc123", status: "pending", queue: "default" }
- \--- Worker process (chạy ở background) ---
- Worker nhận task từ Asynq queue → unmarshal payload (gồm full UserContext để dùng cho RLS)
- Worker thực thi query (dùng lại logic của QE-001): RLS inject → cache check → execute → cache store
- GORM.Update(query, {Status:"running", StartRunningTime:now()}) khi bắt đầu
- Sau khi xong: GORM.Update(query, {Status:"success"/"failed", EndTime:now(), Rows:n, ResultsKey:key})
- Publish lên Redis pub/sub: rdb.Publish("query:status:"+queryID, jsonEvent) để QE-005 forward về browser

### **⚙️ Backend - Yêu cầu kỹ thuật**

- Submit endpoint: POST /api/v1/query/submit - validate request, tạo query record, enqueue, return 202
- Status endpoint: GET /api/v1/query/:id/status - GORM.First(query, id) → return current status
- Asynq task payload: { query_id, sql, database_id, schema, limit, user_context (full: user_id, roles, tenant_id) }
- Queue routing: if hasRole("Admin") → "critical". elif hasRole("Alpha") → "default". else → "low"
- Retry policy: asynq.MaxRetry(3) với exponential backoff: 5s → 25s → 125s
- Status transitions: pending → running → success | failed | timed_out | stopped
- Worker: executeQueryHandler(ctx context.Context, task \*asynq.Task) error
- Worker check cancel: mỗi 500ms: rdb.Exists("query:cancel:"+queryID) → nếu tìm thấy: return error (mark stopped)
- Redis pub/sub event format: { "type": "status", "query_id": "q-abc", "status": "running"/"success"/"failed", "data": {...} }
- Dead letter: sau 3 lần retry fail → query.status = "failed", error_message = last error
- Worker pool: 20 workers cho "default" queue, 10 cho "critical", 5 cho "low" (configurable)

### **🖥️ Frontend - Yêu cầu UI/UX**

**Components sử dụng (shadcn/ui)**

- shadcn Button ("Run Async", với icon khác Run thường): trong SQL Lab toolbar
- shadcn Badge tiến trình: "Queued" (gray) → "Running..." (amber + Loader2) → "Done" (green) / "Failed" (red)
- shadcn Progress bar (indeterminate): xuất hiện bên dưới SQL editor khi status = running
- shadcn Toast: "Query submitted. Results will appear when complete." ngay khi submit
- shadcn Button ("Cancel Query", StopCircle icon, variant=destructive): trigger QE-006
- shadcn Badge (queue name: "Priority" / "Standard" / "Background"): chỉ Admin thấy

**State Management**

- useMutation({ mutationFn: (req) => fetch("/api/v1/query/submit", {method:"POST",...}).then(r=>r.json()), onSuccess: (r) => { setQueryId(r.query_id); setStatus("pending"); subscribeWS(r.query_id) } })
- useQuery({ queryKey:\["query-status", queryId\], queryFn: ()=>fetch("/api/v1/query/"+queryId+"/status").then(r=>r.json()), refetchInterval:2000, enabled: status==="pending"||status==="running" }) - polling fallback khi WS fail
- sqlLabStore (Zustand): activeQueryId, queryStatus per tab - tab vẫn interactive khi query đang chạy

**UX Behaviors - Mô tả trải nghiệm người dùng**

- System detect: nếu query tương tự chạy trước đó > 5s → tự động switch sang async mode, không hỏi user
- Sau khi submit: Progress bar xuất hiện bên dưới editor, Badge "Queued" trong tab header
- Badge animation: "Running..." với amber color + Loader2 spinner khi worker bắt đầu chạy
- Tab vẫn có thể dùng: user có thể mở tab mới, viết SQL khác trong khi chờ
- Kết quả về: DataTable xuất hiện tự động, Badge đổi sang "Done", Toast success
- Nếu browser có permission notification: hiện system notification "Query complete" dù user đang ở tab khác

### **📡 API Contract**

POST /api/v1/query/submit

Body: { "database_id":1, "sql":"SELECT ...", "async":true, "client_id":"uuid" }

Response 202: { "query_id":"q-abc123", "status":"pending", "queue":"default" }

GET /api/v1/query/q-abc123/status

Response 200: { "query_id":"q-abc123", "status":"running",

"start_time":"2024-01-15T14:30:00Z", "elapsed_ms":3420 }

// Khi xong:

Response 200: { "query_id":"q-abc123", "status":"success",

"rows":50000, "results_key":"qcache:a3f2...", "elapsed_ms":18500 }

### **✅ Acceptance Criteria - Điều kiện nghiệm thu**

Requirement này chỉ được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#** | **Điều kiện kiểm tra**                      | **Kết quả mong đợi**                                  |
| ----- | ------------------------------------------- | ----------------------------------------------------- |
| **1** | POST /api/v1/query/submit với valid request | **202 với query_id + status:"pending"**               |
| **2** | Admin submit query                          | **queue = "critical"**                                |
| **3** | Gamma submit query                          | **queue = "low"**                                     |
| **4** | GET /status ngay sau submit                 | **status = "pending" hoặc "running"**                 |
| **5** | Query fail 3 lần liên tiếp                  | **status = "failed", error_message chứa nguyên nhân** |
| **6** | 20 concurrent default workers               | **20 query DB song song cùng lúc**                    |

### **⚠️ Error Responses**

| **HTTP Code** | **Tình huống**                          | **Response body / Hành động**                             |
| ------------- | --------------------------------------- | --------------------------------------------------------- |
| **202**       | Luôn trả 202 khi enqueue thành công     | Async - lỗi execution sẽ reflect qua status endpoint      |
| **500**       | Redis/Asynq không available khi enqueue | { "error": "queue_unavailable" } - không tạo query record |

**QE-005 - WebSocket Result Streaming**

| **Priority**        | **P1**                                                                                  | **Phase** | Phase 2 - sau QE-004 |
| ------------------- | --------------------------------------------------------------------------------------- | --------- | -------------------- |
| **API / Endpoint**  | WS /ws/query/:query_id (WebSocket, không phải HTTP)                                     |
| **Frontend route**  | /sqllab - kết nối WS được quản lý ẩn trong background khi async query chạy              |
| **DB Tables**       | query (đọc ownership để verify)                                                         |
| **Effort ước tính** | **3 ngày (1.5 BE + 1.5 FE)**                                                            |
| **Phụ thuộc vào**   | QE-004 ✓ (async query phải chạy mới có gì để stream), AUTH-004 (JWT trong WS handshake) |

### **📋 Mô tả nghiệp vụ (Business Context)**

Sau khi submit async query (QE-004), browser cần biết khi nào query xong để hiển thị kết quả. Có 2 cách: (1) polling: cứ 2 giây hỏi server một lần (kém hiệu quả), (2) WebSocket: server chủ động push event về browser ngay khi query xong (realtime). QE-005 implement cách 2. Browser mở WS connection đến server, subscribe vào channel của query_id, worker sau khi chạy xong publish event lên Redis pub/sub, server forward về browser qua WS.

### **🔄 Luồng hoạt động chi tiết (Request Flow)**

- Browser: new WebSocket("wss://host/ws/query/q-abc123?token=&lt;jwt&gt;")
- Server: nhận WS upgrade request → validate JWT → kiểm tra user có quyền xem query này không
- Upgrade thành công: rdb.Subscribe(ctx, "query:status:"+queryID)
- Goroutine A: vòng lặp for msg := range redisSub.Channel() { conn.WriteMessage(TextMessage, msg.Payload) }
- Goroutine B (heartbeat): time.NewTicker(30s) → conn.WriteMessage(PingMessage, nil) mỗi 30 giây
- Worker (QE-004) khi query xong: rdb.Publish("query:status:q-abc123", jsonEvent)
- Server nhận message từ Redis → forward về browser qua WS connection
- Nếu result ≤ 1MB: event = { type:"done", data:{rows,columns} } - inline data
- Nếu result > 1MB: event = { type:"result_ready", download_url:"/api/v1/query/q-abc123/result" }
- Browser disconnect: defer redisSub.Close(); conn.Close() - goroutine A kết thúc tự nhiên

### **⚙️ Backend - Yêu cầu kỹ thuật**

- WS upgrader: websocket.Upgrader{CheckOrigin: func(r) bool { return isAllowedOrigin(r.Header.Get("Origin")) }}
- JWT validate TRƯỚC khi upgrade: đọc ?token= từ query string, verify, lấy userID
- Ownership check: GORM.First(&query, queryID). if query.UserID != userID && !isAdmin → 403 (trước upgrade)
- Redis subscribe: redisSub := rdb.Subscribe(ctx, "query:status:"+queryID)
- Forward goroutine: go func() { for msg := range redisSub.Channel() { conn.WriteMessage(websocket.TextMessage, \[\]byte(msg.Payload)) } }()
- Heartbeat: ticker := time.NewTicker(30\*time.Second); defer ticker.Stop() → conn.WriteMessage(websocket.PingMessage, nil)
- Cleanup: defer func() { redisSub.Close(); conn.Close() }() - đảm bảo không goroutine leak
- Size check: nếu result > 1MB → publish { type:"result_ready", url } thay vì inline data
- Multiple tabs: nhiều WS connections có thể subscribe cùng một query_id - Redis fanout tự nhiên

### **🖥️ Frontend - Yêu cầu UI/UX**

**Components sử dụng (shadcn/ui)**

- Không có UI component trực tiếp - WS connection là background process
- shadcn Badge nhỏ trong SQL Lab footer: "WS Connected" (xanh) | "Reconnecting..." (amber)
- shadcn Toast: "Connection lost. Reconnecting..." khi WS bị ngắt

**State Management**

- Zustand wsStore: { connections: Map&lt;queryId, WebSocket&gt;, subscribe(queryId), unsubscribe(queryId) }
- wsStore.subscribe(queryId): tạo new WebSocket → set ws.onmessage → ws.onclose handlers
- ws.onmessage handler: parse JSON event → switch(event.type):
- "progress": cập nhật Badge "Running (42%)..."
- "done": setQueryResult(event.data) → hiện DataTable → toast success
- "result_ready": hiện Button "Download Results" → dẫn vào SQL-008 download flow
- "error": setQueryError(event.message) → hiện Alert đỏ
- ws.onclose: attempt reconnect ×3 với exponential backoff (1s, 2s, 4s). Sau 3 lần fail → fallback sang polling (refetchInterval:2000)

**UX Behaviors - Mô tả trải nghiệm người dùng**

- Sau khi submit async query: WS connection tự mở - user không cần làm gì
- Nếu server push progress event: Badge cập nhật "Running (42%)..." với percentage
- "Done" event ≤ 1MB: DataTable xuất hiện ngay lập tức, không cần user làm gì
- "result_ready" > 1MB: Button "Download Results" xuất hiện để user tải về
- WS bị ngắt: Toast "Connection dropped. Reconnecting..." → auto reconnect im lặng
- Sau 3 lần reconnect fail: chuyển sang polling mỗi 2s - UX không thay đổi, chỉ chậm hơn

### **📡 API Contract**

// WebSocket handshake

GET /ws/query/q-abc123?token=&lt;jwt_token&gt;

Upgrade: websocket

Response 101 Switching Protocols

// Events server → browser (JSON text frames)

{ "type": "progress", "query_id": "q-abc123", "percent": 42 }

{ "type": "done", "query_id": "q-abc123",

"data": { "rows": \[...\], "columns": \[...\] } } // nếu ≤ 1MB

{ "type": "result_ready", "query_id": "q-abc123",

"download_url": "/api/v1/query/q-abc123/result" } // nếu > 1MB

{ "type": "error", "query_id": "q-abc123", "message": "..." }

### **✅ Acceptance Criteria - Điều kiện nghiệm thu**

Requirement này chỉ được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#** | **Điều kiện kiểm tra**                      | **Kết quả mong đợi**                                |
| ----- | ------------------------------------------- | --------------------------------------------------- |
| **1** | WS connect với valid JWT và valid query_id  | **101 Switching Protocols, nhận được events**       |
| **2** | WS connect với invalid/expired JWT          | **401 trả về TRƯỚC KHI upgrade**                    |
| **3** | WS connect với query_id không thuộc về user | **403 trả về trước khi upgrade**                    |
| **4** | Query xong với kết quả ≤ 1MB                | **Event "done" với inline data gửi qua WS**         |
| **5** | Query xong với kết quả > 1MB                | **Event "result_ready" với download_url**           |
| **6** | Heartbeat không được nhận trong 35 giây     | **WS close với code 1001**                          |
| **7** | Browser disconnect                          | **Goroutine cleanup, Redis unsubscribe trong < 1s** |

### **⚠️ Error Responses**

| **HTTP Code** | **Tình huống**                                           | **Response body / Hành động**           |
| ------------- | -------------------------------------------------------- | --------------------------------------- |
| **401**       | JWT invalid hoặc expired - kiểm tra TRƯỚC upgrade        | HTTP 401 response (chưa upgrade)        |
| **403**       | User không phải owner của query và không phải Admin      | HTTP 403 response (chưa upgrade)        |
| **1001**      | WebSocket close vì heartbeat timeout (client không pong) | WS close frame với code 1001 Going Away |

**QE-006 - Query Cancellation**

| **Priority**        | **P1**                                                                                              | **Phase** | Phase 2 - sau QE-004 |
| ------------------- | --------------------------------------------------------------------------------------------------- | --------- | -------------------- |
| **API / Endpoint**  | DELETE /api/v1/query/:id                                                                            |
| **Frontend route**  | /sqllab - nút Cancel trong toolbar, chỉ hiện khi query đang pending hoặc running                    |
| **DB Tables**       | query (đọc status + ownership, update status)                                                       |
| **Effort ước tính** | **2 ngày (1.5 BE + 0.5 FE)**                                                                        |
| **Phụ thuộc vào**   | QE-004 ✓ (cần có async query đang chạy để cancel), DBC-006 (Connection Pool để gọi DB-level cancel) |

### **📋 Mô tả nghiệp vụ (Business Context)**

User submit một query nặng nhưng sau đó nhận ra query sai, hoặc đơn giản là không muốn chờ nữa. Tính năng Cancel Query cho phép dừng query ngay lập tức ở hai tầng: (1) Application layer - đặt flag "cancel" trong Redis, worker tự dừng khi phát hiện, (2) DB layer - gửi lệnh kill thẳng vào database để giải phóng tài nguyên DB ngay. Chỉ chủ sở hữu query hoặc Admin mới được cancel. Idempotent: cancel query đã xong trả về 200 bình thường.

### **🔄 Luồng hoạt động chi tiết (Request Flow)**

- User nhấn "Cancel Query" trong SQL Lab toolbar
- FE gửi: DELETE /api/v1/query/q-abc123
- BE: GORM.First(&query, queryID) - kiểm tra query tồn tại
- Ownership check: if query.UserID != requestUserID && !isAdmin(roles) → 403
- Status check: if query.Status không phải "pending" hoặc "running" → return 200 với current status (idempotent)
- Layer 1 cancel: rdb.Set(ctx, "query:cancel:"+queryID, "1", 5\*time.Minute)
- Layer 2 cancel (DB-level): lấy backendPID từ query record → gọi DB cancel
- PostgreSQL: db.ExecContext(ctx, "SELECT pg_cancel_backend(\$1)", query.BackendPID)
- MySQL: db.ExecContext(ctx, "KILL QUERY "+query.ConnectionID)
- BigQuery: bqClient.Jobs.Cancel(projectID, jobID)
- GORM.Update(&query, {Status:"stopped", EndTime:now()})
- Return 202: { status: "stopping" }
- \--- Worker side (song song) ---
- Worker tick mỗi 500ms: if rdb.Exists("query:cancel:"+queryID) > 0 → cancelFunc() → return error "cancelled"

### **⚙️ Backend - Yêu cầu kỹ thuật**

- Endpoint: DELETE /api/v1/query/:id, Auth: JWT required
- Ownership: query.UserID == authUser.ID || authUser.IsAdmin → proceed. Else → 403
- Idempotency: if status ∉ \["pending", "running"\] → return 200 { status: query.Status, message: "Query already "+status }
- Redis cancel key: "query:cancel:"+queryID với TTL 5 phút (auto cleanup nếu worker không catch)
- Worker poll cancel: time.NewTicker(500ms) → select { case <-ticker.C: check redis }
- PostgreSQL cancel: store pg_backend_pid() vào query.BackendPID khi execution bắt đầu. Cancel: SELECT pg_cancel_backend(\$1)
- MySQL cancel: store connection_id() vào query.ConnectionID. Cancel: KILL QUERY &lt;id&gt;
- BigQuery cancel: store job.ID vào query.ExternalJobID. Cancel: bqClient.Jobs.Cancel()
- Update query status: GORM.Model(&Query{ID:id}).Updates(map\[string\]any{"status":"stopped","end_time":now()})
- Return 202 immediately - không chờ worker confirm stopped (quá trình stop có thể mất thêm < 2s)

### **🖥️ Frontend - Yêu cầu UI/UX**

**Components sử dụng (shadcn/ui)**

- shadcn Button ("Cancel", StopCircle icon, variant=destructive): THAY THẾ Run Button khi query đang chạy
- shadcn AlertDialog (cho query chạy > 10 giây): "Cancel this query? It may take a moment to stop."
- shadcn Badge: transition từ "Running..." → "Cancelled" sau khi cancel thành công
- shadcn Toast: "Query cancelled" sau 200/202 response

**State Management**

- useMutation({ mutationFn: (queryId) => fetch("/api/v1/query/"+queryId, {method:"DELETE"}).then(r=>r.json()), onSuccess: () => { setQueryStatus("stopped"); toast.info("Query cancelled") } })
- Show Cancel Button: queryStatus === "running" || queryStatus === "pending"
- Hide Cancel Button, show Run Button: queryStatus === "idle" || "success" || "error" || "stopped"
- AlertDialog: chỉ xuất hiện nếu query đã chạy > 10 giây (elapsed_ms > 10000)

**UX Behaviors - Mô tả trải nghiệm người dùng**

- Khi query submit: Run Button biến mất, Cancel Button (đỏ) xuất hiện thế chỗ
- Nhấn Cancel (query < 10s): gửi DELETE request ngay, không confirm
- Nhấn Cancel (query > 10s): AlertDialog "Are you sure?" → user confirm → gửi DELETE
- Sau khi nhấn Cancel: Button disabled + Loader2 trong lúc chờ response
- Response về: Badge đổi sang "Cancelled", DataTable trống, Toast "Query cancelled"
- Cancel thành công trong < 2 giây: UX mượt, user thấy phản hồi ngay

### **📡 API Contract**

DELETE /api/v1/query/q-abc123

Authorization: Bearer &lt;jwt&gt;

// Đang chạy → đang stop:

Response 202: { "status": "stopping", "query_id": "q-abc123" }

// Đã xong rồi (idempotent):

Response 200: { "status": "success", "message": "Query already completed" }

// Đã cancelled rồi (idempotent):

Response 200: { "status": "stopped", "message": "Query already stopped" }

### **✅ Acceptance Criteria - Điều kiện nghiệm thu**

Requirement này chỉ được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#** | **Điều kiện kiểm tra**                         | **Kết quả mong đợi**                                            |
| ----- | ---------------------------------------------- | --------------------------------------------------------------- |
| **1** | DELETE query đang ở status "running"           | **202 { status:"stopping" }, query → "stopped" trong < 2s**     |
| **2** | DELETE query của chính mình                    | **202 - success**                                               |
| **3** | DELETE query của người khác (không phải Admin) | **403 Forbidden**                                               |
| **4** | Admin DELETE query của user khác               | **202 - success**                                               |
| **5** | DELETE query đã status "success"               | **200 { status:"success", message:"Query already completed" }** |
| **6** | PostgreSQL query bị cancel                     | **pg_cancel_backend được gọi, DB giải phóng resource**          |
| **7** | Worker detect cancel flag trong 500ms          | **Execution dừng, status update "stopped"**                     |

### **⚠️ Error Responses**

| **HTTP Code** | **Tình huống**                            | **Response body / Hành động**                                              |
| ------------- | ----------------------------------------- | -------------------------------------------------------------------------- |
| **403**       | User không phải owner và không phải Admin | { "error": "forbidden", "message": "Not authorized to cancel this query" } |
| **404**       | query_id không tồn tại trong DB           | { "error": "not_found", "message": "Query not found" }                     |

**QE-007 - Query History & Result Retrieval**

| **Priority**        | **P1**                                                                                  | **Phase** | Phase 2 - sau QE-001 và QE-004 |
| ------------------- | --------------------------------------------------------------------------------------- | --------- | ------------------------------ |
| **API / Endpoint**  | GET /api/v1/query/history · GET /api/v1/query/:id/result · DELETE /api/v1/query/history |
| **Frontend route**  | /sqllab - tab "History" trong results panel                                             |
| **DB Tables**       | query (đọc toàn bộ lịch sử, filter, paginate)                                           |
| **Effort ước tính** | **3 ngày (1.5 BE + 1.5 FE)**                                                            |
| **Phụ thuộc vào**   | QE-001 ✓ và QE-004 ✓ (phải có queries được ghi vào DB trước mới có gì để hiện)          |

### **📋 Mô tả nghiệp vụ (Business Context)**

User cần xem lại các query đã chạy: query nào thành công, query nào fail và lỗi gì, query nào đang chạy, kết quả cũ còn lấy về được không. History tab trong SQL Lab hiện danh sách tất cả queries của user, có thể filter, search trong SQL text, và chạy lại bất kỳ query nào. Admin thấy history của mọi người. Kết quả cũ (stored trong Redis) vẫn có thể tải về nếu chưa hết TTL.

### **🔄 Luồng hoạt động chi tiết (Request Flow)**

- User mở tab "History" trong SQL Lab → trigger useQuery fetch history
- GET /api/v1/query/history với params: status?, database_id?, sql_contains?, page?, page_size?
- BE: lấy userID từ JWT. Build GORM query:
- Non-Admin: WHERE user_id = :uid
- Admin: không có WHERE user_id (thấy tất cả)
- Thêm filter nếu có: AND status = :status, AND database_id = :dbid
- sql_contains: AND sql ILIKE "%:q%" - dùng PostgreSQL GIN index để full-text search
- ORDER BY start_time DESC, LIMIT 20, OFFSET (page-1)\*20
- Trả về: danh sách query records + pagination info
- User nhấn "Download" → GET /api/v1/query/:id/result → fetch từ Redis bằng results_key
- Nếu Redis key tồn tại → trả về data
- Nếu không → 410 Gone (hết TTL, user cần chạy lại)
- User nhấn "Run Again" → load SQL vào editor + tự động execute (gọi QE-001)
- User nhấn "Load SQL" → chỉ load SQL vào editor, không execute
- Admin nhấn "Clear History" → DELETE /api/v1/query/history?older_than=30d

### **⚙️ Backend - Yêu cầu kỹ thuật**

- GET /api/v1/query/history: params: status (enum), database_id (int), sql_contains (string), page (int, default 1), page_size (int, default 20, max 100)
- Response fields per query: id, client_id, database_id, database_name (join), status, sql (first 500 chars), rows, start_time, end_time, duration_ms (end-start), error_message, results_key (để check còn cache không)
- GIN index trên cột sql: CREATE INDEX idx_query_sql_gin ON query USING GIN (to_tsvector("english", sql))
- Full-text search: WHERE to_tsvector("english", sql) @@ plainto_tsquery("english", :q)
- Pagination: GORM.Offset((page-1)\*pageSize).Limit(pageSize)
- GET /api/v1/query/:id/result: GORM.First → rdb.Get("qresult:"+query.ResultsKey) → if nil: 410
- Result response: unmarshal MessagePack → return { data, columns, rows } (giống QE-001 format)
- DELETE /api/v1/query/history?older_than=30d: Admin only. GORM.Where("start_time < ?", now()-30d).Delete(&Query{}) → return { deleted: N }
- History auto-refresh: frontend refetchInterval:5000 - không cần BE làm gì thêm

### **🖥️ Frontend - Yêu cầu UI/UX**

**Components sử dụng (shadcn/ui)**

- shadcn Tabs \[Results | History | Saved Queries\]: trong SQL Lab results panel
- shadcn DataTable (TanStack Table v8): History tab với các cột - Status, SQL (truncated), Database, Duration, Rows, Actions
- shadcn Badge (status): green=success, red=failed, amber=running (pulsing animation), gray=stopped
- shadcn Popover trên SQL cell: hiện full SQL khi hover (SQL bị truncate ở 100 chars trong table)
- shadcn Button "Run Again" per row: load SQL + execute ngay
- shadcn Button "Load SQL" per row: chỉ load SQL vào editor, không chạy
- shadcn Button "Download" per row (chỉ hiện nếu results_key còn valid): trigger download
- shadcn Input + Search icon: filter history theo SQL text content
- shadcn Select: filter theo Status (All / Success / Failed / Running / Stopped)
- shadcn Button "Clear History" (Admin only, variant=destructive): mở AlertDialog confirm trước

**State Management**

- useQuery({ queryKey:\["query-history", {status, q, page, dbId}\], queryFn: ()=>api.getQueryHistory(filters), refetchInterval:5000 }) - auto refresh mỗi 5 giây để thấy running queries update
- useMutation cho "Run Again": load SQL vào sqlLabStore.tabs\[activeTabId\].sql → trigger executeQuery()
- Download button disabled state: results_key null hoặc result đã expired (từ GET result trả 410)

**UX Behaviors - Mô tả trải nghiệm người dùng**

- Mở History tab: danh sách queries hiện ngay, mới nhất ở trên
- Query đang chạy: Badge "Running" với pulsing animation, auto update sau 5s
- Hover vào SQL cell (bị truncate): Popover hiện full SQL text
- Nhấn "Run Again": SQL load vào editor, tab switch sang Results, query tự chạy
- Nhấn "Load SQL": SQL load vào editor, không chạy, user có thể chỉnh trước khi run
- Download icon grayed out + Tooltip "Result expired - rerun query" khi kết quả hết TTL
- Admin: thấy tất cả queries của mọi user. Button "Clear History" xuất hiện
- Search: gõ "orders" vào search box → chỉ hiện queries có chữ "orders" trong SQL

### **📡 API Contract**

GET /api/v1/query/history?status=failed&sql_contains=orders&page=1&page_size=20

Response 200:

{ "queries": \[

{ "id": 42, "status": "success", "sql": "SELECT \* FROM orders...",

"database_name": "prod-postgres", "rows": 127,

"start_time": "2024-01-15T14:30:00Z", "duration_ms": 850,

"results_key": "qcache:a3f2..." }

\],

"total": 156, "page": 1, "page_size": 20 }

GET /api/v1/query/42/result

Response 200: { "data": \[...\], "columns": \[...\], "rows": 127 }

Response 410: { "error": "result_expired", "message": "Result TTL expired. Rerun query." }

DELETE /api/v1/query/history?older_than=30d (Admin only)

Response 200: { "deleted": 1842 }

### **✅ Acceptance Criteria - Điều kiện nghiệm thu**

Requirement này chỉ được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#** | **Điều kiện kiểm tra**                    | **Kết quả mong đợi**                                                    |
| ----- | ----------------------------------------- | ----------------------------------------------------------------------- |
| **1** | GET /history không có filter              | **Danh sách queries của user, sort mới nhất trước, paginated 20/trang** |
| **2** | Admin GET /history                        | **Thấy queries của mọi user**                                           |
| **3** | GET /history?status=failed                | **Chỉ queries có status=failed**                                        |
| **4** | GET /history?sql_contains=orders          | **Queries có chữ "orders" trong SQL (GIN index search)**                |
| **5** | GET /:id/result với results_key còn valid | **200 với data đầy đủ**                                                 |
| **6** | GET /:id/result với results_key hết TTL   | **410 Gone**                                                            |
| **7** | Admin DELETE /history?older_than=30d      | **200 { deleted: N }**                                                  |
| **8** | Non-admin GET history của user khác       | **403 Forbidden**                                                       |

### **⚠️ Error Responses**

| **HTTP Code** | **Tình huống**                                         | **Response body / Hành động**                                                |
| ------------- | ------------------------------------------------------ | ---------------------------------------------------------------------------- |
| **403**       | Non-admin cố GET history của user khác (qua id filter) | { "error": "forbidden" }                                                     |
| **404**       | GET /:id/result với id không tồn tại                   | { "error": "not_found" }                                                     |
| **410**       | GET /:id/result với results_key hết TTL trong Redis    | { "error": "result_expired", "message": "Result TTL expired. Rerun query." } |

**QE-008 - Query Cost Estimation**

| **Priority**        | **P2**                                                                                         | **Phase** | Phase 3 - không block ai, implement sau |
| ------------------- | ---------------------------------------------------------------------------------------------- | --------- | --------------------------------------- |
| **API / Endpoint**  | POST /api/v1/query/estimate                                                                    |
| **Frontend route**  | /sqllab - nút "Estimate Cost" (chỉ hiện với DB hỗ trợ: PostgreSQL, BigQuery, Snowflake, MySQL) |
| **DB Tables**       | dbs (đọc sqlalchemy_uri để detect DB type)                                                     |
| **Effort ước tính** | **2 ngày (1.5 BE + 0.5 FE)**                                                                   |
| **Phụ thuộc vào**   | DBC-006 (Connection Pool để chạy EXPLAIN), DBC-001 (DB type detection từ URI)                  |

### **📋 Mô tả nghiệp vụ (Business Context)**

Trước khi chạy một query lớn (có thể scan hàng triệu rows, tốn nhiều tiền nếu dùng BigQuery), data engineer muốn biết ước tính chi phí. QE-008 chạy EXPLAIN (không execute query thực) để lấy thông tin từ query planner của DB: PostgreSQL cho biết planner cost và ước tính số rows, BigQuery cho biết bytes sẽ scan (và tính ra tiền), Snowflake cho biết partitions. DB không hỗ trợ → trả về supported:false. Rate limit 30 req/phút để tránh spam EXPLAIN.

### **🔄 Luồng hoạt động chi tiết (Request Flow)**

- User gõ SQL xong, nhấn "Estimate Cost" hoặc sau khi gõ 2 giây (debounce) tự estimate
- FE gửi: POST /api/v1/query/estimate { sql, database_id }
- BE: lấy database record → đọc sqlalchemy_uri → detect driver từ scheme (postgresql://, bigquery://, ...)
- Rate limit check: rdb.Incr("rate:estimate:"+userID) → nếu > 30 trong 60s → 429
- Route đến estimator theo driver:
- PostgreSQL: db.QueryRow("EXPLAIN (FORMAT JSON) "+sql) → parse JSON → total_cost, estimated_rows
- BigQuery: bqClient.Jobs.Insert({dryRun:true, query:sql}) → bytes_processed → cost = bytes/1TB \* \$5
- Snowflake: db.QueryRow("EXPLAIN "+sql) → parse output → partitions, bytes
- MySQL/ClickHouse: db.QueryRow("EXPLAIN "+sql) → parse text output
- Khác: return { supported: false }
- Return EstimateResult

### **⚙️ Backend - Yêu cầu kỹ thuật**

- Endpoint: POST /api/v1/query/estimate, Auth: JWT required
- DB type detection: parse sqlalchemy_uri scheme - "postgresql" | "bigquery" | "snowflake" | "mysql" | "clickhouse"
- Rate limit: rdb.Incr("rate:estimate:"+uid); rdb.Expire("rate:estimate:"+uid, 60s). if count > 30 → 429
- PostgreSQL: db.QueryRowContext(ctx, "EXPLAIN (FORMAT JSON) "+sql) → parse \[\]PlanNode\[0\].Plan.TotalCost và RowsEstimate
- BigQuery dry-run: bqClient.Jobs.Insert(projectID, &bigquery.Job{Config:{DryRun:true, Query:{Query:sql}}}) → job.Statistics.TotalBytesProcessed → cost = bytes/1e12\*5
- Snowflake: db.QueryRowContext(ctx, "EXPLAIN "+sql) → parse text output cho partitions + bytes_scanned
- MySQL: db.QueryRowContext(ctx, "EXPLAIN "+sql) → parse tabular output
- Unsupported: return EstimateResult{Supported:false}, nil - không phải error

### **🖥️ Frontend - Yêu cầu UI/UX**

**Components sử dụng (shadcn/ui)**

- shadcn Button ("Estimate Cost", Zap icon, variant=outline): trong SQL Lab toolbar, CHỈ hiện nếu db.backend in \["postgresql","bigquery","snowflake","mysql"\]
- shadcn Popover: mở khi nhấn button hoặc auto sau debounce 2s
- shadcn Card bên trong Popover: hiện metrics - rows, cost, bytes
- shadcn Skeleton: loading state bên trong Popover khi đang fetch
- shadcn Alert (info) bên trong Popover: "Estimate only. Actual execution may differ."
- shadcn Badge trong toolbar: "~50k rows" hoặc "~\$0.005" - summary sau khi estimate xong

**State Management**

- useMutation({ mutationFn: ({sql, db_id}) => fetch("/api/v1/query/estimate", {method:"POST", body:...}).then(r=>r.json()), onSuccess: (r) => setEstimate(r) })
- Show button điều kiện: selectedDB?.backend && \["postgresql","bigquery","snowflake","mysql"\].includes(selectedDB.backend)
- Debounce: useDebounce(sql, 2000) → auto trigger estimate mutation khi SQL thay đổi
- Estimate state: { supported, total_cost?, estimated_rows?, bytes_processed?, estimated_cost_usd?, driver }

**UX Behaviors - Mô tả trải nghiệm người dùng**

- User chọn BigQuery database → "Estimate Cost" button hiện trong toolbar
- User gõ SQL xong, dừng gõ 2 giây → button tự trigger, Popover mở với Skeleton
- Estimate về: Popover hiện "Estimated: 1 GB processed (~\$0.005 at \$5/TB)"
- PostgreSQL: Popover hiện "Estimated: ~50,000 rows, planner cost: 1,250"
- Badge trong toolbar update: "~\$0.005" (BigQuery) hoặc "~50k rows" (PostgreSQL)
- DB không hỗ trợ: Button ẩn hoàn toàn (không phải disabled, mà hidden)
- SQL thay đổi → estimate cũ biến mất → debounce 2s → estimate mới

### **📡 API Contract**

POST /api/v1/query/estimate

Body: { "sql": "SELECT \* FROM orders WHERE ...", "database_id": 3 }

// PostgreSQL response:

Response 200: { "supported": true, "driver": "postgresql",

"total_cost": 1250.5, "estimated_rows": 50000 }

// BigQuery response:

Response 200: { "supported": true, "driver": "bigquery",

"bytes_processed": 1073741824,

"estimated_cost_usd": 0.005 }

// Unsupported DB:

Response 200: { "supported": false }

### **✅ Acceptance Criteria - Điều kiện nghiệm thu**

Requirement này chỉ được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#** | **Điều kiện kiểm tra**                   | **Kết quả mong đợi**                                                         |
| ----- | ---------------------------------------- | ---------------------------------------------------------------------------- |
| **1** | POST với PostgreSQL database             | **{ supported:true, total_cost:1250, estimated_rows:50000 }**                |
| **2** | POST với BigQuery database               | **{ supported:true, bytes_processed:1073741824, estimated_cost_usd:0.005 }** |
| **3** | POST với SQLite hoặc DB không hỗ trợ     | **{ supported:false }**                                                      |
| **4** | User gửi 31 request trong 60 giây        | **Request thứ 31 → 429 Too Many Requests**                                   |
| **5** | SQL syntax sai → EXPLAIN fail            | **422 Unprocessable Entity**                                                 |
| **6** | DB không hỗ trợ: button hiển thị trên FE | **Button KHÔNG hiển thị (hidden, không disabled)**                           |

### **⚠️ Error Responses**

| **HTTP Code** | **Tình huống**                                 | **Response body / Hành động**                                    |
| ------------- | ---------------------------------------------- | ---------------------------------------------------------------- |
| **200**       | DB không hỗ trợ EXPLAIN (MongoDB, Redis, v.v.) | { "supported": false } - không phải error, response bình thường  |
| **422**       | SQL syntax sai khiến EXPLAIN fail              | { "error": "invalid_sql", "message": "syntax error in EXPLAIN" } |
| **429**       | User vượt quá 30 requests trong 60 giây        | { "error": "rate_limited", "retry_after": 45 }                   |

# **6\. Rủi ro & Biện pháp giảm thiểu**

| **Rủi ro**                                                    | **Mức độ**   | **Chi tiết**                                                                                                                                                                                         | **Biện pháp**                                                                                                                                                                                                                                                         |
| ------------------------------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Goroutine leak từ WebSocket connections**                   | **HIGH**     | Mỗi WS connection tạo ít nhất 1 goroutine để forward Redis pub/sub messages. Nếu cleanup không đúng khi disconnect, goroutines tích lũy → memory leak → crash server theo thời gian.                 | Bắt buộc: defer func() { redisSub.Close(); conn.Close() }() trong mọi WS handler. Monitor goroutine count qua expvar/pprof, alert nếu tăng bất thường. Integration test: kết nối 100 WS → disconnect tất cả → verify goroutine count về baseline.                     |
| **SQL Injection qua RLS template rendering**                  | **CRITICAL** | Nếu template {{current_user_id}} được render bằng string concat với giá trị do user control, attacker có thể inject SQL. Ví dụ: user_id = "1 OR 1=1" → WHERE org_id = 1 OR 1=1 → bypass toàn bộ RLS. | Bắt buộc dùng typed values: fmt.Sprintf("%d", userID) với int, không phải interface{}. sqlparser AST injection (không phải string concat SQL). Security test case: đặt user_id = "1; DROP TABLE orders; --" → verify render thành "1" (integer), SQL không bị inject. |
| **Cache poisoning - RLS bypass qua shared cache**             | **HIGH**     | Nếu cache key không include RLS hash, User A (không có RLS) cache kết quả của query, User B (có RLS hạn chế) nhận kết quả của User A từ cache → data leak.                                           | Cache key PHẢI bao gồm rlsHash = sha256(sorted RLS clauses). Test case: User A (Admin, no RLS) chạy query → User B (Gamma, có RLS) chạy cùng query → verify User B nhận kết quả đã filtered, không phải từ cache của User A.                                          |
| **Asynq worker infra chưa sẵn sàng khi Sprint 2 bắt đầu**     | **MED**      | QE-004 depend on Asynq + Redis worker deployment. Nếu infra team chưa deploy, Sprint 2 bị block.                                                                                                     | Xác nhận từ DevOps team trước ngày đầu Sprint 2 rằng: Redis cluster sẵn sàng, Asynq worker containers đã được build và test, HPA config cho worker pod đã setup.                                                                                                      |
| **pg_cancel_backend PID tracking mất khi connection pooling** | **MED**      | Để cancel PostgreSQL query ở DB level, cần biết backend PID. Tuy nhiên connection pool có thể thay đổi connection → PID mất.                                                                         | Store pg_backend_pid() vào query.BackendPID ngay khi BEGIN execution (trong cùng connection). Pool phải expose raw \*sql.Conn để lấy PID. Test: submit async query → cancel → verify pg_stat_activity không còn query đó.                                             |
| **WebSocket fallback về polling không seamless**              | **LOW**      | Nếu WS fail và fallback về polling, user có thể thấy khoảng thời gian chờ lâu hơn hoặc inconsistent state.                                                                                           | Test fallback path: mock WS disconnect → verify polling starts trong < 500ms, results vẫn hiện đúng. Badge "Reconnecting..." phải xuất hiện để user biết.                                                                                                             |

# **7\. Definition of Done**

Một requirement được coi là DONE khi TẤT CẢ các điều kiện sau đều pass:

| **#**  | **Điều kiện**                                                                                     |
| ------ | ------------------------------------------------------------------------------------------------- |
| **1**  | Tất cả Acceptance Criteria trong requirement đều pass (manual test hoặc automated test)           |
| **2**  | Unit tests cho business logic (RLS injection, cache key generation, queue routing) coverage ≥ 80% |
| **3**  | Integration test: end-to-end flow từ HTTP request → DB execute → cache → response                 |
| **4**  | Error cases đều được test: timeout, 403, 404, 410, 429, WS disconnect                             |
| **5**  | Code review đã được approve bởi ít nhất 1 senior engineer                                         |
| **6**  | Không có anti-pattern vi phạm (Go anti-patterns rules, React anti-patterns rules đã định nghĩa)   |
| **7**  | OpenTelemetry traces hoạt động: mỗi query có trace với span từ API → RLS → cache → execute        |
| **8**  | Không có goroutine leak (verified bằng pprof sau 100 requests)                                    |
| **9**  | API response format khớp với API Contract trong tài liệu này                                      |
| **10** | Frontend: không có console error, Lighthouse accessibility score ≥ 90                             |
| **11** | Performance: latency đạt ngưỡng NFR (cache hit < 20ms, sync timeout 30s)                          |