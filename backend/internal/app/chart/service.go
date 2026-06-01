package chart

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

type DatasetRepo interface {
	GetDatasetByID(ctx context.Context, id uint) (*datasetdomain.Dataset, error)
}

type DatasetPermChecker interface {
	CanReadDataset(ctx context.Context, userID uint, dataset *datasetdomain.Dataset) (bool, error)
}

type noopPermChecker struct{}

func (noopPermChecker) CanReadDataset(_ context.Context, _ uint, _ *datasetdomain.Dataset) (bool, error) {
	return true, nil
}

type RoleRepo interface {
	IsAdmin(ctx context.Context, userID uint) (bool, error)
	GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error)
}

type RBACCacheRepo interface {
	GetPermissionSet(ctx context.Context, userID uint) ([]string, error)
}

// UserInfo is a projection of auth.User for API responses.
type UserInfo struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// toUserInfo returns nil for now — Slice model has only FK fields, not GORM relationship fields.
// When Slice gains LastSavedBy *domainauth.User relationship fields, this will map them.
func toUserInfo(_ any) *UserInfo {
	return nil
}

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

type Service struct {
	chartRepo   chartdomain.Repository
	datasetRepo DatasetRepo
	permChecker DatasetPermChecker
	roleRepo    RoleRepo
	rbacCache   RBACCacheRepo
}

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

	if input.Params != "" && !json.Valid([]byte(input.Params)) {
		return nil, fmt.Errorf("invalid params JSON")
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
	if err := s.chartRepo.CreateSliceUser(ctx, &chartdomain.SliceUser{SliceID: slice.ID, UserID: actorID}); err != nil {
		return nil, fmt.Errorf("creating slice_user: %w", err)
	}
	return slice, nil
}

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
			LastSavedBy:    toUserInfo(nil),
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
		LastSavedBy:          toUserInfo(nil),
		CreatedBy:            toUserInfo(nil),
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
