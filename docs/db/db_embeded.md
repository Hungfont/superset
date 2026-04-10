erDiagram
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
        string user_id FK
        string dashboard_id
        string slice_id
        string json
        string duration_ms
        string referrer
        string dtm
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
        datetime changed_on
        int changed_by_fk FK
        datetime created_on
        int created_by_fk FK
        string allowed_domains
    }

    user_attribute {
        int id PK
        int user_id FK
        string welcome_dashboard_id
        string avatar_url
        datetime created_on
        datetime changed_on
    }

    ab_user {
        int id PK
        string username
    }

    dashboards {
        int id PK
        string dashboard_title
    }

    slices {
        int id PK
        string slice_name
    }

    tag ||--o{ tagged_object : "applied to objects"
    ab_user ||--o{ logs : "action logged for user"
    ab_user ||--o{ css_templates : "created_by"
    dashboards ||--o| embedded_dashboards : "embedded config"
    ab_user ||--o{ embedded_dashboards : "created_by"
    ab_user ||--o{ user_attribute : "user preferences"
    ab_user ||--o{ key_value : "created_by"