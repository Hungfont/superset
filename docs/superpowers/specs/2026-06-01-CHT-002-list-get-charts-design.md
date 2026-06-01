# CHT-002: List & Get Charts — Design Spec

**Date:** 2026-06-01
**Priority:** P0 | **Phase:** 2
**Tables:** slices, slice_user, dashboard_slices, ab_user | **Routes:** GET /api/v1/charts · GET /api/v1/charts/:id
**Depends on:** AUTH-011 (visibility filtering by perm)

## Architecture

### Domain Changes

**Slice model** (`domain/chart/entity.go`): Added GORM relationship fields for user name resolution:

```go
LastSavedBy *domainauth.User `gorm:"foreignKey:LastSavedByFK" json:"last_saved_by,omitempty"`
CreatedBy   *domainauth.User `gorm:"foreignKey:CreatedByFK" json:"created_by,omitempty"`
```

Used with `.Preload("LastSavedBy").Preload("CreatedBy")` to hydrate user names in query results.

**Repository interface** (`domain/chart/repository.go`): Expand `SliceListFilter`:

```go
type SliceListFilter struct {
    DatasourceID    uint
    DatasourceType  string
    VizType         string
    OwnerID         uint
    Certified       *bool   // nil=no filter, true/false
    Q               string  // ILIKE search on slice_name + description
    Page            int
    PageSize        int
    // Visibility (set by service, not request)
    VisibilityAll   bool     // Admin — no WHERE restriction
    VisibilityUserID uint    // Alpha/Gamma — filter by slice_user.user_id
    PermissionNames []string // Alpha — additional RBAC perms for OR slices.perm IN(?)
}
```

### Backend — Layers

```
GET /api/v1/charts
  → JWTMiddleware (extract user context)
  → chartHandler.List()
      → c.ShouldBindQuery(&ChartListQuery{})  // form tags
      → chartService.ListCharts(ctx, actorID, query)
          → resolveVisibilityScope(ctx, actorID)
              Admin  → visibilityAll=true
              Alpha  → visibilityUserID=actorID, permissionNames=[rbac perms]
              Gamma  → visibilityUserID=actorID
          → repo.ListSlices(ctx, filter)
              GORM: JOIN slice_user su ON su.slice_id=slices.id
              Admin  → no visibility WHERE
              Alpha  → WHERE (su.user_id=? OR slices.perm IN (?))
              Gamma  → WHERE su.user_id=?
              → filter chain: viz_type, datasource_id, owner, certified, q
              → subquery: dashboard_count per chart
              → COUNT → total
              → ORDER BY last_saved_at DESC, LIMIT, OFFSET
              → Preload("LastSavedBy") for list items
      → c.JSON(200, {data: {items, total, page, page_size}})

GET /api/v1/charts/:id
  → JWTMiddleware
  → chartHandler.Get()
      → parse id from path param
      → chartService.GetChart(ctx, actorID, id)
          → repo.GetSliceByID(ctx, id) with Preload("LastSavedBy").Preload("CreatedBy")
          → visibility check: if gamma && not own → 404
          → subquery dashboard_count
      → c.JSON(200, {data: {...}})
```

### Request DTOs

```go
type ChartListQuery struct {
    Q            string `form:"q"`
    VizType      string `form:"viz_type"`
    DatasourceID uint   `form:"datasource_id"`
    Owner        uint   `form:"owner"`
    Certified    *bool  `form:"certified"`
    Page         int    `form:"page"`
    PageSize     int    `form:"page_size"`
}
```

Defaults: page=1, page_size=20, max page_size=100.

### Response Shapes

**List:**
```json
{
  "data": {
    "items": [{
      "id": 1,
      "slice_name": "Sales Bar",
      "viz_type": "bar",
      "datasource_name": "sales",
      "last_saved_at": "2026-05-30T10:00:00Z",
      "last_saved_by": {"id": 1, "username": "alice", "first_name": "Alice", "last_name": "S"},
      "certified_by": "",
      "dashboard_count": 3
    }],
    "total": 42,
    "page": 1,
    "page_size": 20
  }
}
```

**Detail:**
```json
{
  "data": {
    "id": 1,
    "slice_name": "Sales Bar",
    "viz_type": "bar",
    "datasource_id": "1",
    "datasource_type": "table",
    "datasource_name": "sales",
    "params": "{\"metric\":\"sum__num\"}",
    "query_context": "...",
    "description": "Monthly sales by region",
    "cache_timeout": 0,
    "perm": "[sales](id:1)",
    "certification_details": "",
    "thumbnail_url": "",
    "last_saved_at": "2026-05-30T10:00:00Z",
    "last_saved_by": {"id": 1, "username": "alice", "first_name": "Alice", "last_name": "S"},
    "created_by": {"id": 1, "username": "alice", "first_name": "Alice", "last_name": "S"},
    "dashboard_count": 3
  }
}
```

### Service: Visibility Resolution

```go
func (s *Service) resolveVisibility(ctx context.Context, userID uint) (all bool, userIDFilter uint, perms []string, err error) {
    roles, err := s.roleRepo.GetRoleNamesByUser(ctx, userID)
    // Admin → all=true
    // Alpha → userIDFilter=userID, perms=rbacPermissionNames
    // Gamma → userIDFilter=userID
    return
}
```

### Error Handling

| Case | Status |
|------|--------|
| JWT missing/invalid | 401 (middleware) |
| Invalid query params | 422 (ShouldBindQuery failure) |
| Chart not found or no visibility | 404 |
| Internal error | 500 |

### Acceptance Criteria

- [ ] GET /api/v1/charts → `{ items:[...], total, page }`
- [ ] GET /api/v1/charts?viz_type=bar → only bar charts
- [ ] GET /api/v1/charts?certified=true → only certified
- [ ] GET /api/v1/charts?q=sales → ILIKE match on slice_name + description
- [ ] GET /api/v1/charts?owner=5 → only charts by user 5
- [ ] GET /api/v1/charts/:id → full detail including params JSON, created_by, last_saved_by
- [ ] Gamma without perm → chart excluded (not 403)
- [ ] Alpha sees own + RBAC-permitted charts
- [ ] Admin sees all charts
- [ ] dashboard_count reflects actual dashboard_slices count per chart

## Frontend

### Route

`/charts` — new protected route in `App.tsx`, above the admin routes block.

### API Layer (`api/charts.ts`)

```typescript
export interface ChartListParams {
  q?: string
  viz_type?: string
  datasource_id?: number
  owner?: number
  certified?: boolean
  page?: number
  page_size?: number
}

export interface ChartListItem {
  id: number
  slice_name: string
  viz_type: string
  datasource_name: string
  last_saved_at: string
  last_saved_by: { id: number; username: string; first_name: string; last_name: string } | null
  certified_by: string
  dashboard_count: number
}

export interface ChartListResponse {
  items: ChartListItem[]
  total: number
  page: number
  page_size: number
}

export interface ChartDetail extends ChartListItem {
  datasource_id: string
  datasource_type: string
  params: string
  query_context: string
  description: string
  cache_timeout: number
  perm: string
  certification_details: string
  thumbnail_url: string
  created_by: { id: number; username: string; first_name: string; last_name: string } | null
}

chartsApi.list = (params: ChartListParams) => request<ApiEnvelope<ChartListResponse>>(...)
chartsApi.get = (id: number) => request<ApiEnvelope<ChartDetail>>(...)
```

### Page Component (`pages/charts/ChartsPage.tsx`)

**State:** `{ q, viz_type, owner, certified, page }`

**TanStack Query:**
```typescript
const { data } = useQuery({
  queryKey: ["charts", filters],
  queryFn: () => chartsApi.list(filters),
})
```

**Components (shadcn/ui):**
- `DataTable` — cols: Thumbnail, Name, Type (Badge), Dataset, Dashboards, Modified, Certified, Actions
- `Input` + `Search` icon — search by name (q filter)
- `Select` — viz_type filter ("All Types" | bar | line | pie | table | ...)
- `Select` — owner filter ("All" | "Mine")
- `Switch` — "Certified only" toggle
- `DropdownMenu` per row — Edit (→Explore), Duplicate, Add to Dashboard, Delete
- `Badge` — viz_type color-coded (Line/Area=blue, Bar/Column=green, Pie/Donut=orange, Table=gray, Big Number=purple, Map=teal)
- `Avatar` + `ShieldCheck` — certified_by tooltip
- `Tooltip` on dashboard_count — hover shows dashboard names (Popover)
- `Skeleton` — 6 loading rows
- `Empty` state — BarChart2 icon + "No charts yet" + "Create your first chart" Button
- `Button` "+ Chart" — navigates to /explore
- Checkbox column for bulk select → Delete selected, Add to Dashboard

**UX:**
- Thumbnail: 60×40px image (if generated) else viz_type icon placeholder
- Type Badge: color-coded by chart family
- Dashboard count: click → Popover with dashboard name links
- Duplicate mutation → onSuccess navigates to /explore?slice_id=<new_id>
- Delete mutation → onSuccess invalidate + toast
