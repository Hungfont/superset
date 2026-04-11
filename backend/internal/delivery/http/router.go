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
	pubKey *rsa.PublicKey,
	jwtRepo domain.JWTRepository,
	userRepo domain.UserRepository,
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
		}

		// Protected routes require a valid JWT.
		protected := v1.Group("/")
		protected.Use(middleware.JWTMiddleware(pubKey, jwtRepo, userRepo))
		{
			// Future protected routes are registered here.
			// Example: protected.GET("/me", profileHandler.Me)
		}
	}

	return r
}
