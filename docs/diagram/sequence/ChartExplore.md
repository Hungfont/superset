sequenceDiagram
    autonumber
    actor User
    participant Browser as Browser (Explore UI)
    participant API as Flask REST API
    participant SecMgr as Security Manager
    participant MetaDB as Metadata DB
    participant SL as Semantic Layer
    participant Cache as Redis Cache
    participant DB as Data Source

    rect rgb(219,234,254)
        Note over User,DB: Phase 1 — Open Explore (Dataset → Chart)
        User->>Browser: Click dataset → Explore
        Browser->>API: GET /api/v1/dataset/{id}
        API->>MetaDB: SELECT tables + table_columns + sql_metrics WHERE id=?
        MetaDB-->>API: Dataset definition (columns, metrics, main_dttm_col)
        API-->>Browser: Dataset schema + semantic metadata
        Browser->>Browser: Render Explore panel (dropdowns, filters, time range)
    end

    rect rgb(220,252,231)
        Note over User,DB: Phase 2 — Live Query Preview
        User->>Browser: Select metric, groupby, time grain, filters
        Browser->>API: POST /api/v1/chart/data {query_context}
        API->>SecMgr: Verify datasource_access permission
        SecMgr-->>API: Authorised
        API->>Cache: GET cache_key
        alt Cache MISS
            API->>SL: build_sqla_query(query_context)
            SL->>SL: Resolve metrics → sql_metrics.expression
            SL->>SL: Resolve columns → table_columns
            SL->>SL: Apply time grain (db_engine_spec.time_grain_expressions)
            SL->>SL: Apply RLS WHERE clauses
            SL-->>API: Generated SQL
            API->>DB: Execute SQL
            DB-->>API: Raw result set
            API->>Cache: SET result (TTL)
        end
        API-->>Browser: Result data
        Browser->>Browser: Render chart preview (ECharts)
    end

    rect rgb(252,231,243)
        Note over User,MetaDB: Phase 3 — Save Chart (Slice)
        User->>Browser: Click Save → New Chart
        Browser->>API: POST /api/v1/chart/ {slice_name, viz_type, params, datasource_id, datasource_type}
        API->>SecMgr: can("post", "Chart")
        SecMgr-->>API: Permitted
        API->>MetaDB: INSERT INTO slices (slice_name, viz_type, params, datasource_id ...)
        MetaDB-->>API: {id, slice_name}
        API->>MetaDB: INSERT INTO slice_user (slice_id, user_id)
        API-->>Browser: 201 Created {id}
        Browser->>Browser: Update URL → /explore/?slice_id={id}
    end

    rect rgb(255,237,213)
        Note over User,MetaDB: Phase 4 — Add to Dashboard
        User->>Browser: Save to existing dashboard
        Browser->>API: POST /api/v1/dashboard/{id}/charts {chart_ids:[id]}
        API->>MetaDB: INSERT INTO dashboard_slices (dashboard_id, slice_id)
        API->>MetaDB: UPDATE dashboards SET position_json (layout)
        MetaDB-->>API: OK
        API-->>Browser: Updated dashboard
    end

    rect rgb(240,253,244)
        Note over User,MetaDB: Phase 5 — Explore Permalink
        User->>Browser: Share → Copy Permalink
        Browser->>API: POST /api/v1/explore/permalink {formData}
        API->>MetaDB: INSERT INTO key_value (resource=explore, uuid=generated, value=formData)
        MetaDB-->>API: {key: uuid}
        API-->>Browser: Shareable URL /explore/p/{uuid}
    end