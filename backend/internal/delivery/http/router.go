package http

import (
	"crypto/rsa"

	httpauth "superset/auth-service/internal/delivery/http/auth"
	httpdataset "superset/auth-service/internal/delivery/http/dataset"
	httpdb "superset/auth-service/internal/delivery/http/db"
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
	userHandler *httpauth.UserHandler,
	roleHandler *httpauth.RoleHandler,
	userRoleHandler *httpauth.UserRoleHandler,
	permissionHandler *httpauth.PermissionHandler,
	databaseHandler *httpdb.DatabaseHandler,
	datasetHandler *httpdataset.Handler,
	pubKey *rsa.PublicKey,
	jwtRepo domain.JWTRepository,
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
	rbacPermissionRepo domain.RBACPermissionRepository,
	rbacPermissionCacheRepo domain.RBACPermissionCacheRepository,
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
			require := func(action string, resource string) gin.HandlerFunc {
				return middleware.RequirePermission(roleRepo, rbacPermissionRepo, rbacPermissionCacheRepo, action, resource)
			}

			protected.GET("/datasets", datasetHandler.ListDatasets)
			protected.GET("/datasets/:id", datasetHandler.GetDataset)
			protected.POST("/datasets", datasetHandler.CreatePhysicalDataset)
			protected.POST("/datasets/virtual", datasetHandler.CreateVirtualDataset)

			admin := protected.Group("/admin")
			{
				admin.POST("/databases", databaseHandler.Create)
				admin.GET("/databases", databaseHandler.List)
				admin.GET("/databases/:id", databaseHandler.Get)
				admin.GET("/databases/:id/schemas", databaseHandler.ListSchemas)
				admin.GET("/databases/:id/tables", databaseHandler.ListTables)
				admin.GET("/databases/:id/columns", databaseHandler.ListColumns)
				admin.PUT("/databases/:id", databaseHandler.Update)
				admin.DELETE("/databases/:id", databaseHandler.Delete)
				admin.POST("/databases/test", databaseHandler.TestConnection)
				admin.POST("/databases/:id/test", databaseHandler.TestConnectionByID)

				admin.GET("/users", userHandler.List)
				admin.GET("/users/:id", userHandler.Get)
				admin.POST("/users", userHandler.Create)
				admin.PUT("/users/:id", userHandler.Update)
				admin.DELETE("/users/:id", userHandler.Delete)

				admin.GET("/users/:id/roles", userRoleHandler.List)
				admin.PUT("/users/:id/roles", userRoleHandler.Set)

				admin.GET("/roles", roleHandler.List)
				admin.POST("/roles", roleHandler.Create)
				admin.PUT("/roles/:id", roleHandler.Update)
				admin.DELETE("/roles/:id", roleHandler.Delete)
				admin.GET("/roles/:id/permissions", roleHandler.ListPermissions)
				admin.PUT("/roles/:id/permissions", roleHandler.SetPermissions)
				admin.POST("/roles/:id/permissions/add", roleHandler.AddPermissions)
				admin.DELETE("/roles/:id/permissions/:pv_id", roleHandler.RemovePermission)

				admin.GET("/permissions", require("can_read", "Permission"), permissionHandler.ListPermissions)
				admin.POST("/permissions", permissionHandler.CreatePermission)

				admin.GET("/view-menus", require("can_read", "ViewMenu"), permissionHandler.ListViewMenus)
				admin.POST("/view-menus", permissionHandler.CreateViewMenu)

				admin.GET("/permission-views", require("can_read", "PermissionView"), permissionHandler.ListPermissionViews)
				admin.POST("/permission-views", permissionHandler.CreatePermissionView)
				admin.DELETE("/permission-views/:id", permissionHandler.DeletePermissionView)
			}
		}
	}

	return r
}
