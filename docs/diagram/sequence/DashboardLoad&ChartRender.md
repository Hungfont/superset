sequenceDiagram
    autonumber
    actor User
    participant Browser as Browser (React)
    participant API as Flask REST API
    participant SecMgr as Security Manager
    participant MetaDB as Metadata DB
    participant Cache as Redis Cache
    participant SL as Semantic Layer
    participant DB as Data Source (DW/DB)
    participant WS as WebSocket Server

    rect rgb(219,234,254)
        Note over User,WS: Phase 1 — Dashboard Metadata Load
        User->>Browser: Navigate to /dashboard/{slug}
        Browser->>API: GET /api/v1/dashboard/{id}
        API->>SecMgr: can_access("Dashboard", id)
        SecMgr->>MetaDB: Check ab_user_role + ab_permission_view_role
        MetaDB-->>SecMgr: Roles & permissions
        SecMgr-->>API: Authorised
        API->>MetaDB: SELECT dashboards WHERE id=?
        MetaDB-->>API: Dashboard JSON (position_json, metadata)
        API->>MetaDB: SELECT slices JOIN dashboard_slices WHERE dashboard_id=?
        MetaDB-->>API: Chart list (slice_name, viz_type, params)
        API-->>Browser: Dashboard definition + chart configs
        Browser->>Browser: Render layout skeleton (position_json)
    end

    rect rgb(220,252,231)
        Note over Browser,DB: Phase 2 — Per-Chart Data Query (Sync, Cache Miss)
        loop For each chart in dashboard
            Browser->>API: POST /api/v1/chart/data {query_context, filters}
            API->>SecMgr: can_access_datasource(datasource_id)
            SecMgr->>MetaDB: Check perm / schema_perm on tables
            SecMgr-->>API: Permitted + RLS clauses
            API->>Cache: GET cache_key (hash of query+filters)
            alt Cache HIT
                Cache-->>API: Cached result set
                API-->>Browser: Chart data (from cache)
            else Cache MISS
                API->>SL: build_query(query_context, rls_clauses)
                SL->>MetaDB: Fetch table_columns, sql_metrics for dataset
                MetaDB-->>SL: Column & metric definitions
                SL->>SL: Generate SQL (GROUP BY, WHERE, LIMIT)
                SL-->>API: Final SQL string
                API->>DB: Execute SQL
                DB-->>API: Result set (rows, columns)
                API->>Cache: SET result (TTL=cache_timeout)
                API-->>Browser: Chart data (JSON)
            end
            Browser->>Browser: Render viz (ECharts / D3 / deck.gl)
        end
    end

    rect rgb(252,231,243)
        Note over Browser,WS: Phase 3 — Real-time Filter Interaction
        User->>Browser: Apply dashboard filter
        Browser->>Browser: Compute new query_context per chart
        Browser->>API: POST /api/v1/chart/data (updated filters) x N charts
        API->>Cache: GET new cache_key
        Note right of Cache: Usually MISS on first filter change
        API->>DB: Execute filtered SQL
        DB-->>API: Result
        API->>Cache: SET result
        API-->>Browser: Updated chart data
        Browser->>Browser: Re-render affected charts
    end

    rect rgb(255,237,213)
        Note over Browser,MetaDB: Phase 4 — Dashboard Filter State Persistence
        Browser->>API: PUT /api/v1/dashboard/{id}/filter_state {filters}
        API->>MetaDB: UPSERT key_value (resource=filter_state, uuid=tab_id)
        MetaDB-->>API: OK
        API-->>Browser: {key: uuid}
    end