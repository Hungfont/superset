sequenceDiagram
    autonumber

    %% ─── Auth / Login ───
    rect rgb(219,234,254)
        Note over User,MetaDB: Auth Flow — Login (OAuth / DB / LDAP)
        actor User
        participant Browser as Browser
        participant API as Flask API + FAB
        participant SecMgr as Security Manager
        participant IdP as Identity Provider (OAuth/LDAP)
        participant MetaDB as Metadata DB

        User->>Browser: POST /login (credentials)
        Browser->>API: Auth request
        API->>SecMgr: authenticate()
        alt DB Auth
            SecMgr->>MetaDB: Verify hashed password
            MetaDB-->>SecMgr: User record
        else OAuth / LDAP
            SecMgr->>IdP: Redirect / token exchange
            IdP-->>SecMgr: Token + profile
            SecMgr->>MetaDB: Upsert user & roles
        end
        SecMgr-->>API: Session token (JWT / server session)
        API-->>Browser: Set-Cookie / Bearer token
    end

    %% ─── Alerts & Reports ───
    rect rgb(252,231,243)
        Note over Beat,Channel: Alerts & Reports — Scheduled Execution
        participant Beat as Celery Beat
        participant Worker as Celery Worker
        participant Cache as Redis
        participant DB as Data Source
        participant Thumb as Headless Browser (screenshot)
        participant Channel as Email / Slack

        Beat->>Worker: Trigger scheduled alert/report
        Worker->>Cache: Check prior execution state
        Worker->>DB: Execute alert SQL condition
        DB-->>Worker: Result rows
        alt Condition MET (alert threshold crossed)
            Worker->>Thumb: Render dashboard screenshot
            Thumb-->>Worker: PNG attachment
            Worker->>Channel: Send notification (email/Slack)
        else Condition NOT MET
            Worker->>Cache: Update last_run timestamp
        end
    end

    %% ─── Row-Level Security ───
    rect rgb(220,252,231)
        Note over Browser,DB: Row-Level Security (RLS) Query Path
        participant Browser as Browser
        participant API as Flask API
        participant SecMgr as Security Manager
        participant SL as Semantic Layer
        participant DB as Data Source

        Browser->>API: POST /api/v1/chart/data
        API->>SecMgr: get_rls_filters(user, dataset)
        SecMgr-->>API: RLS WHERE clauses
        API->>SL: Build SQL + inject RLS filters
        SL-->>API: Final SQL with row filters
        API->>DB: Execute filtered SQL
        DB-->>API: Scoped result set
        API-->>Browser: Data (user sees only permitted rows)
    end