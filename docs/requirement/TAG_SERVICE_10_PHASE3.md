**🏷️ Tag & Metadata Service**

Rank #11 · Phase 3 - Enhancement · 5 Requirements · 1 Independent · 4 Dependent

## **Service Overview**

The Tag & Metadata Service handles four distinct cross-cutting concerns: (1) Taxonomic tagging of platform objects (charts, dashboards, datasets, queries) for organization and discovery. (2) Audit logging for compliance and security monitoring. (3) A generic key-value store for UI state persistence (native filter state, explore form state). (4) User attribute/preference management (welcome dashboard, avatar URL).

Most operations are lightweight CRUD with minimal business logic. The frontend surfaces include a Tag Management settings page, an Audit Log viewer (Admin only), and tag input widgets embedded throughout the platform (chart list, dashboard list, dataset list).

Tags are the most user-visible feature here - they appear as colored Badges throughout the app and enable cross-object discovery via the search/filter interfaces.

## **Tech Stack**

| **Layer**     | **Technology / Package**             | **Purpose**                                            |
| ------------- | ------------------------------------ | ------------------------------------------------------ |
| UI Framework  | React 18 + TypeScript                | Type-safe component tree                               |
| Bundler       | Vite 5                               | Fast HMR and build                                     |
| Routing       | React Router v6                      | SPA navigation                                         |
| Server State  | TanStack Query v5                    | API cache, mutations, background refetch               |
| Client State  | Zustand                              | Global UI state                                        |
| Components    | shadcn/ui (Radix UI)                 | ALL components - no custom, no overrides               |
| Forms         | React Hook Form + Zod                | Schema validation, field-level errors                  |
| Data Tables   | TanStack Table v8                    | Sort, filter, paginate                                 |
| Styling       | Tailwind CSS v3                      | Utility-first                                          |
| Icons         | Lucide React                         | Consistent icon set                                    |
| Notifications | shadcn Toaster + useToast            | Toast notifications                                    |
| Charts        | Recharts (for reports/alert history) | Report trend charts                                    |
| Cron UI       | cronstrue (bun)                      | Human-readable cron expression display                 |
| Date/Time     | shadcn Calendar + Popover + date-fns | Date pickers & formatting                              |
| Backend       | Gin + GORM                           | Simple CRUD + audit logging                            |
| Audit Log     | Async goroutine writes               | Non-blocking audit writes for every significant action |
| FE Tags       | Command + Popover (shadcn)           | Tag multi-select with create-on-type                   |

| **Attribute**      | **Detail**                                                                                     |
| ------------------ | ---------------------------------------------------------------------------------------------- |
| Service Name       | Tag & Metadata Service                                                                         |
| Rank               | #11                                                                                            |
| Phase              | Phase 3 - Enhancement                                                                          |
| Backend API Prefix | /api/v1/tags · /api/v1/key-value · /api/v1/logs                                                |
| Frontend Routes    | /settings/tags · /settings/audit-log · (tag widgets embedded in chart/dashboard/dataset pages) |
| Primary DB Tables  | tag, tagged_object, logs, key_value, user_attribute                                            |
| Total Requirements | 5                                                                                              |
| Independent        | 1                                                                                              |
| Dependent          | 4                                                                                              |

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

**✓ INDEPENDENT (1) - no cross-service calls required**

**TAG-001** - **Create and Manage Tags**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                                       |
| ----------------- | ------------ | --------- | ------------- | ------------------------------------------------------------------------------------- |
| **✓ INDEPENDENT** | **P2**       | Phase 3   | tag           | POST /api/v1/tags · GET /api/v1/tags · PUT /api/v1/tags/:id · DELETE /api/v1/tags/:id |

| **⚙️ Backend - Description**
- Tags are org-level named labels with an optional type (type acts as namespace: "owner", "department", "status", "certification", "custom"). Required: name (unique within org+type, max 100 chars). Optional: type (default "custom"), description.
- List: GET /api/v1/tags with filter ?type=X. Response includes usage_count (count of tagged_object rows per tag). Delete guard: if usage_count > 0 → 409. Force delete (Admin only) with ?force=true removes all tagged_object records first.
- Creator or Admin can update/delete. All authenticated users can create and use tags.
**🔄 Request Flow**
1. Create: GORM.Where("name=? AND type=? AND org_id=?").First → 409 if found → GORM.Create.
2. List: GORM.Select("tags.*, (SELECT COUNT(*) FROM tagged_object WHERE tag_id=tags.id) AS usage_count").
3. Delete: GORM.Where("tag_id=?",id).Count → 409 if > 0 && !force. Force: TX delete tagged_object → delete tag.
**⚙️ Go Implementation**
1. GORM.Where("name=? AND type=? AND org_id=?",name,typ,orgID).First → 409
2. GORM.Create(&tag{Name:name,Type:typ,OrgID:orgID,CreatedByFK:uid})
3. List: GORM.Select("tags.*,(SELECT COUNT(*) FROM tagged_object WHERE tag_id=tags.id) AS usage_count")
4. Force delete TX: GORM.Where("tag_id=?",id).Delete(&tagged_object{}); GORM.Delete(&tag{},id) | **✅ Acceptance Criteria**
- POST { name:"Q4 2024", type:"quarter" } → 201 { id, name, type }.
- Duplicate (name+type) in org → 409.
- GET /tags?type=department → filtered list with usage_count.
- DELETE with usage > 0 → 409.
- DELETE ?force=true (Admin) → 204, tagged_objects removed.
**⚠️ Error Responses**
- 409 - Duplicate (name+type) or usage > 0 (without force).
- 403 - Non-Admin using force delete. | **🖥️ Frontend Specification**
**📍 Route & Page**
/settings/tags
**🧩 shadcn/ui Components**
- DataTable - cols: Name, Type (Badge), Description, Usage (count Badge), Created By, Actions
- Button ("+ New Tag") - opens Dialog
- Dialog ("New Tag")
- Input (name) + Select (type: Custom &#124; Owner &#124; Department &#124; Status &#124; Certification) + Textarea (description)
- Button ("Create Tag") - Dialog submit
- DropdownMenu (Actions) - Edit, Delete
- Badge (type, color-coded per type)
- Badge (usage_count) - "Used in 42 objects" with link icon
- Tooltip on usage Badge - "Used in N charts, M dashboards, P datasets"
- AlertDialog - delete confirmation + usage warning
- Checkbox (Admin only, in delete dialog) - "Force delete and untag all objects"
- Input + Search - filter by name/type
- Select (type filter)
**📦 State & TanStack Query**
- useQuery({ queryKey:["tags",{type,q}] })
- useMutation({ mutationFn: api.createTag, onSuccess: ()=>{ queryClient.invalidateQueries(["tags"]); dialog.close() } })
- useMutation({ mutationFn: ({id,force})=>api.deleteTag(id,force), onSuccess: ()=>toast.success("Tag deleted") })
**✨ UX Behaviors**
- Type Badge colors: Custom=gray, Owner=blue, Department=purple, Status=green, Certification=gold.
- Usage count: click → Sheet showing all objects tagged with this tag, grouped by type.
- Delete: if usage > 0 → AlertDialog "This tag is used in 42 places. Removing will untag all of them." with Admin force checkbox.
- Inline type Select uses colored dot + type name pattern matching Badge style.
**🛡️ Client Validation**
- name: z.string().min(1).max(100)
- type: enum ["custom","owner","department","status","certification"]
**🌐 API Calls**
1. useQuery({ queryKey:["tags",{type,q}], queryFn: ()=>fetch("/api/v1/tags?"+qs).then(r=>r.json()) })
2. useMutation({ mutationFn: ({id,force})=>fetch("/api/v1/tags/"+id+(force?"?force=true":""),{method:"DELETE"}) }) |
| --- | --- | --- |


**⚠ DEPENDENT (4) - requires prior services/requirements**

**TAG-002** - **Tag Objects (Attach/Detach Tags to Platform Resources)**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                                                                                                                           |
| --------------- | ------------ | --------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P2**       | Phase 3   | tagged_object | POST /api/v1/tags/:tag_id/objects · DELETE /api/v1/tags/:tag_id/objects/:type/:object_id · GET /api/v1/tagged-objects/:type/:object_id · GET /api/v1/tags/:tag_id/objects |

**⚑ Depends on:** TAG-001 (tag must exist), AUTH-004 (user context)

| **⚙️ Backend - Description**
- Polymorphic tagging. tagged_object: tag_id + object_id + object_type (enum: "chart" &#124; "dashboard" &#124; "dataset" &#124; "query" &#124; "saved_query"). Any user who can read the object can tag it.
- Attach: POST /tags/:tag_id/objects { object_id:5, object_type:"dashboard" } - validate tag exists + user has object read access. Deduplicate: if already tagged → 200 (not 409).
- Detach: DELETE /tags/:tag_id/objects/:type/:object_id → 204.
- Get object tags: GET /tagged-objects/:type/:object_id → [ {id, name, type, color} ] - all tags on this object.
- Get tag objects: GET /tags/:tag_id/objects → paginated objects with this tag, grouped by type.
**🔄 Request Flow**
1. POST: validate tag exists → validate user can read object → GORM.FirstOrCreate(tagged_object) → 200/201.
2. DELETE: GORM.Where("tag_id=? AND object_type=? AND object_id=?").Delete.
3. GET by object: GORM.Where("object_type=? AND object_id=?").Preload("Tag").
**⚙️ Go Implementation**
1. GORM.First(&tag,tagID) → 404 if not found
2. validateObjectAccess(uc,objectType,objectID) // check read perm per type
3. GORM.FirstOrCreate(&tagged_object,tagged_object{TagID:tagID,ObjectType:typ,ObjectID:oid})
4. GET by object: GORM.Where("object_type=? AND object_id=?",typ,oid).Preload("Tag").Find | **✅ Acceptance Criteria**
- POST { object_id:5, object_type:"dashboard" } → 201.
- Double tag same object → 200 (idempotent).
- GET /tagged-objects/dashboard/5 → [ {id, name:"Q4", type:"quarter"} ].
- GET /tags/:id/objects → paginated list by type.
- User without object read access → 403.
**⚠️ Error Responses**
- 403 - No object read access.
- 404 - Tag not found. | **🖥️ Frontend Specification**
**📍 Route & Page**
Embedded in chart/dashboard/dataset detail pages as a "Tags" section
**🧩 shadcn/ui Components**
- - Tag Input Widget (embedded in object pages) -
- Badge × N (existing tags) - each with × remove button
- Command + Popover ("+ Add Tag") - search existing tags, create new inline
- CommandInput - type to search or create new tag
- CommandItem per existing tag - click to attach
- CommandItem ("Create tag '{input}'") - shown when no match, creates new tag then attaches
- Separator in Command - separates "existing tags" from "create new"
- - Tag Filter in List Pages -
- Select ("Filter by Tag") in DataTable toolbar above chart/dashboard/dataset lists
- CommandItem per tag - filter list to tagged objects
- - Tag Settings Link -
- Link "Manage Tags" → /settings/tags
**📦 State & TanStack Query**
- useQuery({ queryKey:["object-tags",type,objectId] }) - current tags on object
- useQuery({ queryKey:["tags"] }) - all tags for Command suggestions
- useMutation({ mutationFn: ({tagId,objectId,objectType})=>api.tagObject(tagId,{object_id:objectId,object_type:objectType}) })
- useMutation({ mutationFn: ({tagId,type,objectId})=>api.untagObject(tagId,type,objectId) })
- useMutation({ mutationFn: api.createTag }) - create-on-type flow
**✨ UX Behaviors**
- Tag input: Command Popover. User types → filters existing tags. If no match → shows "Create tag '{input}'" item.
- Create-on-type: select "Create tag..." → POST /tags → on success: POST /tags/:id/objects → Badge appears.
- Remove tag: × on Badge → DELETE /tags/:id/objects/:type/:objectId → Badge disappears (optimistic).
- Tags shown as colored Badges matching their type color throughout the app.
- Filter by tag in list pages: Select dropdown → re-fetches with ?tag_id=X filter.
**🌐 API Calls**
1. useQuery({ queryKey:["object-tags",type,id], queryFn: ()=>fetch("/api/v1/tagged-objects/"+type+"/"+id).then(r=>r.json()) })
2. useMutation({ mutationFn: ({tagId,data})=>fetch("/api/v1/tags/"+tagId+"/objects",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**TAG-003** - **Audit Log**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**               |
| --------------- | ------------ | --------- | ------------- | ----------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | logs          | GET /api/v1/logs (Admin only) |

**⚑ Depends on:** AUTH-004 (user_id from JWT)

| **⚙️ Backend - Description**
- Write audit log entries asynchronously for significant platform actions: dashboard view, chart save, dataset edit, user login, user role change, database connection create/delete, etc. Async write (goroutine) to avoid blocking request handlers.
- Log fields: action (string, e.g. "dashboard_view"), user_id, dashboard_id (if applicable), slice_id (if applicable), json (additional context as JSON string), duration_ms (how long the action took), referrer (HTTP Referer header), dtm (timestamp).
- GET /api/v1/logs: Admin only. Paginated. Filters: user_id, action, dashboard_id, slice_id, dtm_from/dtm_to. Export: GET /api/v1/logs?format=csv → streaming CSV download.
**🔄 Request Flow**
1. In each handler: go writeAuditLog(AuditEntry{Action,UserID,DashboardID,...}) - non-blocking goroutine.
2. writeAuditLog: GORM.Create(&logs{...}).
3. GET: Admin check → GORM.Where(filters).Order("dtm DESC").Paginate.
**⚙️ Go Implementation**
1. go func(){ GORM.Create(&logs{Action:action,UserID:uid,DashboardID:dashID,JSON:jsonCtx,DtM:time.Now(),DurationMs:dur}) }()
2. GET: RequireRole("Admin") → GORM.Where(filters).Order("dtm DESC").Paginate
3. CSV export: set Content-Type:text/csv → encoding/csv.NewWriter(c.Writer).WriteAll
**🔒 Security**
- Async write: audit log is non-blocking but is best-effort (not 100% guaranteed on crash).
- json field: sanitize before storing - remove passwords, tokens from action context. | **✅ Acceptance Criteria**
- Dashboard view → logs row created within 100ms async.
- GET /logs (Admin) → paginated list.
- GET /logs?user_id=5 → only that user's actions.
- GET /logs?action=dashboard_view → filtered.
- Non-admin GET → 403.
- GET /logs?format=csv → CSV file download.
**⚠️ Error Responses**
- 403 - Non-admin.
- Audit write errors: logged to structured log, not surfaced to API. | **🖥️ Frontend Specification**
**📍 Route & Page**
/settings/audit-log (Admin only)
**🧩 shadcn/ui Components**
- DataTable - cols: Timestamp, User, Action (Badge), Dashboard/Chart, Duration (ms), Details
- Input + Search - search by action or user name
- Select (action filter) - dropdown of common actions
- Select (user filter) - filter by user
- DateRangePicker - filter by dtm range
- Button ("Export CSV", DownloadCloud icon) - GET /logs?format=csv download
- Badge (action, color-coded by category: view=blue, edit=green, delete=red, auth=purple)
- Collapsible row expand - shows json context field formatted as JSON
- Skeleton - loading state
- Alert (info) - "Access to this page is restricted to Admin users"
- Empty state - Shield icon + "No audit log entries match your filters"
**📦 State & TanStack Query**
- useQuery({ queryKey:["audit-logs",filters], queryFn: ()=>api.getAuditLogs(filters) })
- useState: { searchQ, action, userId, dateRange, page }
- useMutation for CSV export: downloadFile("/api/v1/logs?format=csv")
**✨ UX Behaviors**
- Action Badge color categories: view=blue (Eye icon), create=green (Plus), update=amber (Pencil), delete=red (Trash2), auth=purple (Lock).
- Expanded row: JSON context shown with syntax highlighting (Prism or Monaco read-only).
- Duration column: green if  500ms.
- Export CSV: fires download with date-range filename "audit-log-2024-01-01-to-2024-01-31.csv".
- Real-time: refetchInterval:30000 to see recent activity.
- Admin gate: non-admin users see Alert explaining access restriction and are redirected.
**♿ Accessibility**
- DataTable: aria-label="Audit log entries". Sorted by Timestamp descending.
**🌐 API Calls**
1. useQuery({ queryKey:["audit-logs",filters], queryFn: ()=>fetch("/api/v1/logs?"+qs).then(r=>r.json()) })
2. downloadFile("/api/v1/logs?format=csv&"+qs) // streaming CSV download |
| --- | --- | --- |


**TAG-004** - **Key-Value Store (UI State Persistence)**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                |
| --------------- | ------------ | --------- | ------------- | -------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | key_value     | POST /api/v1/key-value · GET /api/v1/key-value/:resource/:uuid |

**⚑ Depends on:** AUTH-004 (user context for created_by)

| **⚙️ Backend - Description**
- Generic key-value store for persisting arbitrary UI state that needs to be shareable via URL. Primary use cases: (1) Dashboard native filter state (DB-007). (2) Explore chart form state (chart config snapshot for sharing). (3) SQL Lab query state (shareable query + DB + schema).
- POST: accept { resource (namespace string, e.g. "filter_state"&#124;"explore"&#124;"sqllab"), value (JSON string, max 100KB) } → generate UUID → store in key_value with expires_on=NOW()+7d → return { uuid }.
- GET: by resource + uuid → validate ExpiresOn > NOW() → return { value }. Expired → 404.
- TTL cleanup: nightly Asynq job deletes key_value records where expires_on  now() → return value.
3. Cleanup: nightly Asynq: GORM.Where("expires_on<?",now()).Delete(&key_value{}).
**⚙️ Go Implementation**
1. uuid:=uuid.New().String()
2. GORM.Create(&key_value{Resource:resource,UUID:uuid,Value:value,CreatedByFK:uid,ExpiresOn:time.Now().Add(7*24*time.Hour)})
3. GET: GORM.Where("resource=? AND uuid=?",resource,uuid).First → if now().After(kv.ExpiresOn): 404
4. Cleanup: asynq periodic "kv:cleanup": GORM.Where("expires_on<?",time.Now()).Delete(&key_value{}) | **✅ Acceptance Criteria**
- POST { resource:"filter_state", value:"{...}" } → 201 { uuid:"abc-123" }.
- GET /key-value/filter_state/abc-123 → { value:"{...}" }.
- Expired UUID → 404.
- UUID shared between users → recipient sees same state (not user-scoped).
- value > 100KB → 413.
**⚠️ Error Responses**
- 404 - Not found or expired.
- 413 - Value > 100KB.
- 422 - Invalid resource name. | **🖥️ Frontend Specification**
**📍 Route & Page**
No dedicated page - used transparently by Dashboard filter sharing + Explore sharing
**🧩 shadcn/ui Components**
- - Explore "Share" Button -
- Button ("Share", Share2 icon) in Explore toolbar
- Popover - shows shareable URL + Copy button
- Input (read-only) - shareable URL with uuid query param
- Button (Copy, ClipboardCopy icon) - copies URL to clipboard
- - SQL Lab "Share" Button -
- Button ("Share Query", Share2 icon) in SQL Lab toolbar
- Same Popover pattern
- Toast - "Link copied to clipboard"
**📦 State & TanStack Query**
- useMutation({ mutationFn: (state)=>api.saveKeyValue({resource:"explore",value:JSON.stringify(state)}) })
- On mount: if URL has ?state=uuid → useQuery(["kv","explore",uuid]) → restore state
- useQuery({ queryKey:["kv",resource,uuid], enabled:!!uuid, queryFn: ()=>fetch("/api/v1/key-value/"+resource+"/"+uuid).then(r=>r.json()) })
**✨ UX Behaviors**
- "Share" Button: saves current state → POST → Popover shows URL with uuid.
- Copy Button: navigator.clipboard.writeText(url) → Toast "Link copied!".
- On URL load: if ?state=uuid → fetch KV → restore chart/filter/query config.
- Expired state URL: Toast "Shared link has expired (7 days). Load default view."
- URL is shareable: any user (or anonymous if allowed) can load the state from the URL.
**🌐 API Calls**
1. useMutation({ mutationFn: ({resource,value})=>fetch("/api/v1/key-value",{method:"POST",body:JSON.stringify({resource,value})}).then(r=>r.json()) })
2. useQuery({ queryKey:["kv",resource,uuid], queryFn: ()=>fetch("/api/v1/key-value/"+resource+"/"+uuid).then(r=>r.json()) }) |
| --- | --- | --- |


**TAG-005** - **User Attributes & Preferences**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**  | **API / Route**                                       |
| --------------- | ------------ | --------- | -------------- | ----------------------------------------------------- |
| **⚠ DEPENDENT** | **P2**       | Phase 3   | user_attribute | GET /api/v1/me/attributes · PUT /api/v1/me/attributes |

**⚑ Depends on:** AUTH-015 (user must exist and be active)

| **⚙️ Backend - Description**
- Store per-user preferences in user_attribute: welcome_dashboard_id (the dashboard shown after login) and avatar_url (profile picture URL stored after upload to object storage).
- GET: GORM.FirstOrCreate(&user_attribute,user_attribute{UserID:uid}) → return record. Auto-creates with defaults if first access.
- PUT: allow updating welcome_dashboard_id (validate dashboard exists + user has access) and avatar_url (URL of uploaded image). Users can only manage their own attributes. Admin can manage any user's attributes via PUT /api/v1/users/:id/attributes.
- Avatar upload (handled in AUTH-015 profile page): multipart upload → resize to 256×256 JPEG → upload to MinIO/S3 → store URL in user_attribute.avatar_url.
**🔄 Request Flow**
1. GET: GORM.FirstOrCreate(&user_attribute{UserID:uid}).
2. PUT: validate welcome_dashboard_id access → GORM.Save(&user_attribute).
**⚙️ Go Implementation**
1. GORM.FirstOrCreate(&ua,user_attribute{UserID:uid,OrgID:orgID})
2. PUT: GORM.First(&dash,welcomeDashID) + visibility check → 422 if inaccessible
3. GORM.Save(&ua) | **✅ Acceptance Criteria**
- GET → { welcome_dashboard_id:null, avatar_url:null } on first access.
- PUT { welcome_dashboard_id:5 } → 200. Dashboard 5 must be accessible to user.
- PUT welcome_dashboard_id for inaccessible dashboard → 422.
- GET /users/:id/attributes (Admin) → that user's attributes.
- Attribute auto-created on first GET (no 404 on first access).
**⚠️ Error Responses**
- 422 - Inaccessible welcome_dashboard_id.
- 403 - Non-admin accessing other user. | **🖥️ Frontend Specification**
**📍 Route & Page**
/settings/profile (Preferences tab)
**🧩 shadcn/ui Components**
- Tabs [Profile &#124; Security &#124; Preferences] - in /settings/profile
- Card ("Preferences") in Preferences tab
- - Welcome Dashboard -
- Label + Command + Popover - searchable dashboard picker
- CommandInput - search dashboards
- CommandItem per dashboard - Title + Published Badge
- CommandItem ("None - show home page") - clear selection
- Badge (selected dashboard name) - shows current selection
- - Avatar -
- Avatar (64×64) - current profile picture or initials fallback
- Button ("Change Photo") - triggers hidden Input[type=file accept="image/*"]
- CropDialog (shadcn Dialog + react-image-crop) - crop to 256×256 square
- Button ("Save Photo") - uploads cropped image via multipart POST
- Button ("Remove Photo") - sets avatar_url to null
- - Save -
- Button ("Save Preferences") - PUT /me/attributes
- Toast - "Preferences saved"
**📦 State & TanStack Query**
- useQuery({ queryKey:["my-attributes"], queryFn: ()=>fetch("/api/v1/me/attributes").then(r=>r.json()) })
- useMutation({ mutationFn: api.updateAttributes, onSuccess: ()=>toast.success("Preferences saved") })
- useQuery({ queryKey:["dashboards",{published:true}] }) - for welcome dashboard picker
- useState: { cropImage:null, croppedBlob:null } - avatar crop state
**✨ UX Behaviors**
- Welcome Dashboard: Command Popover. Type to search. Clear with "None" option.
- After login: if welcome_dashboard_id set → navigate to that dashboard instead of /home.
- Avatar crop: select image → Dialog with crop UI → "Save Photo" uploads cropped JPEG.
- Avatar preview: updates optimistically before upload completes.
- "Remove Photo" → avatar_url=null → Avatar shows initials fallback.
**🛡️ Client Validation**
- Avatar file: max 5MB, image/* MIME types only.
- welcome_dashboard_id: must be a dashboard user has access to (validated server-side + client pre-filters Command list).
**🌐 API Calls**
1. useQuery({ queryKey:["my-attributes"], queryFn: ()=>fetch("/api/v1/me/attributes").then(r=>r.json()) })
2. useMutation({ mutationFn: (data)=>fetch("/api/v1/me/attributes",{method:"PUT",body:JSON.stringify(data)}).then(r=>r.json()) })
3. Avatar upload: useMutation({ mutationFn: (formData)=>fetch("/api/v1/me/avatar",{method:"POST",body:formData}) }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID**  | **Name**                                               | **Priority** | **Dep**       | **FE Route**                                                                         | **Endpoint(s)**                                                                                                                                                           | **Phase** |
| ------- | ------------------------------------------------------ | ------------ | ------------- | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| TAG-001 | Create and Manage Tags                                 | P2           | ✓ INDEPENDENT | /settings/tags                                                                       | POST /api/v1/tags · GET /api/v1/tags · PUT /api/v1/tags/:id · DELETE /api/v1/tags/:id                                                                                     | Phase 3   |
| TAG-002 | Tag Objects (Attach/Detach Tags to Platform Resources) | P2           | ⚠ DEPENDENT   | Embedded in chart/dashboard/dataset detail pages as a "Tags" section                 | POST /api/v1/tags/:tag_id/objects · DELETE /api/v1/tags/:tag_id/objects/:type/:object_id · GET /api/v1/tagged-objects/:type/:object_id · GET /api/v1/tags/:tag_id/objects | Phase 3   |
| TAG-003 | Audit Log                                              | P1           | ⚠ DEPENDENT   | /settings/audit-log (Admin only)                                                     | GET /api/v1/logs (Admin only)                                                                                                                                             | Phase 2   |
| TAG-004 | Key-Value Store (UI State Persistence)                 | P1           | ⚠ DEPENDENT   | No dedicated page - used transparently by Dashboard filter sharing + Explore sharing | POST /api/v1/key-value · GET /api/v1/key-value/:resource/:uuid                                                                                                            | Phase 2   |
| TAG-005 | User Attributes & Preferences                          | P2           | ⚠ DEPENDENT   | /settings/profile (Preferences tab)                                                  | GET /api/v1/me/attributes · PUT /api/v1/me/attributes                                                                                                                     | Phase 3   |