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

func newDatasetRouter(svc *fakeDatasetService) *gin.Engine {
	h := httpdataset.NewHandler(svc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domainauth.UserContext{ID: 10, Active: true})
		c.Next()
	})

	r.POST("/api/v1/datasets", h.CreatePhysicalDataset)
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
	h := httpdataset.NewHandler(&fakeDatasetService{})
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
