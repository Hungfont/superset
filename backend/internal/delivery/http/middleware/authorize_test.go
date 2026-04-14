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
