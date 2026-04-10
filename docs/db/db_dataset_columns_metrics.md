erDiagram
    dbs {
        int id PK
        string database_name
        string sqlalchemy_uri
        text password
        text extra
        bool allow_run_async
        bool allow_dml
        bool allow_file_upload
        bool expose_in_sqllab
        bool allow_ctas
        bool allow_cvas
        bool force_ctas_schema
        int cache_timeout
        string encrypted_extra
        string server_cert
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }

    tables {
        int id PK
        string table_name
        string schema
        string description
        string main_dttm_col
        string default_endpoint
        int database_id FK
        bool is_featured
        bool filter_select_enabled
        bool fetch_values_predicate
        text sql
        text params
        int cache_timeout
        string perm
        string schema_perm
        string catalog
        bool normalize_columns
        bool always_filter_main_dttm
        bool is_managed_externally
        string external_url
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
        int template_params
    }

    table_columns {
        int id PK
        int table_id FK
        string column_name
        string type
        string expression
        string verbose_name
        string description
        bool filterable
        bool groupby
        bool is_active
        bool is_dttm
        bool exported
        string python_date_format
        string extra
        int created_by_fk FK
        int changed_by_fk FK
        datetime created_on
        datetime changed_on
    }

    sql_metrics {
        int id PK
        int table_id FK
        string metric_name
        string verbose_name
        string metric_type
        text expression
        text description
        bool d3format
        bool warning_text
        bool is_restricted
        string extra
        string certification_details
        string certified_by
        int created_by_fk FK
        int changed_by_fk FK
        datetime created_on
        datetime changed_on
    }

    ab_user {
        int id PK
        string first_name
        string last_name
        string username
        string password
        string email
        bool active
        datetime last_login
        int login_count
        int fail_login_count
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }

    dbs ||--o{ tables : "has many datasets"
    tables ||--o{ table_columns : "has columns"
    tables ||--o{ sql_metrics : "has metrics"
    ab_user ||--o{ tables : "created_by / changed_by"
    ab_user ||--o{ dbs : "created_by / changed_by"
    ab_user ||--o{ table_columns : "created_by"
    ab_user ||--o{ sql_metrics : "created_by"