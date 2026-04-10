graph TB
    subgraph CLIENT["🖥️ Client Layer"]
        BR[Browser / Web UI\nReact + TypeScript]
        SDK[Embedded SDK\nsuperset-embedded-sdk]
        EXTAPI[External API Consumers\nREST Clients]
    end

    subgraph FRONTEND["⚛️ Frontend — superset-frontend"]
        direction TB
        EXPLORE[Explore / Chart Builder\nNo-code viz editor]
        SQLAB[SQL Lab\nAdvanced SQL Editor]
        DASH[Dashboard Engine\nDrag-&-Drop Layout]
        VIZ[Viz Plugin System\nECharts / D3 / Custom]
    end

    subgraph BACKEND["🐍 Backend — Flask / Python"]
        direction TB
        WSGI[Gunicorn / WSGI Server]
        subgraph API["REST API Layer (Flask-AppBuilder)"]
            CHARTAPI[Charts API]
            DASHAPI[Dashboards API]
            DSAPI[Datasets API]
            SECAPI[Security & Auth API]
            SQLAPI[SQL Lab API]
        end
        subgraph CORE["Core Services"]
            SL[Semantic Layer\nVirtual Datasets & Metrics]
            QC[Query Context\nQuery Builder]
            CACHE_SVC[Cache Service]
            SEC[Security Manager\nRBAC / OAuth / LDAP]
        end
        subgraph DBENG["Database Engine Layer"]
            SQLA[SQLAlchemy ORM]
            DBAPI[DB-API 2.0 Drivers]
        end
    end

    subgraph ASYNC["⚙️ Async Workers — Celery"]
        BEAT[Celery Beat\nScheduler]
        WORKER[Celery Workers\nAsync Query Executor]
        ALERTS[Alerts & Reports\nEmail / Slack]
        THUMB[Thumbnail Generator]
    end

    subgraph INFRA["🗄️ Infrastructure"]
        METADB[(Metadata DB\nPostgreSQL / MySQL)]
        CACHE[(Cache / Broker\nRedis)]
        WS[WebSocket Server\nsuperset-websocket]
    end

    subgraph DATASOURCES["📦 Data Sources"]
        DW[(Data Warehouses\nSnowflake / BigQuery / Redshift)]
        OLAP[(OLAP Engines\nDruid / Pinot / Trino)]
        RDBMS[(RDBMS\nPostgres / MySQL / MSSQL)]
        OTHER[(Others\nCSV / Google Sheets…)]
    end

    BR --> FRONTEND
    SDK --> WSGI
    EXTAPI --> WSGI

    FRONTEND --> WSGI
    WSGI --> API
    API --> CORE
    CORE --> DBENG
    CORE --> SEC
    CORE --> CACHE_SVC

    CACHE_SVC --> CACHE
    QC --> SQLA
    SQLA --> DBAPI
    DBAPI --> DATASOURCES

    API --> METADB
    SQLA --> METADB

    CACHE --> WORKER
    BEAT --> WORKER
    WORKER --> ALERTS
    WORKER --> THUMB
    WORKER --> CACHE
    WORKER --> DBAPI

    WS --> CACHE
    WSGI --> WS

    style CLIENT fill:#dbeafe,stroke:#3b82f6
    style FRONTEND fill:#dcfce7,stroke:#22c55e
    style BACKEND fill:#fef9c3,stroke:#eab308
    style ASYNC fill:#fce7f3,stroke:#ec4899
    style INFRA fill:#f3e8ff,stroke:#a855f7
    style DATASOURCES fill:#ffedd5,stroke:#f97316