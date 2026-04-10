sequenceDiagram
    autonumber
    actor User
    participant Browser
    participant API as Flask REST API
    participant SecMgr as Security Manager
    participant MetaDB as Metadata DB
    participant SL as Semantic Layer
    participant DB as Data Source

    rect rgb(219,234,254)
        Note over User,DB: Flow A — Create Physical Dataset
        User->>Browser: Data → Datasets → + Dataset
        Browser->>API: GET /api/v1/database/?expose_in_sqllab=true
        API->>MetaDB: SELECT dbs
        MetaDB-->>API: DB list
        API-->>Browser: Databases
        User->>Browser: Select DB → schema → table
        Browser->>API: GET /api/v1/database/{id}/tables/?schema={s}
        API->>DB: SHOW TABLES IN schema
        DB-->>API: Table names
        API-->>Browser: Table list
        User->>Browser: Select table → Create Dataset
        Browser->>API: POST /api/v1/dataset/ {database_id, table_name, schema}
        API->>SecMgr: can("post","Dataset") + database_access check
        SecMgr-->>API: OK
        API->>MetaDB: INSERT INTO tables (table_name, schema, database_id)
        API->>DB: SHOW COLUMNS FROM table (SQLAlchemy inspect)
        DB-->>API: Column list (name, type)
        API->>MetaDB: INSERT INTO table_columns (table_id, column_name, type, is_dttm ...)
        MetaDB-->>API: OK
        API-->>Browser: 201 Created {dataset_id}
    end

    rect rgb(220,252,231)
        Note over User,DB: Flow B — Sync Columns from Source
        User->>Browser: Edit Dataset → Columns → Sync from Source
        Browser->>API: PUT /api/v1/dataset/{id}/refresh
        API->>MetaDB: SELECT tables WHERE id=? (get DB + schema + table)
        API->>DB: SHOW COLUMNS FROM table (fresh inspect)
        DB-->>API: Current column list
        API->>MetaDB: SELECT table_columns WHERE table_id=?
        MetaDB-->>API: Existing columns
        API->>API: Merge (add new, deactivate dropped, keep metadata)
        API->>MetaDB: UPSERT table_columns (merged set)
        MetaDB-->>API: OK
        API-->>Browser: Updated column list
    end

    rect rgb(252,231,243)
        Note over User,DB: Flow C — Create Virtual Dataset (SQL-based)
        User->>Browser: SQL Lab → Run Query → Explore
        Browser->>API: POST /api/v1/dataset/ {sql:"SELECT ...", database_id, table_name}
        API->>SecMgr: can("post","Dataset")
        SecMgr-->>API: OK
        API->>MetaDB: INSERT INTO tables (sql=<virtual SQL>, database_id)
        API->>SL: infer_columns(sql, database_id)
        SL->>DB: Execute LIMIT 0 query (dry run) to infer types
        DB-->>SL: Column metadata
        API->>MetaDB: INSERT INTO table_columns (inferred columns)
        MetaDB-->>API: OK
        API-->>Browser: {dataset_id}
        Browser->>Browser: Open Explore with virtual dataset
    end

    rect rgb(255,237,213)
        Note over User,MetaDB: Flow D — Add Metric to Dataset
        User->>Browser: Edit Dataset → Metrics → + Metric
        Browser->>API: POST /api/v1/dataset/{id}/metric {metric_name, expression, metric_type}
        API->>SecMgr: can("put","Dataset")
        SecMgr-->>API: OK
        API->>MetaDB: INSERT INTO sql_metrics (table_id, metric_name, expression ...)
        MetaDB-->>API: OK
        API-->>Browser: Updated dataset
    end