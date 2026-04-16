package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"superset/auth-service/internal/delivery/http/middleware"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type authorizeRoleRepo struct {
	isAdmin bool
	err     error
}

func (r *authorizeRoleRepo) IsAdmin(_ context.Context, _ uint) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	return r.isAdmin, nil
}

func (r *authorizeRoleRepo) ListWithCounts(_ context.Context) ([]domain.RoleListItem, error) {
	return nil, nil
}

func (r *authorizeRoleRepo) Create(_ context.Context, _ *domain.Role) error { return nil }

func (r *authorizeRoleRepo) UpdateName(_ context.Context, roleID uint, name string) (*domain.Role, error) {
	return &domain.Role{ID: roleID, Name: name}, nil
}

func (r *authorizeRoleRepo) CountUsersByRole(_ context.Context, _ uint) (int64, error) { return 0, nil }

func (r *authorizeRoleRepo) IsBuiltInRole(_ context.Context, _ uint) (bool, error) { return false, nil }

func (r *authorizeRoleRepo) Delete(_ context.Context, _ uint) error { return nil }

func (r *authorizeRoleRepo) RoleExists(_ context.Context, _ uint) (bool, error) { return true, nil }

func (r *authorizeRoleRepo) ListPermissionViewIDsByRole(_ context.Context, _ uint) ([]uint, error) {
	return []uint{}, nil
}

func (r *authorizeRoleRepo) CountExistingPermissionViews(_ context.Context, _ []uint) (int64, error) {
	return 0, nil
}

func (r *authorizeRoleRepo) ReplacePermissionViews(_ context.Context, _ uint, _ []uint) error {
	return nil
}

func (r *authorizeRoleRepo) AddPermissionViews(_ context.Context, _ uint, _ []uint) error { return nil }

func (r *authorizeRoleRepo) RemovePermissionView(_ context.Context, _ uint, _ uint) error { return nil }

type authorizePermissionRepo struct {
	tuples []domain.PermissionTuple
	err    error
	calls  int
}

func (r *authorizePermissionRepo) ListPermissionTuplesByUser(_ context.Context, _ uint) ([]domain.PermissionTuple, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return r.tuples, nil
}

type authorizePermissionCacheRepo struct {
	values    []string
	getErr    error
	setErr    error
	getCalls  int
	setCalls  int
	lastValue []string
}

func (r *authorizePermissionCacheRepo) GetPermissionSet(_ context.Context, _ uint) ([]string, error) {
	r.getCalls++
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.values, nil
}

func (r *authorizePermissionCacheRepo) SetPermissionSet(_ context.Context, _ uint, values []string) error {
	r.setCalls++
	r.lastValue = values
	if r.setErr != nil {
		return r.setErr
	}
	return nil
}

func setupAuthorizeRouter(repo domain.RoleRepository, withActor bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if withActor {
		r.Use(func(c *gin.Context) {
			c.Set(middleware.UserContextKey, domain.UserContext{ID: 1, Active: true})
			c.Next()
		})
	}
	r.Use(middleware.AuthorizeAdminRole(repo))
	r.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": true})
	})
	return r
}

func setupPermissionRouter(
	roleRepo domain.RoleRepository,
	permissionRepo domain.RBACPermissionRepository,
	cacheRepo domain.RBACPermissionCacheRepository,
	withActor bool,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if withActor {
		r.Use(func(c *gin.Context) {
			c.Set(middleware.UserContextKey, domain.UserContext{ID: 1, Active: true})
			c.Next()
		})
	}
	r.Use(middleware.RequirePermission(roleRepo, permissionRepo, cacheRepo, "can_read", "Dashboard"))
	r.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": true})
	})
	return r
}

func TestAuthorizeAdminRole_ForbidsNonAdmin(t *testing.T) {
	r := setupAuthorizeRouter(&authorizeRoleRepo{isAdmin: false}, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuthorizeAdminRole_AllowsAdmin(t *testing.T) {
	r := setupAuthorizeRouter(&authorizeRoleRepo{isAdmin: true}, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthorizeAdminRole_WithoutActorReturns401(t *testing.T) {
	r := setupAuthorizeRouter(&authorizeRoleRepo{isAdmin: true}, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthorizeAdminRole_RepoFailureReturns500(t *testing.T) {
	r := setupAuthorizeRouter(&authorizeRoleRepo{err: errors.New("db down")}, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequirePermission_AllowsAssignedTuple(t *testing.T) {
	roleRepo := &authorizeRoleRepo{isAdmin: false}
	permissionRepo := &authorizePermissionRepo{tuples: []domain.PermissionTuple{{Action: "can_read", Resource: "Dashboard"}}}
	cacheRepo := &authorizePermissionCacheRepo{}
	r := setupPermissionRouter(roleRepo, permissionRepo, cacheRepo, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, permissionRepo.calls)
	assert.Equal(t, 1, cacheRepo.setCalls)
}

func TestRequirePermission_UsesCacheHit(t *testing.T) {
	roleRepo := &authorizeRoleRepo{isAdmin: false}
	permissionRepo := &authorizePermissionRepo{}
	cacheRepo := &authorizePermissionCacheRepo{values: []string{"can_read:dashboard"}}
	r := setupPermissionRouter(roleRepo, permissionRepo, cacheRepo, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, permissionRepo.calls)
	assert.Equal(t, 0, cacheRepo.setCalls)
}

func TestRequirePermission_UsesEmptyCachedSetAsHit(t *testing.T) {
	roleRepo := &authorizeRoleRepo{isAdmin: false}
	permissionRepo := &authorizePermissionRepo{}
	cacheRepo := &authorizePermissionCacheRepo{values: []string{}}
	r := setupPermissionRouter(roleRepo, permissionRepo, cacheRepo, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, 0, permissionRepo.calls)
	assert.Equal(t, 0, cacheRepo.setCalls)
}

func TestRequirePermission_DeniesWhenTupleMissing(t *testing.T) {
	roleRepo := &authorizeRoleRepo{isAdmin: false}
	permissionRepo := &authorizePermissionRepo{tuples: []domain.PermissionTuple{{Action: "can_write", Resource: "Dashboard"}}}
	cacheRepo := &authorizePermissionCacheRepo{}
	r := setupPermissionRouter(roleRepo, permissionRepo, cacheRepo, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_AdminBypassesTupleCheck(t *testing.T) {
	roleRepo := &authorizeRoleRepo{isAdmin: true}
	permissionRepo := &authorizePermissionRepo{}
	cacheRepo := &authorizePermissionCacheRepo{}
	r := setupPermissionRouter(roleRepo, permissionRepo, cacheRepo, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, permissionRepo.calls)
	assert.Equal(t, 0, cacheRepo.getCalls)
}

func TestRequirePermission_WithoutActorReturns401(t *testing.T) {
	roleRepo := &authorizeRoleRepo{isAdmin: false}
	permissionRepo := &authorizePermissionRepo{}
	cacheRepo := &authorizePermissionCacheRepo{}
	r := setupPermissionRouter(roleRepo, permissionRepo, cacheRepo, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePermission_RoleRepoErrorReturns500(t *testing.T) {
	roleRepo := &authorizeRoleRepo{err: errors.New("db down")}
	permissionRepo := &authorizePermissionRepo{}
	cacheRepo := &authorizePermissionCacheRepo{}
	r := setupPermissionRouter(roleRepo, permissionRepo, cacheRepo, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequirePermission_PermissionRepoErrorReturns500(t *testing.T) {
	roleRepo := &authorizeRoleRepo{isAdmin: false}
	permissionRepo := &authorizePermissionRepo{err: errors.New("db down")}
	cacheRepo := &authorizePermissionCacheRepo{}
	r := setupPermissionRouter(roleRepo, permissionRepo, cacheRepo, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
