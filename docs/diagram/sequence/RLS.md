sequenceDiagram
    autonumber
    actor Admin
    actor Analyst
    participant Browser
    participant API as Flask REST API
    participant SecMgr as Security Manager
    participant MetaDB as Metadata DB
    participant SL as Semantic Layer / Query Builder
    participant DB as Data Source

    rect rgb(219,234,254)
        Note over Admin,MetaDB: Phase 1 — Admin Creates RLS Filter
        Admin->>Browser: Security → Row Level Security → +
        Browser->>API: POST /api/v1/rowlevelsecurity/ {name, filter_type, clause:"region='APAC'", tables:[table_id], roles:[role_id]}
        API->>SecMgr: can("post","Row Level Security")
        SecMgr-->>API: Admin permitted
        API->>MetaDB: INSERT INTO row_level_security_filters {name, clause, filter_type, group_key}
        API->>MetaDB: INSERT INTO rls_filter_roles {rls_filter_id, role_id}
        API->>MetaDB: INSERT INTO rls_filter_tables {rls_filter_id, table_id}
        MetaDB-->>API: OK
        API-->>Browser: 201 Created
    end

    rect rgb(220,252,231)
        Note over Analyst,DB: Phase 2 — Analyst Loads Chart (RLS Applied)
        Analyst->>Browser: Open Dashboard
        Browser->>API: POST /api/v1/chart/data {query_context, datasource_id}
        API->>SecMgr: get_rls_filters(current_user, dataset_id)
        SecMgr->>MetaDB: SELECT rlsf.clause FROM row_level_security_filters rlsf\n  JOIN rls_filter_roles rfr ON rlsf.id=rfr.rls_filter_id\n  JOIN rls_filter_tables rft ON rlsf.id=rft.rls_filter_id\n  JOIN ab_user_role aur ON rfr.role_id=aur.role_id\n  WHERE aur.user_id=? AND rft.table_id=?
        MetaDB-->>SecMgr: [clause: "region='APAC'"]
        SecMgr-->>API: RLS WHERE clauses list

        API->>SL: build_sqla_query(query_context, rls_filters=["region='APAC'"])
        SL->>SL: Inject RLS clause into WHERE (AND logic)
        SL->>SL: Generate final SQL:\n  SELECT region, SUM(sales)\n  FROM orders\n  WHERE region='APAC'\n  GROUP BY region
        SL-->>API: SQL with RLS injected
        API->>DB: Execute filtered SQL
        DB-->>API: Scoped result set (APAC rows only)
        API-->>Browser: Data (analyst sees only APAC data)
    end

    rect rgb(252,231,243)
        Note over Analyst,DB: Phase 3 — Multiple RLS Filters (AND logic)
        Note over SecMgr: User has 2 roles: role_apac (region='APAC') + role_2024 (year=2024)
        API->>SecMgr: get_rls_filters(user, table_id)
        SecMgr->>MetaDB: Fetch all matching RLS clauses for user's roles
        MetaDB-->>SecMgr: ["region='APAC'", "year=2024"]
        SecMgr-->>API: Combined clauses
        API->>SL: build_sqla_query(rls_filters=["region='APAC'","year=2024"])
        SL->>SL: WHERE region='APAC' AND year=2024
        SL-->>API: Final SQL
        API->>DB: Execute
        DB-->>API: Doubly-filtered result
        API-->>Browser: Restricted data
    end

    rect rgb(255,237,213)
        Note over Analyst,DB: Phase 4 — Embedded Dashboard RLS via Guest Token
        Browser->>API: POST /api/v1/security/guest_token/ {resources:[{type:dashboard,id}], rls:[{clause:"tenant_id=42"}]}
        API->>SecMgr: create_guest_access_token(user, rls_clauses)
        SecMgr->>SecMgr: Sign JWT with embedded RLS claims
        SecMgr-->>API: Guest JWT
        API-->>Browser: {token}
        Browser->>API: POST /api/v1/chart/data [Authorization: Bearer guest_token]
        API->>SecMgr: decode_guest_token() → extract rls clauses
        SecMgr-->>API: rls_clauses=["tenant_id=42"]
        API->>SL: build_sqla_query(rls_filters=["tenant_id=42"])
        API->>DB: Execute with tenant filter
        DB-->>API: Tenant-scoped data
        API-->>Browser: Filtered chart data
    end