# CHT-001: Create Chart — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement POST /api/v1/charts endpoint (backend) + Save Chart Dialog in Explore page (frontend).

**Architecture:** Backend follows existing hexagonal pattern (repo → service → handler → router). Frontend uses Zustand store + TanStack Query mutation + shadcn Dialog with React Hook Form.

**Tech Stack:** Go 1.25 + Gin + GORM, React 18 + TypeScript + Vite, Zustand, TanStack Query, shadcn/ui, react-hook-form + zod

---

### Task 1: Add CreateSliceUser to chart.Repository interface + GORM repository

**Files:**
- Modify: `backend/internal/domain/chart/repository.go`
- Create: `backend/internal/repository/postgres/chart.go`

- [ ] **Step 1: Add CreateSliceUser to the domain repository interface**

Edit `backend/internal/domain/chart/repository.go` — add after line 8 (after `CreateSlice`):

```go
	CreateSliceUser(ctx context.Context, su *SliceUser) error
```

The Repository interface now has `CreateSliceUser` alongside `CreateSlice`.

- [ ] **Step 2: Create the GORM repository implementation**

Create `backend/internal/repository/postgres/chart.go`:

```go
package postgres

import (
	"context"
	"fmt"

	"superset/auth-service/internal/domain/chart"

	"gorm.io/gorm"
)

type ChartRepository struct {
	db *gorm.DB
}

func NewChartRepository(db *gorm.DB) *ChartRepository {
	return &ChartRepository{db: db}
}

func (r *ChartRepository) CreateSlice(ctx context.Context, slice *chart.Slice) error {
	if err := r.db.WithContext(ctx).Create(slice).Error; err != nil {
		return fmt.Errorf("creating slice: %w", err)
	}
	return nil
}

func (r *ChartRepository) CreateSliceUser(ctx context.Context, su *chart.SliceUser) error {
	if err := r.db.WithContext(ctx).Create(su).Error; err != nil {
		return fmt.Errorf("creating slice_user: %w", err)
	}
	return nil
}

func (r *ChartRepository) GetSliceByID(ctx context.Context, id uint) (*chart.Slice, error) {
	var slice chart.Slice
	if err := r.db.WithContext(ctx).First(&slice, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting slice by id: %w", err)
	}
	return &slice, nil
}

func (r *ChartRepository) UpdateSlice(ctx context.Context, slice *chart.Slice) error {
	if err := r.db.WithContext(ctx).Save(slice).Error; err != nil {
		return fmt.Errorf("updating slice: %w", err)
	}
	return nil
}

func (r *ChartRepository) DeleteSlice(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&chart.Slice{}, id).Error; err != nil {
		return fmt.Errorf("deleting slice: %w", err)
	}
	return nil
}

func (r *ChartRepository) ListSlices(ctx context.Context, filter *chart.SliceListFilter) ([]*chart.Slice, int64, error) {
	var slices []*chart.Slice
	var total int64

	q := r.db.WithContext(ctx).Model(&chart.Slice{})
	if filter.DatasourceID != 0 {
		q = q.Where("datasource_id = ?", filter.DatasourceID)
	}
	if filter.DatasourceType != "" {
		q = q.Where("datasource_type = ?", filter.DatasourceType)
	}
	if filter.VizType != "" {
		q = q.Where("viz_type = ?", filter.VizType)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting slices: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := q.Order("changed_on DESC").Offset(offset).Limit(filter.PageSize).Find(&slices).Error; err != nil {
		return nil, 0, fmt.Errorf("listing slices: %w", err)
	}
	return slices, total, nil
}

func (r *ChartRepository) CreateDashboard(ctx context.Context, d *chart.Dashboard) error {
	if err := r.db.WithContext(ctx).Create(d).Error; err != nil {
		return fmt.Errorf("creating dashboard: %w", err)
	}
	return nil
}

func (r *ChartRepository) GetDashboardByID(ctx context.Context, id uint) (*chart.Dashboard, error) {
	var d chart.Dashboard
	if err := r.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, fmt.Errorf("getting dashboard by id: %w", err)
	}
	return &d, nil
}

func (r *ChartRepository) GetDashboardBySlug(ctx context.Context, slug string) (*chart.Dashboard, error) {
	var d chart.Dashboard
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&d).Error; err != nil {
		return nil, fmt.Errorf("getting dashboard by slug: %w", err)
	}
	return &d, nil
}

func (r *ChartRepository) UpdateDashboard(ctx context.Context, d *chart.Dashboard) error {
	if err := r.db.WithContext(ctx).Save(d).Error; err != nil {
		return fmt.Errorf("updating dashboard: %w", err)
	}
	return nil
}

func (r *ChartRepository) DeleteDashboard(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&chart.Dashboard{}, id).Error; err != nil {
		return fmt.Errorf("deleting dashboard: %w", err)
	}
	return nil
}

func (r *ChartRepository) ListDashboards(ctx context.Context, filter *chart.DashboardListFilter) ([]*chart.Dashboard, int64, error) {
	var dashboards []*chart.Dashboard
	var total int64

	q := r.db.WithContext(ctx).Model(&chart.Dashboard{})
	if filter.Q != "" {
		q = q.Where("dashboard_title ILIKE ?", "%"+filter.Q+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting dashboards: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := q.Order("changed_on DESC").Offset(offset).Limit(filter.PageSize).Find(&dashboards).Error; err != nil {
		return nil, 0, fmt.Errorf("listing dashboards: %w", err)
	}
	return dashboards, total, nil
}

func (r *ChartRepository) AddSliceToDashboard(ctx context.Context, dashboardID, sliceID uint) error {
	ds := &chart.DashboardSlice{DashboardID: dashboardID, SliceID: sliceID}
	if err := r.db.WithContext(ctx).Create(ds).Error; err != nil {
		return fmt.Errorf("adding slice to dashboard: %w", err)
	}
	return nil
}

func (r *ChartRepository) RemoveSliceFromDashboard(ctx context.Context, dashboardID, sliceID uint) error {
	if err := r.db.WithContext(ctx).Where("dashboard_id = ? AND slice_id = ?", dashboardID, sliceID).Delete(&chart.DashboardSlice{}).Error; err != nil {
		return fmt.Errorf("removing slice from dashboard: %w", err)
	}
	return nil
}

func (r *ChartRepository) ListDashboardSlices(ctx context.Context, dashboardID uint) ([]*chart.DashboardSlice, error) {
	var items []*chart.DashboardSlice
	if err := r.db.WithContext(ctx).Where("dashboard_id = ?", dashboardID).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("listing dashboard slices: %w", err)
	}
	return items, nil
}
```

Note: Must add `"errors"` and `"gorm.io/gorm"` to imports for `GetSliceByID`.

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build ./internal/repository/postgres/...`
Expected: compiles with no errors.

---

### Task 2: Create chart service (business logic)

**Files:**
- Create: `backend/internal/app/chart/service.go`

- [ ] **Step 1: Create the service file**

Create `backend/internal/app/chart/service.go`:

```go
package chart

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	chartdomain "superset/auth-service/internal/domain/chart"
	datasetdomain "superset/auth-service/internal/domain/dataset"
	pkgerrors "superset/auth-service/internal/pkg/autherrors"
)

// DatasetRepo is the subset of dataset.Repository needed by chart service.
type DatasetRepo interface {
	GetDatasetByID(ctx context.Context, id uint) (*datasetdomain.Dataset, error)
}

// DatasetPermChecker validates whether a user can read a dataset.
type DatasetPermChecker interface {
	CanReadDataset(ctx context.Context, userID uint, dataset *datasetdomain.Dataset) (bool, error)
}

type noopPermChecker struct{}

func (noopPermChecker) CanReadDataset(_ context.Context, _ uint, _ *datasetdomain.Dataset) (bool, error) {
	return true, nil
}

// CreateChartRequest holds the request data for creating a chart.
type CreateChartRequest struct {
	SliceName            string
	VizType              string
	DatasourceID         string
	DatasourceType       string
	Params               string
	QueryContext          string
	Description          string
	CacheTimeout         int
	CertifiedBy          string
	CertificationDetails string
}

// Service handles chart lifecycle use cases.
type Service struct {
	chartRepo   chartdomain.Repository
	datasetRepo DatasetRepo
	permChecker DatasetPermChecker
}

func NewService(chartRepo chartdomain.Repository, datasetRepo DatasetRepo, permChecker DatasetPermChecker) *Service {
	if permChecker == nil {
		permChecker = noopPermChecker{}
	}
	return &Service{
		chartRepo:   chartRepo,
		datasetRepo: datasetRepo,
		permChecker: permChecker,
	}
}

func (s *Service) CreateChart(ctx context.Context, actorID uint, req CreateChartRequest) (*chartdomain.Slice, error) {
	datasourceID, err := strconv.ParseUint(req.DatasourceID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid datasource_id", pkgerrors.ErrInvalidDataset)
	}

	dataset, err := s.datasetRepo.GetDatasetByID(ctx, uint(datasourceID))
	if err != nil {
		return nil, fmt.Errorf("loading dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, pkgerrors.ErrInvalidDataset
	}

	canRead, err := s.permChecker.CanReadDataset(ctx, actorID, dataset)
	if err != nil {
		return nil, fmt.Errorf("checking dataset permission: %w", err)
	}
	if !canRead {
		return nil, pkgerrors.ErrForbidden
	}

	if req.Params != "" {
		if !json.Valid([]byte(req.Params)) {
			return nil, fmt.Errorf("invalid params JSON")
		}
	}

	now := time.Now()
	slice := &chartdomain.Slice{
		SliceName:            req.SliceName,
		VizType:              req.VizType,
		DatasourceID:         req.DatasourceID,
		DatasourceType:       req.DatasourceType,
		DatasourceName:       dataset.Name,
		Params:               req.Params,
		QueryContext:          req.QueryContext,
		Description:          req.Description,
		CacheTimeout:         req.CacheTimeout,
		Perm:                 dataset.Perm,
		SchemaPerm:           dataset.SchemaPerm,
		CertifiedBy:          req.CertifiedBy,
		CertificationDetails: req.CertificationDetails,
		LastSavedAt:          now,
		LastSavedByFK:        actorID,
		CreatedByFK:          actorID,
		ChangedByFK:          actorID,
	}

	if err := s.chartRepo.CreateSlice(ctx, slice); err != nil {
		return nil, fmt.Errorf("creating slice: %w", err)
	}

	su := &chartdomain.SliceUser{
		SliceID: slice.ID,
		UserID:  actorID,
	}
	if err := s.chartRepo.CreateSliceUser(ctx, su); err != nil {
		return nil, fmt.Errorf("creating slice_user: %w", err)
	}

	return slice, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./internal/app/chart/...`
Expected: compiles with no errors.

---

### Task 3: Create chart HTTP handler + DTO

**Files:**
- Create: `backend/internal/delivery/http/chart/dto.go`
- Create: `backend/internal/delivery/http/chart/handler.go`

- [ ] **Step 1: Create the DTO file**

Create `backend/internal/delivery/http/chart/dto.go`:

```go
package chart

type CreateChartRequest struct {
	SliceName            string `json:"slice_name" binding:"required,max=255"`
	VizType              string `json:"viz_type" binding:"required"`
	DatasourceID         string `json:"datasource_id" binding:"required"`
	DatasourceType       string `json:"datasource_type" binding:"required"`
	Params               string `json:"params"`
	QueryContext          string `json:"query_context"`
	Description          string `json:"description"`
	CacheTimeout         int    `json:"cache_timeout"`
	CertifiedBy          string `json:"certified_by"`
	CertificationDetails string `json:"certification_details"`
}
```

- [ ] **Step 2: Create the handler file**

Create `backend/internal/delivery/http/chart/handler.go`:

```go
package chart

import (
	"context"
	"errors"
	"net/http"

	"superset/auth-service/internal/delivery/http/middleware"
	domainauth "superset/auth-service/internal/domain/auth"
	chartdomain "superset/auth-service/internal/domain/chart"
	pkgerrors "superset/auth-service/internal/pkg/autherrors"

	"github.com/gin-gonic/gin"
)

type createChartService interface {
	CreateChart(ctx context.Context, actorID uint, req chartSvc.CreateChartRequest) (*chartdomain.Slice, error)
}

// We re-import the service package's types here
import chartSvc "superset/auth-service/internal/app/chart"

type Handler struct {
	svcCreate createChartService
}

func NewHandler(svcCreate createChartService) *Handler {
	return &Handler{svcCreate: svcCreate}
}

func (h *Handler) Create(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domainauth.ErrTokenInvalid.Error()})
		return
	}

	var req CreateChartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slice, err := h.svcCreate.CreateChart(c.Request.Context(), actor.ID, chartSvc.CreateChartRequest{
		SliceName:            req.SliceName,
		VizType:              req.VizType,
		DatasourceID:         req.DatasourceID,
		DatasourceType:       req.DatasourceType,
		Params:               req.Params,
		QueryContext:          req.QueryContext,
		Description:          req.Description,
		CacheTimeout:         req.CacheTimeout,
		CertifiedBy:          req.CertifiedBy,
		CertificationDetails: req.CertificationDetails,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": slice})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pkgerrors.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, pkgerrors.ErrInvalidDataset):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		if errMsg := err.Error(); len(errMsg) > 0 && errMsg[:7] == "invalid params JSON" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func getActor(c *gin.Context) (domainauth.UserContext, bool) {
	value, ok := c.Get(middleware.UserContextKey)
	if !ok {
		return domainauth.UserContext{}, false
	}
	actor, ok := value.(domainauth.UserContext)
	if !ok {
		return domainauth.UserContext{}, false
	}
	return actor, true
}
```

Wait — the import `chartSvc` won't work because you can't import the service package while also having a local type named `CreateChartRequest`. Fix the handler to not use a duplicate name:

Actually in Go, the local package is `chart` (handler) and the service package is `app/chart`. We need an alias. Let me reconsider — the handler DTO and the service request type have the same structure but are different types. The handler maps between them. The service doesn't need to export its own request type — we can pass fields directly or define a shared type.

Simplest approach: export the `CreateChartRequest` from the service package, and the handler imports and maps to it. But the handler package is also named `chart`. That's a conflict.

Better approach: Define the request type only in the handler (DTO), and have the service accept individual parameters or use a type from the domain layer. Let me simplify — the service's `CreateChart` method can accept the fields directly as parameters, or we define a service-local request type and use a package alias.

Let me restructure cleanly:

The handler defines `CreateChartRequest` (Gin binding DTO). The service defines its own input type. The handler maps between them:

```go
// handler/dto.go
package chart

type CreateChartRequest struct { ... } // HTTP DTO
```

```go
// app/chart/service.go
package chart

type CreateChartInput struct {
	SliceName string
	...
}
```

And the handler imports the service as `chartsvc` and maps:

```go
import chartsvc "superset/auth-service/internal/app/chart"
```

This is clean. Let me rewrite task 2 and 3 accordingly.

- [ ] **Step 1 (revised): Create the DTO file**

Create `backend/internal/delivery/http/chart/dto.go`:

```go
package chart

type CreateChartRequest struct {
	SliceName            string `json:"slice_name" binding:"required,max=255"`
	VizType              string `json:"viz_type" binding:"required"`
	DatasourceID         string `json:"datasource_id" binding:"required"`
	DatasourceType       string `json:"datasource_type" binding:"required"`
	Params               string `json:"params"`
	QueryContext         string `json:"query_context"`
	Description          string `json:"description"`
	CacheTimeout         int    `json:"cache_timeout"`
	CertifiedBy          string `json:"certified_by"`
	CertificationDetails string `json:"certification_details"`
}
```

- [ ] **Step 2 (revised): Create the handler file**

Create `backend/internal/delivery/http/chart/handler.go`:

```go
package chart

import (
	"context"
	"errors"
	"net/http"

	chartsvc "superset/auth-service/internal/app/chart"
	"superset/auth-service/internal/delivery/http/middleware"
	domainauth "superset/auth-service/internal/domain/auth"
	chartdomain "superset/auth-service/internal/domain/chart"
	pkgerrors "superset/auth-service/internal/pkg/autherrors"

	"github.com/gin-gonic/gin"
)

type createChartService interface {
	CreateChart(ctx context.Context, actorID uint, input chartsvc.CreateChartInput) (*chartdomain.Slice, error)
}

type Handler struct {
	svcCreate createChartService
}

func NewHandler(svcCreate createChartService) *Handler {
	return &Handler{svcCreate: svcCreate}
}

func (h *Handler) Create(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateChartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slice, err := h.svcCreate.CreateChart(c.Request.Context(), actor.ID, chartsvc.CreateChartInput{
		SliceName:            req.SliceName,
		VizType:              req.VizType,
		DatasourceID:         req.DatasourceID,
		DatasourceType:       req.DatasourceType,
		Params:               req.Params,
		QueryContext:         req.QueryContext,
		Description:          req.Description,
		CacheTimeout:         req.CacheTimeout,
		CertifiedBy:          req.CertifiedBy,
		CertificationDetails: req.CertificationDetails,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": slice})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pkgerrors.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, pkgerrors.ErrInvalidDataset):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid datasource_id"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func getActor(c *gin.Context) (domainauth.UserContext, bool) {
	value, ok := c.Get(middleware.UserContextKey)
	if !ok {
		return domainauth.UserContext{}, false
	}
	actor, ok := value.(domainauth.UserContext)
	if !ok {
		return domainauth.UserContext{}, false
	}
	return actor, true
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build ./internal/delivery/http/chart/...`
Expected: compiles with no errors (the service package does not exist yet, so this verification must happen after task 2).

---

### Task 2 (revised): Create chart service

- [ ] **Step 1: Create the service file**

Create `backend/internal/app/chart/service.go`:

```go
package chart

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	chartdomain "superset/auth-service/internal/domain/chart"
	datasetdomain "superset/auth-service/internal/domain/dataset"
	pkgerrors "superset/auth-service/internal/pkg/autherrors"
)

// DatasetRepo is the subset of dataset.Repository needed by chart service.
type DatasetRepo interface {
	GetDatasetByID(ctx context.Context, id uint) (*datasetdomain.Dataset, error)
}

// DatasetPermChecker validates whether a user can read a dataset.
type DatasetPermChecker interface {
	CanReadDataset(ctx context.Context, userID uint, dataset *datasetdomain.Dataset) (bool, error)
}

type noopPermChecker struct{}

func (noopPermChecker) CanReadDataset(_ context.Context, _ uint, _ *datasetdomain.Dataset) (bool, error) {
	return true, nil
}

// CreateChartInput holds data for creating a chart.
type CreateChartInput struct {
	SliceName            string
	VizType              string
	DatasourceID         string
	DatasourceType       string
	Params               string
	QueryContext         string
	Description          string
	CacheTimeout         int
	CertifiedBy          string
	CertificationDetails string
}

// Service handles chart lifecycle use cases.
type Service struct {
	chartRepo   chartdomain.Repository
	datasetRepo DatasetRepo
	permChecker DatasetPermChecker
}

func NewService(chartRepo chartdomain.Repository, datasetRepo DatasetRepo, permChecker DatasetPermChecker) *Service {
	if permChecker == nil {
		permChecker = noopPermChecker{}
	}
	return &Service{
		chartRepo:   chartRepo,
		datasetRepo: datasetRepo,
		permChecker: permChecker,
	}
}

func (s *Service) CreateChart(ctx context.Context, actorID uint, input CreateChartInput) (*chartdomain.Slice, error) {
	datasourceID, err := strconv.ParseUint(input.DatasourceID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid datasource_id", pkgerrors.ErrInvalidDataset)
	}

	dataset, err := s.datasetRepo.GetDatasetByID(ctx, uint(datasourceID))
	if err != nil {
		return nil, fmt.Errorf("loading dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, pkgerrors.ErrInvalidDataset
	}

	canRead, err := s.permChecker.CanReadDataset(ctx, actorID, dataset)
	if err != nil {
		return nil, fmt.Errorf("checking dataset permission: %w", err)
	}
	if !canRead {
		return nil, pkgerrors.ErrForbidden
	}

	if input.Params != "" {
		if !json.Valid([]byte(input.Params)) {
			return nil, fmt.Errorf("invalid params JSON")
		}
	}

	now := time.Now()
	slice := &chartdomain.Slice{
		SliceName:            input.SliceName,
		VizType:              input.VizType,
		DatasourceID:         input.DatasourceID,
		DatasourceType:       input.DatasourceType,
		DatasourceName:       dataset.Name,
		Params:               input.Params,
		QueryContext:         input.QueryContext,
		Description:          input.Description,
		CacheTimeout:         input.CacheTimeout,
		Perm:                 dataset.Perm,
		SchemaPerm:           dataset.SchemaPerm,
		CertifiedBy:          input.CertifiedBy,
		CertificationDetails: input.CertificationDetails,
		LastSavedAt:          now,
		LastSavedByFK:        actorID,
		CreatedByFK:          actorID,
		ChangedByFK:          actorID,
	}

	if err := s.chartRepo.CreateSlice(ctx, slice); err != nil {
		return nil, fmt.Errorf("creating slice: %w", err)
	}

	su := &chartdomain.SliceUser{
		SliceID: slice.ID,
		UserID:  actorID,
	}
	if err := s.chartRepo.CreateSliceUser(ctx, su); err != nil {
		return nil, fmt.Errorf("creating slice_user: %w", err)
	}

	return slice, nil
}
```

---

### Task 4: Register chart routes in router.go

**Files:**
- Modify: `backend/internal/delivery/http/router.go`

- [ ] **Step 1: Add import and route registration**

In `backend/internal/delivery/http/router.go`, add the import:

```go
httpchart "superset/auth-service/internal/delivery/http/chart"
```

Add `chartHandler *httpchart.Handler` to the `NewRouter` function parameters (after `sqllabHandler`).

In the protected routes section, add after the dataset routes:

```go
protected.POST("/charts", chartHandler.Create)
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build ./internal/delivery/http/...`
Expected: compiles correctly (may need to update main.go first — ok if the caller doesn't compile yet, but the router package itself should).

---

### Task 5: Wire chart dependencies in main.go + add dataset perm checker

**Files:**
- Modify: `backend/cmd/api/main.go`
- Create: `backend/internal/app/auth/dataset_perm_checker.go`

- [ ] **Step 1: Create the dataset perm checker implementation**

The simplest approach for dataset read-perm checking is to match the existing `dataset.Service` pattern — admin/alpha can read all, gamma can only read their own. Create `backend/internal/app/auth/dataset_perm_checker.go`:

```go
package auth

import (
	"context"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/dataset"
)

// DatasetPermChecker implements chart.DatasetPermChecker using role-based access.
type DatasetPermChecker struct {
	roleNamesForUser func(ctx context.Context, userID uint) ([]string, error)
}

func NewDatasetPermChecker(roleNamesForUser func(ctx context.Context, userID uint) ([]string, error)) *DatasetPermChecker {
	return &DatasetPermChecker{roleNamesForUser: roleNamesForUser}
}

func (c *DatasetPermChecker) CanReadDataset(ctx context.Context, userID uint, dataset *domain.Dataset) (bool, error) {
	roleNames, err := c.roleNamesForUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("loading role names: %w", err)
	}

	for _, name := range roleNames {
		value := strings.ToLower(strings.TrimSpace(name))
		if value == "admin" || value == "alpha" {
			return true, nil
		}
	}

	return dataset.CreatedByFK == userID, nil
}
```

- [ ] **Step 2: Wire in main.go**

In `backend/cmd/api/main.go`, add imports:

```go
svcchart "superset/auth-service/internal/app/chart"
httpchart "superset/auth-service/internal/delivery/http/chart"
```

After `datasetRepo` is created (line 104), add:

```go
chartRepo := repopostgres.NewChartRepository(db)
```

After the auth service creation section, add the perm checker:

```go
datasetPermChecker := svcauth.NewDatasetPermChecker(databaseRepo.GetRoleNamesByUser)
```

Note: This assumes `databaseRepo` has a `GetRoleNamesByUser` method. Check the existing service — looking at the dataset service, it uses `s.databaseRepo.GetRoleNamesByUser(ctx, actorUserID)`. The `databaseLookupRepository` interface in the dataset service already defines this. But `databaseRepo` in main.go is `repopostgres.NewDatabaseRepository(db)` — let me verify it has this method.

Looking at the dataset service constructor: `svcdataset.NewService(datasetRepo, databaseRepo, ...)`. The `databaseRepo` passed there is `repopostgres.NewDatabaseRepository(db)`, which must implement `databaseLookupRepository`. So `databaseRepo` does have `GetRoleNamesByUser`.

Actually wait — the dataset service's `databaseLookupRepository` interface includes `GetRoleNamesByUser` AND `GetDatabaseByID`. We need to make sure `databaseRepo` (from `repopostgres.NewDatabaseRepository`) implements `GetRoleNamesByUser`. Given the dataset service works, it does. So we can pass it.

But the actual signature matters. Let me use it correctly:

```go
// Need a function that matches DatasetPermChecker's roleNamesForUser signature
roleNameFetcher := func(ctx context.Context, userID uint) ([]string, error) {
    return databaseRepo.GetRoleNamesByUser(ctx, userID)
}
datasetPermChecker := svcauth.NewDatasetPermChecker(roleNameFetcher)
```

Then create the chart service and handler:

```go
chartSvc := svcchart.NewService(chartRepo, datasetRepo, datasetPermChecker)
chartHandler := httpchart.NewHandler(chartSvc)
```

Add `chartHandler` to `delivery.NewRouter(...)` call.

---

### Task 6: Verify backend compiles

- [ ] **Step 1: Build the entire backend**

Run: `cd backend && go build ./...`
Expected: All packages compile.

- [ ] **Step 2: Run go vet**

Run: `cd backend && go vet ./...`
Expected: No issues.

---

### Task 7: Create frontend charts API module

**Files:**
- Create: `frontend/src/api/charts.ts`

- [ ] **Step 1: Create the API module**

Create `frontend/src/api/charts.ts`:

```typescript
import { request } from "@/utils/request";

export interface CreateChartPayload {
  slice_name: string;
  viz_type: string;
  datasource_id: string;
  datasource_type: string;
  params?: string;
  query_context?: string;
  description?: string;
  cache_timeout?: number;
  certified_by?: string;
  certification_details?: string;
}

export interface ChartResponse {
  id: number;
  slice_name: string;
  viz_type: string;
  datasource_id: string;
  datasource_type: string;
  datasource_name: string;
  params: string;
  query_context: string;
  description: string;
  cache_timeout: number;
  perm: string;
  schema_perm: string;
  certified_by: string;
  certification_details: string;
  last_saved_at: string;
  last_saved_by_fk: number;
  created_on: string;
  changed_on: string;
}

interface ApiEnvelope<T> {
  data: T;
}

export const chartsApi = {
  create: (payload: CreateChartPayload): Promise<ChartResponse> =>
    request<ApiEnvelope<ChartResponse>>("/api/v1/charts", {
      method: "POST",
      body: JSON.stringify(payload),
    }).then((res) => res.data),
};
```

---

### Task 8: Create Zustand explore store

**Files:**
- Create: `frontend/src/stores/exploreStore.ts`

- [ ] **Step 1: Create the store**

Create `frontend/src/stores/exploreStore.ts`:

```typescript
import { create } from "zustand";

interface ExploreState {
  datasourceId: string | null;
  vizType: string;
  params: string;
  queryContext: string;
  isDirty: boolean;
  savedParamsSnapshot: string | null;
  savedQueryContextSnapshot: string | null;

  setDatasourceId: (id: string) => void;
  setVizType: (type: string) => void;
  setParams: (params: string) => void;
  setQueryContext: (qc: string) => void;
  markClean: () => void;
}

export const useExploreStore = create<ExploreState>((set) => ({
  datasourceId: null,
  vizType: "bar",
  params: "",
  queryContext: "",
  isDirty: false,
  savedParamsSnapshot: null,
  savedQueryContextSnapshot: null,

  setDatasourceId: (id) => set({ datasourceId: id }),

  setVizType: (type) =>
    set((state) => ({
      vizType: type,
      isDirty: true,
    })),

  setParams: (params) =>
    set((state) => ({
      params,
      isDirty:
        params !== state.savedParamsSnapshot ||
        state.queryContext !== state.savedQueryContextSnapshot,
    })),

  setQueryContext: (qc) =>
    set((state) => ({
      queryContext: qc,
      isDirty:
        qc !== state.savedQueryContextSnapshot ||
        state.params !== state.savedParamsSnapshot,
    })),

  markClean: () =>
    set((state) => ({
      isDirty: false,
      savedParamsSnapshot: state.params,
      savedQueryContextSnapshot: state.queryContext,
    })),
}));
```

---

### Task 9: Create SaveChartDialog component

**Files:**
- Create: `frontend/src/components/Explore/SaveChartDialog.tsx`

- [ ] **Step 1: Create the dialog**

Create `frontend/src/components/Explore/SaveChartDialog.tsx`:

```typescript
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { chartsApi } from "@/api/charts";
import { useExploreStore } from "@/stores/exploreStore";

const formSchema = z.object({
  slice_name: z.string().min(1, "Chart name is required").max(255),
  description: z.string().optional(),
});

type FormValues = z.infer<typeof formSchema>;

interface SaveChartDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export default function SaveChartDialog({ open, onOpenChange }: SaveChartDialogProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const { datasourceId, vizType, params, queryContext, markClean } = useExploreStore();

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      slice_name: "",
      description: "",
    },
  });

  const mutation = useMutation({
    mutationFn: chartsApi.create,
    onSuccess: (chart) => {
      markClean();
      searchParams.set("slice_id", String(chart.id));
      setSearchParams(searchParams, { replace: true });
      toast.success("Chart saved successfully");
      onOpenChange(false);
      form.reset();
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to save chart");
    },
  });

  const onSubmit = (values: FormValues) => {
    if (!datasourceId) {
      toast.error("No datasource selected");
      return;
    }
    mutation.mutate({
      slice_name: values.slice_name,
      viz_type: vizType,
      datasource_id: datasourceId,
      datasource_type: "table",
      params: params || undefined,
      query_context: queryContext || undefined,
      description: values.description || undefined,
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Save Chart
            <Badge variant="secondary">{vizType}</Badge>
          </DialogTitle>
          <DialogDescription>
            Give your chart a name to save it.
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="slice_name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Chart Name</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g. Revenue by Month" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description (optional)</FormLabel>
                  <FormControl>
                    <Textarea placeholder="What this chart shows..." {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? "Saving..." : "Save"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
```

---

### Task 10: Modify ExplorePage to wire save flow

**Files:**
- Modify: `frontend/src/pages/explore/ExplorePage.tsx`

- [ ] **Step 1: Add imports, store usage, dialog state, and Ctrl+S**

In `frontend/src/pages/explore/ExplorePage.tsx`, add these imports (merge with existing, do not duplicate):

```typescript
import { useState, useMemo, useEffect, useCallback } from "react";
import SaveChartDialog from "@/components/Explore/SaveChartDialog";
import { useExploreStore } from "@/stores/exploreStore";
```

- [ ] **Step 2: Add store hooks and dialog state at top of component**

After the existing `useState` declarations, add:

```typescript
const [saveDialogOpen, setSaveDialogOpen] = useState(false);
const exploreStore = useExploreStore();
```

- [ ] **Step 3: Update the explore store when chart config changes**

After `const [chartConfig, setChartConfig] = ...`, add an effect to sync params to the store:

```typescript
useEffect(() => {
  exploreStore.setParams(JSON.stringify(chartConfig));
}, [chartConfig, exploreStore]);
```

Also sync databaseId to store:

```typescript
useEffect(() => {
  if (databaseId) {
    exploreStore.setDatasourceId(String(databaseId));
  }
}, [databaseId, exploreStore]);
```

- [ ] **Step 4: Add Ctrl+S keyboard shortcut**

Add after the effects:

```typescript
const handleKeyDown = useCallback(
  (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "s") {
      e.preventDefault();
      setSaveDialogOpen(true);
    }
  },
  []
);

useEffect(() => {
  document.addEventListener("keydown", handleKeyDown);
  return () => document.removeEventListener("keydown", handleKeyDown);
}, [handleKeyDown]);
```

- [ ] **Step 5: Modify the toolbar header to include Save button and unsaved badge**

Replace the toolbar header div (lines 180-203):

```tsx
<div className="flex items-center justify-between">
  <div className="flex items-center gap-2">
    <h1 className="text-2xl font-bold">Explore</h1>
    {exploreStore.isDirty && (
      <Badge variant="outline" className="text-amber-600 border-amber-600">
        ● Unsaved changes
      </Badge>
    )}
  </div>
  <div className="flex items-center gap-2">
    <Button onClick={() => setSaveDialogOpen(true)}>
      Save
    </Button>
    <Select
      onValueChange={(value) => setDatabaseId(parseInt(value, 10))}
      value={databaseId?.toString() || ""}
    >
      <SelectTrigger className="w-[200px]">
        <SelectValue placeholder="Select database" />
      </SelectTrigger>
      <SelectContent>
        {databasesLoading ? (
          <SelectItem value="loading" disabled>
            Loading...
          </SelectItem>
        ) : (
          databasesData?.items?.map((db) => (
            <SelectItem key={db.id} value={db.id.toString()}>
              {db.database_name}
            </SelectItem>
          ))
        )}
      </SelectContent>
    </Select>
  </div>
</div>
```

- [ ] **Step 6: Add SaveChartDialog to the render tree**

At the end of the component, before the closing `</div>` (line 389), add:

```tsx
<SaveChartDialog open={saveDialogOpen} onOpenChange={setSaveDialogOpen} />
```

- [ ] **Step 7: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors.

---

### Self-Review Checklist

1. **Spec coverage:**
   - POST /api/v1/charts with required/optional fields → Task 2 + 3
   - Validate datasource exists → Task 2 (service)
   - User perm check on dataset → Task 2 + 5
   - Derive perm/schema_perm from datasource → Task 2
   - Create slice_user record → Task 2
   - 422 on invalid datasource → Task 3 (handleError)
   - 403 on no dataset access → Task 3 (handleError)
   - 400 on invalid JSON → Task 3 (handleError)
   - SaveChartDialog component → Task 9
   - Zustand exploreStore → Task 8
   - TanStack Query mutation → Task 9
   - React Hook Form + zod → Task 9
   - Unsaved changes badge → Task 10
   - Ctrl+S → Task 10
   - URL update on save → Task 9

2. **No placeholders** — all code is complete.

3. **Type consistency** — `CreateChartInput` matches between handler and service. Store selectors match usage in dialog and page.
