**🗂️ Dashboard Service**

Rank #07 · Phase 2 - Core · 9 Requirements · 0 Independent · 9 Dependent

## **Service Overview**

Dashboards are flexible grid-based collections of charts. The Dashboard Service manages their lifecycle: creation, chart composition, drag-and-drop layout, access control (role-based and owner-based), native filter state, and iframe embedding.

The frontend provides two major surfaces: (1) the Dashboard List page for management, and (2) the Dashboard View/Edit page - a full-featured canvas with @dnd-kit drag-and-drop grid, chart panels, native filter panel, and a real-time publish/edit mode toggle.

Dashboards do not query data directly. On load, the frontend fetches dashboard config then independently requests each chart's data via CHT-006. The Dashboard Service only manages layout configs and metadata.

## **Tech Stack**

| **Layer**         | **Technology / Package**                     | **Purpose**                                           |
| ----------------- | -------------------------------------------- | ----------------------------------------------------- |
| UI Framework      | React 18 + TypeScript                        | Type-safe component tree                              |
| Bundler           | Vite 5                                       | Fast HMR and build                                    |
| Routing           | React Router v6                              | SPA navigation + nested routes                        |
| Server State      | TanStack Query v5                            | API cache, mutations, background refetch              |
| Client State      | Zustand                                      | Global UI state (sidebar, user, theme)                |
| Component Library | shadcn/ui (Radix UI primitives)              | Accessible - ALL components from here, no custom      |
| Forms             | React Hook Form + Zod                        | Schema validation, field-level errors                 |
| Data Tables       | TanStack Table v8                            | Sort, filter, paginate, row selection, virtualization |
| Styling           | Tailwind CSS v3                              | Utility-first, no custom CSS                          |
| Icons             | Lucide React                                 | Consistent icon set                                   |
| API Client        | TanStack Query (fetch)                       | No raw fetch/axios in components                      |
| Notifications     | shadcn Toaster + useToast                    | Success/error/info toasts                             |
| Charts            | Apache ECharts / Recharts (same as Superset) | Chart rendering in Explore view                       |
| DnD               | @dnd-kit/core + @dnd-kit/sortable            | Dashboard grid drag-and-drop                          |
| Layout            | shadcn ResizablePanel                        | Resizable pane layouts (SQL Lab, Explore)             |
| Backend           | Gin + GORM + encoding/json + uuid            | Dashboard CRUD + embed tokens                         |
| DnD               | @dnd-kit/core + @dnd-kit/sortable            | Drag-and-drop chart grid in edit mode                 |
| Layout            | react-grid-layout (Superset pattern)         | Resizable chart panels in dashboard                   |
| Embed             | JWT guest tokens (RS256)                     | Secure iframe embed                                   |
| Cache             | go-redis                                     | Filter state KV store                                 |

| **Attribute**      | **Detail**                                                                                                   |
| ------------------ | ------------------------------------------------------------------------------------------------------------ |
| Service Name       | Dashboard Service                                                                                            |
| Rank               | #07                                                                                                          |
| Phase              | Phase 2 - Core                                                                                               |
| Backend API Prefix | /api/v1/dashboards                                                                                           |
| Frontend Routes    | /dashboards · /dashboards/:idOrSlug · /dashboards/:id/edit · /dashboard/:slug (public view)                  |
| Primary DB Tables  | dashboards, dashboard_slices, dashboard_user, dashboard_roles, embedded_dashboards, key_value, css_templates |
| Total Requirements | 9                                                                                                            |
| Independent        | 0                                                                                                            |
| Dependent          | 9                                                                                                            |

## **Frontend Stack Notes**

Frontend stack mirrors Apache Superset: React 18 + TypeScript, Vite 5, TanStack Query v5 for all server state, Zustand for global client state, React Router v6.

Component library: shadcn/ui ONLY - no custom components. Use: Button, Input, Form, Select, Dialog, Sheet, Tabs, Table, Toast, DropdownMenu, Command, Popover, Badge, Card, Skeleton, Alert, AlertDialog, Tooltip, ScrollArea, Separator, ResizablePanel, Slider.

Forms: React Hook Form + Zod schema. All inputs via shadcn FormField/FormControl/FormMessage for consistent error display.

Data tables: shadcn DataTable + TanStack Table v8. Never raw HTML tables.

Toasts: shadcn Toaster + useToast. Success=green, error=destructive, info=default.

Loading: shadcn Skeleton for initial load. Button loading via disabled + Lucide Loader2 animate-spin. No full-page blocking spinners.

Styling: Tailwind CSS utility classes only. No inline styles, no CSS modules.

Icons: Lucide React exclusively. Semantic: Plus=create, Pencil=edit, Trash2=delete, RefreshCw=sync, LayoutDashboard=dashboard, BarChart2=chart, Play=run.

API: all calls via TanStack Query hooks (useQuery/useMutation). Never raw fetch in components. Hooks in /hooks directory.

Error handling: React Error Boundary at page level. API errors via toast onError in useMutation.

## **Requirements**

**✓ INDEPENDENT (0) - no cross-service calls required**

**⚠ DEPENDENT (9) - requires prior services/requirements**

**DB-001** - **Create Dashboard**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**              | **API / Route**         |
| --------------- | ------------ | --------- | -------------------------- | ----------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | dashboards, dashboard_user | POST /api/v1/dashboards |

**⚑ Depends on:** AUTH-004 (user context)

| **⚙️ Backend - Description**
- Create a new dashboard. Required: dashboard_title (3-255 chars). Optional: description (markdown), slug (auto-generated if omitted: slugify(title)+"-"+rand.Hex(3), unique per org), json_metadata (JSON: color_scheme, cross_filters_enabled, native_filter_configuration, auto_refresh_interval), css, position_json (initial grid layout, default empty).
- published=false on creation (draft). Create dashboard_user record linking creator as owner. Validate json_metadata is valid JSON if provided. Slug uniqueness: auto-increment suffix on conflict.
**🔄 Request Flow**
1. Validate title → slug generate/check → validate json_metadata JSON.
2. GORM.Create(&dashboards{Published:false}).
3. GORM.Create(&dashboard_user{DashboardID:id,UserID:uid}).
4. Return 201.
**⚙️ Go Implementation**
1. if no slug: slug=slugify(title)+"-"+rand.Hex(3)
2. GORM.Where("slug=? AND org_id=?",slug,orgID).First → 409 if conflict (custom slug only)
3. GORM.Create(&dashboards{...Published:false})
4. GORM.Create(&dashboard_user{DashboardID:id,UserID:uid}) | **✅ Acceptance Criteria**
- POST { dashboard_title:"Q4 KPIs" } → 201 { id, slug:"q4-kpis-x7k2", published:false }.
- Custom slug conflict → 409.
- Invalid json_metadata → 422.
- dashboard_user record created.
**⚠️ Error Responses**
- 409 - Slug conflict.
- 422 - Invalid json_metadata. | **🖥️ Frontend Specification**
**📍 Route & Page**
/dashboards (+ Button opens Dialog) and /dashboards/:id/edit (after create)
**🧩 shadcn/ui Components**
- Dialog ("New Dashboard") - lightweight create form
- Form + Input - dashboard_title
- Input (optional) - slug with helper "Leave blank to auto-generate"
- Textarea (optional) - description
- Button ("Create") - submits, then navigates to edit page
- Toast - "Dashboard created. Add charts to get started."
**📦 State & TanStack Query**
- useMutation({ mutationFn: api.createDashboard, onSuccess: (d)=>navigate("/dashboards/"+d.id+"/edit") })
- React Hook Form: { dashboard_title: z.string().min(3).max(255), slug: z.string().regex(/^[a-z0-9-]*$/).optional() }
**✨ UX Behaviors**
- Dialog is minimal - only title required. All other config done in the edit page.
- After create: immediately navigate to /dashboards/:id/edit (edit mode, empty canvas).
- Edit page shows empty state: "Add charts to your dashboard" with a "Add Charts" Button.
- Slug field: real-time preview shows generated slug as user types title.
**🛡️ Client-Side Validation**
- dashboard_title: min 3, max 255 chars.
- slug: lowercase alphanumeric + hyphens only (regex).
**🌐 API Calls**
1. useMutation({ mutationFn: (data)=>fetch("/api/v1/dashboards",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**DB-002** - **Add and Remove Charts from Dashboard**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**    | **API / Route**                                                                                                             |
| --------------- | ------------ | --------- | ---------------- | --------------------------------------------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | dashboard_slices | PUT /api/v1/dashboards/:id/charts · POST /api/v1/dashboards/:id/charts/add · DELETE /api/v1/dashboards/:id/charts/:chart_id |

**⚑ Depends on:** CHT-001 (charts must exist), DB-001 (dashboard must exist), AUTH-011 (owner)

| **⚙️ Backend - Description**
- Manage the dashboard_slices join table. Replace-all (PUT): atomic TX delete existing + bulk insert. Validate all chart_ids exist and caller has read access. Add (POST /add): append without clearing, deduplicate. Remove (DELETE): single chart removal.
- Position_json layout NOT auto-updated - frontend manages layout separately. Added charts appear in a "pending" state until positioned in the grid.
**🔄 Request Flow**
1. Validate ownership → validate all chart IDs exist + accessible.
2. Replace-all TX: DELETE dashboard_slices WHERE dashboard_id=X; bulk INSERT.
3. Return {total, added, removed}.
**⚙️ Go Implementation**
1. Validate: GORM.Where("id IN ? AND org_id=?",ids,orgID).Count → 422 if mismatch
2. TX: GORM.Where("dashboard_id=?",id).Delete(&dashboard_slices{}); CreateInBatches(newRows)
3. Add: GORM.Clauses(OnConflict{DoNothing:true}).CreateInBatches | **✅ Acceptance Criteria**
- PUT { chart_ids:[1,2,3] } → 200 { total:3 }.
- Non-existent chart_id → 422.
- Unauthorized chart → 422.
- POST /add { chart_ids:[4] } → 200 { added:1, skipped:[] }.
- DELETE /charts/4 → 204.
**⚠️ Error Responses**
- 403 - Not owner.
- 422 - Invalid or inaccessible chart IDs. | **🖥️ Frontend Specification**
**📍 Route & Page**
/dashboards/:id/edit ("Add Charts" workflow)
**🧩 shadcn/ui Components**
- Sheet ("Add Charts to Dashboard") - slide-in from right, triggered by toolbar "Add Charts" Button
- Input + Search icon - search charts by name (inside Sheet)
- DataTable (inside Sheet) - columns: Thumbnail, Name, Type, Dataset, checkbox column
- Checkbox column - multi-select charts
- Badge (viz_type, color-coded) - chart type
- Button ("Add Selected Charts") in Sheet footer - calls POST /add
- Toast - "3 charts added to dashboard. Drag them to position."
- Chart tile (unpositioned) - added charts appear in a "tray" at bottom of canvas
**📦 State & TanStack Query**
- useQuery({ queryKey:["charts",{owner:"all"}] }) - chart search inside Sheet
- useState: { selectedChartIds: Set }
- useMutation({ mutationFn: (ids)=>api.addChartsToDashboard(dashId,ids), onSuccess: ()=>{ dashStore.addUnpositionedCharts(ids); sheet.close() } })
- Zustand dashStore: { layout, unpositionedCharts } - tracks pending charts
**✨ UX Behaviors**
- Sheet: searchable chart list with thumbnails. Checkbox multi-select.
- After add: Sheet closes. New charts appear as tiles in a "Components Tray" below the canvas.
- User drags tiles from tray onto the canvas grid to position them.
- Remove from dashboard: hover chart panel → DropdownMenu → "Remove from dashboard" → DELETE /charts/:id.
**🌐 API Calls**
1. useMutation({ mutationFn: (ids)=>fetch("/api/v1/dashboards/"+id+"/charts/add",{method:"POST",body:JSON.stringify({chart_ids:ids})}).then(r=>r.json()) })
2. useMutation({ mutationFn: (chartId)=>fetch("/api/v1/dashboards/"+id+"/charts/"+chartId,{method:"DELETE"}) }) |
| --- | --- | --- |


**DB-003** - **Update Dashboard Layout and Metadata**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**            |
| --------------- | ------------ | --------- | ------------- | -------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | dashboards    | PUT /api/v1/dashboards/:id |

**⚑ Depends on:** DB-001 (dashboard must exist)

| **⚙️ Backend - Description**
- Save layout (position_json - grid component positions), metadata (json_metadata: color_scheme, filters config, auto_refresh), css, description, dashboard_title. position_json validation: valid JSON + all chartId refs exist in dashboard_slices. json_metadata: key-level merge (not full replace). css: max 100KB. Auto-broadcast update event via Redis pub/sub if published.
**🔄 Request Flow**
1. Ownership → validate position_json JSON → validate chartId refs exist.
2. json.Merge(existing.JsonMetadata, incoming).
3. GORM.Model.Updates(fields).
4. if published: rdb.Publish("dashboard:updated:"+id, event).
**⚙️ Go Implementation**
1. json.Unmarshal(positionJSON) → validate structure
2. Extract chartIDs from position_json → GORM.Where("slice_id IN ? AND dashboard_id=?",ids,dashID).Count → 422 if mismatch
3. json.Merge(existing,incoming) per key
4. GORM.Model(&dash).Updates(fields)
5. if dash.Published: rdb.Publish("dashboard:updated:"+id,...) | **✅ Acceptance Criteria**
- PUT { position_json:{...}, css:"body{...}" } → 200.
- Invalid JSON in position_json → 422.
- chartId ref not in dashboard_slices → 422.
- json_metadata merge: existing native_filter_configuration preserved.
- Published dashboard: Redis event published.
**⚠️ Error Responses**
- 403 - Not owner.
- 422 - Invalid JSON or broken chart ref. | **🖥️ Frontend Specification**
**📍 Route & Page**
/dashboards/:id/edit (auto-saved as user drags/resizes)
**🧩 shadcn/ui Components**
- - Dashboard Canvas (edit mode) -
- react-grid-layout GridLayout - main canvas with draggable+resizable chart panels
- Chart panel (card) - each chart: header with name + drag handle + resize handle + panel menu
- DropdownMenu on panel - "View Chart", "Edit Chart", "Refresh", "Remove from Dashboard"
- - Edit Toolbar (top) -
- Button ("Save") - triggers PUT /dashboards/:id with current layout
- Button ("Discard Changes") - reverts local layout to server state
- Toggle ("Edit" / "View") - switch between edit and view mode
- Badge ("● Unsaved") - dirty state indicator
- - Dashboard Properties Slide-out -
- Sheet ("Dashboard Properties") - opened from toolbar gear icon
- Form + Input - dashboard_title
- Textarea - description
- Select - color_scheme (preset palette options)
- Switch - cross_filters_enabled
- Select - auto_refresh_interval (Off / 10s / 30s / 1m / 5m)
- Textarea - css (custom CSS, Monaco Editor mini)
- Button ("Apply") - saves to dashStore locally (PUT on main Save)
**📦 State & TanStack Query**
- Zustand dashStore: { layout, isDirty, charts, metadata }
- react-grid-layout onLayoutChange → dashStore.setLayout(newLayout) + isDirty=true
- useMutation({ mutationFn: api.updateDashboard, onSuccess: ()=>{ dashStore.clearDirty(); toast.success("Dashboard saved") } })
- Auto-save option: useDebounce 2000ms on layout change → PUT
**✨ UX Behaviors**
- Edit mode: charts have drag handles (GripVertical icon), resize handles at corners.
- Drag: @dnd-kit sensors with smooth preview shadow. React-grid-layout handles actual grid.
- Resize: drag corner handle → chart panel resizes → chart re-renders at new dimensions.
- Save: manual "Save" Button collects full layout + metadata → PUT.
- Properties Sheet: live-preview color_scheme change on canvas without saving.
- CSS editor: Monaco mini with CSS syntax highlighting.
- Keyboard shortcut: Ctrl+S saves. Escape exits edit mode (with "Discard?" confirmation if dirty).
**♿ Accessibility**
- Drag handles: aria-label="Drag to reposition {chart_name}".
- Edit/View toggle: role="switch" aria-checked.
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,...data})=>fetch("/api/v1/dashboards/"+id,{method:"PUT",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**DB-004** - **Publish and Unpublish Dashboard**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                    |
| --------------- | ------------ | --------- | ------------- | ---------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | dashboards    | PUT /api/v1/dashboards/:id/publish |

**⚑ Depends on:** DB-001 (dashboard must exist), AUTH-011 (owner)

| **⚙️ Backend - Description**
- Toggle published state. Publish validation: must have ≥1 chart + non-empty position_json + slug set. On unpublish: deactivate active report schedules targeting this dashboard. Emit audit log.
**🔄 Request Flow**
1. Ownership → count dashboard_slices → validate position_json not empty → validate slug.
2. GORM.Update("published",true/false).
3. On unpublish: GORM.Where("dashboard_id=?",id).Update("active",false) on report_schedule.
4. Audit log.
**⚙️ Go Implementation**
1. GORM.Where("dashboard_id=?",id).Count(&n) on dashboard_slices → 422 if 0
2. json.Unmarshal(positionJSON) → check len > 2 → 422 if empty
3. GORM.Model(&dash).Update("published",true)
4. go auditLog(uid,"dashboard_published",id) | **✅ Acceptance Criteria**
- PUT /publish → 200 { published:true, slug:"q4-kpis" }.
- 0 charts → 422.
- Empty layout → 422.
- Unpublish → active reports deactivated.
**⚠️ Error Responses**
- 403 - Not owner.
- 422 - Validation failure. | **🖥️ Frontend Specification**
**📍 Route & Page**
/dashboards/:id/edit (toolbar Publish button)
**🧩 shadcn/ui Components**
- Button ("Publish", variant=default) - primary CTA in edit toolbar
- Button ("Unpublish", variant=outline) - shown when already published
- AlertDialog - "Publish dashboard?" confirmation with checklist
- AlertDialogDescription - shows validation checklist: ✓ Has charts, ✓ Layout saved, ✓ Slug set
- Badge ("Draft") or Badge ("Published", green) - status indicator in toolbar
- Toast - "Dashboard published. Share the link: {url}"
- Input (read-only, with Copy icon) - share URL shown in success Toast/Dialog
**📦 State & TanStack Query**
- useMutation({ mutationFn: ({id,publish})=>api.publishDashboard(id,publish), onSuccess: (d)=>{ dashStore.setPublished(d.published); toast.success(d.published?"Dashboard published!":"Dashboard unpublished") } })
- Pre-check: validate locally (charts.length>0, layout not empty) before API call
**✨ UX Behaviors**
- Publish Button: pre-validates locally → if fails: show checklist Dialog "Complete these steps first".
- After publish: Toast with shareable URL + copy Button.
- Published Badge in toolbar: green "Published" → click → "Unpublish?" AlertDialog.
- "Draft" Badge: amber "Draft" with tooltip "Only you and admins can see this dashboard".
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,publish})=>fetch("/api/v1/dashboards/"+id+"/publish",{method:"PUT",body:JSON.stringify({published:publish})}).then(r=>r.json()) }) |
| --- | --- | --- |


**DB-005** - **List and Get Dashboards**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                               | **API / Route**                                     |
| --------------- | ------------ | --------- | ------------------------------------------- | --------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | dashboards, dashboard_user, dashboard_roles | GET /api/v1/dashboards · GET /api/v1/dashboards/:id |

**⚑ Depends on:** AUTH-011 (RBAC visibility), DB-006 (dashboard_roles for role access)

| **⚙️ Backend - Description**
- Paginated list with visibility rules: published=true AND (no role restriction OR user has matching role) OR owner OR Admin. List response: id, title, slug, published, chart_count, thumbnail_url, owner_name, changed_on, certified_by. Filters: published, owner, q (ILIKE title/description), changed_since. Detail includes: position_json, json_metadata, css, charts list (with viz_type), owners, roles with access, embedded uuid.
- Performance: composite index on (org_id, published, changed_on).
**🔄 Request Flow**
1. Visibility WHERE: (published AND (role match OR no restriction)) OR owner OR Admin.
2. Joins: dashboard_user (owner), ab_user (owner name).
3. Detail: Preload DashboardSlices.Slice, DashboardUsers, DashboardRoles.
**⚙️ Go Implementation**
1. GORM.Where("(d.published=true AND (dr.role_id IN ? OR NOT EXISTS(SELECT 1 FROM dashboard_roles WHERE dashboard_id=d.id))) OR du.user_id=? OR ?",userRoles,uid,isAdmin)
2. GORM.Preload("DashboardSlices.Slice").Preload("DashboardUsers.User").Preload("DashboardRoles.Role").First | **✅ Acceptance Criteria**
- GET → paginated + visibility filtered.
- GET ?published=true → only published.
- GET ?q=q4 → title search.
- GET /:id → full detail with charts + owners + roles.
- Gamma without role access → excluded.
- Admin → sees all including unpublished.
**⚠️ Error Responses**
- 404 - Not found or no access. | **🖥️ Frontend Specification**
**📍 Route & Page**
/dashboards
**🧩 shadcn/ui Components**
- DataTable - cols: Thumbnail, Title, Status (Badge), Charts (count), Owner, Modified, Actions
- Button ("+ Dashboard") - opens create Dialog (DB-001)
- Input + Search - search by title
- Select (Status: All &#124; Published &#124; Draft)
- Select (Owner: All &#124; Mine)
- DropdownMenu (Actions) - View, Edit, Duplicate, Share, Delete
- Badge ("Published"=green / "Draft"=amber) - status per row
- Tooltip on Charts count - hover shows chart names
- Avatar - owner display with AvatarFallback initials
- Skeleton - 4 loading rows with thumbnail placeholder
- Empty state - LayoutDashboard icon + "No dashboards yet" + CTA
- Card Grid mode (toggle) - alternative to DataTable, shows thumbnail cards
**📦 State & TanStack Query**
- useQuery({ queryKey:["dashboards",filters] })
- useState: { viewMode:"table"&#124;"grid", searchQ, status, owner, page }
- useMutation for delete/duplicate
- localStorage.getItem("dashboard_view_mode") → persist user's view preference
**✨ UX Behaviors**
- View toggle: DataTable ↔ Card Grid. Card shows thumbnail + title + status + chart count.
- Card Grid: 3-column responsive grid. Hover card shows "View" + "Edit" action buttons.
- Thumbnail: 240×160px screenshot (if generated), else gradient placeholder with viz type icons.
- "Share" action: copies dashboard URL to clipboard + Toast "Link copied!".
- Duplicate: creates copy → navigate to edit page.
**♿ Accessibility**
- Card grid: role="grid". Card: role="gridcell" with aria-label="{title} dashboard".
**🌐 API Calls**
1. useQuery({ queryKey:["dashboards",{q,published,owner,page}], queryFn: ()=>fetch("/api/v1/dashboards?"+new URLSearchParams(filters)).then(r=>r.json()) }) |
| --- | --- | --- |


**DB-006** - **Role-Based Dashboard Access**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**   | **API / Route**                                                     |
| --------------- | ------------ | --------- | --------------- | ------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | dashboard_roles | PUT /api/v1/dashboards/:id/roles · GET /api/v1/dashboards/:id/roles |

**⚑ Depends on:** AUTH-007 (roles must exist), DB-001 (dashboard exists), AUTH-011 (owner)

| **⚙️ Backend - Description**
- Admin: replace-all role assignments for dashboard access. If role_ids empty → open to all authenticated users. Validate all role_ids exist. Invalidate any dashboard access cache.
**🔄 Request Flow**
1. Admin check → validate role IDs → TX delete+insert → return assigned roles
**⚙️ Go Implementation**
1. isAdmin → 403
2. Validate: GORM.Where("id IN ?",ids).Count → 422
3. TX: GORM.Where("dashboard_id=?",id).Delete(&dashboard_roles{}); CreateInBatches(newRows) | **✅ Acceptance Criteria**
- PUT { role_ids:[2,4] } → 200 { assigned:2, roles:[...] }.
- PUT { role_ids:[] } → open to all.
- Non-admin → 403.
- Invalid role_id → 422.
- Users with role 2 now see dashboard in list.
**⚠️ Error Responses**
- 403 - Non-admin.
- 422 - Invalid role_id. | **🖥️ Frontend Specification**
**📍 Route & Page**
/dashboards/:id/edit (Properties Sheet → "Access" section)
**🧩 shadcn/ui Components**
- Card ("Access Control") in Properties Sheet
- MultiSelect (Command + Popover pattern) - role picker
- Badge × N (removable) - selected roles
- Alert (info) - "Empty = accessible to all authenticated users"
- Alert (warning) - "Restricting access will immediately affect who can see this dashboard"
- Button ("Save Access") - triggers PUT /roles
- Separator
- Section "Owners" - list of dashboard_user records with Avatar+name
**📦 State & TanStack Query**
- useQuery({ queryKey:["dashboard-roles",id] }) - current roles
- useQuery({ queryKey:["roles"] }) - all available roles for MultiSelect
- useState: { selectedRoleIds: number[] }
- useMutation({ mutationFn: (ids)=>api.setDashboardRoles(id,ids) })
**✨ UX Behaviors**
- MultiSelect: Command + Popover. Type to filter roles. Click to select/deselect.
- Empty selection: Alert "Dashboard will be visible to all authenticated users with the can_read Dashboard permission."
- Warning Alert shown when removing roles (making more restrictive).
**🌐 API Calls**
1. useMutation({ mutationFn: (ids)=>fetch("/api/v1/dashboards/"+id+"/roles",{method:"PUT",body:JSON.stringify({role_ids:ids})}).then(r=>r.json()) }) |
| --- | --- | --- |


**DB-007** - **Native Filter State Persistence**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                         | **API / Route**                                                                     |
| --------------- | ------------ | --------- | ------------------------------------- | ----------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | key_value, dashboards (json_metadata) | POST /api/v1/dashboards/:id/filter-state · GET /api/v1/key-value/filter_state/:uuid |

**⚑ Depends on:** DB-001 (dashboard exists), AUTH-004 (user context)

| **⚙️ Backend - Description**
- Persist active native filter selections as a shareable key_value record (resource="filter_state", uuid=new UUID, value=JSON state, expires_on=NOW()+7d). Return UUID stored in URL. GET by UUID restores state. Filter config stored in json_metadata.native_filter_configuration (via DB-003).
**🔄 Request Flow**
1. POST: uuid.New() → GORM.Create(&key_value{Resource:"filter_state",UUID:uuid,...}) → return {uuid}.
2. GET: GORM.Where("resource=? AND uuid=?").First → check ExpiresOn > now() → return value.
**⚙️ Go Implementation**
1. uuid.New().String()
2. GORM.Create(&key_value{Resource:"filter_state",UUID:uuid,Value:jsonState,ExpiresOn:now().Add(7*24*time.Hour)})
3. GET: GORM.Where("resource=? AND uuid=?","filter_state",uuid).First → check ExpiresOn | **✅ Acceptance Criteria**
- POST → 201 { uuid:"abc-123" }.
- GET /key-value/filter_state/abc-123 → { state:{...} }.
- Expired UUID → 404.
- Shared URL → recipient sees same filters.
**⚠️ Error Responses**
- 404 - UUID not found or expired.
- 422 - Invalid state JSON. | **🖥️ Frontend Specification**
**📍 Route & Page**
/dashboard/:slug or /dashboards/:id (dashboard view mode)
**🧩 shadcn/ui Components**
- - Native Filter Panel (left sidebar in dashboard view) -
- Sheet (left side, collapsible) - filter sidebar
- Card per filter - each native filter as a Card
- Select / DateRangePicker / Input / MultiSelect - filter control based on filter type (value/time/range)
- Button ("Apply Filters") - saves state + updates URL
- Button ("Clear All") - resets all filters
- Button ("Share Filters", Share2 icon) - saves filter state + copies URL with ?filter_state=uuid
- - URL-based state restore -
- Toast - "Filter state loaded from shared link"
**📦 State & TanStack Query**
- Zustand dashFilterStore: { filters: Record, isDirty }
- On filter change: dashFilterStore.setFilter(id, value) + isDirty=true
- useMutation({ mutationFn: ()=>api.saveFilterState(dashId, dashFilterStore.filters), onSuccess: (r)=>{ updateURL("?filter_state="+r.uuid); navigator.clipboard.writeText(window.location.href) } })
- On mount: if URL has ?filter_state=uuid → useQuery to restore
- useQuery({ queryKey:["filter-state",uuid], enabled:!!uuid, onSuccess: (s)=>dashFilterStore.setFilters(s.state) })
**✨ UX Behaviors**
- Filter sidebar: collapsible (ResizablePanel), shows filter controls based on json_metadata.native_filter_configuration.
- Cross-filter: chart click → sets filter value → all linked charts re-query.
- "Share" Button: saves state → copies URL with UUID to clipboard → Toast "Link copied with current filters".
- URL restore: on load if ?filter_state=uuid → fetch → apply → Toast "Filters restored from shared link".
- Expired filter state URL → Toast "Shared filter link has expired" + load dashboard without filters.
**🌐 API Calls**
1. useMutation({ mutationFn: (state)=>fetch("/api/v1/dashboards/"+id+"/filter-state",{method:"POST",body:JSON.stringify({state})}).then(r=>r.json()) })
2. useQuery({ queryKey:["filter-state",uuid], queryFn: ()=>fetch("/api/v1/key-value/filter_state/"+uuid).then(r=>r.json()) }) |
| --- | --- | --- |


**DB-008** - **Embed Dashboard (Guest Token)**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**       | **API / Route**                                                                                            |
| --------------- | ------------ | --------- | ------------------- | ---------------------------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 3   | embedded_dashboards | POST /api/v1/dashboards/:id/embed · POST /api/v1/dashboards/guest-token · GET /api/v1/dashboards/:id/embed |

**⚑ Depends on:** DB-004 (dashboard must be published), AUTH-004 (guest token generation)

| **⚙️ Backend - Description**
- Create embed config with UUID + allowed_domains list. Guest token endpoint: validate dashboard_uuid, check Referer against allowed_domains, issue short-lived JWT (15min) with embedded_guest role. The embedding application's server calls this endpoint (not the browser). The iframe authenticates with the guest token - read-only access, no SQL Lab, no admin.
**🔄 Request Flow**
1. POST /embed: GORM.Upsert(embedded_dashboards{UUID:uuid.New(),AllowedDomains:join(domains)}).
2. POST /guest-token: validate uuid → check origin in allowed_domains → jwt.Sign(embedded_guest,dashID,15min).
3. Middleware: embedded_guest role → restrict to chart query endpoints only.
**⚙️ Go Implementation**
1. GORM.Clauses(OnConflict DoUpdates[allowed_domains]).Create(&embedded_dashboards{UUID:uuid.New()})
2. check origin: strings.Contains(embed.AllowedDomains,extractDomain(r.Header.Get("Origin")))
3. jwt.NewWithClaims(RS256,Claims{Role:"embedded_guest",DashboardID:id,Exp:now()+15min}) | **✅ Acceptance Criteria**
- POST /embed → 201 { uuid, allowed_domains }.
- POST /guest-token from allowed domain → 200 { token, expires_in:900 }.
- From non-allowed domain → 403.
- Guest JWT can render charts but cannot access SQL Lab.
- Expired guest token → 401.
**⚠️ Error Responses**
- 403 - Non-allowed domain or non-owner.
- 401 - Expired guest token. | **🖥️ Frontend Specification**
**📍 Route & Page**
/dashboards/:id/edit (Properties Sheet → "Embed" section)
**🧩 shadcn/ui Components**
- Card ("Embed Dashboard") in Properties Sheet - Admin/owner only
- Input (read-only) - embed UUID display
- Input - allowed_domains (comma-separated)
- Button ("Generate Embed Config") - POST /embed
- Textarea (read-only) - iframe embed code snippet
- Button (Copy icon) - copies iframe snippet to clipboard
- Alert (warning) - "Only share the guest token with trusted servers, never expose in browser code"
- Badge ("Embedded" or "Not configured") - embed status
- - Embed Code Preview -
- CodeBlock - shows sample iframe HTML
- CodeBlock - shows sample server-side guest token fetch code (Node.js example)
**📦 State & TanStack Query**
- useQuery({ queryKey:["embed-config",id] }) - load existing config
- useMutation({ mutationFn: ({id,domains})=>api.createEmbedConfig(id,domains), onSuccess: (r)=>setEmbedConfig(r) })
- useState: { allowedDomains: string }
**✨ UX Behaviors**
- After generating: shows iframe code + server-side code example in CodeBlock.
- iframe src="https://yourapp.com/dashboard/{uuid}?guest_token={token}".
- Warning: "Guest tokens must be generated server-side. Never expose API keys in the browser.".
- Allowed domains Input: comma-separated, validate domain format (regex).
**🛡️ Client-Side Validation**
- allowed_domains: each domain matches /^[a-zA-Z0-9.-]+$/ regex.
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,domains})=>fetch("/api/v1/dashboards/"+id+"/embed",{method:"POST",body:JSON.stringify({allowed_domains:domains.split(",").map(d=>d.trim())})}).then(r=>r.json()) }) |
| --- | --- | --- |


**DB-009** - **Delete Dashboard**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**                                                                      | **API / Route**               |
| --------------- | ------------ | --------- | ---------------------------------------------------------------------------------- | ----------------------------- |
| **⚠ DEPENDENT** | **P0**       | Phase 2   | dashboards, dashboard_slices, dashboard_user, dashboard_roles, embedded_dashboards | DELETE /api/v1/dashboards/:id |

**⚑ Depends on:** AUTH-011 (owner), cascade across dashboard_slices/user/roles

| **⚙️ Backend - Description**
- Guard: active report schedules targeting this dashboard → 409. TX cascade: delete dashboard_slices, dashboard_user, dashboard_roles, embedded_dashboards, key_value (filter_states), then dashboards. Charts NOT deleted. Publish Redis event "dashboard:deleted:id". Audit log.
**🔄 Request Flow**
1. Ownership → count active reports → 409 if any.
2. TX: delete all related tables → delete dashboards.
3. Redis pub/sub "dashboard:deleted:id".
4. Audit log.
**⚙️ Go Implementation**
1. GORM.Where("dashboard_id=? AND active=true",id).Count on report_schedule → 409
2. TX: delete dashboard_slices,dashboard_user,dashboard_roles,embedded_dashboards,key_value (filter_state),dashboards
3. rdb.Publish("dashboard:deleted:"+id,"removed") | **✅ Acceptance Criteria**
- DELETE → 204.
- Has active reports → 409 with report list.
- Non-owner → 403.
- Charts still exist after delete.
- Redis event published.
**⚠️ Error Responses**
- 403 - Not owner.
- 409 - Active reports target this dashboard. | **🖥️ Frontend Specification**
**📍 Route & Page**
AlertDialog from /dashboards list or /dashboards/:id/edit toolbar
**🧩 shadcn/ui Components**
- AlertDialog - full confirmation with dashboard name
- AlertDialogDescription - lists active reports if any: "2 active reports will be stopped"
- AlertDialogAction (destructive) - "Delete Dashboard"
- Alert (info, inside dialog) - "Charts on this dashboard will NOT be deleted"
**📦 State & TanStack Query**
- useMutation({ mutationFn: (id)=>api.deleteDashboard(id), onSuccess: ()=>{ navigate("/dashboards"); toast.success("Dashboard deleted") } })
**✨ UX Behaviors**
- AlertDialog: "Delete {title}? This cannot be undone. Charts will remain."
- If reports: expand dialog with report list. Action still allowed (reports will be deactivated).
- After delete: navigate /dashboards + Toast.
- Other clients viewing deleted dashboard: WebSocket event → Toast "This dashboard has been removed."
**🌐 API Calls**
1. useMutation({ mutationFn: (id)=>fetch("/api/v1/dashboards/"+id,{method:"DELETE"}) }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID** | **Name**                             | **Priority** | **Dep**     | **FE Route**                                                                | **Endpoint(s)**                                                                                                             | **Phase** |
| ------ | ------------------------------------ | ------------ | ----------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | --------- |
| DB-001 | Create Dashboard                     | P0           | ⚠ DEPENDENT | /dashboards (+ Button opens Dialog) and /dashboards/:id/edit (after create) | POST /api/v1/dashboards                                                                                                     | Phase 2   |
| DB-002 | Add and Remove Charts from Dashboard | P0           | ⚠ DEPENDENT | /dashboards/:id/edit ("Add Charts" workflow)                                | PUT /api/v1/dashboards/:id/charts · POST /api/v1/dashboards/:id/charts/add · DELETE /api/v1/dashboards/:id/charts/:chart_id | Phase 2   |
| DB-003 | Update Dashboard Layout and Metadata | P0           | ⚠ DEPENDENT | /dashboards/:id/edit (auto-saved as user drags/resizes)                     | PUT /api/v1/dashboards/:id                                                                                                  | Phase 2   |
| DB-004 | Publish and Unpublish Dashboard      | P0           | ⚠ DEPENDENT | /dashboards/:id/edit (toolbar Publish button)                               | PUT /api/v1/dashboards/:id/publish                                                                                          | Phase 2   |
| DB-005 | List and Get Dashboards              | P0           | ⚠ DEPENDENT | /dashboards                                                                 | GET /api/v1/dashboards · GET /api/v1/dashboards/:id                                                                         | Phase 2   |
| DB-006 | Role-Based Dashboard Access          | P0           | ⚠ DEPENDENT | /dashboards/:id/edit (Properties Sheet → "Access" section)                  | PUT /api/v1/dashboards/:id/roles · GET /api/v1/dashboards/:id/roles                                                         | Phase 2   |
| DB-007 | Native Filter State Persistence      | P1           | ⚠ DEPENDENT | /dashboard/:slug or /dashboards/:id (dashboard view mode)                   | POST /api/v1/dashboards/:id/filter-state · GET /api/v1/key-value/filter_state/:uuid                                         | Phase 2   |
| DB-008 | Embed Dashboard (Guest Token)        | P1           | ⚠ DEPENDENT | /dashboards/:id/edit (Properties Sheet → "Embed" section)                   | POST /api/v1/dashboards/:id/embed · POST /api/v1/dashboards/guest-token · GET /api/v1/dashboards/:id/embed                  | Phase 3   |
| DB-009 | Delete Dashboard                     | P0           | ⚠ DEPENDENT | AlertDialog from /dashboards list or /dashboards/:id/edit toolbar           | DELETE /api/v1/dashboards/:id                                                                                               | Phase 2   |