package auth

import (
	"errors"
	"net/http"
	"strconv"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

// UserRoleHandler handles /api/v1/admin/users/:id/roles endpoints.
type UserRoleHandler struct {
	svc *svcauth.UserRoleService
}

func NewUserRoleHandler(svc *svcauth.UserRoleService) *UserRoleHandler {
	return &UserRoleHandler{svc: svc}
}

func (h *UserRoleHandler) List(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	userID, ok := parseUserID(c)
	if !ok {
		return
	}

	roleIDs, err := h.svc.ListUserRoles(c.Request.Context(), actor.ID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": domain.UserRolesPayload{UserID: userID, RoleIDs: roleIDs}})
}

func (h *UserRoleHandler) Set(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	userID, ok := parseUserID(c)
	if !ok {
		return
	}

	var req domain.UpsertUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	roleIDs, err := h.svc.SetUserRoles(c.Request.Context(), actor.ID, userID, req.RoleIDs)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": domain.UserRolesPayload{UserID: userID, RoleIDs: roleIDs}})
}

func (h *UserRoleHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUserMustHaveRole), errors.Is(err, domain.ErrInvalidRole):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func parseUserID(c *gin.Context) (uint, bool) {
	parsed, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return 0, false
	}
	return uint(parsed), true
}
