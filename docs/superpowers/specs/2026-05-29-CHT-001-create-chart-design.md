# CHT-001: Create Chart — Design Spec

**Date:** 2026-05-29
**Priority:** P0 | **Phase:** 2
**Tables:** slices, slice_user | **Route:** POST /api/v1/charts

## Architecture

### Request DTO

```go
type CreateChartRequest struct {
    SliceName            string `json:"slice_name" binding:"required,max=255"`
    VizType              string `json:"viz_type" binding:"required"`
    DatasourceID         string `json:"datasource_id" binding:"required"`
    DatasourceType       string `json:"datasource_type" binding:"required"`
    Params               string `json:"params"`               // optional, any valid JSON string
    QueryContext          string `json:"query_context"`         // optional
    Description          string `json:"description"`           // optional
    CacheTimeout         int    `json:"cache_timeout"`         // optional
    CertifiedBy          string `json:"certified_by"`          // optional
    CertificationDetails string `json:"certification_details"` // optional
}
```

### Backend — Layers

```
POST /api/v1/charts
  → JWTMiddleware (extract user context)
  → chartHandler.Create()
      → dto.ShouldBindJSON(&CreateChartRequest{})
      → if params != "" && !json.Valid(params) → 400 "Invalid params JSON"
      → chartService.CreateChart(ctx, actorID, req)
          → datasetRepo.GetByID(datasourceID) → 422 if not found
          → authService.CheckDatasetReadPerm(actorID, dataset.Perm) → 403
          → derive perm/schema_perm from dataset
          → populate datasource_name from dataset.Name
          → chartRepo.CreateSlice(&Slice{
                SliceName, VizType, DatasourceID, DatasourceType,
                DatasourceName, Params, QueryContext, Description,
                CacheTimeout, Perm, SchemaPerm,
                CertifiedBy, CertificationDetails,
                LastSavedAt: time.Now(),
                LastSavedByFK: uid,
                CreatedByFK: uid,
                ChangedByFK: uid,
            })
          → chartRepo.CreateSliceUser(&SliceUser{SliceID, UserID: uid})
      → c.JSON(201, slice)
```

### Files

| File | Action | Purpose |
|------|--------|---------|
| `backend/internal/repository/postgres/chart.go` | Create | GORM implementation of `chart.Repository` |
| `backend/internal/app/chart/service.go` | Create | Business logic: validation, perm derivation, creation |
| `backend/internal/delivery/http/chart/handler.go` | Create | HTTP handler: bind, call service, respond |
| `backend/internal/delivery/http/chart/dto.go` | Create | CreateChartRequest with binding tags |
| `backend/internal/delivery/http/router.go` | Modify | Register chart routes, add chartHandler param |
| `backend/cmd/api/main.go` | Modify | Wire chartRepo → chartService → chartHandler |

### Service Constructor

Follows `dataset.Service` pattern: accepts interfaces, nil-safe defaults (noop for auth), all dependencies injectable.

```
func NewService(chartRepo chart.Repository, datasetRepo dataset.Repository, authService AuthService) *Service
```

`datasource_name` is populated from `dataset.Name` on create (not from the request body).

### Error Mapping

| Condition | HTTP |
|-----------|------|
| Invalid JSON body / invalid params JSON | 400 |
| datasource_id not found | 422 |
| No dataset read access | 403 |

### Perm Derivation

```
slice.Perm = dataset.Perm
slice.SchemaPerm = dataset.SchemaPerm
```

`CreatedByFK` and `ChangedByFK` set to calling user's ID on create.

---

## Frontend — Components

### Files

| File | Action | Purpose |
|------|--------|---------|
| `frontend/src/api/charts.ts` | Create | `createChart(data)` fetch wrapper |
| `frontend/src/components/Explore/SaveChartDialog.tsx` | Create | Dialog: slice_name, description, save |
| `frontend/src/store/exploreStore.ts` | Create/Modify | Zustand: datasourceId, vizType, params, queryContext, isDirty, savedParamsSnapshot |
| `frontend/src/pages/Explore/ExplorePage.tsx` | Modify | Wire save flow, Dialog, mutation, Ctrl+S |

### Component Tree

```
ExplorePage
  └── Zustand exploreStore
  ├── Toolbar
  │     ├── "● Unsaved changes" Badge (isDirty)
  │     └── Save Button → opens SaveChartDialog
  └── SaveChartDialog (shadcn Dialog)
        ├── Badge: viz_type
        ├── FormField: slice_name (required, max 255)
        ├── Textarea: description (optional)
        └── Button "Save" → useMutation(createChart)
```

Notes:
- CHT-001 scope: save always opens Dialog (no existing-chart PUT path yet — future CHT-002)
- If `slice_id` already in URL search params → chart already exists → skip for CHT-001

### State

- `isDirty` → true when chart config (params/queryContext) changes from last saved baseline
- After successful create: `isDirty = false`, URL updated to `/explore?slice_id=<id>`
- `isDirty` false initially (no saved chart yet — empty form), true once user modifies config
- Detect dirty via deep equality check of current params/queryContext vs saved snapshot

### TanStack Mutation

```ts
useMutation({
  mutationFn: (data) => api.createChart(data),
  onSuccess: (chart) => {
    updateSearchParams({ slice_id: chart.id });
    toast.success("Chart saved successfully");
  }
})
```

### Client Validation

```ts
slice_name: z.string().min(1, "Chart name required").max(255)
```

### Keyboard

Ctrl+S triggers save flow (same as Save button click).
