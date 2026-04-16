**📌 Annotation Service**

Rank #09 · Phase 3 - Enhancement · 4 Requirements · 1 Independent · 3 Dependent

## **Service Overview**

The Annotation Service manages time-based event markers shown directly on charts. Annotations are organized in named layers (annotation_layer) and consist of individual events with time points or ranges (annotation). Charts display annotation layers as vertical lines or shaded regions on the time axis.

This is a relatively self-contained service with minimal dependencies. The frontend provides an Annotation Layer management page and a per-layer annotation editor. Annotations are referenced by chart params - the Explore view lets users add annotation layers to charts.

The service is deliberately simple: a straightforward CRUD with a time-range query for chart rendering. The most interesting UX is the annotation timeline editor, which provides a visual way to add and edit time-based events.

## **Tech Stack**

| **Layer**     | **Technology / Package**               | **Purpose**                                    |
| ------------- | -------------------------------------- | ---------------------------------------------- |
| UI Framework  | React 18 + TypeScript                  | Type-safe component tree                       |
| Bundler       | Vite 5                                 | Fast HMR and build                             |
| Routing       | React Router v6                        | SPA navigation                                 |
| Server State  | TanStack Query v5                      | API cache, mutations, background refetch       |
| Client State  | Zustand                                | Global UI state                                |
| Components    | shadcn/ui (Radix UI)                   | ALL components - no custom, no overrides       |
| Forms         | React Hook Form + Zod                  | Schema validation, field-level errors          |
| Data Tables   | TanStack Table v8                      | Sort, filter, paginate                         |
| Styling       | Tailwind CSS v3                        | Utility-first                                  |
| Icons         | Lucide React                           | Consistent icon set                            |
| Notifications | shadcn Toaster + useToast              | Toast notifications                            |
| Charts        | Recharts (for reports/alert history)   | Report trend charts                            |
| Cron UI       | cronstrue (bun)                        | Human-readable cron expression display         |
| Date/Time     | shadcn Calendar + Popover + date-fns   | Date pickers & formatting                      |
| Backend       | Gin + GORM                             | Simple CRUD with time-range filtering          |
| FE Timeline   | Recharts ReferenceArea + ReferenceLine | Annotation rendering in chart previews         |
| FE Date       | shadcn Calendar + Popover + date-fns   | Date/time range pickers for annotation editing |

| **Attribute**      | **Detail**                                                                       |
| ------------------ | -------------------------------------------------------------------------------- |
| Service Name       | Annotation Service                                                               |
| Rank               | #09                                                                              |
| Phase              | Phase 3 - Enhancement                                                            |
| Backend API Prefix | /api/v1/annotation-layers                                                        |
| Frontend Routes    | /annotation-layers · /annotation-layers/:id · /annotation-layers/:id/annotations |
| Primary DB Tables  | annotation_layer, annotation                                                     |
| Total Requirements | 4                                                                                |
| Independent        | 1                                                                                |
| Dependent          | 3                                                                                |

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

**ANN-001** - **Create Annotation Layer**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**    | **API / Route**                |
| ----------------- | ------------ | --------- | ---------------- | ------------------------------ |
| **✓ INDEPENDENT** | **P2**       | Phase 3   | annotation_layer | POST /api/v1/annotation-layers |

| **⚙️ Backend - Description**
- Create a named annotation layer. Required: name (unique per org, 3-100 chars). Optional: descr (description, markdown). Layers are org-level shared resources - all authenticated users can use them when configuring charts. Owner = created_by_fk.
- Use cases: "Marketing Campaigns" layer for launch dates, "Incidents" for outage periods, "Releases" for deployments. Layers are reused across multiple charts.
**🔄 Request Flow**
1. Validate name uniqueness → GORM.Create(&annotation_layer) → 201.
**⚙️ Go Implementation**
1. GORM.Where("name=? AND org_id=?",name,orgID).First → 409 if found
2. GORM.Create(&annotation_layer{Name:name,Descr:descr,CreatedByFK:uid,OrgID:orgID}) | **✅ Acceptance Criteria**
- POST { name:"Marketing Campaigns", descr:"All campaign launch dates" } → 201 { id, name }.
- Duplicate name in org → 409.
- GET /annotation-layers → all org layers.
- Non-authenticated → 401.
**⚠️ Error Responses**
- 409 - Duplicate name in org.
- 422 - Name too short or too long. | **🖥️ Frontend Specification**
**📍 Route & Page**
/annotation-layers (+ Button opens Dialog)
**🧩 shadcn/ui Components**
- DataTable - cols: Name, Description, Annotations (count Badge), Created By, Modified, Actions
- Button ("+ New Layer") - opens Dialog
- Dialog ("New Annotation Layer")
- Form + Input (name) + Textarea (descr) inside Dialog
- Button ("Create Layer") - Dialog submit
- DropdownMenu (Actions per row) - View Annotations, Edit, Delete
- Badge (annotation_count) - click → navigates to /annotation-layers/:id
- Skeleton - 3 loading rows
- Empty state - Tag icon + "No annotation layers yet" + CTA
**📦 State & TanStack Query**
- useQuery({ queryKey:["annotation-layers"] })
- useMutation({ mutationFn: api.createLayer, onSuccess: ()=>{ queryClient.invalidateQueries(["annotation-layers"]); dialog.close(); toast.success("Layer created") } })
**✨ UX Behaviors**
- Dialog: Input auto-focused on open. Enter submits.
- After create: row appears in DataTable (optimistic update).
- annotation_count Badge: click → navigate to /annotation-layers/:id to manage annotations.
**🛡️ Client Validation**
- name: z.string().min(3,"Min 3 characters").max(100,"Max 100 characters")
**🌐 API Calls**
1. useMutation({ mutationFn: (data)=>fetch("/api/v1/annotation-layers",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**⚠ DEPENDENT (3) - requires prior services/requirements**

**ANN-002** - **Update and Delete Annotation Layer**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**    | **API / Route**                                                          |
| --------------- | ------------ | --------- | ---------------- | ------------------------------------------------------------------------ |
| **⚠ DEPENDENT** | **P2**       | Phase 3   | annotation_layer | PUT /api/v1/annotation-layers/:id · DELETE /api/v1/annotation-layers/:id |

**⚑ Depends on:** ANN-001 (layer must exist)

| **⚙️ Backend - Description**
- Update layer name (re-check uniqueness) or description. Owner or Admin.
- Delete: pre-delete guard - count annotations in layer. If count > 0 → 409 { count, error:"Layer has N annotations. Delete them first." }. If count = 0 → hard delete. Non-owner, non-Admin → 403.
**🔄 Request Flow**
1. Ownership check → validate name uniqueness if changed.
2. GORM.Model(&layer).Updates(fields).
3. Delete: GORM.Where("layer_id=?",id).Count → 409 if > 0.
4. GORM.Delete(&annotation_layer{},id).
**⚙️ Go Implementation**
1. GORM.Where("name=? AND org_id=? AND id!=?",name,orgID,id).First → 409 if found
2. GORM.Model(&annotation_layer{ID:id}).Updates(fields)
3. GORM.Where("layer_id=?",id).Count(&n) → 409 if n>0
4. GORM.Delete(&annotation_layer{},id) | **✅ Acceptance Criteria**
- PUT { name:"Updated Name" } → 200.
- PUT name conflict → 409.
- DELETE with 5 annotations → 409 { count:5, error:"Delete annotations first." }.
- DELETE empty layer → 204.
- Non-owner → 403.
**⚠️ Error Responses**
- 403 - Not owner.
- 404 - Not found.
- 409 - Name conflict or has annotations. | **🖥️ Frontend Specification**
**📍 Route & Page**
/annotation-layers (edit via inline row actions)
**🧩 shadcn/ui Components**
- DropdownMenu ("Edit" action) - opens Sheet with edit form
- Sheet ("Edit Layer") - pre-filled name + descr form
- Form + Input + Textarea inside Sheet
- Button ("Save Changes") inside Sheet footer
- AlertDialog - delete confirmation: "Delete {name}? All N annotations will also need to be deleted first."
- AlertDialog (variant: if has annotations) - "This layer has N annotations. Delete all annotations first, then delete the layer."
- Button (AlertDialogAction, disabled if has annotations) - "Delete Layer"
**📦 State & TanStack Query**
- useMutation({ mutationFn: ({id,...data})=>api.updateLayer(id,data) })
- useMutation({ mutationFn: (id)=>api.deleteLayer(id), onSuccess: ()=>{ queryClient.invalidateQueries(["annotation-layers"]); toast.success("Layer deleted") } })
- Pre-fetch annotation count before opening delete AlertDialog
**✨ UX Behaviors**
- Edit Sheet: opens on "Edit" action, pre-fills fields, saves via PUT.
- Delete: check annotation count first. If > 0: AlertDialog with count + "Delete annotations first" - Action button disabled.
- If count = 0: standard AlertDialog confirmation.
**🌐 API Calls**
1. useMutation({ mutationFn: ({id,...d})=>fetch("/api/v1/annotation-layers/"+id,{method:"PUT",body:JSON.stringify(d)}).then(r=>r.json()) }) |
| --- | --- | --- |


**ANN-003** - **Create Annotation**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                |
| --------------- | ------------ | --------- | ------------- | ---------------------------------------------- |
| **⚠ DEPENDENT** | **P2**       | Phase 3   | annotation    | POST /api/v1/annotation-layers/:id/annotations |

**⚑ Depends on:** ANN-001 (layer must exist), AUTH-004 (user context)

| **⚙️ Backend - Description**
- Create an annotation within a layer. Required: layer_id, short_descr (label shown on chart, max 100 chars). Optional: long_descr (tooltip, markdown), start_dttm (default NOW()), end_dttm (if set → range annotation, must be > start_dttm), json_metadata { color:"#hex", stroke_width:number, opacity:0-1 }.
- Point annotation (no end_dttm): chart renders vertical line. Range annotation (start+end): chart renders shaded region. Validate end_dttm > start_dttm if both provided.
**🔄 Request Flow**
1. GORM.First(&layer,layerID) → 422 if not found.
2. Validate: if end_dttm != nil && end_dttm <= start_dttm → 422.
3. json.Unmarshal(json_metadata) → 422 if invalid.
4. GORM.Create(&annotation{...}) → 201.
**⚙️ Go Implementation**
1. GORM.First(&layer,layerID) → 422 if err
2. if req.EndDttm!=nil && req.EndDttm.Before(*req.StartDttm): return 422 "end_dttm must be after start_dttm"
3. if req.JsonMetadata!="": json.Unmarshal([]byte(req.JsonMetadata),&map[string]interface{}{}) → 422 if err
4. GORM.Create(&annotation{LayerID:layerID,...}) | **✅ Acceptance Criteria**
- POST { short_descr:"Q4 Launch", start_dttm:"2024-10-01T00:00:00Z" } → 201 { id, short_descr, start_dttm, end_dttm:null }.
- end_dttm < start_dttm → 422.
- Invalid json_metadata → 422.
- Layer not found → 422.
- GET /annotation-layers/:id/annotations?start=2024-01-01&end=2024-12-31 → time-filtered list.
**⚠️ Error Responses**
- 422 - Invalid time range, invalid metadata, or layer not found. | **🖥️ Frontend Specification**
**📍 Route & Page**
/annotation-layers/:id (annotation management page)
**🧩 shadcn/ui Components**
- - Page Layout -
- Breadcrumb - "Annotation Layers / {layer_name}"
- Card (header) - layer name + description + annotation count
- Button ("+ Add Annotation") - opens Dialog
- - Create/Edit Dialog -
- Dialog ("Add Annotation")
- Form + Input (short_descr) + Textarea (long_descr)
- DatePicker (shadcn Calendar + Popover) - start_dttm (date + time)
- DatePicker (optional) - end_dttm with Switch "Range Annotation" to toggle
- Switch ("Range Annotation") - enables end_dttm picker
- Input (type=color, shadcn styled) - annotation color picker
- Slider - opacity 0-1
- Input (type=number) - stroke_width 1-5
- Card ("Preview") - shows how annotation will look on chart (line or shaded region)
- Button ("Add Annotation") - submit
- - Annotation List -
- DataTable - cols: Label, Type (Point/Range), Start, End, Color (swatch), Actions
- Badge (Point &#124; Range, color-coded)
- Color swatch cell - 16×16px colored square matching annotation color
- Tooltip on Label - shows long_descr preview
**📦 State & TanStack Query**
- useQuery({ queryKey:["annotations",layerId,dateRange] })
- useMutation({ mutationFn: api.createAnnotation, onSuccess: ()=>{ queryClient.invalidateQueries(["annotations",layerId]); dialog.close() } })
- useState: { isRange:false, startDttm, endDttm, color:"#1890FF", opacity:0.3, strokeWidth:2 }
**✨ UX Behaviors**
- "Range Annotation" Switch: toggle → second DatePicker appears for end_dttm.
- Preview Card: shows a mini timeline with either a vertical line (point) or shaded region (range) at chosen color/opacity.
- Color picker: native Input[type=color] styled with shadcn border/radius.
- Date+Time picker: shadcn Calendar for date + Input[type=time] for time - combined in Popover.
- DateRange filter above table: filter annotations to a specific time window.
**🛡️ Client Validation**
- short_descr: z.string().min(1).max(100).
- end_dttm (if isRange): z.date().min(startDttm,"End must be after start").
- opacity: z.number().min(0).max(1).
- stroke_width: z.number().int().min(1).max(5).
**🌐 API Calls**
1. useMutation({ mutationFn: ({layerId,...data})=>fetch("/api/v1/annotation-layers/"+layerId+"/annotations",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**ANN-004** - **List, Update and Delete Annotations**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                                                                                                                  |
| --------------- | ------------ | --------- | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P2**       | Phase 3   | annotation    | GET /api/v1/annotation-layers/:id/annotations · PUT /api/v1/annotation-layers/:id/annotations/:ann_id · DELETE /api/v1/annotation-layers/:id/annotations/:ann_id |

**⚑ Depends on:** ANN-003 (annotations must exist)

| **⚙️ Backend - Description**
- List: paginated annotations in a layer. Time range filter: WHERE start_dttm BETWEEN filter.start AND filter.end OR (end_dttm IS NOT NULL AND end_dttm BETWEEN ...) - overlap detection. Sorted by start_dttm ASC.
- Update: allow changing short_descr, long_descr, start_dttm, end_dttm, json_metadata. Re-validate time range. Creator or Admin.
- Delete: hard delete, no guard needed (annotations have no downstream FKs). Creator or Admin.
- Bulk delete: DELETE /annotation-layers/:id/annotations?before=ISO8601 - Admin only. Deletes all annotations in layer older than given date.
**🔄 Request Flow**
1. GET: GORM.Where("layer_id=? AND (start_dttm BETWEEN ? AND ? OR ...)",layerID,start,end).Order("start_dttm ASC").Paginate.
2. PUT: ownership → validate time range → GORM.Model.Updates.
3. DELETE: ownership → GORM.Delete.
4. Bulk DELETE: Admin → GORM.Where("layer_id=? AND start_dttm<?",id,before).Delete.
**⚙️ Go Implementation**
1. GORM.Where("layer_id=? AND (start_dttm BETWEEN ? AND ? OR (end_dttm IS NOT NULL AND end_dttm BETWEEN ? AND ?))",id,s,e,s,e)
2. .Order("start_dttm ASC").Offset(off).Limit(sz).Find(&annotations)
3. Bulk delete: GORM.Where("layer_id=? AND start_dttm<?",id,before).Delete(&annotation{}) → RowsAffected | **✅ Acceptance Criteria**
- GET → list sorted by start_dttm.
- GET ?start=2024-01-01&end=2024-06-30 → H1 2024 annotations.
- PUT { short_descr:"Updated" } → 200.
- DELETE → 204.
- DELETE ?before=2023-01-01 (Admin) → 200 { deleted:150 }.
- Non-creator on PUT/DELETE → 403.
**⚠️ Error Responses**
- 403 - Not creator.
- 404 - Not found. | **🖥️ Frontend Specification**
**📍 Route & Page**
/annotation-layers/:id (annotation list + edit)
**🧩 shadcn/ui Components**
- DataTable - with time filter DateRangePicker above
- DateRangePicker (shadcn Calendar range mode) - filter annotations by time window
- Button ("Clear Filter") - reset date range
- DropdownMenu (Actions) - Edit (opens Sheet), Delete
- Sheet ("Edit Annotation") - pre-filled edit form
- AlertDialog - delete confirmation "Delete this annotation?"
- Button ("Bulk Delete Old") - Admin only, opens DatePicker + confirms count
- Badge (count) - "Showing {N} annotations"
- - Timeline Visualization (above DataTable) -
- Recharts ComposedChart - mini timeline showing annotation positions
- ReferenceLine per point annotation - at start_dttm x position
- ReferenceArea per range annotation - from start to end x with fill color
- Tooltip on chart element - shows short_descr + long_descr preview
**📦 State & TanStack Query**
- useQuery({ queryKey:["annotations",layerId,{start,end,page}], queryFn: ()=>api.getAnnotations(layerId,{start,end}) })
- useMutation({ mutationFn: ({layerId,annId,...data})=>api.updateAnnotation(layerId,annId,data) })
- useMutation({ mutationFn: ({layerId,annId})=>api.deleteAnnotation(layerId,annId) })
- useState: { dateRange:null, selectedAnnotation:null }
**✨ UX Behaviors**
- Timeline chart: x-axis shows time, annotations shown as colored markers. Click marker → selects row in DataTable.
- DateRangePicker filters both the timeline chart and the DataTable simultaneously.
- Edit Sheet: pre-fills all fields. Same form as create Dialog.
- "Bulk Delete Old" (Admin): DatePicker → "Delete all annotations before {date}" → AlertDialog "This will delete N annotations. Continue?"
- Optimistic delete: row disappears immediately from table, restored on error.
**🌐 API Calls**
1. useQuery({ queryKey:["annotations",layerId,dateRange], queryFn: ()=>fetch("/api/v1/annotation-layers/"+layerId+"/annotations?start="+start+"&end="+end).then(r=>r.json()) })
2. useMutation({ mutationFn: ({layerId,annId,...d})=>fetch("/api/v1/annotation-layers/"+layerId+"/annotations/"+annId,{method:"PUT",body:JSON.stringify(d)}).then(r=>r.json()) }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID**  | **Name**                            | **Priority** | **Dep**       | **FE Route**                                        | **Endpoint(s)**                                                                                                                                                  | **Phase** |
| ------- | ----------------------------------- | ------------ | ------------- | --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| ANN-001 | Create Annotation Layer             | P2           | ✓ INDEPENDENT | /annotation-layers (+ Button opens Dialog)          | POST /api/v1/annotation-layers                                                                                                                                   | Phase 3   |
| ANN-002 | Update and Delete Annotation Layer  | P2           | ⚠ DEPENDENT   | /annotation-layers (edit via inline row actions)    | PUT /api/v1/annotation-layers/:id · DELETE /api/v1/annotation-layers/:id                                                                                         | Phase 3   |
| ANN-003 | Create Annotation                   | P2           | ⚠ DEPENDENT   | /annotation-layers/:id (annotation management page) | POST /api/v1/annotation-layers/:id/annotations                                                                                                                   | Phase 3   |
| ANN-004 | List, Update and Delete Annotations | P2           | ⚠ DEPENDENT   | /annotation-layers/:id (annotation list + edit)     | GET /api/v1/annotation-layers/:id/annotations · PUT /api/v1/annotation-layers/:id/annotations/:ann_id · DELETE /api/v1/annotation-layers/:id/annotations/:ann_id | Phase 3   |