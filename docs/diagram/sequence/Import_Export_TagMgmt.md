sequenceDiagram
    autonumber
    actor Admin
    participant Browser
    participant API as Flask REST API
    participant SecMgr as Security Manager
    participant MetaDB as Metadata DB
    participant FileStore as ZIP Bundle

    rect rgb(219,234,254)
        Note over Admin,FileStore: Flow A — Export Dashboard Bundle
        Admin->>Browser: Dashboard → Export
        Browser->>API: GET /api/v1/dashboard/{id}/export/
        API->>SecMgr: can("export","Dashboard")
        SecMgr-->>API: Permitted
        API->>MetaDB: SELECT dashboards WHERE id=?
        API->>MetaDB: SELECT slices JOIN dashboard_slices WHERE dashboard_id=?
        API->>MetaDB: SELECT tables JOIN slices (datasources)
        API->>MetaDB: SELECT table_columns, sql_metrics WHERE table_id IN (...)
        API->>MetaDB: SELECT dbs WHERE id IN (...)
        MetaDB-->>API: All related entities
        API->>FileStore: Serialize to YAML per entity type
        Note right of FileStore: Structure:\n/dashboards/my_dash.yaml\n/charts/my_chart.yaml\n/datasets/my_table.yaml\n/databases/my_db.yaml
        FileStore-->>API: ZIP bytes
        API-->>Browser: dashboard_export_{timestamp}.zip (download)
    end

    rect rgb(220,252,231)
        Note over Admin,MetaDB: Flow B — Import Dashboard Bundle
        Admin->>Browser: Import → Upload ZIP
        Browser->>API: POST /api/v1/dashboard/import/ {formData: zip_file, passwords: {db_conn_str}}
        API->>SecMgr: can("import","Dashboard")
        SecMgr-->>API: Permitted
        API->>FileStore: Unzip + parse YAML files
        FileStore-->>API: Entity objects (dashboards, charts, datasets, dbs)
        API->>API: Validate schema versions + checksums
        loop For each database
            API->>MetaDB: UPSERT dbs (match by name or uuid)
        end
        loop For each dataset
            API->>MetaDB: UPSERT tables + table_columns + sql_metrics
        end
        loop For each chart
            API->>MetaDB: UPSERT slices (match by uuid)
        end
        loop For each dashboard
            API->>MetaDB: UPSERT dashboards + dashboard_slices
        end
        MetaDB-->>API: All upserted
        API-->>Browser: 200 OK — Import complete
    end

    rect rgb(252,231,243)
        Note over Admin,MetaDB: Flow C — Tag Asset
        Admin->>Browser: Chart detail → Tags → Add tag
        Browser->>API: POST /api/v1/tag/ {name, type:"custom"}
        API->>MetaDB: INSERT INTO tag {name, type}
        MetaDB-->>API: {tag_id}
        Browser->>API: POST /api/v1/tag/{tag_id}/objects/ {object_type:"chart", object_id}
        API->>MetaDB: INSERT INTO tagged_object {tag_id, object_type, object_id}
        MetaDB-->>API: OK
        API-->>Browser: Tagged

        Note over Admin,MetaDB: Search by tag
        Browser->>API: GET /api/v1/chart/?q=(filters:!((col:tags,opr:ChartTagsFilter,val:my_tag)))
        API->>MetaDB: SELECT slices JOIN tagged_object JOIN tag WHERE tag.name=?
        MetaDB-->>API: Filtered chart list
        API-->>Browser: Charts with that tag
    end

    rect rgb(255,237,213)
        Note over Admin,MetaDB: Flow D — CSS Template Apply
        Admin->>Browser: Settings → CSS Templates → + New
        Browser->>API: POST /api/v1/css_template/ {template_name, css}
        API->>MetaDB: INSERT INTO css_templates
        MetaDB-->>API: {id}
        Admin->>Browser: Dashboard Edit → Apply CSS Template
        Browser->>API: PUT /api/v1/dashboard/{id} {css: "<template CSS>"}
        API->>MetaDB: UPDATE dashboards SET css=?
        MetaDB-->>API: OK
        API-->>Browser: Dashboard updated
    end