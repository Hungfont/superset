sequenceDiagram
    autonumber
    actor User
    participant Browser as Browser (React)
    participant API as Flask REST API
    participant SecMgr as Security Manager
    participant Cache as Redis Cache
    participant SL as Semantic Layer /<br/>Query Context
    participant Celery as Celery Worker
    participant DB as Data Source (DB/DW)
    participant MetaDB as Metadata DB

    %% ─── Dashboard Load ───
    rect rgb(219,234,254)
        Note over Browser,MetaDB: Phase 1 — Dashboard Load
        User->>Browser: Navigate to Dashboard
        Browser->>API: GET /api/v1/dashboard/{id}
        API->>SecMgr: Verify session & RBAC permissions
        SecMgr-->>API: Authorised
        API->>MetaDB: Fetch dashboard definition (charts, layout, filters)
        MetaDB-->>API: Dashboard JSON
        API-->>Browser: Dashboard metadata
        Browser->>Browser: Render layout skeleton
    end

    %% ─── Sync Chart Query ───
    rect rgb(220,252,231)
        Note over Browser,DB: Phase 2a — Synchronous Chart Data Query
        Browser->>API: POST /api/v1/chart/data (query_context payload)
        API->>SecMgr: Check dataset-level permissions
        SecMgr-->>API: Permitted
        API->>Cache: GET cache_key (chart + filter hash)
        alt Cache HIT
            Cache-->>API: Cached result set
        else Cache MISS
            API->>SL: Build SQL from virtual dataset / metrics
            SL-->>API: Generated SQL
            API->>DB: Execute SQL query
            DB-->>API: Raw result set
            API->>Cache: SET result (TTL configurable)
        end
        API-->>Browser: Chart data (JSON)
        Browser->>Browser: Render visualization (ECharts/D3)
    end

    %% ─── Async Query Path ───
    rect rgb(252,231,243)
        Note over Browser,Celery: Phase 2b — Async Query (SQL Lab / large datasets)
        Browser->>API: POST /api/v1/sqllab/execute (async=true)
        API->>Cache: Publish job to Redis queue
        API-->>Browser: { status: "pending", job_id }
        Cache->>Celery: Dequeue job
        Celery->>DB: Execute SQL
        DB-->>Celery: Results
        Celery->>Cache: Store result keyed by job_id
        Browser->>API: GET /api/v1/sqllab/results/{job_id} (polling / WS)
        API->>Cache: Fetch result
        Cache-->>API: Result rows
        API-->>Browser: Final result set
        Browser->>Browser: Render SQL Lab results table
    end

    %% ─── Save Chart ───
    rect rgb(255,237,213)
        Note over Browser,MetaDB: Phase 3 — Save Chart / Dashboard
        User->>Browser: Click Save
        Browser->>API: POST /api/v1/chart/ (chart definition)
        API->>SecMgr: Write permission check
        SecMgr-->>API: Granted
        API->>MetaDB: Persist chart (slice) record
        MetaDB-->>API: 201 Created
        API-->>Browser: Updated chart object
    end