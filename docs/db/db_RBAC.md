erDiagram
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
        string password
        string email
        string registration_hash
        datetime registration_date
        bool hashed_password
    }

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

    tables {
        int id PK
        string table_name
        string perm
        string schema_perm
    }

    ab_user ||--o{ ab_user_role : "assigned roles"
    ab_role ||--o{ ab_user_role : "has users"
    ab_role ||--o{ ab_permission_view_role : "granted permissions"
    ab_permission_view ||--o{ ab_permission_view_role : "granted to roles"
    ab_permission ||--o{ ab_permission_view : "applied to views"
    ab_view_menu ||--o{ ab_permission_view : "has permissions"
    row_level_security_filters ||--o{ rls_filter_roles : "applied to roles"
    ab_role ||--o{ rls_filter_roles : "has RLS filters"
    row_level_security_filters ||--o{ rls_filter_tables : "applied to tables"
    tables ||--o{ rls_filter_tables : "filtered by RLS"
    ab_user ||--o{ row_level_security_filters : "created_by"