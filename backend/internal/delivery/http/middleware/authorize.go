package middleware

import (
	"net/http"
	"strings"
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

// RequirePermission checks whether the authenticated actor can perform action on resource.
// Admin users bypass tuple verification.
func RequirePermission(
	roleRepo domain.RoleRepository,
	permissionRepo domain.RBACPermissionRepository,
	permissionCacheRepo domain.RBACPermissionCacheRepository,
	action string,
	resource string,
) gin.HandlerFunc {
	required := permissionKey(action, resource)

	return func(c *gin.Context) {
		actor, ok := getActor(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
			return
		}

		isAdmin, err := roleRepo.IsAdmin(c.Request.Context(), actor.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if isAdmin {
			c.Next()
			return
		}

		permissionSet, err := resolvePermissionSet(c, permissionRepo, permissionCacheRepo, actor.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if _, hasRequiredPermission := permissionSet[required]; !hasRequiredPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrForbidden.Error()})
			return
		}

		c.Next()
	}
}

func getActor(c *gin.Context) (domain.UserContext, bool) {
	value, ok := c.Get(UserContextKey)
	if !ok {
		return domain.UserContext{}, false
	}

	actor, ok := value.(domain.UserContext)
	if !ok {
		return domain.UserContext{}, false
	}

	return actor, true
}

func resolvePermissionSet(
	c *gin.Context,
	permissionRepo domain.RBACPermissionRepository,
	permissionCacheRepo domain.RBACPermissionCacheRepository,
	userID uint,
) (map[string]struct{}, error) {
	cached, err := permissionCacheRepo.GetPermissionSet(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return buildPermissionSet(cached), nil
	}

	tuples, err := permissionRepo.ListPermissionTuplesByUser(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}

	values := make([]string, 0, len(tuples))
	for _, tuple := range tuples {
		values = append(values, permissionKey(tuple.Action, tuple.Resource))
	}

	// Cache set is best-effort; authorization should still proceed on cache write errors.
	_ = permissionCacheRepo.SetPermissionSet(c.Request.Context(), userID, values)

	return buildPermissionSet(values), nil
}

func buildPermissionSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizePermissionPart(value)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func permissionKey(action string, resource string) string {
	normalizedAction := normalizePermissionPart(action)
	normalizedResource := normalizePermissionPart(resource)
	return normalizedAction + ":" + normalizedResource
}

func normalizePermissionPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
