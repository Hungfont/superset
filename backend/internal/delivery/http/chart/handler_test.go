package chart_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	chartsvc "superset/auth-service/internal/app/chart"
	"superset/auth-service/internal/delivery/http/chart"
	"superset/auth-service/internal/delivery/http/middleware"
	chartdomain "superset/auth-service/internal/domain/chart"
	domainauth "superset/auth-service/internal/domain/auth"
	pkgerrors "superset/auth-service/internal/pkg/autherrors"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

type fakeCreateChartService struct {
	slice *chartdomain.Slice
	err   error
}

func (f *fakeCreateChartService) CreateChart(_ context.Context, _ uint, _ chartsvc.CreateChartInput) (*chartdomain.Slice, error) {
	return f.slice, f.err
}

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

func newRouter(svc *fakeCreateChartService) *gin.Engine {
	h := chart.NewHandler(svc, &fakeListChartsService{}, &fakeGetChartService{})
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domainauth.UserContext{ID: 10, Active: true})
		c.Next()
	})

	r.POST("/api/v1/charts", h.Create)
	r.GET("/api/v1/charts", h.List)
	r.GET("/api/v1/charts/:id", h.Get)
	return r
}

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

func postChart(router *gin.Engine, body interface{}, contentType string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/charts", bytes.NewReader(b))
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)
	router.ServeHTTP(w, req)
	return w
}

func TestCreateChart_Success_Returns201(t *testing.T) {
	now := time.Now()
	svc := &fakeCreateChartService{
		slice: &chartdomain.Slice{
			ID:                   1,
			SliceName:            "Revenue by Month",
			VizType:             "bar",
			DatasourceID:         "3",
			DatasourceType:       "table",
			DatasourceName:       "sales",
			Perm:                "[sales](id:1)",
			LastSavedAt:          now,
			LastSavedByFK:        10,
			CreatedByFK:          10,
		},
	}
	router := newRouter(svc)
	w := postChart(router, map[string]interface{}{
		"slice_name":      "Revenue by Month",
		"viz_type":        "bar",
		"datasource_id":   "3",
		"datasource_type": "table",
	}, "")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data envelope, got %v", resp)
	}
	if data["id"] != float64(1) {
		t.Errorf("expected id=1, got %v", data["id"])
	}
	if data["slice_name"] != "Revenue by Month" {
		t.Errorf("expected slice_name='Revenue by Month', got %v", data["slice_name"])
	}
}

func TestCreateChart_Unauthorized_Returns401(t *testing.T) {
	h := chart.NewHandler(&fakeCreateChartService{}, &fakeListChartsService{}, &fakeGetChartService{})
	r := gin.New()
	r.POST("/api/v1/charts", h.Create)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/charts", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateChart_InvalidJSON_Returns400(t *testing.T) {
	svc := &fakeCreateChartService{}
	router := newRouter(svc)
	w := postChart(router, `{bad json`, "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateChart_Forbidden_Returns403(t *testing.T) {
	svc := &fakeCreateChartService{err: pkgerrors.ErrForbidden}
	router := newRouter(svc)
	w := postChart(router, map[string]interface{}{
		"slice_name":      "Test",
		"viz_type":        "bar",
		"datasource_id":   "1",
		"datasource_type": "table",
	}, "")

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateChart_InvalidDatasource_Returns422(t *testing.T) {
	svc := &fakeCreateChartService{err: errors.New("invalid datasource_id")}
	router := newRouter(svc)
	w := postChart(router, map[string]interface{}{
		"slice_name":      "Test",
		"viz_type":        "bar",
		"datasource_id":   "999",
		"datasource_type": "table",
	}, "")

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateChart_DatasetNotFound_Returns422(t *testing.T) {
	svc := &fakeCreateChartService{err: errors.New("dataset not found: 999")}
	router := newRouter(svc)
	w := postChart(router, map[string]interface{}{
		"slice_name":      "Test",
		"viz_type":        "bar",
		"datasource_id":   "999",
		"datasource_type": "table",
	}, "")

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateChart_InvalidParamsJSON_Returns400(t *testing.T) {
	svc := &fakeCreateChartService{err: errors.New("invalid params JSON")}
	router := newRouter(svc)
	w := postChart(router, map[string]interface{}{
		"slice_name":      "Test",
		"viz_type":        "bar",
		"datasource_id":   "1",
		"datasource_type": "table",
		"params":          "{invalid",
	}, "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateChart_UnexpectedError_Returns500(t *testing.T) {
	svc := &fakeCreateChartService{err: errors.New("database connection lost")}
	router := newRouter(svc)
	w := postChart(router, map[string]interface{}{
		"slice_name":      "Test",
		"viz_type":        "bar",
		"datasource_id":   "1",
		"datasource_type": "table",
	}, "")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

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
