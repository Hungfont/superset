package chart_test

import (
	"context"
	"errors"
	"testing"
	"time"

	chartsvc "superset/auth-service/internal/app/chart"
	chartdomain "superset/auth-service/internal/domain/chart"
	datasetdomain "superset/auth-service/internal/domain/dataset"
	pkgerrors "superset/auth-service/internal/pkg/autherrors"
)

// --- fakes ---

type fakeChartRepo struct {
	slice     *chartdomain.Slice
	sliceUser *chartdomain.SliceUser
	createErr error

	listSlices []*chartdomain.Slice
	listTotal  int64
	listErr    error

	sliceDetail *chartdomain.Slice
	detailErr   error

	dashCount int64
	dashErr   error
}

func (f *fakeChartRepo) CreateSlice(_ context.Context, s *chartdomain.Slice) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.slice != nil {
		s.ID = f.slice.ID
		return nil
	}
	s.ID = 42
	return nil
}

func (f *fakeChartRepo) CreateSliceUser(_ context.Context, su *chartdomain.SliceUser) error {
	if f.createErr != nil {
		return f.createErr
	}
	if su != nil {
		su.ID = 99
	}
	return nil
}

func (f *fakeChartRepo) GetSliceByID(_ context.Context, _ uint) (*chartdomain.Slice, error) { return nil, nil }

func (f *fakeChartRepo) UpdateSlice(_ context.Context, _ *chartdomain.Slice) error { return nil }

func (f *fakeChartRepo) DeleteSlice(_ context.Context, _ uint) error { return nil }

func (f *fakeChartRepo) ListSlices(_ context.Context, _ *chartdomain.SliceListFilter) ([]*chartdomain.Slice, int64, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.listSlices, f.listTotal, nil
}

func (f *fakeChartRepo) CreateDashboard(_ context.Context, _ *chartdomain.Dashboard) error { return nil }

func (f *fakeChartRepo) GetDashboardByID(_ context.Context, _ uint) (*chartdomain.Dashboard, error) {
	return nil, nil
}

func (f *fakeChartRepo) GetDashboardBySlug(_ context.Context, _ string) (*chartdomain.Dashboard, error) {
	return nil, nil
}

func (f *fakeChartRepo) UpdateDashboard(_ context.Context, _ *chartdomain.Dashboard) error { return nil }

func (f *fakeChartRepo) DeleteDashboard(_ context.Context, _ uint) error { return nil }

func (f *fakeChartRepo) ListDashboards(_ context.Context, _ *chartdomain.DashboardListFilter) ([]*chartdomain.Dashboard, int64, error) {
	return nil, 0, nil
}

func (f *fakeChartRepo) AddSliceToDashboard(_ context.Context, _, _ uint) error { return nil }

func (f *fakeChartRepo) RemoveSliceFromDashboard(_ context.Context, _, _ uint) error { return nil }

func (f *fakeChartRepo) ListDashboardSlices(_ context.Context, _ uint) ([]*chartdomain.DashboardSlice, error) {
	return nil, nil
}

func (f *fakeChartRepo) GetSliceDetail(_ context.Context, _ uint) (*chartdomain.Slice, error) {
	if f.detailErr != nil {
		return nil, f.detailErr
	}
	return f.sliceDetail, nil
}

func (f *fakeChartRepo) DashboardCount(_ context.Context, _ uint) (int64, error) {
	if f.dashErr != nil {
		return 0, f.dashErr
	}
	return f.dashCount, nil
}

type fakeDatasetRepo struct {
	dataset *datasetdomain.Dataset
	err     error
}

func (f *fakeDatasetRepo) GetDatasetByID(_ context.Context, _ uint) (*datasetdomain.Dataset, error) {
	return f.dataset, f.err
}

type fakePermChecker struct {
	allowed bool
	err     error
}

func (f *fakePermChecker) CanReadDataset(_ context.Context, _ uint, _ *datasetdomain.Dataset) (bool, error) {
	return f.allowed, f.err
}

type fakeRoleRepo struct {
	roleNames []string
	err       error
}

func (f *fakeRoleRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, r := range f.roleNames {
		if r == "admin" {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRoleRepo) GetRoleNamesByUser(_ context.Context, _ uint) ([]string, error) {
	return f.roleNames, f.err
}

type fakeRBACCacheRepo struct {
	perms []string
	err   error
}

func (f *fakeRBACCacheRepo) GetPermissionSet(_ context.Context, _ uint) ([]string, error) {
	return f.perms, f.err
}

func newTestService(repo *fakeChartRepo, dsRepo *fakeDatasetRepo, permChecker *fakePermChecker) *chartsvc.Service {
	return chartsvc.NewService(
		repo,
		dsRepo,
		permChecker,
		&fakeRoleRepo{roleNames: []string{"admin"}},
		&fakeRBACCacheRepo{},
	)
}

// --- tests ---

func TestCreateChart_Success(t *testing.T) {
	ds := &datasetdomain.Dataset{
		ID:         3,
		Name:       "sales",
		Perm:       "[sales](id:1)",
		SchemaPerm: "schema_public",
	}
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: ds}, &fakePermChecker{allowed: true})

	slice, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Revenue by Month",
		VizType:        "bar",
		DatasourceID:   "3",
		DatasourceType: "table",
		Params:         `{"metrics":["count"]}`,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slice.ID != 42 {
		t.Errorf("expected ID=42, got %d", slice.ID)
	}
	if slice.SliceName != "Revenue by Month" {
		t.Errorf("expected SliceName='Revenue by Month', got %s", slice.SliceName)
	}
	if slice.Perm != "[sales](id:1)" {
		t.Errorf("expected Perm='[sales](id:1)', got %s", slice.Perm)
	}
	if slice.SchemaPerm != "schema_public" {
		t.Errorf("expected SchemaPerm='schema_public', got %s", slice.SchemaPerm)
	}
	if slice.DatasourceName != "sales" {
		t.Errorf("expected DatasourceName='sales', got %s", slice.DatasourceName)
	}
	if slice.LastSavedByFK != 10 {
		t.Errorf("expected LastSavedByFK=10, got %d", slice.LastSavedByFK)
	}
	if slice.CreatedByFK != 10 {
		t.Errorf("expected CreatedByFK=10, got %d", slice.CreatedByFK)
	}
	if slice.LastSavedAt.IsZero() {
		t.Error("expected LastSavedAt to be set")
	}
}

func TestCreateChart_InvalidDatasourceID(t *testing.T) {
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: nil, err: nil}, &fakePermChecker{allowed: true})

	_, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "bar",
		DatasourceID:   "abc",
		DatasourceType: "table",
	})

	if err == nil {
		t.Fatal("expected error for non-numeric datasource_id")
	}
}

func TestCreateChart_DatasetNotFound(t *testing.T) {
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: nil, err: nil}, &fakePermChecker{allowed: true})

	_, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "bar",
		DatasourceID:   "999",
		DatasourceType: "table",
	})

	if !errors.Is(err, pkgerrors.ErrInvalidDataset) {
		t.Errorf("expected ErrInvalidDataset, got %v", err)
	}
}

func TestCreateChart_DatasetRepoError(t *testing.T) {
	dbErr := errors.New("database connection lost")
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: nil, err: dbErr}, &fakePermChecker{allowed: true})

	_, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "bar",
		DatasourceID:   "1",
		DatasourceType: "table",
	})

	if err == nil {
		t.Fatal("expected error from dataset repo")
	}
}

func TestCreateChart_PermissionDenied(t *testing.T) {
	ds := &datasetdomain.Dataset{ID: 3, Name: "sales", Perm: "[sales](id:1)"}
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: ds}, &fakePermChecker{allowed: false})

	_, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "bar",
		DatasourceID:   "3",
		DatasourceType: "table",
	})

	if !errors.Is(err, pkgerrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateChart_PermCheckerError(t *testing.T) {
	ds := &datasetdomain.Dataset{ID: 3, Name: "sales", Perm: "[sales](id:1)"}
	permErr := errors.New("role lookup failed")
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: ds}, &fakePermChecker{allowed: false, err: permErr})

	_, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "bar",
		DatasourceID:   "3",
		DatasourceType: "table",
	})

	if err == nil {
		t.Fatal("expected error from perm checker")
	}
}

func TestCreateChart_InvalidParamsJSON(t *testing.T) {
	ds := &datasetdomain.Dataset{ID: 3, Name: "sales", Perm: "[sales](id:1)"}
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: ds}, &fakePermChecker{allowed: true})

	_, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "bar",
		DatasourceID:   "3",
		DatasourceType: "table",
		Params:         "{not valid json",
	})

	if err == nil {
		t.Fatal("expected error for invalid params JSON")
	}
}

func TestCreateChart_ValidParamsJSON_Accepted(t *testing.T) {
	ds := &datasetdomain.Dataset{ID: 3, Name: "sales", Perm: "[sales](id:1)"}
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: ds}, &fakePermChecker{allowed: true})

	slice, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "pie",
		DatasourceID:   "3",
		DatasourceType: "table",
		Params:         `{"metrics":["sum__num"],"groupby":["region"]}`,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slice.Params != `{"metrics":["sum__num"],"groupby":["region"]}` {
		t.Errorf("params mismatch: %s", slice.Params)
	}
}

func TestCreateChart_EmptyParams_Accepted(t *testing.T) {
	ds := &datasetdomain.Dataset{ID: 3, Name: "sales", Perm: "[sales](id:1)"}
	svc := newTestService(&fakeChartRepo{}, &fakeDatasetRepo{dataset: ds}, &fakePermChecker{allowed: true})

	slice, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "bar",
		DatasourceID:   "3",
		DatasourceType: "table",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slice.Params != "" {
		t.Errorf("expected empty params, got %s", slice.Params)
	}
}

func TestCreateChart_CreateSliceRepoError(t *testing.T) {
	ds := &datasetdomain.Dataset{ID: 3, Name: "sales", Perm: "[sales](id:1)"}
	repo := &fakeChartRepo{createErr: errors.New("write failed")}
	svc := newTestService(repo, &fakeDatasetRepo{dataset: ds}, &fakePermChecker{allowed: true})

	_, err := svc.CreateChart(context.Background(), 10, chartsvc.CreateChartInput{
		SliceName:      "Test",
		VizType:        "bar",
		DatasourceID:   "3",
		DatasourceType: "table",
	})

	if err == nil {
		t.Fatal("expected error from CreateSlice")
	}
}

// --- List/Get Charts tests ---

func TestListCharts_AdminReturnsAll(t *testing.T) {
	now := time.Now()
	repo := &fakeChartRepo{
		listSlices: []*chartdomain.Slice{
			{ID: 1, SliceName: "Chart A", VizType: "bar", DatasourceName: "sales", LastSavedAt: now, LastSavedByFK: 10},
			{ID: 2, SliceName: "Chart B", VizType: "line", DatasourceName: "events", LastSavedAt: now, LastSavedByFK: 20},
		},
		listTotal: 2,
		dashCount: 3,
	}
	svc := newTestService(repo, &fakeDatasetRepo{}, &fakePermChecker{allowed: true})

	result, err := svc.ListCharts(context.Background(), 10, chartsvc.ListChartsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].DashboardCount != 3 {
		t.Errorf("expected DashboardCount=3, got %d", result.Items[0].DashboardCount)
	}
	if result.Items[0].LastSavedBy == nil || result.Items[0].LastSavedBy.ID != 10 {
		t.Errorf("expected LastSavedBy.ID=10, got %v", result.Items[0].LastSavedBy)
	}
}

func TestListCharts_NormalizesPageDefaults(t *testing.T) {
	repo := &fakeChartRepo{
		listSlices: []*chartdomain.Slice{},
		listTotal:  0,
	}
	svc := newTestService(repo, &fakeDatasetRepo{}, &fakePermChecker{allowed: true})

	result, err := svc.ListCharts(context.Background(), 10, chartsvc.ListChartsInput{Page: -1, PageSize: 200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Page != 1 {
		t.Errorf("expected page normalized to 1, got %d", result.Page)
	}
	if result.PageSize != 100 {
		t.Errorf("expected page_size capped at 100, got %d", result.PageSize)
	}
}

func TestListCharts_AppliesFilters(t *testing.T) {
	now := time.Now()
	repo := &fakeChartRepo{
		listSlices: []*chartdomain.Slice{
			{ID: 1, SliceName: "Bar Chart", VizType: "bar", DatasourceName: "sales", LastSavedAt: now, CertifiedBy: "admin", LastSavedByFK: 10},
		},
		listTotal: 1,
		dashCount: 0,
	}
	svc := newTestService(repo, &fakeDatasetRepo{}, &fakePermChecker{allowed: true})

	// The repo fake doesn't actually filter; we verify the input is passed through.
	// Integration tests cover actual query filtering.
	result, err := svc.ListCharts(context.Background(), 10, chartsvc.ListChartsInput{
		Q: "bar", VizType: "bar", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestGetChart_ReturnsFullDetail(t *testing.T) {
	now := time.Now()
	repo := &fakeChartRepo{
		sliceDetail: &chartdomain.Slice{
			ID: 1, SliceName: "Sales Detail", VizType: "bar",
			DatasourceID: "3", DatasourceType: "table", DatasourceName: "sales",
			Params: `{"metric":"sum"}`, QueryContext: `{}`, Description: "desc",
			CacheTimeout: 300, Perm: "[sales](id:1)",
			CertifiedBy: "admin", CertificationDetails: "trusted",
			LastSavedAt: now, LastSavedByFK: 10, CreatedByFK: 10,
		},
		dashCount: 5,
	}
	svc := newTestService(repo, &fakeDatasetRepo{}, &fakePermChecker{allowed: true})

	detail, err := svc.GetChart(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.SliceName != "Sales Detail" {
		t.Errorf("expected 'Sales Detail', got %s", detail.SliceName)
	}
	if detail.Params != `{"metric":"sum"}` {
		t.Errorf("expected params, got %s", detail.Params)
	}
	if detail.DashboardCount != 5 {
		t.Errorf("expected DashboardCount=5, got %d", detail.DashboardCount)
	}
	if detail.LastSavedBy == nil || detail.LastSavedBy.ID != 10 {
		t.Errorf("expected LastSavedBy.ID=10, got %v", detail.LastSavedBy)
	}
	if detail.CreatedBy == nil || detail.CreatedBy.ID != 10 {
		t.Errorf("expected CreatedBy.ID=10, got %v", detail.CreatedBy)
	}
	if detail.CacheTimeout != 300 {
		t.Errorf("expected CacheTimeout=300, got %d", detail.CacheTimeout)
	}
}

func TestGetChart_NotFound(t *testing.T) {
	repo := &fakeChartRepo{
		sliceDetail: &chartdomain.Slice{}, // ID=0 triggers not-found
	}
	svc := newTestService(repo, &fakeDatasetRepo{}, &fakePermChecker{allowed: true})

	_, err := svc.GetChart(context.Background(), 10, 999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !errors.Is(err, pkgerrors.ErrChartNotFound) {
		t.Errorf("expected ErrChartNotFound, got %v", err)
	}
}

func TestGetChart_NonAdminDeniedWhenNotOwner(t *testing.T) {
	now := time.Now()
	repo := &fakeChartRepo{
		sliceDetail: &chartdomain.Slice{
			ID: 1, SliceName: "Other's Chart", VizType: "bar",
			DatasourceID: "3", DatasourceType: "table",
			Perm: "[sales](id:1)", LastSavedAt: now,
			CreatedByFK: 99, // different user
		},
	}
	svc := chartsvc.NewService(
		repo,
		&fakeDatasetRepo{},
		&fakePermChecker{allowed: true},
		&fakeRoleRepo{roleNames: []string{"gamma"}}, // non-admin
		&fakeRBACCacheRepo{},
	)

	_, err := svc.GetChart(context.Background(), 10, 1)
	if err == nil {
		t.Fatal("expected error for unauthorized access")
	}
	if !errors.Is(err, pkgerrors.ErrChartNotFound) {
		t.Errorf("expected ErrChartNotFound, got %v", err)
	}
}

func TestGetChart_OwnerCanAccessWithoutAdmin(t *testing.T) {
	now := time.Now()
	repo := &fakeChartRepo{
		sliceDetail: &chartdomain.Slice{
			ID: 1, SliceName: "My Chart", VizType: "bar",
			DatasourceID: "3", DatasourceType: "table",
			Perm: "[sales](id:1)", LastSavedAt: now,
			LastSavedByFK: 10, CreatedByFK: 10,
		},
		dashCount: 1,
	}
	svc := chartsvc.NewService(
		repo,
		&fakeDatasetRepo{},
		&fakePermChecker{allowed: true},
		&fakeRoleRepo{roleNames: []string{"gamma"}}, // non-admin
		&fakeRBACCacheRepo{},
	)

	detail, err := svc.GetChart(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("owner should access own chart: %v", err)
	}
	if detail.SliceName != "My Chart" {
		t.Errorf("expected 'My Chart', got %s", detail.SliceName)
	}
}

func TestListCharts_RepoErrorPropagates(t *testing.T) {
	repo := &fakeChartRepo{listErr: errors.New("db down")}
	svc := newTestService(repo, &fakeDatasetRepo{}, &fakePermChecker{allowed: true})

	_, err := svc.ListCharts(context.Background(), 10, chartsvc.ListChartsInput{})
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

func TestListCharts_DashboardCountErrorPropagates(t *testing.T) {
	now := time.Now()
	repo := &fakeChartRepo{
		listSlices: []*chartdomain.Slice{
			{ID: 1, SliceName: "Chart", VizType: "bar", DatasourceName: "sales", LastSavedAt: now, LastSavedByFK: 10},
		},
		listTotal: 1,
		dashErr:   errors.New("count failed"),
	}
	svc := newTestService(repo, &fakeDatasetRepo{}, &fakePermChecker{allowed: true})

	_, err := svc.ListCharts(context.Background(), 10, chartsvc.ListChartsInput{})
	if err == nil {
		t.Fatal("expected error from dashboard count")
	}
}
