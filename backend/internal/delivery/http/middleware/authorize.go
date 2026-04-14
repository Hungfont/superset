package middleware

import (
	"net/http"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

// AuthorizeAdminRole ensures the authenticated actor has the Admin role.
func AuthorizeAdminRole(roleRepo domain.RoleRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get(UserContextKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
			return
		}

		actor, ok := value.(domain.UserContext)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
			return
		}

		isAdmin, err := roleRepo.IsAdmin(c.Request.Context(), actor.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrForbidden.Error()})
			return
		}

		c.Next()
	}
}
