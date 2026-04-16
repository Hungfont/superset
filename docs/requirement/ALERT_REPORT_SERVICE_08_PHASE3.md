**🔔 Alert & Report Service**

Rank #08 · Phase 3 - Async & Scale · 9 Requirements · 0 Independent · 9 Dependent

## **Service Overview**

The Alert & Report Service handles two closely related features: (1) Reports - scheduled captures of charts or dashboards delivered as PDF/PNG/CSV to recipients via email or Slack. (2) Alerts - scheduled SQL metric evaluations that send notifications when a threshold condition is met.

Both are powered by Asynq cron scheduling. A scheduler process enqueues execution tasks at the correct time per timezone; workers handle screenshot capture, metric evaluation, and notification delivery.

The frontend provides a unified Alerts & Reports management page where users create, configure, and monitor both types. An execution log viewer tracks every run with timing, outcome, and error detail.

## **Tech Stack**

| **Layer**     | **Technology / Package**             | **Purpose**                                    |
| ------------- | ------------------------------------ | ---------------------------------------------- |
| UI Framework  | React 18 + TypeScript                | Type-safe component tree                       |
| Bundler       | Vite 5                               | Fast HMR and build                             |
| Routing       | React Router v6                      | SPA navigation                                 |
| Server State  | TanStack Query v5                    | API cache, mutations, background refetch       |
| Client State  | Zustand                              | Global UI state                                |
| Components    | shadcn/ui (Radix UI)                 | ALL components - no custom, no overrides       |
| Forms         | React Hook Form + Zod                | Schema validation, field-level errors          |
| Data Tables   | TanStack Table v8                    | Sort, filter, paginate                         |
| Styling       | Tailwind CSS v3                      | Utility-first                                  |
| Icons         | Lucide React                         | Consistent icon set                            |
| Notifications | shadcn Toaster + useToast            | Toast notifications                            |
| Charts        | Recharts (for reports/alert history) | Report trend charts                            |
| Cron UI       | cronstrue (bun)                      | Human-readable cron expression display         |
| Date/Time     | shadcn Calendar + Popover + date-fns | Date pickers & formatting                      |
| Backend       | Gin + GORM + Asynq + robfig/cron     | Schedule management & execution pipeline       |
| Screenshots   | chromedp (headless Chrome)           | Dashboard/chart screenshot capture for PDF/PNG |
| Email         | gomail (SMTP)                        | HTML email with attachments                    |
| Slack         | http.Post webhook                    | Slack message delivery                         |
| Scheduler     | Asynq Scheduler + robfig/cron parser | Cron-based job scheduling with timezone        |
| FE Cron       | cronstrue (bun library)              | Convert cron string to human-readable text     |
| FE Charts     | Recharts                             | Execution history trend charts                 |

| **Attribute**      | **Detail**                                                                             |
| ------------------ | -------------------------------------------------------------------------------------- |
| Service Name       | Alert & Report Service                                                                 |
| Rank               | #08                                                                                    |
| Phase              | Phase 3 - Async & Scale                                                                |
| Backend API Prefix | /api/v1/reports · /api/v1/alerts                                                       |
| Frontend Routes    | /alerts-reports · /alerts-reports/new · /alerts-reports/:id · /alerts-reports/:id/logs |
| Primary DB Tables  | report_schedule, report_recipient, report_execution_log, report_schedule_user          |
| Total Requirements | 9                                                                                      |
| Independent        | 0                                                                                      |
| Dependent          | 9                                                                                      |

## **Frontend Stack Notes**

Frontend stack mirrors Apache Superset: React 18 + TypeScript, Vite 5, TanStack Query v5 for all server state, Zustand for global client state, React Router v6.

Component library: shadcn/ui ONLY - no custom components. Use: Button, Input, Form, Select, Dialog, Sheet, Tabs, Table, Toast, DropdownMenu, Command, Popover, Badge, Card, Skeleton, Alert, AlertDialog, Tooltip, ScrollArea, Separator, Switch, Checkbox, RadioGroup, Calendar.

Forms: React Hook Form + Zod. All inputs via shadcn FormField / FormControl / FormMessage.

Data tables: shadcn DataTable pattern with TanStack Table v8. Never raw HTML tables.

Toasts: shadcn Toaster + useToast. success=green, destructive=red, info=default.

Loading: shadcn Skeleton for initial load. Button disabled + Loader2 animate-spin during mutation.

Styling: Tailwind CSS utility classes only. No inline styles, no CSS modules.

Icons: Lucide React exclusively.

API: all calls via TanStack Query useQuery / useMutation. Never raw fetch in components.

Error handling: React Error Boundary at page level. API errors via toast onError in useMutation.

## **Requirements**

**✓ INDEPENDENT (0) - no cross-service calls required**

**⚠ DEPENDENT (9) - requires prior services/requirements**

**AR-001** - **Create Report Schedule**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                         | **API / Route**      |
| --------------- | ------------ | --------- | ------------------------------------- | -------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_schedule, report_schedule_user | POST /api/v1/reports |

**⚑ Depends on:** AUTH-004 (user context), CHT-001 or DB-001 (target must exist and be published)

| **⚙️ Backend - Description**
- Create a cron-based report schedule. Required: name (unique per org), type="Report", crontab (validated via robfig/cron.ParseStandard), timezone (IANA string, e.g. "Asia/Ho_Chi_Minh"). Target: chart_id OR dashboard_id (not both) - must exist and be published. report_format: "PNG" &#124; "PDF" &#124; "CSV".
- Optional: description, force_screenshot (bool), log_retention (days, default 90), grace_period (seconds, default 3600), working_timeout (seconds, default 3600). Set active=true on creation.
- Compute next_run: use robfig/cron.Schedule.Next(time.Now().In(timezone)) and store as display info in response. Create report_schedule_user for owner tracking.
**🔄 Request Flow**
1. robfig/cron.ParseStandard(crontab) → 422 if err.
2. Validate target: GORM.First(chart or dashboard) → check published.
3. GORM.Create(&report_schedule{Active:true}).
4. GORM.Create(&report_schedule_user{ReportScheduleID:id,UserID:uid}).
5. Compute next_run from cron schedule + timezone.
6. Return 201 { id, name, crontab, next_run }.
**⚙️ Go Implementation**
1. sched,err:=robfig_cron.ParseStandard(crontab); if err: return 422
2. loc,_:=time.LoadLocation(timezone)
3. nextRun:=sched.Next(time.Now().In(loc)).Format(time.RFC3339)
4. GORM.Create(&report_schedule{...Active:true})
5. GORM.Create(&report_schedule_user{ReportScheduleID:id,UserID:uid}) | **✅ Acceptance Criteria**
- POST { name:"Weekly KPIs", crontab:"0 8 * * 1", timezone:"Asia/Ho_Chi_Minh", dashboard_id:3, report_format:"PDF" } → 201 { id, next_run:"Mon Jan 15 08:00 ICT" }.
- Invalid crontab → 422 { error:"Invalid crontab: ..." }.
- Both chart_id + dashboard_id → 422.
- Unpublished dashboard → 422.
- report_schedule_user record created.
- next_run correctly computed in timezone.
**⚠️ Error Responses**
- 422 - Invalid crontab, invalid target, unpublished target, or both targets provided. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports/new (wizard, "Report" type tab)
**🧩 shadcn/ui Components**
- Dialog or Page wizard - 4-step: "Type" → "Target" → "Schedule" → "Recipients"
- Tabs ("Report" &#124; "Alert") - type selection at top of wizard
- - Step 1: Target -
- RadioGroup ("Dashboard" &#124; "Chart") - target type
- Command + CommandInput - searchable dashboard/chart picker
- CommandItem per result - thumbnail + name + published Badge
- - Step 2: Schedule -
- Select (report_format: PNG &#124; PDF &#124; CSV)
- Input (crontab expression) - with live preview below
- p tag (cronstrue output) - "Every Monday at 8:00 AM" below crontab Input
- Select (timezone) - searchable timezone list (100+ options)
- Badge ("Next run: Mon Jan 15 08:00 ICT") - computed next run preview
- - Step 3: Options -
- Input (name) - report name
- Textarea (description)
- Input (log_retention) - days, default 90
- Switch (force_screenshot) - always capture screenshot even for CSV
- - Navigation -
- Button ("Next") + Button ("Back") - step navigation
- Button ("Create Report") - final step submit
- Stepper (shadcn Steps pattern using Badge + Separator) - progress indicator
**📦 State & TanStack Query**
- useState: { step:0-3, formData:{} } - wizard state
- useQuery({ queryKey:["dashboards",{published:true}] }) - dashboard picker
- useQuery({ queryKey:["charts",{published:true}] }) - chart picker
- useMutation({ mutationFn: api.createReport, onSuccess: (r)=>navigate("/alerts-reports/"+r.id) })
- cronstrue.toString(crontab, {use24HourTimeFormat:true}) - live cron preview
**✨ UX Behaviors**
- Step 1: select target type (Dashboard/Chart) → Command search list.
- Step 2: crontab Input → live human-readable preview via cronstrue.
- Bad crontab: cronstrue throws → show red Alert "Invalid cron expression".
- Timezone: Select with search (Cmdk inside Select).
- "Next run" Badge updates live as crontab/timezone changes.
- Step validation: each step validates before allowing Next.
- Final step: summary card showing all config before submit.
**🛡️ Client Validation**
- name: z.string().min(1).max(255).
- crontab: custom Zod refine → attempt cronstrue.toString(), throw if error.
- timezone: must be in IANA tz list.
- chart_id or dashboard_id required (at least one).
**🌐 API Calls**
1. useMutation({ mutationFn: (data)=>fetch("/api/v1/reports",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**AR-002** - **Create Alert Schedule**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**   | **API / Route**     |
| --------------- | ------------ | --------- | --------------- | ------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_schedule | POST /api/v1/alerts |

**⚑ Depends on:** AUTH-004 (user context), DBC-001 (database_id for metric SQL), QE-001 (SQL eval at trigger time)

| **⚙️ Backend - Description**
- Create a metric-based alert. Required: name, type="Alert", crontab, timezone, database_id, sql (metric SQL - must be SELECT, single-row, single-column), validator_type="operator", validator_config_json { op: ">" &#124; ">=" &#124; "":true,">=":true,"<":true,"<=":true,"==":true,"!=":true}
5. if !validOps[cfg.Op]: 422
6. GORM.Create(&report_schedule{Type:"Alert",SQL:sql,...}) | **✅ Acceptance Criteria**
- POST { name:"Error Rate Alert", crontab:"*/15 * * * *", database_id:1, sql:"SELECT COUNT(*) FROM errors WHERE created_at > NOW()-INTERVAL 15 MIN", validator_config_json:{"op":">","threshold":50} } → 201.
- Non-SELECT SQL → 422.
- Invalid op → 422 { error:"op must be one of >, >=,  50.
**⚠️ Error Responses**
- 422 - Non-SELECT, semicolon, invalid op, invalid database_id. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports/new ("Alert" type tab)
**🧩 shadcn/ui Components**
- - Step 1: Metric SQL (Alert-specific steps) -
- Select (database) - pick database connection for metric evaluation
- Monaco Editor (SQL mode, mini) - metric SQL query editor
- Button ("Validate SQL") - fires sqlparser check + optional LIMIT 2 test
- Alert (green/red) - validation result: "SQL valid and returns 1 row" or error
- - Step 2: Condition -
- Select (operator: ">", ">=", " 50"
- - Step 3: Schedule (same as AR-001 Step 2) -
- Input (crontab) + cronstrue preview
- Select (timezone)
- Input (grace_period) - "Suppress repeat notifications for N seconds"
- Input (working_timeout) - "Cancel evaluation after N seconds"
- - Step 4: Name + Recipients (same as AR-001) -
**📦 State & TanStack Query**
- useState: { step, sql, database_id, operator, threshold, crontab, timezone, gracePeriod }
- useMutation({ mutationFn: api.validateSQL }) - "Validate SQL" button
- useMutation({ mutationFn: api.createAlert, onSuccess: (r)=>navigate("/alerts-reports/"+r.id) })
- sqlValidated: bool - gates Step 2 access
**✨ UX Behaviors**
- Alert wizard has different Step 1 from Report wizard (SQL editor instead of target picker).
- "Validate SQL" Button → shows Loader2 → result Alert.
- Valid SQL + single-row check → "SQL valid (returns 1 numeric value)" green Alert.
- Condition Preview Card updates live as operator and threshold change.
- grace_period Input helper text: "e.g. 3600 = 1 hour. Alert will fire at most once per grace period."
**🛡️ Client Validation**
- sql: required, non-empty.
- Validate via useMutation before allowing Step 2.
- threshold: z.number() - numeric.
- operator: must be one of the 6 valid operators.
- grace_period: z.number().min(0).max(86400).
**🌐 API Calls**
1. useMutation({ mutationFn: (data)=>fetch("/api/v1/alerts",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**AR-003** - **Manage Report Recipients**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**    | **API / Route**                                                                                                                |
| --------------- | ------------ | --------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_recipient | POST /api/v1/reports/:id/recipients · GET /api/v1/reports/:id/recipients · DELETE /api/v1/reports/:id/recipients/:recipient_id |

**⚑ Depends on:** AR-001/AR-002 (schedule must exist)

| **⚙️ Backend - Description**
- Manage notification recipients. Types: Email (recipient_config_json: { target:"user@company.com" }) and Slack (recipient_config_json: { target:"https://hooks.slack.com/..." }). Validate email format for Email type. Validate Slack URL prefix for Slack type.
- List recipients. Delete: if deleting last recipient, allow but return warning { warning:"No recipients remain. Report will run but send to nobody." }. Only owner or Admin.
**🔄 Request Flow**
1. Ownership check → validate type+config.
2. Email: regexp.MatchString(emailRegex,target).
3. Slack: strings.HasPrefix(target,"https://hooks.slack.com/").
4. GORM.Create(&report_recipient{}) → 201.
5. Delete: count remaining after delete → warn if 0.
**⚙️ Go Implementation**
1. switch req.Type { case "Email": matched,_:=regexp.MatchString(emailRe,target); if !matched: 422
2. case "Slack": if !strings.HasPrefix(target,"https://hooks.slack.com/"): 422 }
3. GORM.Create(&report_recipient{...})
4. On delete: GORM.Where("report_schedule_id=?",id).Count(&remaining) → warn if 0 | **✅ Acceptance Criteria**
- POST { type:"Email", recipient_config_json:{"target":"ceo@company.com"} } → 201.
- Invalid email → 422.
- Invalid Slack URL → 422.
- GET → list of recipients (config_json masked for Slack: show only domain).
- DELETE last recipient → 200 { deleted:true, warning:"No recipients remain" }.
- Non-owner → 403.
**⚠️ Error Responses**
- 403 - Not owner.
- 422 - Invalid email or Slack URL. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports/:id (Recipients section / wizard Step 4)
**🧩 shadcn/ui Components**
- - Recipients Card (in detail page + wizard Step 4) -
- Card ("Notification Recipients")
- Button ("+ Add Recipient") - opens small Dialog
- Dialog - add recipient form
- Select (type: Email &#124; Slack) inside Dialog
- Input (email or Slack URL, conditional based on type) inside Dialog
- Button ("Add") - Dialog submit
- - Recipients list (existing) -
- Badge per recipient - Mail icon + email OR Slack icon + workspace name
- Button (X, size=icon, variant=ghost) - delete recipient per Badge
- Alert (warning, shown if 0 recipients) - "No recipients. Add at least one to receive notifications."
- Separator - visual separation
**📦 State & TanStack Query**
- useQuery({ queryKey:["recipients",scheduleId] })
- useMutation({ mutationFn: api.addRecipient, onSuccess: ()=>{ queryClient.invalidateQueries(["recipients",id]); dialog.close() } })
- useMutation({ mutationFn: api.deleteRecipient, onSuccess: (r)=>{ if(r.warning) toast.warning(r.warning) } })
- useState: { recipientType:"Email"&#124;"Slack", value:"" }
**✨ UX Behaviors**
- Type Select: "Email" shows email Input. "Slack" shows webhook URL Input.
- Slack URL: helper text "Find this in Slack: App settings → Incoming Webhooks".
- Recipients shown as Badge row: Mail icon + "john@co.com" or Slack icon + "hooks.slack.com".
- Delete last → Toast warning "No recipients remain. Report will run silently."
- Inline in wizard Step 4: same UI embedded in wizard Card.
**🛡️ Client Validation**
- Email: z.string().email("Enter a valid email address").
- Slack: z.string().startsWith("https://hooks.slack.com/","Slack URL must start with https://hooks.slack.com/").
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,data})=>fetch("/api/v1/reports/"+id+"/recipients",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) })
2. useMutation({ mutationFn: ({schedId,recId})=>fetch("/api/v1/reports/"+schedId+"/recipients/"+recId,{method:"DELETE"}).then(r=>r.json()) }) |
| --- | --- | --- |


**AR-004** - **Cron Scheduler (Asynq Scheduler)**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                         | **API / Route**                                |
| --------------- | ------------ | --------- | ------------------------------------- | ---------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_schedule, report_execution_log | Internal background process - no HTTP endpoint |

**⚑ Depends on:** AR-001/AR-002 (schedules must exist), Asynq infrastructure running

| **⚙️ Backend - Description**
- Background Asynq Scheduler process: loads all active report_schedule records at startup, registers each as a cron job in the Asynq scheduler with its timezone-aware crontab. Dynamic reload: polls DB every 60 seconds for added/updated/deleted schedules, diffs against in-memory registry, and adds/removes/updates accordingly.
- Before enqueueing: idempotency check - query report_execution_log for existing record with same (report_schedule_id, scheduled_dttm). If exists: skip (prevents duplicate executions on restart). Create execution log record state="working" before enqueue, update to "success"/"error" after worker completes.
**🔄 Request Flow**
1. Startup: GORM.Where("active=true").Find(&schedules) → register each in asynq.Scheduler.
2. Every 60s: re-fetch → diff → add new, remove deleted, update changed crontab.
3. On fire: idempotency check → GORM.Create(log{state:"working",scheduled_dttm}) → asynq.Enqueue.
**⚙️ Go Implementation**
1. scheduler:=asynq.NewScheduler(redisConn,&asynq.SchedulerOpts{})
2. for _,s:=range schedules { scheduler.Register(s.Crontab,asynq.NewTask("report:execute",payload)) }
3. go func(){ t:=time.NewTicker(60*time.Second); for range t.C { reloadSchedules(scheduler) } }()
4. Idempotency: GORM.Where("report_schedule_id=? AND scheduled_dttm=?",id,t).First → skip if found | **✅ Acceptance Criteria**
- "0 8 * * 1" fires Monday 08:00 in configured timezone (±30s).
- New schedule → picked up within 60s.
- Deleted schedule → no longer fires.
- Scheduler restart → no duplicate executions.
- Execution log created for every fire.
**⚠️ Error Responses**
- Internal - scheduler errors logged at ERROR level, do not surface to API. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports/:id (shows "Next Run" and "Last Run" metadata)
**🧩 shadcn/ui Components**
- Card ("Schedule Status") in detail page
- Badge ("Active" green &#124; "Paused" amber &#124; "Error" red) - active state
- p text ("Next run: Monday Jan 20 at 08:00 ICT") - computed next run
- p text ("Last run: 3 hours ago") - last_eval_dttm relative time
- Switch ("Active / Paused") - toggles schedule active state via PUT
- Button ("Run Now") - manual trigger via POST /reports/:id/run
- Tooltip on "Run Now" - "Trigger an immediate execution outside the schedule"
**📦 State & TanStack Query**
- useQuery({ queryKey:["schedule",id] }) - schedule metadata including next_run, last_eval_dttm
- useMutation({ mutationFn: ({id,active})=>api.updateSchedule(id,{active}) }) - toggle active
- useMutation({ mutationFn: (id)=>api.runNow(id), onSuccess: ()=>toast.info("Execution triggered. Check logs for result.") })
**✨ UX Behaviors**
- Switch: "Active → Paused" confirmation AlertDialog: "Pause this report? It will stop sending until reactivated."
- "Run Now" Button: triggers immediate execution → Toast "Manual run triggered. Results appear in logs."
- next_run computed on frontend via cronstrue + timezone-aware calculation.
- Refresh polling: useQuery refetchInterval:30000 to keep next_run + last_eval fresh.
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,active})=>fetch("/api/v1/reports/"+id,{method:"PUT",body:JSON.stringify({active})}).then(r=>r.json()) })
2. useMutation({ mutationFn: (id)=>fetch("/api/v1/reports/"+id+"/run",{method:"POST"}).then(r=>r.json()) }) |
| --- | --- | --- |


**AR-005** - **Screenshot Report Worker**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**        | **API / Route**                                                                   |
| --------------- | ------------ | --------- | -------------------- | --------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_execution_log | Asynq worker - no HTTP endpoint. Guest token: POST /api/v1/dashboards/guest-token |

**⚑ Depends on:** AR-004 (Asynq task enqueued), DB-008 (guest token for browser auth), CHT-001/DB-001 (target must be published)

| **⚙️ Backend - Description**
- Asynq worker for screenshot-based reports (PNG, PDF). Flow: (1) Fetch schedule. (2) Generate guest token for target chart/dashboard. (3) chromedp.NewContext with allocator (reuse headless Chrome). (4) Navigate to target URL with guest token. (5) Wait for chart render completion (.loading-spinner disappears, timeout = working_timeout). (6) Capture: PNG via chromedp.FullScreenshot or PDF via page.PrintToPDF. (7) Pass screenshot bytes to AR-006 (notification). Update execution log state.
- Error handling: if render fails or times out → update log state="error", error_message="Render timeout after Xs". Retry ×2 with 5-minute delay. After max retries → dead-letter queue.
**🔄 Request Flow**
1. Fetch schedule → generate guest token.
2. chromedp.Navigate(url+"?guest_token="+token).
3. chromedp.WaitNotVisible(".loading-spinner",chromedp.ByQuery) → timeout=working_timeout.
4. PNG: chromedp.FullScreenshot(&buf,90). PDF: page.PrintToPDF.
5. Pass buf to AR-006.Notify(schedule,recipients,screenshot).
6. GORM.Update(log{State:"success",EndDttm:now()}).
**⚙️ Go Implementation**
1. allocCtx:=chromedp.NewRemoteAllocator(ctx,chromedpURL) // reuse existing Chrome
2. chromedpCtx,cancel:=chromedp.NewContext(allocCtx)
3. chromedp.Run(chromedpCtx,chromedp.Navigate(url),chromedp.WaitNotVisible(".loading-spinner",chromedp.ByQuery,chromedp.BySearch))
4. var buf []byte; chromedp.Run(chromedpCtx,chromedp.FullScreenshot(&buf,90))
5. asynq task retry: asynq.Retry(2), asynq.ProcessIn(5*time.Minute) | **✅ Acceptance Criteria**
- PNG report: non-empty byte slice received by AR-006.
- PDF report: valid PDF with rendered charts.
- Render timeout → log state="error", error_message="Render timeout after 60s".
- Worker completes → log.end_dttm set.
- Retry ×2 on failure → dead-letter after max retries.
**⚠️ Error Responses**
- Worker log state=error on timeout or Chrome failure.
- Retry ×2 then dead-letter. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports/:id/logs (execution log viewer shows screenshot outcome)
**🧩 shadcn/ui Components**
- Card in log row - "Screenshot Captured" or "Render Failed" status
- Button ("Preview") - if screenshot stored, opens in Dialog/lightbox
- Dialog - shows screenshot preview image
- Badge ("PNG" &#124; "PDF" &#124; "CSV") - report format indicator
- Alert (destructive, in log row) - render error message on failure
**📦 State & TanStack Query**
- Screenshot preview: useQuery({ queryKey:["screenshot",logId] }) - fetches screenshot bytes if stored
**✨ UX Behaviors**
- Log row expands to show: rendered screenshot thumbnail, delivery status, error details.
- Click thumbnail → Dialog with full-size screenshot preview.
**🌐 API Calls**
1. useQuery({ queryKey:["log-screenshot",logId], queryFn: ()=>fetch("/api/v1/reports/logs/"+logId+"/screenshot").then(r=>r.blob()) }) |
| --- | --- | --- |


**AR-006** - **Email and Slack Notification Delivery**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                          | **API / Route**                             |
| --------------- | ------------ | --------- | -------------------------------------- | ------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_recipient, report_execution_log | Internal worker function - no HTTP endpoint |

**⚑ Depends on:** AR-003 (recipients), AR-005 (screenshot bytes for reports), AR-002 (alert metric value)

| **⚙️ Backend - Description**
- Send captured report or alert notification to all configured recipients.
- Email (gomail + SMTP): HTML email with subject "{report_name} - {date}", body with inline preview image (PNG embedded as cid: attachment), and link to live chart/dashboard. Attach full PDF or PNG as file attachment. For CSV reports, attach CSV. SMTP config from environment. For >10MB PDF: attach as download link instead of attachment.
- Slack (http.Post to webhook URL): Block Kit message with report name, date, and image URL (or text summary for alerts). Alert emails: "ALERT: {metric_name} is {value}, exceeding threshold of {threshold}."
- Retry per recipient: on SMTP/Slack failure → retry ×3 exponential backoff (30s, 90s, 270s). Log each attempt in report_execution_log.
**🔄 Request Flow**
1. For each recipient: switch type { case "Email": sendEmail(); case "Slack": sendSlack() }.
2. Email: compose gomail.Message → SMTP Dial → Send.
3. Slack: json.Marshal(blockKit) → http.Post(webhookURL).
4. Retry on error (30s,90s,270s).
5. GORM.Update(log{EndDttm:now()}).
**⚙️ Go Implementation**
1. gomail.NewMessage().SetHeader("Subject",subject).SetBody("text/html",htmlBody)
2. m.Attach(filename,gomail.SetCopyFunc(func(w io.Writer) error{ _,err:=w.Write(screenshotBytes); return err }))
3. gomail.NewDialer(smtpHost,port,user,pass).DialAndSend(m)
4. Slack: http.NewRequest("POST",webhookURL,bytes.NewReader(payload))
5. Retry: for attempt:=0;attempt<3;attempt++{ if err:=send(); err==nil: break; time.Sleep(backoff(attempt)) } | **✅ Acceptance Criteria**
- Email recipient receives email with chart PNG attachment.
- Slack recipient receives formatted Block Kit message.
- SMTP failure → retry ×3, log state=error after max.
- Alert email: "ALERT: Error Count is 250, exceeds threshold of 100."
- PDF >10MB → attached as download link (not direct attachment).
**⚠️ Error Responses**
- Log state=error after max retries. Recipients without valid config skipped with warning. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports/:id/logs (delivery status visible in log rows)
**🧩 shadcn/ui Components**
- Badge per recipient in log row - Mail/Slack icon + "Delivered" green or "Failed" red
- Tooltip on Failed Badge - retry count + last error message
- Collapsible (log row expand) - shows per-recipient delivery details
**📦 State & TanStack Query**
- Log detail from GET /api/v1/reports/:id/logs includes per-recipient delivery status
**✨ UX Behaviors**
- Log row expanded view: table of recipients with delivery status + timestamp per recipient.
**🌐 API Calls**
1. useQuery({ queryKey:["report-logs",id] }) - includes delivery_status per recipient in response |
| --- | --- | --- |


**AR-007** - **Execution Log and Retention**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**        | **API / Route**                                              |
| --------------- | ------------ | --------- | -------------------- | ------------------------------------------------------------ |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_execution_log | GET /api/v1/reports/:id/logs · GET /api/v1/reports/:id/stats |

**⚑ Depends on:** AR-004 (scheduler creates log entries), AR-005/AR-006 (workers update log)

| **⚙️ Backend - Description**
- Every schedule execution writes a log: state (working&#124;success&#124;noop&#124;error), value (metric value for alerts), value_row_json, error_message, uuid, start_dttm, end_dttm, scheduled_dttm.
- Retention: nightly Asynq job "log:prune" deletes entries older than report_schedule.log_retention days per schedule. Paginated GET /logs endpoint. Filter by state. Stats endpoint: { success_count, error_count, avg_duration_ms, last_success_at, last_error_at } from recent 30 days - cached in Redis 5min.
**🔄 Request Flow**
1. GET logs: GORM.Where("report_schedule_id=?",id).Where(stateFilter).Order("scheduled_dttm DESC").Paginate.
2. GET stats: SELECT aggregate query → redis.Set(TTL 5min) → return.
3. Prune (Asynq periodic): DELETE WHERE scheduled_dttm < NOW()-log_retention days per schedule.
**⚙️ Go Implementation**
1. GORM.Where("report_schedule_id=? AND state=?",id,state).Order("scheduled_dttm DESC").Offset(off).Limit(sz)
2. statsKey:="stats:report:"+strconv.Itoa(id)
3. rdb.Get(statsKey) → if miss: aggregate query → rdb.Set(statsKey,result,5*time.Minute)
4. Prune: GORM.Where("report_schedule_id=? AND scheduled_dttm<?",id,cutoff).Delete(&report_execution_log{}) | **✅ Acceptance Criteria**
- GET /logs → paginated list newest first.
- GET /logs?state=error → only error entries.
- GET /stats → { success_count:45, error_count:2, avg_duration_ms:8200 }.
- Stats cached 5min.
- Nightly prune removes old entries per log_retention setting.
- Non-owner → 403.
**⚠️ Error Responses**
- 403 - Non-owner.
- 404 - Schedule not found. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports/:id/logs
**🧩 shadcn/ui Components**
- - Log List (main content) -
- DataTable - cols: Status (Badge), Scheduled At, Duration, Rows/Value, Recipients, Actions
- Badge (state: green=success, red=error, amber=working/noop) - status per row
- Collapsible row expand - shows error_message, executed SQL (for alerts), screenshot thumbnail
- Select (state filter: All &#124; Success &#124; Error &#124; Noop)
- DateRangePicker - filter by scheduled_dttm range
- - Stats Panel (top summary) -
- Card grid (4 stat cards): "Success Rate", "Avg Duration", "Last Success", "Last Error"
- Recharts LineChart - success/error trend over last 30 days
- Badge (success_count, error_count) - highlighted counts
- - Empty State -
- Clock icon + "No executions yet. This report/alert has not run yet."
**📦 State & TanStack Query**
- useQuery({ queryKey:["report-logs",id,{state,dateRange,page}], queryFn: ()=>api.getReportLogs(id,filters), refetchInterval:30000 })
- useQuery({ queryKey:["report-stats",id], queryFn: ()=>api.getReportStats(id) })
- useState: { stateFilter:"all", dateRange:null, page:1 }
**✨ UX Behaviors**
- Stats cards at top: "98% success rate (45/46 runs)". Color: green if >95%, amber if >80%, red if  100 ✓ TRIGGERED").
- Auto-refresh: useQuery refetchInterval:30000 while any log has state="working".
**🌐 API Calls**
1. useQuery({ queryKey:["report-logs",id,filters], queryFn: ()=>fetch("/api/v1/reports/"+id+"/logs?"+qs).then(r=>r.json()), refetchInterval:30000 })
2. useQuery({ queryKey:["report-stats",id], queryFn: ()=>fetch("/api/v1/reports/"+id+"/stats").then(r=>r.json()) }) |
| --- | --- | --- |


**AR-008** - **Alert Evaluation Worker**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                         | **API / Route**                 |
| --------------- | ------------ | --------- | ------------------------------------- | ------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_schedule, report_execution_log | Asynq worker - no HTTP endpoint |

**⚑ Depends on:** AR-004 (Asynq task), QE-001 (metric SQL exec), AR-006 (notification on trigger)

| **⚙️ Backend - Description**
- Worker evaluates alert condition and triggers notifications when threshold exceeded. Flow: (1) Fetch schedule. (2) Grace period check: if last_eval_dttm + grace_period > now() → log state="noop", skip. (3) Execute metric SQL via QE-001 with timeout = working_timeout. (4) Validate single-row single-column result (else log error). (5) Evaluate: compare float64 value against threshold using operator. (6) If condition TRUE: AR-006.Notify. Update last_eval_dttm, last_state. Log execution.
- value_row_json: full result row as JSON stored in log for audit. log.value: the scalar metric value as string ("250").
**🔄 Request Flow**
1. Fetch schedule → grace_period check → if within grace: noop log.
2. QE.Execute(ctx,{DatabaseID,SQL:schedule.SQL,Limit:2,Timeout:working_timeout}).
3. Validate len(result.Data)==1 → if not: log error "Must return exactly 1 row".
4. val:=toFloat64(result.Data[0][0]) → evaluate(val,operator,threshold).
5. if triggered: AR006.Notify(schedule,recipients,val,threshold).
6. GORM.Update(schedule,{LastEvalDttm:now(),LastState:state}).
7. GORM.Create(log{State:state,Value:strconv.FormatFloat(val),ScheduledDttm:scheduled}).
**⚙️ Go Implementation**
1. if sched.LastEvalDttm!=nil && time.Since(*sched.LastEvalDttm)<gracePeriod: createLog("noop"); return
2. result:=QE.Execute(ctx,req)
3. if len(result.Data)!=1: createLog("error","Must return exactly 1 row"); return
4. val,_:=strconv.ParseFloat(fmt.Sprint(result.Data[0][0]),64)
5. triggered:=evaluate(val,cfg.Op,cfg.Threshold)
6. if triggered: AR006.Notify(...) | **✅ Acceptance Criteria**
- COUNT=250 > threshold=100 → notification sent, log state="triggered", value="250".
- COUNT=30 < threshold=100 → no notification, log state="pass".
- Within grace_period → log state="noop", no notification.
- Multi-row result → log state="error" "Must return exactly 1 row".
- Timeout → log state="error" "Query timed out after 60s".
- last_eval_dttm updated after every evaluation.
**⚠️ Error Responses**
- Log state=error on SQL failure, multi-row, or timeout. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports/:id (alert detail page shows current status)
**🧩 shadcn/ui Components**
- Card ("Current Status") - in alert detail page
- Badge ("Triggered" red &#124; "Passing" green &#124; "Error" amber) - last_state display
- p ("Last checked: 15 minutes ago") - relative time from last_eval_dttm
- p ("Last value: 250") - value from last evaluation log
- Gauge chart (Recharts RadialBarChart) - shows current value vs threshold visually
- Alert (destructive) - shown if last_state="error" with error_message
- Badge ("Grace Period Active") - shown if within grace_period window
- Tooltip on Grace Badge - "Alert already triggered. Next notification after {time}."
**📦 State & TanStack Query**
- useQuery({ queryKey:["alert-status",id], refetchInterval:60000 }) - polls last_eval_dttm + value + state
**✨ UX Behaviors**
- Gauge chart: needle points to current value, threshold line marked in red.
- "Triggered": red Badge + pulsing animation. "Passing": green Badge.
- Grace period countdown: Badge with timer "Next notification in 42 min".
- Last 5 values: mini sparkline showing trend of metric value over time.
**🌐 API Calls**
1. useQuery({ queryKey:["alert-detail",id], queryFn: ()=>fetch("/api/v1/alerts/"+id).then(r=>r.json()), refetchInterval:60000 }) |
| --- | --- | --- |


**AR-009** - **List, Edit and Delete Schedules**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**   | **API / Route**                                                            |
| --------------- | ------------ | --------- | --------------- | -------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | report_schedule | GET /api/v1/reports · PUT /api/v1/reports/:id · DELETE /api/v1/reports/:id |

**⚑ Depends on:** AR-001/AR-002 (schedules must exist)

| **⚙️ Backend - Description**
- Paginated list of all report and alert schedules visible to the user (own + Admin sees all). Filters: type (Report&#124;Alert), active (bool), owner. Each item: id, name, type, crontab, timezone, active, last_state, last_eval_dttm, target (chart/dashboard name), recipient_count, next_run.
- Update: allow changing name, crontab, timezone, active, report_format, force_screenshot, log_retention, grace_period, working_timeout. Crontab change → re-register in Asynq scheduler within 60s (via reload goroutine). Delete: hard delete + cascade to execution_logs and recipients. Guard: if type=Alert and last_state="triggered" → 409 (must acknowledge first, or Admin force=true).
**🔄 Request Flow**
1. GET: GORM.Where(owner OR isAdmin).Where(filters).Order("changed_on DESC").Paginate.
2. PUT: validate changed crontab → GORM.Updates → scheduler reloads within 60s.
3. DELETE: GORM.Where("report_schedule_id=?",id).Delete(recipients,logs) → GORM.Delete(schedule).
**⚙️ Go Implementation**
1. GORM.Where("created_by_fk=? OR ?",uid,isAdmin).Where(filters).Paginate
2. recipient_count: SELECT COUNT(*) FROM report_recipient WHERE report_schedule_id=id
3. PUT: GORM.Model(&sched).Updates(fields) // scheduler goroutine re-reads DB every 60s
4. DELETE TX: GORM.Where("report_schedule_id=?",id).Delete(&report_recipient{},&report_execution_log{}); GORM.Delete(&report_schedule{}) | **✅ Acceptance Criteria**
- GET → paginated list with recipient_count + next_run + last_state.
- PUT { crontab:"0 9 * * 1" } → 200. Scheduler picks up change within 60s.
- DELETE → 204. Logs and recipients cascade-deleted.
- Non-owner → 403.
- GET ?type=Alert → only alerts.
- GET ?active=false → paused schedules.
**⚠️ Error Responses**
- 403 - Not owner.
- 422 - Invalid crontab on update.
- 404 - Not found. | **🖥️ Frontend Specification**
**📍 Route & Page**
/alerts-reports (main list page)
**🧩 shadcn/ui Components**
- DataTable - cols: Type (Badge), Name, Target, Schedule (human-readable), Status, Last Run, Recipients, Actions
- Button ("+ New") → opens wizard (AR-001 or AR-002 based on type tab)
- Tabs ("All" &#124; "Reports" &#124; "Alerts") - type filter tabs above table
- Select ("Active" &#124; "Paused" &#124; "All") - active state filter
- Input + Search - filter by name
- Badge (type: Bell=Alert blue / FileText=Report green)
- Badge (status: Active green &#124; Paused amber &#124; Error red)
- Badge (last_state per row: Triggered / Passing / Success / Error)
- Tooltip on "Schedule" cell - cronstrue output + timezone
- DropdownMenu (Actions) - View Logs, Edit, Pause/Resume, Run Now, Delete
- Switch (inline in row) - toggle active/paused directly in table
- Skeleton - 5 loading rows
- Empty state - BellOff icon + "No alerts or reports yet" + CTA
**📦 State & TanStack Query**
- useQuery({ queryKey:["schedules",{type,active,q,page}] })
- useMutation for toggle active: onSuccess → invalidate + toast
- useMutation for delete: AlertDialog confirmation → onSuccess → invalidate + toast
- useMutation for run now: onSuccess → toast "Execution triggered"
**✨ UX Behaviors**
- Type Badge: Bell icon for Alert, FileText icon for Report.
- Inline Switch: toggle active state directly in table row without navigating.
- Switch confirm: AlertDialog "Pause {name}?" before deactivating.
- "Last Run" cell: relative time + state color dot (green/red/amber).
- Schedule cell: cronstrue output ("Every Monday at 8:00 AM ICT") - Tooltip shows raw crontab.
- "Run Now" in DropdownMenu → Toast "Running... check logs for result."
- Bulk actions: select rows → "Pause Selected", "Delete Selected" in DataTable toolbar.
**🌐 API Calls**
1. useQuery({ queryKey:["schedules",filters], queryFn: ()=>fetch("/api/v1/reports?"+qs).then(r=>r.json()) })
2. useMutation({ mutationFn: ({id,active})=>fetch("/api/v1/reports/"+id,{method:"PUT",body:JSON.stringify({active})}).then(r=>r.json()) })
3. useMutation({ mutationFn: (id)=>fetch("/api/v1/reports/"+id,{method:"DELETE"}) }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID** | **Name**                              | **Priority** | **Dep**     | **FE Route**                                                             | **Endpoint(s)**                                                                                                                | **Phase** |
| ------ | ------------------------------------- | ------------ | ----------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------ | --------- |
| AR-001 | Create Report Schedule                | P1           | ⚠ DEPENDENT | /alerts-reports/new (wizard, "Report" type tab)                          | POST /api/v1/reports                                                                                                           | Phase 3   |
| AR-002 | Create Alert Schedule                 | P1           | ⚠ DEPENDENT | /alerts-reports/new ("Alert" type tab)                                   | POST /api/v1/alerts                                                                                                            | Phase 3   |
| AR-003 | Manage Report Recipients              | P1           | ⚠ DEPENDENT | /alerts-reports/:id (Recipients section / wizard Step 4)                 | POST /api/v1/reports/:id/recipients · GET /api/v1/reports/:id/recipients · DELETE /api/v1/reports/:id/recipients/:recipient_id | Phase 3   |
| AR-004 | Cron Scheduler (Asynq Scheduler)      | P1           | ⚠ DEPENDENT | /alerts-reports/:id (shows "Next Run" and "Last Run" metadata)           | Internal background process - no HTTP endpoint                                                                                 | Phase 3   |
| AR-005 | Screenshot Report Worker              | P1           | ⚠ DEPENDENT | /alerts-reports/:id/logs (execution log viewer shows screenshot outcome) | Asynq worker - no HTTP endpoint. Guest token: POST /api/v1/dashboards/guest-token                                              | Phase 3   |
| AR-006 | Email and Slack Notification Delivery | P1           | ⚠ DEPENDENT | /alerts-reports/:id/logs (delivery status visible in log rows)           | Internal worker function - no HTTP endpoint                                                                                    | Phase 3   |
| AR-007 | Execution Log and Retention           | P1           | ⚠ DEPENDENT | /alerts-reports/:id/logs                                                 | GET /api/v1/reports/:id/logs · GET /api/v1/reports/:id/stats                                                                   | Phase 3   |
| AR-008 | Alert Evaluation Worker               | P1           | ⚠ DEPENDENT | /alerts-reports/:id (alert detail page shows current status)             | Asynq worker - no HTTP endpoint                                                                                                | Phase 3   |
| AR-009 | List, Edit and Delete Schedules       | P1           | ⚠ DEPENDENT | /alerts-reports (main list page)                                         | GET /api/v1/reports · PUT /api/v1/reports/:id · DELETE /api/v1/reports/:id                                                     | Phase 3   |