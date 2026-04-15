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

type handlerUserRoleRepo struct {
	isAdmin bool

	roleIDsByUser map[uint][]uint
	validRoleIDs  map[uint]bool
}

func (f *handlerUserRoleRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	return f.isAdmin, nil
}

func (f *handlerUserRoleRepo) ListRoleIDsByUser(_ context.Context, userID uint) ([]uint, error) {
	if f.roleIDsByUser == nil {
		return []uint{}, nil
	}
	ids := f.roleIDsByUser[userID]
	cloned := make([]uint, len(ids))
	copy(cloned, ids)
	return cloned, nil
}

func (f *handlerUserRoleRepo) CountExistingRoles(_ context.Context, roleIDs []uint) (int64, error) {
	if f.validRoleIDs == nil {
		return int64(len(roleIDs)), nil
	}
	count := int64(0)
	for _, roleID := range roleIDs {
		if f.validRoleIDs[roleID] {
			count++
		}
	}
	return count, nil
}

func (f *handlerUserRoleRepo) ReplaceUserRoles(_ context.Context, userID uint, roleIDs []uint) error {
	if f.roleIDsByUser == nil {
		f.roleIDsByUser = map[uint][]uint{}
	}
	f.roleIDsByUser[userID] = append([]uint{}, roleIDs...)
	return nil
}

type handlerUserRoleCacheRepo struct{}

func (h *handlerUserRoleCacheRepo) BustRBAC(_ context.Context) error {
	return nil
}

func (h *handlerUserRoleCacheRepo) BustRBACForUser(_ context.Context, _ uint) error {
	return nil
}

func newUserRoleRouter(repo *handlerUserRoleRepo, withAuthorizeMiddleware bool) *gin.Engine {
	svc := svcauth.NewUserRoleService(repo, &handlerUserRoleCacheRepo{})
	h := httpauth.NewUserRoleHandler(svc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domain.UserContext{ID: 1, Active: true})
		c.Next()
	})

	admin := r.Group("/api/v1/admin")
	if withAuthorizeMiddleware {
		admin.Use(func(c *gin.Context) {
			c.Next()
		})
	}
	admin.GET("/users/:id/roles", h.List)
	admin.PUT("/users/:id/roles", h.Set)

	return r
}

func TestUserRoleHandler_PutReturns200(t *testing.T) {
	r := newUserRoleRouter(&handlerUserRoleRepo{
		isAdmin: true,
		validRoleIDs: map[uint]bool{
			1: true,
			3: true,
		},
	}, false)

	payload := []byte(`{"role_ids":[1,3]}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/users/7/roles", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Data domain.UserRolesPayload `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Data.UserID != 7 || len(body.Data.RoleIDs) != 2 {
		t.Fatalf("unexpected response payload: %+v", body.Data)
	}
}

func TestUserRoleHandler_PutEmptyRolesReturns422(t *testing.T) {
	r := newUserRoleRouter(&handlerUserRoleRepo{isAdmin: true}, false)

	payload := []byte(`{"role_ids":[]}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/users/7/roles", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserRoleHandler_PutInvalidRoleIDReturns422(t *testing.T) {
	r := newUserRoleRouter(&handlerUserRoleRepo{
		isAdmin: true,
		validRoleIDs: map[uint]bool{
			1: true,
		},
	}, false)

	payload := []byte(`{"role_ids":[1,999]}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/users/7/roles", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserRoleHandler_GetReturns200(t *testing.T) {
	r := newUserRoleRouter(&handlerUserRoleRepo{
		isAdmin:       true,
		roleIDsByUser: map[uint][]uint{7: []uint{1, 3}},
	}, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users/7/roles", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserRoleHandler_NonAdminReturns403(t *testing.T) {
	r := newUserRoleRouter(&handlerUserRoleRepo{isAdmin: false}, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/users/7/roles", bytes.NewReader([]byte(`{"role_ids":[1]}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
