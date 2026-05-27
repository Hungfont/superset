-- ============================================================================
-- Superset Database Constraints & Entity Relationships
-- Generated from GORM models in backend/internal/domain/*
-- PostgreSQL syntax
-- ============================================================================

-- ============================================================================
-- 1. UNIQUE CONSTRAINTS
-- ============================================================================

-- Identity / RBAC
ALTER TABLE ab_user              ADD CONSTRAINT uq_ab_user_username       UNIQUE (username);
ALTER TABLE ab_user              ADD CONSTRAINT uq_ab_user_email          UNIQUE (email);
ALTER TABLE ab_register_user     ADD CONSTRAINT uq_ab_register_username   UNIQUE (username);
ALTER TABLE ab_register_user     ADD CONSTRAINT uq_ab_register_email      UNIQUE (email);
ALTER TABLE ab_register_user     ADD CONSTRAINT uq_ab_register_hash       UNIQUE (registration_hash);
ALTER TABLE ab_role              ADD CONSTRAINT uq_ab_role_name           UNIQUE (name);
ALTER TABLE ab_permission        ADD CONSTRAINT uq_ab_permission_name     UNIQUE (name);
ALTER TABLE ab_view_menu         ADD CONSTRAINT uq_ab_view_menu_name      UNIQUE (name);
ALTER TABLE ab_permission_view   ADD CONSTRAINT uq_ab_perm_view           UNIQUE (permission_id, view_menu_id);

-- Data sources
ALTER TABLE dbs                  ADD CONSTRAINT uq_dbs_name               UNIQUE (database_name);
ALTER TABLE tables               ADD CONSTRAINT uq_tables_db_table        UNIQUE (database_id, table_name);
ALTER TABLE table_columns        ADD CONSTRAINT uq_cols_table_col         UNIQUE (table_id, column_name);
ALTER TABLE sql_metrics          ADD CONSTRAINT uq_metrics_tbl_name       UNIQUE (table_id, metric_name);

-- Charts & Dashboards
ALTER TABLE dashboards           ADD CONSTRAINT uq_dashboards_slug        UNIQUE (slug);
ALTER TABLE dashboard_slices     ADD CONSTRAINT uq_dash_slice             UNIQUE (dashboard_id, slice_id);

-- Tags
ALTER TABLE tagged_object        ADD CONSTRAINT uq_tag_obj                UNIQUE (tag_id, object_id, object_type);

-- RLS
ALTER TABLE row_level_security_filters ADD CONSTRAINT uq_rls_name         UNIQUE (name);

-- Embedded / KeyValue
ALTER TABLE key_value            ADD CONSTRAINT uq_key_value_uuid         UNIQUE (uuid);
ALTER TABLE embedded_dashboards  ADD CONSTRAINT uq_embedded_uuid          UNIQUE (uuid);

-- User preferences
ALTER TABLE user_attribute       ADD CONSTRAINT uq_user_attr_user_id      UNIQUE (user_id);


-- ============================================================================
-- 2. NOT NULL CONSTRAINTS (for columns that should never be null)
-- ============================================================================

-- Identity / RBAC
ALTER TABLE ab_user              ALTER COLUMN first_name     SET NOT NULL;
ALTER TABLE ab_user              ALTER COLUMN last_name      SET NOT NULL;
ALTER TABLE ab_user              ALTER COLUMN username       SET NOT NULL;
ALTER TABLE ab_user              ALTER COLUMN email          SET NOT NULL;
ALTER TABLE ab_user              ALTER COLUMN password       SET NOT NULL;
ALTER TABLE ab_register_user     ALTER COLUMN first_name     SET NOT NULL;
ALTER TABLE ab_register_user     ALTER COLUMN last_name      SET NOT NULL;
ALTER TABLE ab_register_user     ALTER COLUMN username       SET NOT NULL;
ALTER TABLE ab_register_user     ALTER COLUMN email          SET NOT NULL;
ALTER TABLE ab_register_user     ALTER COLUMN password       SET NOT NULL;
ALTER TABLE ab_register_user     ALTER COLUMN registration_hash SET NOT NULL;
ALTER TABLE ab_role              ALTER COLUMN name           SET NOT NULL;
ALTER TABLE ab_permission        ALTER COLUMN name           SET NOT NULL;
ALTER TABLE ab_view_menu         ALTER COLUMN name           SET NOT NULL;
ALTER TABLE ab_permission_view   ALTER COLUMN permission_id  SET NOT NULL;
ALTER TABLE ab_permission_view   ALTER COLUMN view_menu_id   SET NOT NULL;
ALTER TABLE ab_permission_view_role ALTER COLUMN permission_view_id SET NOT NULL;
ALTER TABLE ab_permission_view_role ALTER COLUMN role_id     SET NOT NULL;
ALTER TABLE ab_user_role         ALTER COLUMN user_id        SET NOT NULL;
ALTER TABLE ab_user_role         ALTER COLUMN role_id        SET NOT NULL;

-- Data sources
ALTER TABLE dbs                   ALTER COLUMN database_name SET NOT NULL;
ALTER TABLE dbs                   ALTER COLUMN sqlalchemy_uri SET NOT NULL;
ALTER TABLE tables                ALTER COLUMN table_name    SET NOT NULL;
ALTER TABLE tables                ALTER COLUMN database_id  SET NOT NULL;
ALTER TABLE table_columns         ALTER COLUMN table_id     SET NOT NULL;
ALTER TABLE table_columns         ALTER COLUMN column_name  SET NOT NULL;
ALTER TABLE sql_metrics           ALTER COLUMN table_id     SET NOT NULL;
ALTER TABLE sql_metrics           ALTER COLUMN metric_name  SET NOT NULL;
ALTER TABLE sql_metrics           ALTER COLUMN metric_type  SET NOT NULL;
ALTER TABLE sql_metrics           ALTER COLUMN expression   SET NOT NULL;

-- Charts & Dashboards
ALTER TABLE slices                ALTER COLUMN slice_name   SET NOT NULL;
ALTER TABLE slices                ALTER COLUMN viz_type     SET NOT NULL;
ALTER TABLE slices                ALTER COLUMN datasource_id SET NOT NULL;
ALTER TABLE slices                ALTER COLUMN datasource_type SET NOT NULL;
ALTER TABLE slices                ALTER COLUMN datasource_name SET NOT NULL;
ALTER TABLE slices                ALTER COLUMN perm         SET NOT NULL;
ALTER TABLE dashboards            ALTER COLUMN dashboard_title SET NOT NULL;
ALTER TABLE dashboard_slices      ALTER COLUMN dashboard_id SET NOT NULL;
ALTER TABLE dashboard_slices      ALTER COLUMN slice_id     SET NOT NULL;
ALTER TABLE dashboard_user        ALTER COLUMN dashboard_id SET NOT NULL;
ALTER TABLE dashboard_user        ALTER COLUMN user_id      SET NOT NULL;
ALTER TABLE slice_user            ALTER COLUMN slice_id     SET NOT NULL;
ALTER TABLE slice_user            ALTER COLUMN user_id      SET NOT NULL;
ALTER TABLE dashboard_roles       ALTER COLUMN dashboard_id SET NOT NULL;
ALTER TABLE dashboard_roles       ALTER COLUMN role_id      SET NOT NULL;

-- SQL Lab
ALTER TABLE saved_query           ALTER COLUMN label        SET NOT NULL;
ALTER TABLE tab_state             ALTER COLUMN user_id      SET NOT NULL;
ALTER TABLE tab_state             ALTER COLUMN db_id        SET NOT NULL;
ALTER TABLE table_schema          ALTER COLUMN tab_state_id SET NOT NULL;
ALTER TABLE table_schema          ALTER COLUMN db_id        SET NOT NULL;
ALTER TABLE table_schema          ALTER COLUMN "table"      SET NOT NULL;

-- Alerts & Reports
ALTER TABLE report_schedule       ALTER COLUMN type         SET NOT NULL;
ALTER TABLE report_schedule       ALTER COLUMN name         SET NOT NULL;
ALTER TABLE report_schedule       ALTER COLUMN crontab      SET NOT NULL;
ALTER TABLE report_recipient      ALTER COLUMN report_schedule_id SET NOT NULL;
ALTER TABLE report_recipient      ALTER COLUMN type         SET NOT NULL;
ALTER TABLE report_execution_log  ALTER COLUMN report_schedule_id SET NOT NULL;
ALTER TABLE report_execution_log  ALTER COLUMN state        SET NOT NULL;
ALTER TABLE report_schedule_user  ALTER COLUMN report_schedule_id SET NOT NULL;
ALTER TABLE report_schedule_user  ALTER COLUMN user_id      SET NOT NULL;

-- Annotations
ALTER TABLE annotation_layer      ALTER COLUMN name         SET NOT NULL;
ALTER TABLE annotation            ALTER COLUMN layer_id     SET NOT NULL;
ALTER TABLE annotation            ALTER COLUMN short_descr  SET NOT NULL;
ALTER TABLE annotation            ALTER COLUMN start_dttm   SET NOT NULL;
ALTER TABLE annotation            ALTER COLUMN end_dttm     SET NOT NULL;

-- RLS
ALTER TABLE row_level_security_filters ALTER COLUMN name        SET NOT NULL;
ALTER TABLE row_level_security_filters ALTER COLUMN filter_type SET NOT NULL;
ALTER TABLE row_level_security_filters ALTER COLUMN clause      SET NOT NULL;
ALTER TABLE rls_filter_roles       ALTER COLUMN rls_id       SET NOT NULL;
ALTER TABLE rls_filter_roles       ALTER COLUMN role_id      SET NOT NULL;
ALTER TABLE rls_filter_tables      ALTER COLUMN rls_id       SET NOT NULL;
ALTER TABLE rls_filter_tables      ALTER COLUMN datasource_id SET NOT NULL;
ALTER TABLE rls_filter_tables      ALTER COLUMN datasource_type SET NOT NULL;
ALTER TABLE rls_filter_tables      ALTER COLUMN table_name   SET NOT NULL;
ALTER TABLE rls_filter_tables      ALTER COLUMN database_name SET NOT NULL;
ALTER TABLE rls_audit_log          ALTER COLUMN rls_id       SET NOT NULL;
ALTER TABLE rls_audit_log          ALTER COLUMN rls_name     SET NOT NULL;
ALTER TABLE rls_audit_log          ALTER COLUMN event_type   SET NOT NULL;
ALTER TABLE rls_audit_log          ALTER COLUMN changed_by   SET NOT NULL;

-- Tags
ALTER TABLE tag                    ALTER COLUMN name         SET NOT NULL;
ALTER TABLE tag                    ALTER COLUMN type         SET NOT NULL;
ALTER TABLE tagged_object           ALTER COLUMN tag_id      SET NOT NULL;
ALTER TABLE tagged_object           ALTER COLUMN object_id   SET NOT NULL;
ALTER TABLE tagged_object           ALTER COLUMN object_type SET NOT NULL;

-- Logs
ALTER TABLE logs                   ALTER COLUMN action       SET NOT NULL;
ALTER TABLE logs                   ALTER COLUMN dttm         SET NOT NULL;

-- Embedded / Misc
ALTER TABLE css_templates          ALTER COLUMN template_name SET NOT NULL;
ALTER TABLE key_value              ALTER COLUMN resource     SET NOT NULL;
ALTER TABLE embedded_dashboards    ALTER COLUMN uuid         SET NOT NULL;
ALTER TABLE embedded_dashboards    ALTER COLUMN dashboard_id SET NOT NULL;
ALTER TABLE user_attribute         ALTER COLUMN user_id      SET NOT NULL;


-- ============================================================================
-- 3. FOREIGN KEY CONSTRAINTS (Entity Relationships)
-- ============================================================================

-- ── Identity & RBAC ─────────────────────────────────────────────────────────

-- ab_user self-references
ALTER TABLE ab_user
    ADD CONSTRAINT fk_ab_user_created_by  FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_ab_user_changed_by  FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

ALTER TABLE ab_user_role
    ADD CONSTRAINT fk_user_role_user  FOREIGN KEY (user_id) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_user_role_role  FOREIGN KEY (role_id) REFERENCES ab_role(id);

ALTER TABLE ab_permission_view
    ADD CONSTRAINT fk_perm_view_permission FOREIGN KEY (permission_id) REFERENCES ab_permission(id),
    ADD CONSTRAINT fk_perm_view_viewmenu   FOREIGN KEY (view_menu_id)   REFERENCES ab_view_menu(id);

ALTER TABLE ab_permission_view_role
    ADD CONSTRAINT fk_pv_role_perm_view FOREIGN KEY (permission_view_id) REFERENCES ab_permission_view(id),
    ADD CONSTRAINT fk_pv_role_role     FOREIGN KEY (role_id)            REFERENCES ab_role(id);

-- ── Databases ───────────────────────────────────────────────────────────────

ALTER TABLE dbs
    ADD CONSTRAINT fk_dbs_created_by  FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_dbs_changed_by  FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

-- ── Datasets (tables) ────────────────────────────────────────────────────────

ALTER TABLE tables
    ADD CONSTRAINT fk_tables_database    FOREIGN KEY (database_id)   REFERENCES dbs(id),
    ADD CONSTRAINT fk_tables_created_by  FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_tables_changed_by  FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

ALTER TABLE table_columns
    ADD CONSTRAINT fk_columns_table       FOREIGN KEY (table_id)      REFERENCES tables(id),
    ADD CONSTRAINT fk_columns_created_by  FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_columns_changed_by  FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

ALTER TABLE sql_metrics
    ADD CONSTRAINT fk_metrics_table       FOREIGN KEY (table_id)      REFERENCES tables(id),
    ADD CONSTRAINT fk_metrics_created_by  FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_metrics_changed_by  FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

-- ── Charts & Dashboards ─────────────────────────────────────────────────────

ALTER TABLE slices
    ADD CONSTRAINT fk_slices_created_by    FOREIGN KEY (created_by_fk)  REFERENCES ab_user(id),
    ADD CONSTRAINT fk_slices_changed_by    FOREIGN KEY (changed_by_fk)  REFERENCES ab_user(id),
    ADD CONSTRAINT fk_slices_last_saved_by FOREIGN KEY (last_saved_by_fk) REFERENCES ab_user(id);

ALTER TABLE dashboards
    ADD CONSTRAINT fk_dashboards_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_dashboards_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

ALTER TABLE dashboard_slices
    ADD CONSTRAINT fk_dash_slice_dashboard FOREIGN KEY (dashboard_id) REFERENCES dashboards(id),
    ADD CONSTRAINT fk_dash_slice_slice     FOREIGN KEY (slice_id)     REFERENCES slices(id);

ALTER TABLE dashboard_user
    ADD CONSTRAINT fk_dash_user_dashboard FOREIGN KEY (dashboard_id) REFERENCES dashboards(id),
    ADD CONSTRAINT fk_dash_user_user      FOREIGN KEY (user_id)      REFERENCES ab_user(id);

ALTER TABLE slice_user
    ADD CONSTRAINT fk_slice_user_slice FOREIGN KEY (slice_id) REFERENCES slices(id),
    ADD CONSTRAINT fk_slice_user_user  FOREIGN KEY (user_id)  REFERENCES ab_user(id);

ALTER TABLE dashboard_roles
    ADD CONSTRAINT fk_dash_role_dashboard FOREIGN KEY (dashboard_id) REFERENCES dashboards(id),
    ADD CONSTRAINT fk_dash_role_role      FOREIGN KEY (role_id)      REFERENCES ab_role(id);

-- ── SQL Lab ─────────────────────────────────────────────────────────────────

ALTER TABLE query
    ADD CONSTRAINT fk_query_database FOREIGN KEY (database_id) REFERENCES dbs(id),
    ADD CONSTRAINT fk_query_user     FOREIGN KEY (user_id)     REFERENCES ab_user(id);

ALTER TABLE saved_query
    ADD CONSTRAINT fk_saved_query_db          FOREIGN KEY (db_id)           REFERENCES dbs(id),
    ADD CONSTRAINT fk_saved_query_user        FOREIGN KEY (user_id)         REFERENCES ab_user(id),
    ADD CONSTRAINT fk_saved_query_created_by  FOREIGN KEY (created_by_fk)   REFERENCES ab_user(id),
    ADD CONSTRAINT fk_saved_query_changed_by  FOREIGN KEY (changed_by_fk)   REFERENCES ab_user(id);

ALTER TABLE tab_state
    ADD CONSTRAINT fk_tab_state_user          FOREIGN KEY (user_id)         REFERENCES ab_user(id),
    ADD CONSTRAINT fk_tab_state_db            FOREIGN KEY (db_id)           REFERENCES dbs(id),
    ADD CONSTRAINT fk_tab_state_latest_query  FOREIGN KEY (latest_query_id) REFERENCES query(id),
    ADD CONSTRAINT fk_tab_state_saved_query   FOREIGN KEY (saved_query_id)  REFERENCES saved_query(id),
    ADD CONSTRAINT fk_tab_state_created_by    FOREIGN KEY (created_by_fk)   REFERENCES ab_user(id),
    ADD CONSTRAINT fk_tab_state_changed_by    FOREIGN KEY (changed_by_fk)   REFERENCES ab_user(id);

ALTER TABLE table_schema
    ADD CONSTRAINT fk_table_schema_tab_state FOREIGN KEY (tab_state_id) REFERENCES tab_state(id),
    ADD CONSTRAINT fk_table_schema_db        FOREIGN KEY (db_id)        REFERENCES dbs(id);

-- SQL-006: upsert target for schema browser expand state
ALTER TABLE table_schema
    ADD CONSTRAINT uq_table_schema_tab_db_schema_table
    UNIQUE (tab_state_id, db_id, schema, "table");

-- ── Alerts & Reports ────────────────────────────────────────────────────────

ALTER TABLE report_schedule
    ADD CONSTRAINT fk_report_schedule_chart     FOREIGN KEY (chart_id)      REFERENCES slices(id),
    ADD CONSTRAINT fk_report_schedule_dashboard FOREIGN KEY (dashboard_id)  REFERENCES dashboards(id),
    ADD CONSTRAINT fk_report_schedule_db        FOREIGN KEY (database_id)   REFERENCES dbs(id),
    ADD CONSTRAINT fk_report_schedule_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_report_schedule_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

ALTER TABLE report_recipient
    ADD CONSTRAINT fk_recipient_schedule   FOREIGN KEY (report_schedule_id) REFERENCES report_schedule(id),
    ADD CONSTRAINT fk_recipient_created_by FOREIGN KEY (created_by_fk)      REFERENCES ab_user(id),
    ADD CONSTRAINT fk_recipient_changed_by FOREIGN KEY (changed_by_fk)      REFERENCES ab_user(id);

ALTER TABLE report_execution_log
    ADD CONSTRAINT fk_exec_log_schedule FOREIGN KEY (report_schedule_id) REFERENCES report_schedule(id);

ALTER TABLE report_schedule_user
    ADD CONSTRAINT fk_report_user_schedule FOREIGN KEY (report_schedule_id) REFERENCES report_schedule(id),
    ADD CONSTRAINT fk_report_user_user     FOREIGN KEY (user_id)            REFERENCES ab_user(id);

-- ── Annotations ─────────────────────────────────────────────────────────────

ALTER TABLE annotation_layer
    ADD CONSTRAINT fk_anno_layer_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_anno_layer_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

ALTER TABLE annotation
    ADD CONSTRAINT fk_annotation_layer      FOREIGN KEY (layer_id)      REFERENCES annotation_layer(id),
    ADD CONSTRAINT fk_annotation_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_annotation_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

-- ── Row Level Security ─────────────────────────────────────────────────────

ALTER TABLE row_level_security_filters
    ADD CONSTRAINT fk_rls_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_rls_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

ALTER TABLE rls_filter_roles
    ADD CONSTRAINT fk_rls_filter_roles_filter FOREIGN KEY (rls_id)  REFERENCES row_level_security_filters(id),
    ADD CONSTRAINT fk_rls_filter_roles_role   FOREIGN KEY (role_id) REFERENCES ab_role(id);

ALTER TABLE rls_filter_tables
    ADD CONSTRAINT fk_rls_filter_tables_filter FOREIGN KEY (rls_id)        REFERENCES row_level_security_filters(id),
    ADD CONSTRAINT fk_rls_filter_tables_table  FOREIGN KEY (datasource_id) REFERENCES tables(id);

ALTER TABLE rls_audit_log
    ADD CONSTRAINT fk_rls_audit_filter    FOREIGN KEY (rls_id)     REFERENCES row_level_security_filters(id),
    ADD CONSTRAINT fk_rls_audit_changed_by FOREIGN KEY (changed_by) REFERENCES ab_user(id);

-- ── Tags ────────────────────────────────────────────────────────────────────

ALTER TABLE tag
    ADD CONSTRAINT fk_tag_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_tag_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

ALTER TABLE tagged_object
    ADD CONSTRAINT fk_tagged_obj_tag FOREIGN KEY (tag_id) REFERENCES tag(id);

-- ── Logs ────────────────────────────────────────────────────────────────────

ALTER TABLE logs
    ADD CONSTRAINT fk_log_user      FOREIGN KEY (user_id)      REFERENCES ab_user(id),
    ADD CONSTRAINT fk_log_dashboard FOREIGN KEY (dashboard_id) REFERENCES dashboards(id),
    ADD CONSTRAINT fk_log_slice     FOREIGN KEY (slice_id)     REFERENCES slices(id);

-- ── CSS Templates ───────────────────────────────────────────────────────────

ALTER TABLE css_templates
    ADD CONSTRAINT fk_css_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_css_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

-- ── Key-Value Store ─────────────────────────────────────────────────────────

ALTER TABLE key_value
    ADD CONSTRAINT fk_key_value_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_key_value_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

-- ── Embedded Dashboards ─────────────────────────────────────────────────────

ALTER TABLE embedded_dashboards
    ADD CONSTRAINT fk_embedded_dashboard  FOREIGN KEY (dashboard_id) REFERENCES dashboards(id),
    ADD CONSTRAINT fk_embedded_created_by FOREIGN KEY (created_by_fk) REFERENCES ab_user(id),
    ADD CONSTRAINT fk_embedded_changed_by FOREIGN KEY (changed_by_fk) REFERENCES ab_user(id);

-- ── User Attributes ─────────────────────────────────────────────────────────

ALTER TABLE user_attribute
    ADD CONSTRAINT fk_user_attr_user FOREIGN KEY (user_id) REFERENCES ab_user(id);


-- ============================================================================
-- 4. CHECK CONSTRAINTS
-- ============================================================================

-- RLS filter_type must be 'Regular' or 'Base'
ALTER TABLE row_level_security_filters
    ADD CONSTRAINT ck_rls_filter_type CHECK (filter_type IN ('Regular', 'Base'));

-- report_schedule.type (common values: 'Alert', 'Report')
ALTER TABLE report_schedule
    ADD CONSTRAINT ck_report_type CHECK (type IN ('Alert', 'Report'));

-- report_schedule.validator_type common values
ALTER TABLE report_schedule
    ADD CONSTRAINT ck_report_validator CHECK (validator_type IN ('', 'not null', 'operator', 'schema'));

-- report_recipient.type (common values: 'Email', 'Slack')
ALTER TABLE report_recipient
    ADD CONSTRAINT ck_recipient_type CHECK (type IN ('Email', 'Slack'));

-- report_execution_log.state values
ALTER TABLE report_execution_log
    ADD CONSTRAINT ck_exec_log_state CHECK (state IN ('success', 'error', 'working', 'grace', 'timeout'));

-- query.status values
ALTER TABLE query
    ADD CONSTRAINT ck_query_status CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled', 'timed_out'));

-- tagged_object.object_type (common domain types)
ALTER TABLE tagged_object
    ADD CONSTRAINT ck_tagged_obj_type CHECK (object_type IN ('dashboard', 'chart', 'dataset', 'saved_query'));

-- rls_audit_log.event_type
ALTER TABLE rls_audit_log
    ADD CONSTRAINT ck_rls_audit_event CHECK (event_type IN ('filter_created', 'filter_updated', 'filter_deleted'));

-- sql_metrics.metric_type
ALTER TABLE sql_metrics
    ADD CONSTRAINT ck_metric_type CHECK (metric_type IN ('SUM', 'COUNT', 'AVG', 'MAX', 'MIN', 'COUNT_DISTINCT', 'CUSTOM'));


-- ============================================================================
-- 5. ADDITIONAL PERFORMANCE INDEXES (columns referenced in queries)
-- ============================================================================

-- Query history lookups
CREATE INDEX IF NOT EXISTS idx_query_status_created ON query(status, created_at);
CREATE INDEX IF NOT EXISTS idx_query_user_created    ON query(user_id, created_at);

-- Saved query search
CREATE INDEX IF NOT EXISTS idx_saved_query_label  ON saved_query(label);
CREATE INDEX IF NOT EXISTS idx_saved_query_user   ON saved_query(user_id);

-- Tag lookups by type
CREATE INDEX IF NOT EXISTS idx_tag_type ON tag(type);

-- Log filtering
CREATE INDEX IF NOT EXISTS idx_log_action_dttm ON logs(action, dttm);

-- Report schedule active lookups
CREATE INDEX IF NOT EXISTS idx_report_schedule_active ON report_schedule(active);

-- RLS audit lookups by date
CREATE INDEX IF NOT EXISTS idx_rls_audit_created ON rls_audit_log(created_on);

-- Annotation time-range lookups
CREATE INDEX IF NOT EXISTS idx_annotation_start ON annotation(start_dttm);
CREATE INDEX IF NOT EXISTS idx_annotation_end   ON annotation(end_dttm);
