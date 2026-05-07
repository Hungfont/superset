1. Overall Project Structure
The Go backend lives entirely under D:\superset\backend. The module is superset/auth-service (Go 1.25.7, using Gin for HTTP, GORM for ORM with PostgreSQL, Redis via go-redis/v9, and JWT RS256 for auth).
D:\superset\backend\
├── cmd\api\main.go                    # Application entry point
├── configs\config.go                  # Environment-based configuration loader
├── internal\
│   ├── app\                           # APPLICATION LAYER (business logic / use cases)
│   │   ├── auth\                      # Auth services (login, register, RLS, roles, etc.)
│   │   ├── db\                        # Database service, connection pool, schema inspection
│   │   ├── dataset\                   # Dataset service
│   │   └── query\                     # Query execution, RLS injection, caching, async queue
│   ├── delivery\http\                 # TRANSPORT/DELIVERY LAYER (HTTP handlers + middleware)
│   │   ├── router.go                  # Central route registration (Gin engine)
│   │   ├── middleware\                # JWT middleware, permission/authorize middleware
│   │   ├── auth\                      # Auth HTTP handlers (login, register, users, roles, etc.)
│   │   ├── db\                        # Database HTTP handler
│   │   ├── dataset\                   # Dataset HTTP handler
│   │   ├── query\                     # Query HTTP handler (sync + async)
│   │   └── rls\                       # RLS filter HTTP handler
│   ├── domain\                        # DOMAIN LAYER (entities, value objects, repository interfaces)
│   │   ├── auth\                      # Auth domain (User, Role, Permission, RLSFilter entities)
│   │   ├── db\                        # DB domain (Database entity)
│   │   ├── dataset\                   # Dataset domain (Dataset, Column, SqlMetric entities)
│   │   └── query\                     # Query domain (Query entity, request/response types)
│   ├── repository\                    # INFRASTRUCTURE LAYER (repository implementations)
│   │   ├── postgres\                  # PostgreSQL repos (User, Role, DB, Dataset, Query, RLS repo)
│   │   └── redis\                     # Redis repos (JWT, rate, refresh, cache, async queues)
│   ├── worker\                        # Background workers
│   │   ├── query_worker.go            # Async query worker (Redis BRPop consumer)
│   │   └── column_sync.go             # Column sync worker for datasets
│   └── pkg\                           # Shared internal utilities
│       ├── email\                     # SMTP email sender
│       └── validator\                 # Password validator
├── pkg\                               # Public package (empty, just README)
├── test\                              # Test helpers
├── examples\                          # Example files
├── go.mod, go.sum                     # Go module definition
├── Makefile                           # Build commands
└── .air.toml                          # Hot reload config
---
2. Architecture Pattern: Clean Architecture / Domain-Driven Design (DDD)
The codebase follows a clean, layered architecture with clear separation of concerns:
Directory
internal/domain/
internal/app/
internal/repository/
internal/delivery/http/
Dependencies flow inward: delivery -> app -> domain. The repository layer implements domain interfaces. DI is manual in main.go (no DI framework).
---
3. Main Entry Point
File: D:\superset\backend\cmd\api\main.go
The main() function:
1. Loads .env config via godotenv
2. Loads typed config via configs.Load()
3. Opens GORM Postgres connection and auto-migrates all domain entities
4. Connects to Redis
5. Parses RSA key pair for JWT RS256
6. Manually wires all dependencies (repos -> services -> handlers)
7. Creates the Gin router via delivery.NewRouter(...) with all route groups
8. Starts the query background worker (worker.NewQueryWorker)
9. Starts the column sync background worker (worker.NewColumnSyncWorker)
10. Starts http.Server on the configured port
11. Handles graceful shutdown on SIGINT/SIGTERM
---
4. All GORM Model / Entity Definitions
4a. Auth Domain (internal/domain/auth/entity.go)
Struct	Table
RegisterUser	ab_register_user
User	ab_user
Role	ab_role
Permission	ab_permission
ViewMenu	ab_view_menu
PermissionView	ab_permission_view
RLSFilter	row_level_security_filters
RLSFilterRoleJunction	rls_filter_roles
RLSFilterTableJunction	rls_filter_tables
RLSAuditLog	rls_audit_log
4b. Database Domain (internal/domain/db/database.go)
Description
External database connection config (SQLAlchemy URI, permissions)
4c. Dataset Domain (internal/domain/dataset/dataset.go)
Table
tables
table_columns
sql_metrics
4d. Query Domain (internal/domain/query/entity.go)
Struct	Table
Query	query
Key fields on Query:
- ID (varchar PK), ClientID, DatabaseID, UserID, TenantID
- SQL (original), ExecutedSQL (after RLS injection)
- Status (pending/running/success/failed/timed_out/stopped)
- StartTime, EndTime, Rows, ResultsKey, ErrorMessage
- Schema, CreatedAt, UpdatedAt
---
5. All Query-Related Files
5a. Domain Layer
File	Description
D:\superset\backend\internal\domain\query\entity.go	Query GORM entity + all request/response DTOs (ExecuteRequest, ExecuteResponse, ExecuteMeta, AsyncSubmitRequest, AsyncSubmitResponse, QueryStatusResponse, QueryTask, ListFilter, etc.)
D:\superset\backend\internal\domain\query\repository.go	Repository interface: Create, GetByID, Update, List
5b. Application Layer
Description
QueryExecutor - Main sync query engine: RLS injection, Redis caching (CheckCache/SetCache/FlushCache), SQL normalization, query recording, connection pool execution, role-based row limits (Admin=10M, Alpha=100K, Gamma=10K), 30s timeout, QueryError type for HTTP status mapping. Also contains RLSInjector for injecting WHERE clauses.
AsyncQueryExecutor - Async query submission via Redis queues (queue:query:critical, queue:query:default, queue:query:low), WorkerPool per queue (10/20/5 slots), retry logic (max 3 attempts, 5s/25s/125s backoff), cancel support, Redis pub/sub status events, query lifecycle management (pending -> running -> success/failed/stopped).
Unit tests for SQL normalization, cache key generation, cache size validation, TTL, nil Redis handling
Unit tests for RLS injection (Admin bypass, Regular/Base filter types, UNION queries, template rendering, caching)
Integration tests: RLS with mock repos, cache flush, cross-user cache key isolation, acceptance criteria verification
Unit tests for async executor
5c. Delivery/HTTP Layer
File	Description
D:\superset\backend\internal\delivery\http\query\handler.go	Handler - HTTP handlers: Execute (sync POST), Submit (async POST), GetStatus (GET), Cancel (DELETE), GetResult (GET). Extracts UserContext from Gin context, maps domain errors to HTTP status codes (400/403/408/500/503).
D:\superset\backend\internal\delivery\http\query\handler_test.go	Unit tests for handler (Submit returns 202, nil-safety, status response format, queue resolution)
5d. Repository Layer
File	Description
D:\superset\backend\internal\repository\postgres\query_repo.go	queryRepo - GORM implementation of query.Repository: Create, GetByID, Update, List (with filtering by UserID, Status, DatabaseID, SQL like, pagination)
5e. Worker Layer
File
D:\superset\backend\internal\worker\query_worker.go
D:\superset\backend\internal\worker\query_worker_test.go
---
6. All API Routes / Endpoints
All routes are defined in D:\superset\backend\internal\delivery\http\router.go under /api/v1:
Public (no auth required)
Path
/api/v1/auth/register
/api/v1/auth/verify
/api/v1/auth/login
/api/v1/auth/refresh
/api/v1/auth/logout
Protected (JWT required) - Dataset CRUD
Path
/api/v1/datasets
/api/v1/datasets/:id
/api/v1/datasets
/api/v1/datasets/virtual
/api/v1/datasets/:id
/api/v1/datasets/:id
/api/v1/datasets/:id/columns/:col_id
/api/v1/datasets/:id/columns
/api/v1/datasets/:id/metrics
/api/v1/datasets/:id/metrics
/api/v1/datasets/:id/metrics
/api/v1/datasets/:id/metrics/:metric_id
/api/v1/datasets/:id/metrics/:metric_id
/api/v1/datasets/:id/refresh
/api/v1/datasets/:id/cache/flush
Protected - Admin routes (/api/v1/admin)
Database management
Path
/admin/databases
/admin/databases
/admin/databases/:id
/admin/databases/:id/schemas
/admin/databases/:id/tables
/admin/databases/:id/columns
/admin/databases/:id
/admin/databases/:id
/admin/databases/test
/admin/databases/:id/test
User management
Path
/admin/users
/admin/users/:id
/admin/users
/admin/users/:id
/admin/users/:id
/admin/users/:id/roles
/admin/users/:id/roles
Role management
Path
/admin/roles
/admin/roles
/admin/roles/:id
/admin/roles/:id
/admin/roles/:id/permissions
/admin/roles/:id/permissions
/admin/roles/:id/permissions/add
/admin/roles/:id/permissions/:pv_id
Permission management
Path
/admin/permissions
/admin/permissions
/admin/view-menus
/admin/view-menus
/admin/permission-views
/admin/permission-views
/admin/permission-views/:id
RLS Filter management (Admin-only)
Method	Path
GET	/admin/rls
GET	/admin/rls/:id
POST	/admin/rls
PUT	/admin/rls/:id
DELETE	/admin/rls/:id
Query endpoints (protected, under /api/v1 prefix)
Method	Path
POST	/query/execute
POST	/query/submit
GET	/query/:id/status
GET	/query/:id/result
DELETE	/query/:id
Note: The query routes appear inside protected.POST/G calls within the admin block (line 128-133 of router.go), which means they are nested under /api/v1 and protected by JWT middleware but also inside the admin sub-group. They are accessible as /api/v1/query/execute, /api/v1/query/submit, etc.
---
7. Key Design Patterns
- Manual Dependency Injection in main.go -- all components are created and wired explicitly
- Repository pattern -- domain interfaces in internal/domain/*/repository.go, concrete implementations in internal/repository/postgres/ and internal/repository/redis/
- CQRS-lite -- separate sync (QueryExecutor.Execute) and async (AsyncQueryExecutor.Submit + QueryWorker) paths
- Priority queue routing -- Admin queries go to queue:query:critical, Alpha to queue:query:default, Gamma to queue:query:low
- RLS (Row-Level Security) -- SQL injection of WHERE clauses based on user roles, rendered from templates with {{current_user_id}} and {{current_username}}
- Query result caching -- SHA256-based cache keys incorporating normalized SQL, database ID, schema, and RLS hash; stored in Redis with configurable TTL (10MB max, 24h default)
- Role-based row limits -- Admin: 10M rows, Alpha: 100K, Gamma: 10K
- Graceful error handling -- QueryError type maps DB errors to HTTP status codes (400 for SQL syntax, 408 for timeout, 403 for forbidden, 500 for internal)
- Graceful shutdown -- context-based worker and connection pool shutdown on OS signals