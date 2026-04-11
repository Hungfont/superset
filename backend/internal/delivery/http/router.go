package http

import (
	httpauth "superset/auth-service/internal/delivery/http/auth"

	"github.com/gin-gonic/gin"
)

// NewRouter wires all routes and returns the configured Gin engine.
func NewRouter(
	registerHandler *httpauth.RegisterHandler,
	verifyHandler *httpauth.VerifyHandler,
	loginHandler *httpauth.LoginHandler,
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
	}

	return r
}
