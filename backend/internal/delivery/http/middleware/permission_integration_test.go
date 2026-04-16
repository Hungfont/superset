package middleware_test

import (
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"superset/auth-service/internal/delivery/http/middleware"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupProtectedPermissionRouter(
	pubKey *rsa.PublicKey,
	jwtRepo domain.JWTRepository,
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
	permissionRepo domain.RBACPermissionRepository,
	permissionCacheRepo domain.RBACPermissionCacheRepository,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTMiddleware(pubKey, jwtRepo, userRepo))
	r.GET(
		"/api/v1/admin/users",
		middleware.RequirePermission(roleRepo, permissionRepo, permissionCacheRepo, "can_read", "User"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": true})
		},
	)
	return r
}

func TestTupleProtectedRoute_AllowsWhenPermissionExists(t *testing.T) {
	priv, pub := generateKeyPair(t)
	jwtRepo := newFakeJWTRepo()
	userRepo := &fakeUserRepo{user: &domain.User{ID: 42, Username: "testuser", Email: "user@example.com", Active: true}}
	roleRepo := &authorizeRoleRepo{isAdmin: false}
	permissionRepo := &authorizePermissionRepo{tuples: []domain.PermissionTuple{{Action: "can_read", Resource: "User"}}}
	permissionCacheRepo := &authorizePermissionCacheRepo{}

	r := setupProtectedPermissionRouter(pub, jwtRepo, userRepo, roleRepo, permissionRepo, permissionCacheRepo)
	token := makeToken(t, priv, 42, uuid.NewString(), 15*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
