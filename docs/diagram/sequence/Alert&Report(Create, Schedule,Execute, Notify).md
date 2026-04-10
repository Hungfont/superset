sequenceDiagram
    autonumber
    actor Admin
    participant Browser
    participant API as Flask REST API
    participant MetaDB as Metadata DB
    participant Beat as Celery Beat (Scheduler)
    participant Worker as Celery Worker
    participant DB as Data Source
    participant Cache as Redis
    participant Headless as Headless Browser (Playwright)
    participant Channel as Email / Slack

    rect rgb(219,234,254)
        Note over Admin,MetaDB: Phase 1 — Create Alert / Report
        Admin->>Browser: Manage → Alerts & Reports → +
        Browser->>API: POST /api/v1/report/ {name, type, crontab, chart_id|dashboard_id, validator_type, validator_config, recipients}
        API->>MetaDB: INSERT INTO report_schedule
        API->>MetaDB: INSERT INTO report_recipient (report_schedule_id, type, recipient_config_json)
        API->>MetaDB: INSERT INTO report_schedule_user (report_schedule_id, user_id)
        MetaDB-->>API: {id}
        API-->>Browser: 201 Created
    end

    rect rgb(220,252,231)
        Note over Beat,Worker: Phase 2 — Celery Beat Scheduling
        loop Every minute
            Beat->>Beat: Evaluate cron expressions vs current time
            Beat->>Worker: Dispatch scheduled_reports task (report_schedule_id)
        end
    end

    rect rgb(252,231,243)
        Note over Worker,Channel: Phase 3A — Alert Execution (Condition Check)
        Worker->>MetaDB: SELECT report_schedule WHERE id=? (get sql, validator_config)
        MetaDB-->>Worker: Alert config
        Worker->>MetaDB: UPDATE report_schedule SET last_eval_dttm=now
        Worker->>DB: Execute alert SQL query
        DB-->>Worker: Scalar result value
        Worker->>Worker: Evaluate validator (operator: >, <, ==, not null ...)
        alt Condition NOT MET (no alert)
            Worker->>MetaDB: INSERT report_execution_log {state=noop}
            Worker->>MetaDB: UPDATE report_schedule SET last_state=noop
        else Condition MET — within grace period
            Worker->>MetaDB: INSERT report_execution_log {state=grace}
            Note right of Worker: Grace period prevents duplicate alerts
        else Condition MET — outside grace period
            Worker->>MetaDB: INSERT report_execution_log {state=triggered}
            Worker->>Worker: Proceed to notification (Phase 4)
        end
    end

    rect rgb(255,237,213)
        Note over Worker,Channel: Phase 3B — Report Execution (Scheduled Screenshot)
        Worker->>MetaDB: SELECT report_schedule WHERE id=? (get dashboard_id/chart_id)
        Worker->>Headless: render_screenshot(dashboard_url, user_context)
        Headless->>API: GET /superset/dashboard/{id}/?standalone=true (authenticated)
        API->>MetaDB: Load dashboard + charts
        API-->>Headless: Rendered HTML
        Headless->>Headless: Wait for charts to load, take PNG screenshot
        Headless-->>Worker: PNG bytes
        Worker->>Cache: Store screenshot (thumbnail cache)
        Worker->>MetaDB: UPDATE report_execution_log {state=success}
    end

    rect rgb(240,253,244)
        Note over Worker,Channel: Phase 4 — Notification Dispatch
        Worker->>MetaDB: SELECT report_recipient WHERE report_schedule_id=?
        MetaDB-->>Worker: Recipients (email/slack configs)
        loop For each recipient
            alt Email
                Worker->>Channel: SMTP sendmail (PNG attachment + chart link)
            else Slack
                Worker->>Channel: POST /api/chat.postMessage (PNG + text)
            end
            Channel-->>Worker: Delivery receipt
        end
        Worker->>MetaDB: UPDATE report_schedule SET last_state=success
        Worker->>MetaDB: INSERT report_execution_log {state=success, end_dttm}
    end