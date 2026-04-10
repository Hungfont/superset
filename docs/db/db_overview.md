erDiagram

    %% ═══════════════════════════════════════════
    %% IDENTITY & RBAC
    %% ═══════════════════════════════════════════
    ab_user {
        int id PK
        string username
        string email
        string first_name
        string last_name
        string password
        bool active
        datetime last_login
        int login_count
        int fail_login_count
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    ab_role {
        int id PK
        string name
    }
    ab_user_role {
        int id PK
        int user_id FK
        int role_id FK
    }
    ab_permission {
        int id PK
        string name
    }
    ab_view_menu {
        int id PK
        string name
    }
    ab_permission_view {
        int id PK
        int permission_id FK
        int view_menu_id FK
    }
    ab_permission_view_role {
        int id PK
        int permission_view_id FK
        int role_id FK
    }
    ab_register_user {
        int id PK
        string first_name
        string last_name
        string username
        string email
        string registration_hash
        datetime registration_date
    }

    %% ═══════════════════════════════════════════
    %% DATA SOURCES
    %% ═══════════════════════════════════════════
    dbs {
        int id PK
        string database_name
        string sqlalchemy_uri
        text password
        text extra
        text encrypted_extra
        string server_cert
        bool allow_run_async
        bool allow_dml
        bool allow_file_upload
        bool expose_in_sqllab
        bool allow_ctas
        bool allow_cvas
        int cache_timeout
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    tables {
        int id PK
        string table_name
        string schema
        string catalog
        string description
        string main_dttm_col
        int database_id FK
        bool is_featured
        bool filter_select_enabled
        text sql
        text params
        int cache_timeout
        string perm
        string schema_perm
        bool normalize_columns
        bool is_managed_externally
        string external_url
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
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
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    sql_metrics {
        int id PK
        int table_id FK
        string metric_name
        string verbose_name
        string metric_type
        text expression
        text description
        string d3format
        string warning_text
        bool is_restricted
        string extra
        string certified_by
        string certification_details
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }

    %% ═══════════════════════════════════════════
    %% CHARTS & DASHBOARDS
    %% ═══════════════════════════════════════════
    slices {
        int id PK
        string slice_name
        string viz_type
        string datasource_id
        string datasource_type
        string datasource_name
        text params
        text query_context
        string description
        int cache_timeout
        string perm
        string schema_perm
        string certified_by
        string certification_details
        bool is_managed_externally
        string external_url
        datetime last_saved_at
        int last_saved_by_fk FK
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    dashboards {
        int id PK
        string dashboard_title
        text position_json
        text css
        string description
        string slug
        text json_metadata
        bool published
        bool is_managed_externally
        string external_url
        string certified_by
        string certification_details
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    dashboard_slices {
        int id PK
        int dashboard_id FK
        int slice_id FK
    }
    dashboard_user {
        int id PK
        int dashboard_id FK
        int user_id FK
    }
    slice_user {
        int id PK
        int slice_id FK
        int user_id FK
    }
    dashboard_roles {
        int id PK
        int dashboard_id FK
        int role_id FK
    }

    %% ═══════════════════════════════════════════
    %% SQL LAB
    %% ═══════════════════════════════════════════
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
        text executed_sql
        int limit
        int rows
        string error_message
        string results_key
        datetime start_time
        datetime start_running_time
        datetime end_time
        string tmp_table_name
        string tracking_url
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
        int query_limit
        int latest_query_id FK
        bool hide_left_bar
        int saved_query_id FK
        string extra_json
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
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

    %% ═══════════════════════════════════════════
    %% ALERTS & REPORTS
    %% ═══════════════════════════════════════════
    report_schedule {
        int id PK
        string type
        string name
        string description
        bool active
        string crontab
        string timezone
        text sql
        int chart_id FK
        int dashboard_id FK
        int database_id FK
        string last_eval_dttm
        string last_state
        string validator_type
        string validator_config_json
        int log_retention
        int grace_period
        int working_timeout
        string creation_method
        bool force_screenshot
        string report_format
        string extra
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    report_recipient {
        int id PK
        int report_schedule_id FK
        string type
        text recipient_config_json
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    report_execution_log {
        int id PK
        int report_schedule_id FK
        string state
        string value
        string value_row_json
        string error_message
        string uuid
        datetime start_dttm
        datetime end_dttm
        datetime scheduled_dttm
    }
    report_schedule_user {
        int id PK
        int report_schedule_id FK
        int user_id FK
    }

    %% ═══════════════════════════════════════════
    %% ANNOTATIONS
    %% ═══════════════════════════════════════════
    annotation_layer {
        int id PK
        string name
        string descr
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    annotation {
        int id PK
        int layer_id FK
        string short_descr
        text long_descr
        datetime start_dttm
        datetime end_dttm
        string json_metadata
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }

    %% ═══════════════════════════════════════════
    %% ROW LEVEL SECURITY
    %% ═══════════════════════════════════════════
    row_level_security_filters {
        int id PK
        string name
        string description
        string filter_type
        string group_key
        text clause
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    rls_filter_roles {
        int id PK
        int rls_filter_id FK
        int role_id FK
    }
    rls_filter_tables {
        int id PK
        int rls_filter_id FK
        int table_id FK
    }

    %% ═══════════════════════════════════════════
    %% TAGS, LOGS, CSS, KEY-VALUE, EMBEDDED
    %% ═══════════════════════════════════════════
    tag {
        int id PK
        string name
        string type
        string description
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    tagged_object {
        int id PK
        int tag_id FK
        int object_id
        string object_type
        datetime created_on
        datetime changed_on
    }
    logs {
        int id PK
        string action
        int user_id FK
        int dashboard_id
        int slice_id
        string json
        int duration_ms
        string referrer
        datetime dtm
    }
    css_templates {
        int id PK
        string template_name
        text css
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    key_value {
        int id PK
        string resource
        string uuid
        text value
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
        datetime expires_on
    }
    embedded_dashboards {
        int id PK
        string uuid
        int dashboard_id FK
        string allowed_domains
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
    }
    user_attribute {
        int id PK
        int user_id FK
        string welcome_dashboard_id
        string avatar_url
        datetime created_on
        datetime changed_on
    }

    %% ═══════════════════════════════════════════
    %% RELATIONSHIPS
    %% ═══════════════════════════════════════════

    %% Identity
    ab_user ||--o{ ab_user_role : "assigned to"
    ab_role ||--o{ ab_user_role : "has"
    ab_role ||--o{ ab_permission_view_role : "grants"
    ab_permission_view ||--o{ ab_permission_view_role : "granted via"
    ab_permission ||--o{ ab_permission_view : "on"
    ab_view_menu ||--o{ ab_permission_view : "has"

    %% Data Sources
    dbs ||--o{ tables : "hosts"
    tables ||--o{ table_columns : "has columns"
    tables ||--o{ sql_metrics : "has metrics"
    ab_user ||--o{ dbs : "created_by"
    ab_user ||--o{ tables : "created_by"
    ab_user ||--o{ table_columns : "created_by"
    ab_user ||--o{ sql_metrics : "created_by"

    %% Charts & Dashboards
    tables ||--o{ slices : "datasource for"
    slices ||--o{ dashboard_slices : "in"
    dashboards ||--o{ dashboard_slices : "contains"
    dashboards ||--o{ dashboard_user : "owned by"
    ab_user ||--o{ dashboard_user : "owns"
    slices ||--o{ slice_user : "owned by"
    ab_user ||--o{ slice_user : "owns"
    dashboards ||--o{ dashboard_roles : "accessible by"
    ab_role ||--o{ dashboard_roles : "accesses"
    ab_user ||--o{ slices : "created_by"
    ab_user ||--o{ dashboards : "created_by"

    %% SQL Lab
    dbs ||--o{ query : "executed on"
    ab_user ||--o{ query : "run by"
    dbs ||--o{ saved_query : "belongs to"
    ab_user ||--o{ saved_query : "saved by"
    ab_user ||--o{ tab_state : "owns tab"
    dbs ||--o{ tab_state : "connected to"
    tab_state ||--o{ table_schema : "shows"
    dbs ||--o{ table_schema : "from"
    query ||--o| tab_state : "latest query"
    saved_query ||--o| tab_state : "linked query"

    %% Alerts & Reports
    report_schedule }o--o| slices : "monitors"
    report_schedule }o--o| dashboards : "screenshots"
    report_schedule }o--o| dbs : "alert db"
    report_schedule ||--o{ report_recipient : "notifies"
    report_schedule ||--o{ report_execution_log : "logs"
    report_schedule ||--o{ report_schedule_user : "owned by"
    ab_user ||--o{ report_schedule_user : "owns"
    ab_user ||--o{ report_schedule : "created_by"

    %% Annotations
    annotation_layer ||--o{ annotation : "contains"
    ab_user ||--o{ annotation_layer : "created_by"
    ab_user ||--o{ annotation : "created_by"

    %% RLS
    row_level_security_filters ||--o{ rls_filter_roles : "applied to"
    ab_role ||--o{ rls_filter_roles : "has filter"
    row_level_security_filters ||--o{ rls_filter_tables : "filters"
    tables ||--o{ rls_filter_tables : "filtered by"
    ab_user ||--o{ row_level_security_filters : "created_by"

    %% Tags, Logs, Misc
    tag ||--o{ tagged_object : "labels"
    ab_user ||--o{ logs : "actor"
    ab_user ||--o{ css_templates : "created_by"
    ab_user ||--o{ key_value : "created_by"
    dashboards ||--o| embedded_dashboards : "embedded as"
    ab_user ||--o{ embedded_dashboards : "created_by"
    ab_user ||--o{ user_attribute : "has prefs"