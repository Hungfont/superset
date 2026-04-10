sequenceDiagram
    autonumber
    actor EndUser as End User
    participant HostApp as Host Application (3rd Party)
    participant HostBE as Host App Backend
    participant SDK as @superset-ui/embedded-sdk
    participant API as Superset REST API
    participant SecMgr as Security Manager
    participant MetaDB as Metadata DB
    participant Cache as Redis Cache
    participant DB as Data Source

    rect rgb(219,234,254)
        Note over HostApp,MetaDB: Phase 1 — Admin Enables Embedded Dashboard
        Note over API,MetaDB: (One-time setup by Superset Admin)
        API->>MetaDB: INSERT INTO embedded_dashboards {uuid, dashboard_id, allowed_domains}
        MetaDB-->>API: OK
    end

    rect rgb(220,252,231)
        Note over EndUser,SecMgr: Phase 2 — Host App Requests Guest Token
        EndUser->>HostApp: Opens page with embedded dashboard
        HostApp->>HostBE: Authenticated request (session)
        HostBE->>HostBE: Determine user context + tenant filters
        HostBE->>API: POST /api/v1/security/guest_token/ [Authorization: Bearer admin_token]\n{user:{username,first_name,last_name}, resources:[{type:dashboard,id:uuid}], rls:[{clause:"tenant_id=42"}]}
        API->>SecMgr: Validate admin_token
        SecMgr-->>API: Admin verified
        API->>MetaDB: SELECT embedded_dashboards WHERE uuid=? (verify embed enabled)
        MetaDB-->>API: Embedded config + allowed_domains
        API->>SecMgr: create_guest_access_token(user, resources, rls_clauses)
        SecMgr->>SecMgr: Sign JWT {sub:guest, resources, rls, exp:+5min}
        SecMgr-->>API: Guest JWT (short-lived)
        API-->>HostBE: {token: guest_jwt}
        HostBE-->>HostApp: Guest token
    end

    rect rgb(252,231,243)
        Note over EndUser,DB: Phase 3 — SDK Renders Embedded Dashboard
        HostApp->>SDK: embedDashboard({id, supersetDomain, fetchGuestToken})
        SDK->>HostBE: fetchGuestToken() (callback)
        HostBE-->>SDK: guest_jwt
        SDK->>SDK: Create hidden <iframe> src=/embedded/{uuid}
        SDK->>API: GET /embedded/{uuid} [Authorization: Bearer guest_jwt]
        API->>SecMgr: decode_guest_token() → validate resources + expiry
        SecMgr-->>API: Guest context (user, rls_clauses)
        API->>MetaDB: SELECT dashboards JOIN embedded_dashboards WHERE uuid=?
        MetaDB-->>API: Dashboard definition
        API-->>SDK: Dashboard HTML (standalone mode)
        SDK->>SDK: Inject into iframe
        SDK-->>EndUser: Embedded dashboard rendered
    end

    rect rgb(255,237,213)
        Note over EndUser,DB: Phase 4 — Chart Data Queries (with RLS)
        SDK->>API: POST /api/v1/chart/data {query_context} [Bearer guest_jwt]
        API->>SecMgr: decode_guest_token() → rls_clauses=["tenant_id=42"]
        SecMgr-->>API: Scoped user + RLS
        API->>Cache: GET cache_key (query + rls hash)
        alt Cache MISS
            API->>DB: Execute SQL WHERE tenant_id=42
            DB-->>API: Filtered rows
            API->>Cache: SET result
        end
        API-->>SDK: Chart data (tenant-scoped)
        SDK-->>EndUser: Rendered chart
    end

    rect rgb(240,253,244)
        Note over SDK,API: Phase 5 — Token Refresh (Auto-renew before expiry)
        SDK->>SDK: Token expiry timer fires (before exp)
        SDK->>HostBE: fetchGuestToken() callback
        HostBE->>API: POST /api/v1/security/guest_token/ (new token)
        API-->>HostBE: New guest_jwt
        HostBE-->>SDK: Refreshed token
        SDK->>SDK: Update iframe Authorization header
    end