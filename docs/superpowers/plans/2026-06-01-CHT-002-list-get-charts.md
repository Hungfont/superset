# CHT-002: List & Get Charts — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement GET /api/v1/charts (paginated list with role-based visibility) and GET /api/v1/charts/:id (full detail) backend + /charts frontend page.

**Architecture:** Existing hexagonal Go layers + React/TypeScript frontend. List endpoint uses visibility-scoped GORM query with slice_user join + RBAC perm check for Alpha. Detail endpoint preloads user relationships. Frontend uses TanStack Query + shadcn DataTable.

**Tech Stack:** Go + Gin + GORM (backend), React 18 + TypeScript + TanStack Query v5 + shadcn/ui (frontend)

---

### Task 1: Expand SliceListFilter in Domain Repository Interface

**Files:**
- Modify: `backend/internal/domain/chart/repository.go`

- [ ] **Step 1: Replace the existing SliceListFilter struct**

Replace lines 27-33:

```go
// SliceListFilter defines filters for listing slices.
type SliceListFilter struct {
	DatasourceID   uint
	DatasourceType string
	VizType        string
	Page           int
	PageSize       int
}
```

With:

```go
// SliceListFilter defines filters for listing slices.
type SliceListFilter struct {
	DatasourceID     uint
	DatasourceType   string
	VizType          string
	OwnerID          uint
	Certified        *bool
	Q                string
	Page             int
	PageSize         int
	VisibilityAll    bool
	VisibilityUserID uint
	PermissionNames  []string
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/domain/chart/repository.go
git commit -m "feat(CHT-002): expand SliceListFilter with visibility and search fields"
```

---

### Task 2: Add Repository Interface Methods

**Files:**
- Modify: `backend/internal/domain/chart/repository.go`

- [ ] **Step 1: Add GetSliceDetail and DashboardCount to Repository interface**

After line 12 (`ListSlices(...)`), insert:

```go
	GetSliceDetail(ctx context.Context, id uint) (*Slice, error)
	DashboardCount(ctx context.Context, sliceID uint) (int64, error)
```

- [ ] **Step 2: Run `go build ./...` to verify**

Run: `cd backend && go build ./...`
Expected: FAIL — methods not yet implemented (will be fixed in Task 3)

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/chart/repository.go
git commit -m "feat(CHT-002): add GetSliceDetail and DashboardCount to Repository interface"
```

---

### Task 3: Implement ListSlices + GetSliceDetail in GORM Repository

**Files:**
- Modify: `backend/internal/repository/postgres/chart.go`

- [ ] **Step 1: Replace the ListSlices method (lines 44-63)**

Replace the entire `ListSlices` method with:

```go
func (r *ChartRepository) ListSlices(ctx context.Context, f *chart.SliceListFilter) ([]*chart.Slice, int64, error) {
	var total int64

	base := r.db.WithContext(ctx).
		Table("slices").
		Joins("LEFT JOIN slice_user su ON su.slice_id = slices.id")

	if f.VisibilityAll {
		// Admin — no restriction
	} else {
		base = base.Distinct("slices.*")
		if len(f.PermissionNames) > 0 {
			base = base.Where("su.user_id = ? OR slices.perm IN (?)", f.VisibilityUserID, f.PermissionNames)
		} else {
			base = base.Where("su.user_id = ?", f.VisibilityUserID)
		}
	}

	if f.DatasourceID != 0 {
		base = base.Where("slices.datasource_id = ?", fmt.Sprintf("%d", f.DatasourceID))
	}
	if f.DatasourceType != "" {
		base = base.Where("slices.datasource_type = ?", f.DatasourceType)
	}
	if f.VizType != "" {
		base = base.Where("slices.viz_type = ?", f.VizType)
	}
	if f.OwnerID != 0 {
		base = base.Where("slices.created_by_fk = ?", f.OwnerID)
	}
	if f.Certified != nil {
		if *f.Certified {
			base = base.Where("slices.certified_by <> ''")
		} else {
			base = base.Where("slices.certified_by = ''")
		}
	}
	if f.Q != "" {
		base = base.Where("(slices.slice_name ILIKE ? OR slices.description ILIKE ?)", "%"+f.Q+"%", "%"+f.Q+"%")
	}

	countQ := r.db.WithContext(ctx).Table("(?) AS filtered", base)
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting slices: %w", err)
	}

	off := (f.Page - 1) * f.PageSize
	var slices []*chart.Slice
	if err := base.Order("slices.last_saved_at DESC").
		Offset(off).Limit(f.PageSize).
		Preload("LastSavedBy").
		Find(&slices).Error; err != nil {
		return nil, 0, fmt.Errorf("listing slices: %w", err)
	}

	return slices, total, nil
}
```

- [ ] **Step 2: Add GetSliceDetail method (after GetSliceByID, line 34)**

Insert after `GetSliceByID`:

```go
func (r *ChartRepository) GetSliceDetail(ctx context.Context, id uint) (*chart.Slice, error) {
	var s chart.Slice
	if err := r.db.WithContext(ctx).
		Preload("LastSavedBy").
		Preload("CreatedBy").
		First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting slice detail: %w", err)
	}
	return &s, nil
}
```

- [ ] **Step 3: Add DashboardCount (before `var _` assertion)**

Insert before `var _ chart.Repository = (*ChartRepository)(nil)`:

```go
func (r *ChartRepository) DashboardCount(ctx context.Context, sliceID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&chart.DashboardSlice{}).
		Where("slice_id = ?", sliceID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting dashboards for slice %d: %w", sliceID, err)
	}
	return count, nil
}
```

- [ ] **Step 4: Run `go build ./...`**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/postgres/chart.go
git commit -m "feat(CHT-002): implement visibility-scoped ListSlices, GetSliceDetail, DashboardCount"
```

---

### Task 4: Add ListCharts and GetChart to Chart Service

**Files:**
- Modify: `backend/internal/app/chart/service.go`

- [ ] **Step 1: Replace imports (lines 3-13)**

Replace the import block with:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	chartdomain "superset/auth-service/internal/domain/chart"
	datasetdomain "superset/auth-service/internal/domain/dataset"
	pkgerrors "superset/auth-service/internal/pkg/autherrors"
)
```

- [ ] **Step 2: Add RoleRepo and RBACCacheRepo interfaces**

Add after `noopPermChecker` (after line 28):

```go
type RoleRepo interface {
	GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error)
}

type RBACCacheRepo interface {
	GetPermissionSet(ctx context.Context, userID uint) ([]string, error)
}
```

- [ ] **Step 3: Add UserInfo helper type**

Add after the new interfaces:

```go
// UserInfo is a projection of auth.User for API responses.
type UserInfo struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func toUserInfo(u *chartdomain.User) *UserInfo {
	if u == nil {
		return nil
	}
	return &UserInfo{
		ID:        u.ID,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
}
```

- [ ] **Step 4: Replace Service struct (lines 42-46)**

Replace the Service struct:

```go
type Service struct {
	chartRepo   chartdomain.Repository
	datasetRepo DatasetRepo
	permChecker DatasetPermChecker
	roleRepo    RoleRepo
	rbacCache   RBACCacheRepo
}
```

- [ ] **Step 5: Replace NewService (lines 48-53)**

Replace `NewService`:

```go
func NewService(
	chartRepo chartdomain.Repository,
	datasetRepo DatasetRepo,
	permChecker DatasetPermChecker,
	roleRepo RoleRepo,
	rbacCache RBACCacheRepo,
) *Service {
	if permChecker == nil {
		permChecker = noopPermChecker{}
	}
	return &Service{
		chartRepo:   chartRepo,
		datasetRepo: datasetRepo,
		permChecker: permChecker,
		roleRepo:    roleRepo,
		rbacCache:   rbacCache,
	}
}
```

- [ ] **Step 6: Add DTO types and methods at end of file**

Append to `service.go`:

```go
// --- List/Get Charts DTOs and methods ---

type ListChartsInput struct {
	Q            string
	VizType      string
	DatasourceID uint
	Owner        uint
	Certified    *bool
	Page         int
	PageSize     int
}

type ChartListItem struct {
	ID             uint      `json:"id"`
	SliceName      string    `json:"slice_name"`
	VizType        string    `json:"viz_type"`
	DatasourceName string    `json:"datasource_name"`
	LastSavedAt    time.Time `json:"last_saved_at"`
	LastSavedBy    *UserInfo `json:"last_saved_by,omitempty"`
	CertifiedBy    string    `json:"certified_by"`
	DashboardCount int64     `json:"dashboard_count"`
}

type ChartListResult struct {
	Items    []ChartListItem `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

type ChartDetail struct {
	ID                   uint      `json:"id"`
	SliceName            string    `json:"slice_name"`
	VizType              string    `json:"viz_type"`
	DatasourceID         string    `json:"datasource_id"`
	DatasourceType       string    `json:"datasource_type"`
	DatasourceName       string    `json:"datasource_name"`
	Params               string    `json:"params"`
	QueryContext         string    `json:"query_context"`
	Description          string    `json:"description"`
	CacheTimeout         int       `json:"cache_timeout"`
	Perm                 string    `json:"perm"`
	CertifiedBy          string    `json:"certified_by"`
	CertificationDetails string    `json:"certification_details"`
	LastSavedAt          time.Time `json:"last_saved_at"`
	LastSavedBy          *UserInfo `json:"last_saved_by,omitempty"`
	CreatedBy            *UserInfo `json:"created_by,omitempty"`
	DashboardCount       int64     `json:"dashboard_count"`
}

func normalizeListInput(input ListChartsInput) ListChartsInput {
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}
	return input
}

func (s *Service) ListCharts(ctx context.Context, actorID uint, input ListChartsInput) (*ChartListResult, error) {
	input = normalizeListInput(input)

	visibilityAll, visibilityUserID, permissionNames, err := s.resolveVisibility(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("resolving visibility: %w", err)
	}

	slices, total, err := s.chartRepo.ListSlices(ctx, &chartdomain.SliceListFilter{
		DatasourceID:     input.DatasourceID,
		VizType:          input.VizType,
		OwnerID:          input.Owner,
		Certified:        input.Certified,
		Q:                input.Q,
		Page:             input.Page,
		PageSize:         input.PageSize,
		VisibilityAll:    visibilityAll,
		VisibilityUserID: visibilityUserID,
		PermissionNames:  permissionNames,
	})
	if err != nil {
		return nil, fmt.Errorf("listing charts: %w", err)
	}

	items := make([]ChartListItem, 0, len(slices))
	for _, sl := range slices {
		dashCount, err := s.chartRepo.DashboardCount(ctx, sl.ID)
		if err != nil {
			return nil, fmt.Errorf("counting dashboards for chart %d: %w", sl.ID, err)
		}
		item := ChartListItem{
			ID:             sl.ID,
			SliceName:      sl.SliceName,
			VizType:        sl.VizType,
			DatasourceName: sl.DatasourceName,
			LastSavedAt:    sl.LastSavedAt,
			LastSavedBy:    toUserInfo(sl.LastSavedBy),
			CertifiedBy:    sl.CertifiedBy,
			DashboardCount: dashCount,
		}
		items = append(items, item)
	}

	return &ChartListResult{
		Items:    items,
		Total:    total,
		Page:     input.Page,
		PageSize: input.PageSize,
	}, nil
}

func (s *Service) GetChart(ctx context.Context, actorID uint, id uint) (*ChartDetail, error) {
	slice, err := s.chartRepo.GetSliceDetail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting chart: %w", err)
	}
	if slice == nil || slice.ID == 0 {
		return nil, pkgerrors.ErrChartNotFound
	}

	isAdmin, err := s.roleRepo.IsAdmin(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("checking admin: %w", err)
	}
	if !isAdmin {
		visibilityAll, _, permissionNames, visErr := s.resolveVisibility(ctx, actorID)
		if visErr != nil {
			return nil, fmt.Errorf("resolving visibility: %w", visErr)
		}
		if !visibilityAll {
			// Alpha/Gamma — check own or perm
			if slice.CreatedByFK != actorID {
				hasAccess := false
				for _, perm := range permissionNames {
					if slice.Perm == perm {
						hasAccess = true
						break
					}
				}
				if !hasAccess {
					return nil, pkgerrors.ErrChartNotFound
				}
			}
		}
	}

	dashCount, err := s.chartRepo.DashboardCount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("counting dashboards: %w", err)
	}

	return &ChartDetail{
		ID:                   slice.ID,
		SliceName:            slice.SliceName,
		VizType:              slice.VizType,
		DatasourceID:         slice.DatasourceID,
		DatasourceType:       slice.DatasourceType,
		DatasourceName:       slice.DatasourceName,
		Params:               slice.Params,
		QueryContext:         slice.QueryContext,
		Description:          slice.Description,
		CacheTimeout:         slice.CacheTimeout,
		Perm:                 slice.Perm,
		CertifiedBy:          slice.CertifiedBy,
		CertificationDetails: slice.CertificationDetails,
		LastSavedAt:          slice.LastSavedAt,
		LastSavedBy:          toUserInfo(slice.LastSavedBy),
		CreatedBy:            toUserInfo(slice.CreatedBy),
		DashboardCount:       dashCount,
	}, nil
}

func (s *Service) resolveVisibility(ctx context.Context, userID uint) (all bool, userIDFilter uint, perms []string, err error) {
	roleNames, err := s.roleRepo.GetRoleNamesByUser(ctx, userID)
	if err != nil {
		return false, 0, nil, fmt.Errorf("loading role names: %w", err)
	}

	for _, r := range roleNames {
		v := strings.ToLower(strings.TrimSpace(r))
		if v == "admin" {
			return true, 0, nil, nil
		}
	}

	perms = make([]string, 0)
	for _, r := range roleNames {
		v := strings.ToLower(strings.TrimSpace(r))
		if v == "alpha" {
			if s.rbacCache != nil {
				permSet, cacheErr := s.rbacCache.GetPermissionSet(ctx, userID)
				if cacheErr == nil {
					perms = permSet
				}
			}
			return false, userID, perms, nil
		}
	}

	return false, userID, nil, nil
}
```

- [ ] **Note: The `toUserInfo` function references `chartdomain.User`. This is valid because `domain/chart/entity.go` already imports `domain/auth` and exports the relationship field `LastSavedBy *domainauth.User`. The `chartdomain` package alias gives access to `chartdomain.User` which resolves to the re-exported `domainauth.User` type.**

- [ ] **Step 7: Run `go build ./...`**

Run: `cd backend && go build ./...`
Expected: FAIL — main.go wiring not yet updated (Task 7 fixes this)

- [ ] **Step 8: Commit**

```bash
git add backend/internal/app/chart/service.go
git commit -m "feat(CHT-002): add ListCharts and GetChart service methods with visibility"
```

---

### Task 5: Add DTO and Handler Methods for List/Get

**Files:**
- Modify: `backend/internal/delivery/http/chart/dto.go`
- Modify: `backend/internal/delivery/http/chart/handler.go`

- [ ] **Step 1: Add list query DTO to dto.go**

Append to `dto.go`:

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

- [ ] **Step 2: Update handler.go imports (lines 3-14)**

Replace imports in handler.go:

```go
import (
	"context"
	"fmt"
	"net/http"
	"strings"

	chartsvc "superset/auth-service/internal/app/chart"
	"superset/auth-service/internal/delivery/http/middleware"
	domainauth "superset/auth-service/internal/domain/auth"
	chartdomain "superset/auth-service/internal/domain/chart"

	"github.com/gin-gonic/gin"
)
```

- [ ] **Step 3: Update interface types and Handler struct (lines 16-22)**

Replace the handler struct and constructor:

```go
type createChartService interface {
	CreateChart(ctx context.Context, actorID uint, input chartsvc.CreateChartInput) (*chartdomain.Slice, error)
}

type listChartsService interface {
	ListCharts(ctx context.Context, actorID uint, input chartsvc.ListChartsInput) (*chartsvc.ChartListResult, error)
}

type getChartService interface {
	GetChart(ctx context.Context, actorID uint, id uint) (*chartsvc.ChartDetail, error)
}

type Handler struct {
	svcCreate createChartService
	svcList   listChartsService
	svcGet    getChartService
}

func NewHandler(svcCreate createChartService, svcList listChartsService, svcGet getChartService) *Handler {
	return &Handler{svcCreate: svcCreate, svcList: svcList, svcGet: svcGet}
}
```

- [ ] **Step 4: Add List handler method (after Create, before handleError)**

Insert:

```go
func (h *Handler) List(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var query ChartListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svcList.ListCharts(c.Request.Context(), actor.ID, chartsvc.ListChartsInput{
		Q:            query.Q,
		VizType:      query.VizType,
		DatasourceID: query.DatasourceID,
		Owner:        query.Owner,
		Certified:    query.Certified,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
```

- [ ] **Step 5: Add Get handler method (after List)**

Insert:

```go
func (h *Handler) Get(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil || id == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	detail, err := h.svcGet.GetChart(c.Request.Context(), actor.ID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": detail})
}
```

- [ ] **Step 6: Update handleError to add chart-not-found case**

Before the `default:` case in `handleError`, add:

```go
	case strings.Contains(msg, "chart not found"):
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
```

- [ ] **Step 7: Run `go build ./...`**

Run: `cd backend && go build ./...`
Expected: FAIL — NewHandler signature changed (Task 7 fixes this)

- [ ] **Step 8: Commit**

```bash
git add backend/internal/delivery/http/chart/dto.go backend/internal/delivery/http/chart/handler.go
git commit -m "feat(CHT-002): add List and Get handler methods with DTO"
```

---

### Task 6: Write Handler Tests

**Files:**
- Modify: `backend/internal/delivery/http/chart/handler_test.go`

- [ ] **Step 1: Add fake services for list and get (after fakeCreateChartService)**

Insert after line 32:

```go
type fakeListChartsService struct {
	result *chartsvc.ChartListResult
	err    error
}

func (f *fakeListChartsService) ListCharts(_ context.Context, _ uint, _ chartsvc.ListChartsInput) (*chartsvc.ChartListResult, error) {
	return f.result, f.err
}

type fakeGetChartService struct {
	detail *chartsvc.ChartDetail
	err    error
}

func (f *fakeGetChartService) GetChart(_ context.Context, _ uint, _ uint) (*chartsvc.ChartDetail, error) {
	return f.detail, f.err
}
```

- [ ] **Step 2: Add helper router for list/get (after newRouter)**

Insert after `newRouter`:

```go
func newListGetRouter(listSvc *fakeListChartsService, getSvc *fakeGetChartService) *gin.Engine {
	h := chart.NewHandler(&fakeCreateChartService{}, listSvc, getSvc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domainauth.UserContext{ID: 10, Active: true})
		c.Next()
	})

	r.GET("/api/v1/charts", h.List)
	r.GET("/api/v1/charts/:id", h.Get)
	return r
}
```

- [ ] **Step 3: Add tests at end of file**

Append:

```go
func TestListCharts_Success(t *testing.T) {
	now := time.Now()
	svc := &fakeListChartsService{
		result: &chartsvc.ChartListResult{
			Items: []chartsvc.ChartListItem{
				{
					ID: 1, SliceName: "Sales Bar", VizType: "bar",
					DatasourceName: "sales", LastSavedAt: now,
					CertifiedBy: "", DashboardCount: 2,
				},
			},
			Total: 1, Page: 1, PageSize: 20,
		},
	}
	router := newListGetRouter(svc, &fakeGetChartService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/charts", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	data := resp["data"].(map[string]interface{})
	if data["total"] != float64(1) {
		t.Errorf("expected total=1, got %v", data["total"])
	}
}

func TestListCharts_Unauthorized(t *testing.T) {
	h := chart.NewHandler(&fakeCreateChartService{}, &fakeListChartsService{}, &fakeGetChartService{})
	r := gin.New()
	r.GET("/api/v1/charts", h.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/charts", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetChart_Success(t *testing.T) {
	now := time.Now()
	svc := &fakeGetChartService{
		detail: &chartsvc.ChartDetail{
			ID: 1, SliceName: "Sales Bar", VizType: "bar",
			DatasourceID: "3", DatasourceType: "table",
			DatasourceName: "sales", Params: `{"metric":"sum"}`,
			Description: "Monthly sales", LastSavedAt: now, DashboardCount: 2,
		},
	}
	router := newListGetRouter(&fakeListChartsService{}, svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/charts/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["params"] != `{"metric":"sum"}` {
		t.Errorf("expected params, got %v", data["params"])
	}
}

func TestGetChart_NotFound(t *testing.T) {
	svc := &fakeGetChartService{err: pkgerrors.ErrChartNotFound}
	router := newListGetRouter(&fakeListChartsService{}, svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/charts/999", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetChart_InvalidID(t *testing.T) {
	router := newListGetRouter(&fakeListChartsService{}, &fakeGetChartService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/charts/abc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 4: Run handler tests**

Run: `cd backend && go test ./internal/delivery/http/chart/... -v`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/chart/handler_test.go
git commit -m "test(CHT-002): add handler tests for List and Get chart endpoints"
```

---

### Task 7: Wire Dependencies in main.go + Router

**Files:**
- Modify: `backend/cmd/api/main.go`
- Modify: `backend/internal/delivery/http/router.go`

- [ ] **Step 1: Update chartSvc constructor in main.go (line 142)**

Replace:

```go
chartSvc := svcchart.NewService(chartRepo, datasetRepo, permChecker)
```

With:

```go
chartSvc := svcchart.NewService(chartRepo, datasetRepo, permChecker, databaseRepo, rbacPermissionCacheRepo)
```

- [ ] **Step 2: Update chartHandler constructor in main.go (line 143)**

Replace:

```go
chartHandler := httpchart.NewHandler(chartSvc)
```

With:

```go
chartHandler := httpchart.NewHandler(chartSvc, chartSvc, chartSvc)
```

- [ ] **Step 3: Add GET routes in router.go (after line 174)**

After `protected.POST("/charts", chartHandler.Create)`, add:

```go
				protected.GET("/charts", chartHandler.List)
				protected.GET("/charts/:id", chartHandler.Get)
```

- [ ] **Step 4: Run `go build ./...`**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/api/main.go backend/internal/delivery/http/router.go
git commit -m "feat(CHT-002): wire chart service deps and register list/get routes"
```

---

### Task 8: Write Repository Tests

**Files:**
- Modify: `backend/internal/repository/postgres/chart_test.go`

- [ ] **Step 1: Add tests at end of file**

Append:

```go
func TestGetSliceDetail_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`SELECT.*FROM "slices".*Preload.*LastSavedBy.*Preload.*CreatedBy`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "slice_name", "viz_type", "datasource_id", "datasource_type",
			"datasource_name", "perm", "last_saved_at",
		}).AddRow(1, "Test", "bar", "3", "table", "sales", "[sales](id:1)", time.Now()))

	slice, err := repo.GetSliceDetail(context.Background(), 1)
	assert.NoError(t, err)
	assert.NotNil(t, slice)
	assert.Equal(t, uint(1), slice.ID)
}

func TestGetSliceDetail_NotFound(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`SELECT.*FROM "slices"`).
		WillReturnError(gorm.ErrRecordNotFound)

	slice, err := repo.GetSliceDetail(context.Background(), 999)
	assert.NoError(t, err)
	assert.Nil(t, slice)
}

func TestDashboardCount_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "dashboard_slices"`).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.DashboardCount(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestListSlices_AdminVisibility(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`SELECT count\(\*\).*FROM "slices"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`SELECT.*FROM "slices".*Preload.*ORDER BY.*last_saved_at DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "slice_name", "viz_type", "perm", "last_saved_at"}).
			AddRow(1, "Test", "bar", "[sales](id:1)", time.Now()))

	slices, total, err := repo.ListSlices(context.Background(), &chart.SliceListFilter{
		VisibilityAll: true, Page: 1, PageSize: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, slices, 1)
}
```

- [ ] **Step 2: Run repo tests**

Run: `cd backend && go test ./internal/repository/postgres/... -v -run "Chart"`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/postgres/chart_test.go
git commit -m "test(CHT-002): add repo tests for GetSliceDetail, DashboardCount, ListSlices"
```

---

### Task 9: Add Frontend API Methods

**Files:**
- Modify: `frontend/src/api/charts.ts`

- [ ] **Step 1: Add types and list/get methods**

After the `chartsApi` object's `create` method, insert before the closing `};`:

```typescript
export interface ChartListParams {
  q?: string;
  viz_type?: string;
  datasource_id?: number;
  owner?: number;
  certified?: boolean;
  page?: number;
  page_size?: number;
}

export interface ChartListItem {
  id: number;
  slice_name: string;
  viz_type: string;
  datasource_name: string;
  last_saved_at: string;
  last_saved_by: { id: number; username: string; first_name: string; last_name: string } | null;
  certified_by: string;
  dashboard_count: number;
}

export interface ChartListResponse {
  items: ChartListItem[];
  total: number;
  page: number;
  page_size: number;
}

export interface ChartDetail extends ChartListItem {
  datasource_id: string;
  datasource_type: string;
  params: string;
  query_context: string;
  description: string;
  cache_timeout: number;
  perm: string;
  certification_details: string;
  created_by: { id: number; username: string; first_name: string; last_name: string } | null;
}

  list: (params: ChartListParams = {}): Promise<ChartListResponse> => {
    const sp = new URLSearchParams();
    if (params.q) sp.set("q", params.q);
    if (params.viz_type) sp.set("viz_type", params.viz_type);
    if (params.datasource_id) sp.set("datasource_id", String(params.datasource_id));
    if (params.owner) sp.set("owner", String(params.owner));
    if (params.certified !== undefined) sp.set("certified", String(params.certified));
    if (params.page) sp.set("page", String(params.page));
    if (params.page_size) sp.set("page_size", String(params.page_size));
    const qs = sp.toString();
    return request<ApiEnvelope<ChartListResponse>>(`/api/v1/charts${qs ? "?" + qs : ""}`, {
      method: "GET", credentials: "include", headers: getAuthHeaders(),
    }).then((r) => r.data);
  },

  get: (id: number): Promise<ChartDetail> =>
    request<ApiEnvelope<ChartDetail>>(`/api/v1/charts/${id}`, {
      method: "GET", credentials: "include", headers: getAuthHeaders(),
    }).then((r) => r.data),
```

- [ ] **Step 2: Run TypeScript check**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/charts.ts
git commit -m "feat(CHT-002): add list and get chart API methods with types"
```

---

### Task 10: Create ChartsPage Component

**Files:**
- Create: `frontend/src/pages/charts/ChartsPage.tsx`

- [ ] **Step 1: Install needed shadcn components**

Run:
```
cd frontend
npx shadcn@latest add badge
npx shadcn@latest add switch
npx shadcn@latest add dropdown-menu
npx shadcn@latest add tooltip
npx shadcn@latest add avatar
npx shadcn@latest add select
npx shadcn@latest add input
npx shadcn@latest add button
npx shadcn@latest add skeleton
npx shadcn@latest add table
```

- [ ] **Step 2: Create ChartsPage.tsx**

Create `frontend/src/pages/charts/ChartsPage.tsx`:

```tsx
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Search, Plus, BarChart2, ShieldCheck, MoreHorizontal } from "lucide-react";
import { chartsApi, type ChartListItem, type ChartListParams } from "@/api/charts";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const VIZ_TYPE_COLORS: Record<string, string> = {
  line: "bg-blue-100 text-blue-800",
  area: "bg-blue-100 text-blue-800",
  bar: "bg-green-100 text-green-800",
  column: "bg-green-100 text-green-800",
  pie: "bg-orange-100 text-orange-800",
  donut: "bg-orange-100 text-orange-800",
  table: "bg-gray-100 text-gray-800",
  big_number: "bg-purple-100 text-purple-800",
  big_number_total: "bg-purple-100 text-purple-800",
  map: "bg-teal-100 text-teal-800",
};

function vizTypeColor(vizType: string): string {
  return VIZ_TYPE_COLORS[vizType] ?? "bg-gray-100 text-gray-800";
}

function formatDate(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function ChartsPage() {
  const navigate = useNavigate();
  const [q, setQ] = useState("");
  const [vizType, setVizType] = useState("");
  const [owner, setOwner] = useState("");
  const [certified, setCertified] = useState(false);
  const [page, setPage] = useState(1);

  const filters: ChartListParams = {
    ...(q ? { q } : {}),
    ...(vizType && vizType !== "all" ? { viz_type: vizType } : {}),
    ...(owner === "mine" ? { owner: 0 } : {}),
    ...(certified ? { certified: true } : {}),
    page,
    page_size: 20,
  };

  const { data, isLoading } = useQuery({
    queryKey: ["charts", filters],
    queryFn: () => chartsApi.list(filters),
  });

  const totalPages = data ? Math.ceil(data.total / data.page_size) : 0;

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Charts</h1>
        <Button onClick={() => navigate("/explore")}>
          <Plus className="mr-2 h-4 w-4" />
          Chart
        </Button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 flex-wrap">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search charts..."
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setPage(1);
            }}
            className="pl-9"
            aria-label="Search charts"
          />
        </div>
        <Select
          value={vizType}
          onValueChange={(v) => {
            setVizType(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[160px]">
            <SelectValue placeholder="All Types" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Types</SelectItem>
            <SelectItem value="bar">Bar</SelectItem>
            <SelectItem value="line">Line</SelectItem>
            <SelectItem value="pie">Pie</SelectItem>
            <SelectItem value="table">Table</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={owner}
          onValueChange={(v) => {
            setOwner(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[130px]">
            <SelectValue placeholder="All Owners" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="mine">Mine</SelectItem>
          </SelectContent>
        </Select>
        <div className="flex items-center gap-2">
          <Switch
            id="certified-only"
            checked={certified}
            onCheckedChange={(v) => {
              setCertified(v);
              setPage(1);
            }}
          />
          <label
            htmlFor="certified-only"
            className="text-sm text-muted-foreground cursor-pointer"
          >
            Certified only
          </label>
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : !data || data.items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <BarChart2 className="h-12 w-12 text-muted-foreground mb-4" />
          <p className="text-lg font-medium">No charts yet</p>
          <p className="text-sm text-muted-foreground mb-4">
            Create your first chart to get started.
          </p>
          <Button onClick={() => navigate("/explore")}>
            <Plus className="mr-2 h-4 w-4" />
            Create your first chart
          </Button>
        </div>
      ) : (
        <>
          <Table aria-label="Charts list">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[80px]">Thumbnail</TableHead>
                <TableHead aria-sort="descending">Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Dataset</TableHead>
                <TableHead>Dashboards</TableHead>
                <TableHead>Modified</TableHead>
                <TableHead>Certified</TableHead>
                <TableHead className="w-[60px]">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.items.map((chart) => (
                <ChartRow
                  key={chart.id}
                  chart={chart}
                  onEdit={() => navigate(`/explore?slice_id=${chart.id}`)}
                />
              ))}
            </TableBody>
          </Table>

          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-4">
              <p className="text-sm text-muted-foreground">
                Showing {(page - 1) * 20 + 1}–
                {Math.min(page * 20, data.total)} of {data.total}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function ChartRow({
  chart,
  onEdit,
}: {
  chart: ChartListItem;
  onEdit: () => void;
}) {
  return (
    <TableRow>
      <TableCell>
        <div className="h-10 w-16 bg-muted rounded flex items-center justify-center overflow-hidden">
          <BarChart2 className="h-6 w-6 text-muted-foreground" />
        </div>
      </TableCell>
      <TableCell className="font-medium">{chart.slice_name}</TableCell>
      <TableCell>
        <Badge variant="secondary" className={vizTypeColor(chart.viz_type)}>
          {chart.viz_type}
        </Badge>
      </TableCell>
      <TableCell className="text-muted-foreground">
        {chart.datasource_name}
      </TableCell>
      <TableCell>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="secondary" className="cursor-pointer">
                {chart.dashboard_count}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              <p>
                Used in {chart.dashboard_count} dashboard
                {chart.dashboard_count !== 1 ? "s" : ""}
              </p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </TableCell>
      <TableCell className="text-muted-foreground text-sm">
        {formatDate(chart.last_saved_at)}
      </TableCell>
      <TableCell>
        {chart.certified_by ? (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Avatar className="h-6 w-6">
                  <AvatarFallback>
                    <ShieldCheck className="h-4 w-4 text-green-600" />
                  </AvatarFallback>
                </Avatar>
              </TooltipTrigger>
              <TooltipContent>
                <p>Certified by {chart.certified_by}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ) : null}
      </TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuItem>Duplicate</DropdownMenuItem>
            <DropdownMenuItem>Add to Dashboard</DropdownMenuItem>
            <DropdownMenuItem className="text-destructive">
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
}
```

- [ ] **Step 2: Run TypeScript check**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/charts/ChartsPage.tsx
git commit -m "feat(CHT-002): create ChartsPage with DataTable, filters, and empty state"
```

---

### Task 11: Add /charts Route + Final Verification

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Add import (after existing page imports)**

After line 23 (`import ExplorePage from "@/pages/explore/ExplorePage";`):

```tsx
import ChartsPage from "@/pages/charts/ChartsPage";
```

- [ ] **Step 2: Add route (after ExplorePage route)**

After `<Route path="/explore" element={<ExplorePage />} />`:

```tsx
          <Route path="/charts" element={<ChartsPage />} />
```

- [ ] **Step 3: Verify frontend build**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 4: Run all backend tests**

Run: `cd backend && go test ./... -cover 2>&1 | Select-String -Pattern "FAIL|ok|coverage"`
Expected: all `ok`, no `FAIL`

- [ ] **Step 5: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(CHT-002): add /charts route, all tests pass"
```
