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

type handlerUserAdminRepo struct {
	isAdmin bool

	users []domain.UserListItem
	user  *domain.UserDetail

	validRoleIDs map[uint]bool

	notFoundOnGet        bool
	notFoundOnUpdate     bool
	notFoundOnDeactivate bool
}

func (h *handlerUserAdminRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	return h.isAdmin, nil
}

func (h *handlerUserAdminRepo) ListUsers(_ context.Context) ([]domain.UserListItem, error) {
	return h.users, nil
}

func (h *handlerUserAdminRepo) GetUserByID(_ context.Context, userID uint) (*domain.UserDetail, error) {
	if h.notFoundOnGet {
		return nil, domain.ErrUserNotFound
	}
	if h.user != nil {
		return h.user, nil
	}
	return &domain.UserDetail{ID: userID, Username: "demo", Email: "demo@example.com", Active: true, RoleIDs: []uint{1}}, nil
}

func (h *handlerUserAdminRepo) CreateUser(_ context.Context, _ domain.CreateUserRequest) (uint, error) {
	return 88, nil
}

func (h *handlerUserAdminRepo) UpdateUser(_ context.Context, _ uint, _ domain.UpdateUserRequest) error {
	if h.notFoundOnUpdate {
		return domain.ErrUserNotFound
	}
	return nil
}

func (h *handlerUserAdminRepo) DeactivateUser(_ context.Context, _ uint) error {
	if h.notFoundOnDeactivate {
		return domain.ErrUserNotFound
	}
	return nil
}

func (h *handlerUserAdminRepo) CountExistingRoles(_ context.Context, roleIDs []uint) (int64, error) {
	if h.validRoleIDs == nil {
		return int64(len(roleIDs)), nil
	}
	count := int64(0)
	for _, roleID := range roleIDs {
		if h.validRoleIDs[roleID] {
			count++
		}
	}
	return count, nil
}

func (h *handlerUserAdminRepo) ReplaceUserRoles(_ context.Context, _ uint, _ []uint) error {
	return nil
}

type handlerUserAdminCacheRepo struct{}

func (h *handlerUserAdminCacheRepo) BustRBAC(_ context.Context) error {
	return nil
}

func (h *handlerUserAdminCacheRepo) BustRBACForUser(_ context.Context, _ uint) error {
	return nil
}

func newUserAdminRouter(repo *handlerUserAdminRepo) *gin.Engine {
	svc := svcauth.NewUserService(repo, &handlerUserAdminCacheRepo{})
	h := httpauth.NewUserHandler(svc)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, domain.UserContext{ID: 1, Active: true})
		c.Next()
	})

	admin := r.Group("/api/v1/admin")
	admin.GET("/users", h.List)
	admin.GET("/users/:id", h.Get)
	admin.POST("/users", h.Create)
	admin.PUT("/users/:id", h.Update)
	admin.DELETE("/users/:id", h.Delete)

	return r
}

func TestUserHandler_GetListReturns200(t *testing.T) {
	r := newUserAdminRouter(&handlerUserAdminRepo{isAdmin: true, users: []domain.UserListItem{{
		ID: 1, Username: "admin", Email: "admin@example.com", Active: true, RoleIDs: []uint{1},
	}}})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_PostReturns201(t *testing.T) {
	r := newUserAdminRouter(&handlerUserAdminRepo{isAdmin: true, validRoleIDs: map[uint]bool{1: true}})

	payload := []byte(`{"first_name":"New","last_name":"User","username":"newuser","email":"new@example.com","password":"StrongPass@123","role_ids":[1]}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Data domain.UserDetail `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Data.ID != 88 {
		t.Fatalf("expected new user id 88, got %d", body.Data.ID)
	}
}

func TestUserHandler_PutInvalidRoleReturns422(t *testing.T) {
	r := newUserAdminRouter(&handlerUserAdminRepo{isAdmin: true, validRoleIDs: map[uint]bool{1: true}})

	payload := []byte(`{"first_name":"Edit","last_name":"User","username":"edituser","email":"edit@example.com","active":true,"role_ids":[1,99]}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/users/5", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_DeleteNotFoundReturns404(t *testing.T) {
	r := newUserAdminRouter(&handlerUserAdminRepo{isAdmin: true, notFoundOnDeactivate: true})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/users/5", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_DoesNotApplyRoleGate(t *testing.T) {
	r := newUserAdminRouter(&handlerUserAdminRepo{isAdmin: false})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
