erDiagram
    query {
        int id PK
        string client_id
        int database_id FK
        int user_id FK
        string status
        string tab_name
        string sql_editor_id
        string schema
        string catalog
        text sql
        text select_sql
        text executed_sql
        int limit
        int limiting_factor
        int select_as_cta
        bool select_as_cta_used
        string progress
        int rows
        int error_message
        string results_key
        datetime start_time
        datetime start_running_time
        datetime end_time
        int end_result_backend_time
        datetime tmp_table_name
        int tracking_url
        bool tmp_schema_name
        string cached_data
        bool is_saved
        string extra_json
        datetime changed_on
    }

    saved_query {
        int id PK
        int db_id FK
        int user_id FK
        string label
        string schema
        string catalog
        text sql
        text description
        string sql_tables
        string extra_json
        bool published
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
        string tags
    }

    tab_state {
        int id PK
        int user_id FK
        int db_id FK
        string schema
        string catalog
        string label
        bool active
        text sql
        string query_limit
        int latest_query_id FK
        bool hide_left_bar
        bool saved_query_id FK
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
        string extra_json
    }

    table_schema {
        int id PK
        int tab_state_id FK
        int db_id FK
        string schema
        string catalog
        string table
        text description
        bool expanded
        datetime created_on
        datetime changed_on
    }

    dbs {
        int id PK
        string database_name
        string sqlalchemy_uri
    }

    ab_user {
        int id PK
        string username
        string email
    }

    dbs ||--o{ query : "queries executed on"
    ab_user ||--o{ query : "run by user"
    dbs ||--o{ saved_query : "queries belong to db"
    ab_user ||--o{ saved_query : "saved by user"
    ab_user ||--o{ tab_state : "tab owned by user"
    dbs ||--o{ tab_state : "tab connected to db"
    saved_query ||--o| tab_state : "linked saved query"
    query ||--o| tab_state : "latest query in tab"
    tab_state ||--o{ table_schema : "expanded schemas"
    dbs ||--o{ table_schema : "schema from db"