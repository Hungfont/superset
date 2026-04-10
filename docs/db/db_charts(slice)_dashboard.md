erDiagram
    slices {
        int id PK
        string slice_name
        string datasource_id
        string datasource_type
        string datasource_name
        string viz_type
        text params
        text query_context
        string description
        string cache_timeout
        string perm
        string schema_perm
        string certified_by
        string certification_details
        bool is_managed_externally
        string external_url
        datetime created_on
        datetime changed_on
        int created_by_fk FK
        int changed_by_fk FK
        int last_saved_by_fk FK
        datetime last_saved_at
    }

    dashboards {
        int id PK
        string dashboard_title
        string position_json
        string css
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

    ab_user {
        int id PK
        string username
        string email
        string first_name
        string last_name
    }

    ab_role {
        int id PK
        string name
    }

    tables {
        int id PK
        string table_name
        string schema
    }

    dashboards ||--o{ dashboard_slices : "contains charts"
    slices ||--o{ dashboard_slices : "belongs to dashboards"
    dashboards ||--o{ dashboard_user : "owned by users"
    ab_user ||--o{ dashboard_user : "owns dashboards"
    slices ||--o{ slice_user : "owned by users"
    ab_user ||--o{ slice_user : "owns slices"
    dashboards ||--o{ dashboard_roles : "accessible by roles"
    ab_role ||--o{ dashboard_roles : "has dashboard access"
    ab_user ||--o{ slices : "created_by / changed_by"
    ab_user ||--o{ dashboards : "created_by / changed_by"
    slices }o--|| tables : "datasource (SqlaTable)"