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

type handlerRoleRepo struct {
	isAdmin         bool
	roles           []domain.RoleListItem
	userCountByRole map[uint]int64
	builtInByRole   map[uint]bool
}

func (f *handlerRoleRepo) IsAdmin(_ context.Context, _ uint) (bool, error) { return f.isAdmin, nil }
func (f *handlerRoleRepo) ListWithCounts(_ context.Context) ([]domain.RoleListItem, error) {
	return f.roles, nil
}
func (f *handlerRoleRepo) Create(_ context.Context, role *domain.Role) error {
	role.ID = 99
	return nil
}
func (f *handlerRoleRepo) UpdateName(_ context.Context, roleID uint, name string) (*domain.Role, error) {
	return &domain.Role{ID: roleID, Name: name}, nil
}
func (f *handlerRoleRepo) CountUsersByRole(_ context.Context, roleID uint) (int64, error) {
	if f.userCountByRole == nil {
		return 0, nil
	}
	return f.userCountByRole[roleID], nil
}
func (f *handlerRoleRepo) IsBuiltInRole(_ context.Context, roleID uint) (bool, error) {
	if f.builtInByRole == nil {
		return false, nil
	}
	return f.builtInByRole[roleID], nil
}
func (f *handlerRoleRepo) Delete(_ context.Context, _ uint) error { return nil }

type handlerCacheRepo struct{}

func (c *handlerCacheRepo) BustRBAC(_ context.Context) error { return nil }

func newRoleRouter(repo *handlerRoleRepo) *gin.Engine {
	svc := svcauth.NewRoleService(repo, &handlerCacheRepo{})
	h := httpauth.NewRoleHandler(svc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domain.UserContext{ID: 1, Active: true})
		c.Next()
	})

	v1 := r.Group("/api/v1")
	admin := v1.Group("/admin")
	admin.GET("/roles", h.List)
	admin.POST("/roles", h.Create)
	admin.PUT("/roles/:id", h.Update)
	admin.DELETE("/roles/:id", h.Delete)
	return r
}

func TestRoleHandler_PostReturns201(t *testing.T) {
	r := newRoleRouter(&handlerRoleRepo{isAdmin: true})
	payload := []byte(`{"name":"Analyst"}`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/roles", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRoleHandler_DeleteWithUsersReturns409(t *testing.T) {
	r := newRoleRouter(&handlerRoleRepo{isAdmin: true, userCountByRole: map[uint]int64{7: 1}})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/roles/7", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestRoleHandler_DeleteBuiltInReturns403(t *testing.T) {
	r := newRoleRouter(&handlerRoleRepo{isAdmin: true, builtInByRole: map[uint]bool{3: true}})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/roles/3", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRoleHandler_GetListWithCounts(t *testing.T) {
	r := newRoleRouter(&handlerRoleRepo{isAdmin: true, roles: []domain.RoleListItem{{
		ID: 1, Name: "Admin", UserCount: 1, PermissionCount: 10, BuiltIn: true,
	}}})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Data []domain.RoleListItem `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].PermissionCount != 10 {
		t.Fatalf("expected list with counts, got %+v", body.Data)
	}
}
