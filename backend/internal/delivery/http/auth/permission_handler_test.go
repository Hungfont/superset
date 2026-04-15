package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	httpauth "superset/auth-service/internal/delivery/http/auth"
	"superset/auth-service/internal/delivery/http/middleware"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

type handlerPermissionRepo struct {
	permissionViewUseByID map[uint]int64
	createPermissionErr   error
	createPVErr           error
}

func (f *handlerPermissionRepo) ListPermissions(_ context.Context) ([]domain.Permission, error) {
	return []domain.Permission{{ID: 1, Name: "can_read"}}, nil
}

func (f *handlerPermissionRepo) CreatePermission(_ context.Context, permission *domain.Permission) error {
	if f.createPermissionErr != nil {
		return f.createPermissionErr
	}
	permission.ID = 1
	return nil
}

func (f *handlerPermissionRepo) ListViewMenus(_ context.Context) ([]domain.ViewMenu, error) {
	return []domain.ViewMenu{{ID: 1, Name: "Dashboard"}}, nil
}

func (f *handlerPermissionRepo) CreateViewMenu(_ context.Context, viewMenu *domain.ViewMenu) error {
	viewMenu.ID = 1
	return nil
}

func (f *handlerPermissionRepo) ListPermissionViews(_ context.Context) ([]domain.PermissionView, error) {
	return []domain.PermissionView{{
		ID:             1,
		PermissionID:   1,
		ViewMenuID:     1,
		PermissionName: "can_read",
		ViewMenuName:   "Dashboard",
	}}, nil
}

func (f *handlerPermissionRepo) CreatePermissionView(_ context.Context, permissionView *domain.PermissionView) error {
	if f.createPVErr != nil {
		return f.createPVErr
	}
	permissionView.ID = 1
	return nil
}

func (f *handlerPermissionRepo) CountRoleAssignmentsByPermissionView(_ context.Context, permissionViewID uint) (int64, error) {
	if f.permissionViewUseByID == nil {
		return 0, nil
	}
	return f.permissionViewUseByID[permissionViewID], nil
}

func (f *handlerPermissionRepo) DeletePermissionView(_ context.Context, _ uint) error {
	return nil
}

func (f *handlerPermissionRepo) SeedPermissionViews(_ context.Context, _ []domain.PermissionViewSeed) error {
	return nil
}

type handlerPermissionCacheRepo struct{}

func (h *handlerPermissionCacheRepo) BustRBAC(_ context.Context) error { return nil }

func (h *handlerPermissionCacheRepo) BustRBACForUser(_ context.Context, _ uint) error { return nil }

func newPermissionRouter(repo *handlerPermissionRepo) *gin.Engine {
	svc := svcauth.NewPermissionService(repo, &handlerPermissionCacheRepo{})
	h := httpauth.NewPermissionHandler(svc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domain.UserContext{ID: 1, Active: true})
		c.Next()
	})

	v1 := r.Group("/api/v1")
	admin := v1.Group("/admin")
	admin.GET("/permission-views", h.ListPermissionViews)
	admin.POST("/permissions", h.CreatePermission)
	admin.POST("/permission-views", h.CreatePermissionView)
	admin.DELETE("/permission-views/:id", h.DeletePermissionView)

	return r
}

func TestPermissionHandler_PostPermissionReturns201(t *testing.T) {
	r := newPermissionRouter(&handlerPermissionRepo{})
	payload := []byte(`{"name":"can_edit"}`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/permissions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPermissionHandler_ListPermissionViewsIncludesNames(t *testing.T) {
	r := newPermissionRouter(&handlerPermissionRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/permission-views", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	type permissionViewResponse struct {
		Data []domain.PermissionView `json:"data"`
	}

	var payload permissionViewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json, got %v", err)
	}

	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 permission view, got %d", len(payload.Data))
	}

	if payload.Data[0].PermissionName != "can_read" || payload.Data[0].ViewMenuName != "Dashboard" {
		t.Fatalf("expected permission/view names for UI, got %+v", payload.Data[0])
	}
}

func TestPermissionHandler_PostPermissionViewDuplicateReturns409(t *testing.T) {
	r := newPermissionRouter(&handlerPermissionRepo{createPVErr: domain.ErrPermissionViewDuplicate})
	payload := []byte(`{"permission_id":1,"view_menu_id":2}`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/permission-views", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPermissionHandler_DeleteAssignedPermissionViewReturns409(t *testing.T) {
	r := newPermissionRouter(&handlerPermissionRepo{permissionViewUseByID: map[uint]int64{5: 1}})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/permission-views/5", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}
