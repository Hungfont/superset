sequenceDiagram
    autonumber
    actor User
    participant Browser
    participant API as Flask REST API
    participant CacheSvc as Cache Service
    participant Redis as Redis (Cache Backend)
    participant SL as Semantic Layer
    participant DB as Data Source
    participant Worker as Celery Worker
    participant Headless as Headless Browser

    rect rgb(219,234,254)
        Note over User,DB: Flow A — Chart Query Cache (Cache Miss → Hit)
        User->>Browser: Load chart
        Browser->>API: POST /api/v1/chart/data {query_context}
        API->>CacheSvc: compute_cache_key(query_context + rls + user_id)
        Note right of CacheSvc: Key = MD5(sql + filters + datasource + rls)
        CacheSvc->>Redis: GET cache_key
        Redis-->>CacheSvc: (nil) — MISS
        CacheSvc-->>API: Cache MISS
        API->>SL: build SQL
        SL-->>API: SQL
        API->>DB: Execute SQL
        DB-->>API: Result rows
        API->>CacheSvc: store(cache_key, result, timeout=cache_timeout)
        CacheSvc->>Redis: SET cache_key EX cache_timeout
        Redis-->>CacheSvc: OK
        API-->>Browser: Chart data

        Note over User,Redis: Second load — same chart, same filters
        User->>Browser: Reload / revisit chart
        Browser->>API: POST /api/v1/chart/data {same query_context}
        API->>CacheSvc: compute_cache_key(...)
        CacheSvc->>Redis: GET cache_key
        Redis-->>CacheSvc: Cached result — HIT
        CacheSvc-->>API: Cached data
        API-->>Browser: Chart data (no DB hit)
    end

    rect rgb(220,252,231)
        Note over User,Redis: Flow B — Explore Form Data Cache (Temporary State)
        User->>Browser: Share Explore state
        Browser->>API: POST /api/v1/explore/form_data {formData}
        API->>Redis: SET key_value(resource=form_data, uuid, value=formData, expires_on=+24h)
        Redis-->>API: OK
        API-->>Browser: {key: uuid}
        Browser->>Browser: Share URL /explore/p/{uuid}

        Note over User,Redis: Recipient opens shared link
        Browser->>API: GET /api/v1/explore/form_data/{key}
        API->>Redis: GET key_value(resource=form_data, uuid)
        Redis-->>API: formData JSON
        API-->>Browser: Restore Explore state
    end

    rect rgb(252,231,243)
        Note over Worker,Redis: Flow C — Dashboard Thumbnail Cache (Async)
        Worker->>Headless: render_dashboard_thumbnail(dashboard_id)
        Headless->>API: GET /superset/dashboard/{id}/?standalone=true
        API->>DB: Execute all chart queries
        DB-->>API: Data
        API-->>Headless: Rendered page
        Headless->>Headless: Capture PNG (1200x800)
        Headless-->>Worker: PNG bytes
        Worker->>Redis: SET thumbnail:dashboard:{digest} = PNG bytes (TTL=24h)
        Redis-->>Worker: OK

        Note over User,Redis: User requests thumbnail
        Browser->>API: GET /api/v1/dashboard/{id}/thumbnail/{digest}
        API->>Redis: GET thumbnail:dashboard:{digest}
        alt Thumbnail HIT
            Redis-->>API: PNG bytes
            API-->>Browser: 200 image/png
        else Thumbnail MISS (stale/first request)
            API->>Worker: Dispatch cache_dashboard_thumbnail task
            API-->>Browser: 202 Accepted (thumbnail being generated)
        end
    end

    rect rgb(255,237,213)
        Note over Admin,Redis: Flow D — Cache Invalidation
        Admin->>Browser: Edit & Save dataset (column change)
        Browser->>API: PUT /api/v1/dataset/{id}
        API->>Redis: DELETE all keys matching datasource:{id}:*
        Redis-->>API: Flushed
        API-->>Browser: OK (next chart load will re-query DB)
    end