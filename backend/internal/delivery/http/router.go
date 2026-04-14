package http

import (
	"crypto/rsa"

	httpauth "superset/auth-service/internal/delivery/http/auth"
	"superset/auth-service/internal/delivery/http/middleware"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

// NewRouter wires all routes and returns the configured Gin engine.
func NewRouter(
	registerHandler *httpauth.RegisterHandler,
	verifyHandler *httpauth.VerifyHandler,
	loginHandler *httpauth.LoginHandler,
	refreshHandler *httpauth.RefreshHandler,
	logoutHandler *httpauth.LogoutHandler,
	roleHandler *httpauth.RoleHandler,
	pubKey *rsa.PublicKey,
	jwtRepo domain.JWTRepository,
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", registerHandler.Register)
			authGroup.GET("/verify", verifyHandler.Verify)
			authGroup.POST("/login", loginHandler.Login)
			authGroup.POST("/refresh", refreshHandler.Refresh)
			authGroup.POST("/logout", logoutHandler.Logout)
		}

		// Protected routes require a valid JWT.
		protected := v1.Group("/")
		protected.Use(middleware.JWTMiddleware(pubKey, jwtRepo, userRepo))
		{
			admin := protected.Group("/admin")
			admin.Use(middleware.AuthorizeAdminRole(roleRepo))
			{
				admin.GET("/roles", roleHandler.List)
				admin.POST("/roles", roleHandler.Create)
				admin.PUT("/roles/:id", roleHandler.Update)
				admin.DELETE("/roles/:id", roleHandler.Delete)
			}
		}
	}

	return r
}
