package dataset_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httpdataset "superset/auth-service/internal/delivery/http/dataset"
	"superset/auth-service/internal/delivery/http/middleware"
	domainauth "superset/auth-service/internal/domain/auth"
	domain "superset/auth-service/internal/domain/dataset"

	"github.com/gin-gonic/gin"
)

type fakeDatasetService struct {
	created *domain.CreatePhysicalDatasetResponse
	err     error

	called bool
	req    domain.CreatePhysicalDatasetRequest
	actor  uint

	// Multi-endpoint support
	virtualCreated *domain.CreateVirtualDatasetResponse
	listResult     *domain.DatasetListResult
	detail         *domain.DatasetDetail
	updateResult   *domain.UpdateDatasetMetadataResponse
	metrics        []domain.SqlMetric
	metricCreated  *domain.CreateMetricResponse
	refreshResult  *domain.RefreshDatasetResponse
	deleteResult   *domain.DeleteDatasetResponse
	deletedKeys    int64
}

func (f *fakeDatasetService) CreatePhysicalDataset(_ context.Context, actorUserID uint, req domain.CreatePhysicalDatasetRequest) (*domain.CreatePhysicalDatasetResponse, error) {
	f.called = true
	f.req = req
	f.actor = actorUserID
	if f.err != nil {
		return nil, f.err
	}
	if f.created == nil {
		return &domain.CreatePhysicalDatasetResponse{ID: 42, TableName: "orders", BackgroundSync: true}, nil
	}
	copyValue := *f.created
	return &copyValue, nil
}

func (f *fakeDatasetService) CreateVirtualDataset(_ context.Context, _ uint, _ domain.CreateVirtualDatasetRequest) (*domain.CreateVirtualDatasetResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.virtualCreated == nil {
		return &domain.CreateVirtualDatasetResponse{ID: 43, TableName: "v_dataset", BackgroundSync: true}, nil
	}
	copyValue := *f.virtualCreated
	return &copyValue, nil
}

func (f *fakeDatasetService) ListDatasets(_ context.Context, _ uint, _ domain.DatasetListQuery) (*domain.DatasetListResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.listResult == nil {
		return &domain.DatasetListResult{}, nil
	}
	return f.listResult, nil
}

func (f *fakeDatasetService) GetDatasetDetail(_ context.Context, _ uint, _ uint) (*domain.DatasetDetail, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.detail, nil
}

func (f *fakeDatasetService) UpdateDatasetMetadata(_ context.Context, _ uint, _ uint, _ domain.UpdateDatasetMetadataRequest) (*domain.UpdateDatasetMetadataResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.updateResult == nil {
		return &domain.UpdateDatasetMetadataResponse{ID: 1, TableName: "updated"}, nil
	}
	return f.updateResult, nil
}

func (f *fakeDatasetService) UpdateColumn(_ context.Context, _ uint, _ uint, _ uint, _ domain.UpdateColumnRequest) (*domain.UpdateColumnResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.UpdateColumnResponse{ID: 1}, nil
}

func (f *fakeDatasetService) BulkUpdateColumns(_ context.Context, _ uint, _ uint, _ domain.BulkUpdateColumnRequest) (*domain.BulkUpdateColumnResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.BulkUpdateColumnResponse{UpdatedCount: 1}, nil
}

func (f *fakeDatasetService) GetMetrics(_ context.Context, _ uint, _ uint) ([]domain.SqlMetric, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.metrics == nil {
		return []domain.SqlMetric{}, nil
	}
	return f.metrics, nil
}

func (f *fakeDatasetService) CreateMetric(_ context.Context, _ uint, _ uint, _ domain.CreateMetricRequest) (*domain.CreateMetricResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.metricCreated == nil {
		return &domain.CreateMetricResponse{ID: 1}, nil
	}
	return f.metricCreated, nil
}

func (f *fakeDatasetService) UpdateMetric(_ context.Context, _ uint, _ uint, _ uint, _ domain.UpdateMetricRequest) (*domain.UpdateMetricResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.UpdateMetricResponse{ID: 1}, nil
}

func (f *fakeDatasetService) DeleteMetric(_ context.Context, _ uint, _ uint, _ uint) (*domain.DeleteMetricResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.DeleteMetricResponse{}, nil
}

func (f *fakeDatasetService) BulkUpdateMetrics(_ context.Context, _ uint, _ uint, _ domain.BulkUpdateMetricsRequest) (*domain.BulkUpdateMetricsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.BulkUpdateMetricsResponse{UpdatedCount: 1}, nil
}

func (f *fakeDatasetService) DeleteDataset(_ context.Context, _ uint, _ uint, _ domain.DeleteDatasetRequest) (*domain.DeleteDatasetResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.deleteResult == nil {
		return &domain.DeleteDatasetResponse{Deleted: true}, nil
	}
	return f.deleteResult, nil
}

func (f *fakeDatasetService) RefreshDataset(_ context.Context, _ uint, _ uint) (*domain.RefreshDatasetResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.refreshResult == nil {
		return &domain.RefreshDatasetResponse{JobID: "job-1", BackgroundSync: true}, nil
	}
	return f.refreshResult, nil
}

func (f *fakeDatasetService) FlushCache(_ context.Context, _ uint) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.deletedKeys, nil
}

type fakeWSHandler struct{}

func newDatasetRouter(svc *fakeDatasetService) *gin.Engine {
	h := httpdataset.NewHandler(svc, svc, svc, svc, svc, svc, svc, svc, svc, svc, svc, svc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domainauth.UserContext{ID: 10, Active: true})
		c.Next()
	})

	r.POST("/api/v1/datasets", h.CreatePhysicalDataset)
	r.POST("/api/v1/datasets/virtual", h.CreateVirtualDataset)
	r.GET("/api/v1/datasets", h.ListDatasets)
	r.GET("/api/v1/datasets/:id", h.GetDataset)
	r.PUT("/api/v1/datasets/:id", h.UpdateDataset)
	r.PUT("/api/v1/datasets/:id/columns/:col_id", h.UpdateColumn)
	r.PUT("/api/v1/datasets/:id/columns", h.BulkUpdateColumns)
	r.GET("/api/v1/datasets/:id/metrics", h.GetMetrics)
	r.POST("/api/v1/datasets/:id/metrics", h.CreateMetric)
	r.PUT("/api/v1/datasets/:id/metrics/:metric_id", h.UpdateMetric)
	r.DELETE("/api/v1/datasets/:id/metrics/:metric_id", h.DeleteMetric)
	r.PUT("/api/v1/datasets/:id/metrics", h.BulkUpdateMetrics)
	r.DELETE("/api/v1/datasets/:id", h.DeleteDataset)
	r.POST("/api/v1/datasets/:id/refresh", h.RefreshDataset)
	r.POST("/api/v1/datasets/:id/cache/flush", h.FlushCache)
	return r
}

func TestDatasetHandler_CreatePhysicalDatasetReturns201(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{})

	payload := []byte(`{"database_id":7,"schema":"public","table_name":"orders"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"background_sync":true`)) {
		t.Fatalf("expected background_sync true response, got %s", w.Body.String())
	}
}

func TestDatasetHandler_CreatePhysicalDatasetReturns403ForGamma(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{err: domain.ErrForbidden})

	payload := []byte(`{"database_id":7,"schema":"public","table_name":"orders"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatasetHandler_CreatePhysicalDatasetReturns409ForDuplicate(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{err: domain.ErrDatasetDuplicate})

	payload := []byte(`{"database_id":7,"schema":"public","table_name":"orders"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatasetHandler_CreatePhysicalDatasetReturns422ForInvalidDatabase(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{err: domain.ErrInvalidDatabase})

	payload := []byte(`{"database_id":999,"schema":"public","table_name":"orders"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatasetHandler_CreatePhysicalDatasetReturns401WithoutActor(t *testing.T) {
	svc := &fakeDatasetService{}
	h := httpdataset.NewHandler(svc, svc, svc, svc, svc, svc, svc, svc, svc, svc, svc, svc)
	r := gin.New()
	r.POST("/api/v1/datasets", h.CreatePhysicalDataset)

	payload := []byte(`{"database_id":7,"schema":"public","table_name":"orders"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatasetHandler_CreatePhysicalDatasetBindsRequestPayload(t *testing.T) {
	svc := &fakeDatasetService{created: &domain.CreatePhysicalDatasetResponse{ID: 11, TableName: "events", BackgroundSync: true}}
	r := newDatasetRouter(svc)

	payload := []byte(`{"database_id":5,"schema":"core","table_name":"events"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !svc.called {
		t.Fatal("expected service called")
	}
	if svc.actor != 10 {
		t.Fatalf("expected actor id 10, got %d", svc.actor)
	}
	if svc.req.DatabaseID != 5 || svc.req.Schema != "core" || svc.req.TableName != "events" {
		t.Fatalf("unexpected payload bound: %+v", svc.req)
	}
}

func TestDatasetHandler_CreatePhysicalDatasetReturns500ForUnexpectedError(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{err: errors.New("boom")})

	payload := []byte(`{"database_id":7,"schema":"public","table_name":"orders"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatasetHandler_CreateVirtualDatasetReturns201(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{})

	payload := []byte(`{"database_id":7,"table_name":"my_view","sql":"SELECT * FROM orders"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets/virtual", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatasetHandler_GetMetricsReturns200(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{
		metrics: []domain.SqlMetric{
			{ID: 1, MetricName: "total_count", MetricType: "COUNT", Expression: "COUNT(*)"},
		},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/datasets/1/metrics", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"total_count"`)) {
		t.Fatalf("expected metric in response, got %s", w.Body.String())
	}
}

func TestDatasetHandler_DeleteDatasetReturns204(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/datasets/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatasetHandler_DeleteDatasetReturns409WhenReferenced(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{err: domain.ErrDatasetReferencedByCharts})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/datasets/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatasetHandler_RefreshDatasetReturns202(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets/1/refresh", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"job_id"`)) {
		t.Fatalf("expected job_id in response, got %s", w.Body.String())
	}
}

func TestDatasetHandler_FlushCacheReturns200(t *testing.T) {
	r := newDatasetRouter(&fakeDatasetService{deletedKeys: 5})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datasets/1/cache/flush", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"keys_deleted":5`)) {
		t.Fatalf("expected keys_deleted 5 response, got %s", w.Body.String())
	}
}
