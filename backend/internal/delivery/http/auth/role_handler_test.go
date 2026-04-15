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
	roleExistsByID  map[uint]bool

	rolePermissionIDs      map[uint][]uint
	validPermissionViewIDs map[uint]bool
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

func (f *handlerRoleRepo) RoleExists(_ context.Context, roleID uint) (bool, error) {
	if f.roleExistsByID == nil {
		return true, nil
	}
	exists, ok := f.roleExistsByID[roleID]
	if !ok {
		return false, nil
	}
	return exists, nil
}

func (f *handlerRoleRepo) ListPermissionViewIDsByRole(_ context.Context, roleID uint) ([]uint, error) {
	if f.rolePermissionIDs == nil {
		return []uint{}, nil
	}
	ids := f.rolePermissionIDs[roleID]
	cloned := make([]uint, len(ids))
	copy(cloned, ids)
	return cloned, nil
}

func (f *handlerRoleRepo) CountExistingPermissionViews(_ context.Context, permissionViewIDs []uint) (int64, error) {
	if f.validPermissionViewIDs == nil {
		return int64(len(permissionViewIDs)), nil
	}
	count := int64(0)
	for _, id := range permissionViewIDs {
		if f.validPermissionViewIDs[id] {
			count++
		}
	}
	return count, nil
}

func (f *handlerRoleRepo) ReplacePermissionViews(_ context.Context, roleID uint, permissionViewIDs []uint) error {
	if f.rolePermissionIDs == nil {
		f.rolePermissionIDs = map[uint][]uint{}
	}
	f.rolePermissionIDs[roleID] = append([]uint{}, permissionViewIDs...)
	return nil
}

func (f *handlerRoleRepo) AddPermissionViews(_ context.Context, roleID uint, permissionViewIDs []uint) error {
	if f.rolePermissionIDs == nil {
		f.rolePermissionIDs = map[uint][]uint{}
	}
	existing := f.rolePermissionIDs[roleID]
	for _, candidate := range permissionViewIDs {
		found := false
		for _, current := range existing {
			if current == candidate {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, candidate)
		}
	}
	f.rolePermissionIDs[roleID] = existing
	return nil
}

func (f *handlerRoleRepo) RemovePermissionView(_ context.Context, roleID uint, permissionViewID uint) error {
	if f.rolePermissionIDs == nil {
		return nil
	}
	existing := f.rolePermissionIDs[roleID]
	filtered := make([]uint, 0, len(existing))
	for _, id := range existing {
		if id != permissionViewID {
			filtered = append(filtered, id)
		}
	}
	f.rolePermissionIDs[roleID] = filtered
	return nil
}

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
	admin.GET("/roles/:id/permissions", h.ListPermissions)
	admin.PUT("/roles/:id/permissions", h.SetPermissions)
	admin.POST("/roles/:id/permissions/add", h.AddPermissions)
	admin.DELETE("/roles/:id/permissions/:pv_id", h.RemovePermission)
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

func TestRoleHandler_PutPermissionsReturns422OnInvalidPermissionViewID(t *testing.T) {
	r := newRoleRouter(&handlerRoleRepo{
		isAdmin:                true,
		roleExistsByID:         map[uint]bool{2: true},
		validPermissionViewIDs: map[uint]bool{1: true},
	})

	payload := []byte(`{"permission_view_ids":[1,999]}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/roles/2/permissions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRoleHandler_ListAndMutatePermissions(t *testing.T) {
	r := newRoleRouter(&handlerRoleRepo{
		isAdmin:                true,
		roleExistsByID:         map[uint]bool{5: true},
		rolePermissionIDs:      map[uint][]uint{5: []uint{1, 2}},
		validPermissionViewIDs: map[uint]bool{1: true, 2: true, 3: true},
	})

	listRecorder := httptest.NewRecorder()
	listReq, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/roles/5/permissions", nil)
	r.ServeHTTP(listRecorder, listReq)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from list, got %d", listRecorder.Code)
	}

	addRecorder := httptest.NewRecorder()
	addReq, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/roles/5/permissions/add", bytes.NewReader([]byte(`{"permission_view_ids":[3]}`)))
	addReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(addRecorder, addReq)
	if addRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from add, got %d", addRecorder.Code)
	}

	deleteRecorder := httptest.NewRecorder()
	deleteReq, _ := http.NewRequest(http.MethodDelete, "/api/v1/admin/roles/5/permissions/2", nil)
	r.ServeHTTP(deleteRecorder, deleteReq)
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from delete, got %d", deleteRecorder.Code)
	}
}
