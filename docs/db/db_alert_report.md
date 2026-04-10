erDiagram
    report_schedule {
        int id PK
        string type
        string name
        string description
        string context_markdown
        bool active
        string crontab
        string timezone
        int sql
        string chart_id FK
        int dashboard_id FK
        int database_id FK
        string last_eval_dttm
        string last_state
        string last_value
        string last_value_row_json
        string validator_type
        string validator_config_json
        string log_retention
        string grace_period
        string working_timeout
        string creation_method
        string force_screenshot
        bool extra
        string report_format
        string recipients
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

    slices {
        int id PK
        string slice_name
        string viz_type
    }

    dashboards {
        int id PK
        string dashboard_title
    }

    ab_user {
        int id PK
        string username
    }

    report_schedule ||--o{ report_recipient : "notifies via"
    report_schedule ||--o{ report_execution_log : "execution history"
    report_schedule ||--o{ report_schedule_user : "owned by users"
    ab_user ||--o{ report_schedule_user : "owns reports"
    report_schedule }o--o| slices : "monitors chart"
    report_schedule }o--o| dashboards : "screenshots dashboard"
    annotation_layer ||--o{ annotation : "contains annotations"
    ab_user ||--o{ annotation : "created_by"
    ab_user ||--o{ annotation_layer : "created_by"