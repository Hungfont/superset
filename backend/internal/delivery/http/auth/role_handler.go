package auth

import (
	"errors"
	"net/http"
	"strconv"

	svcauth "superset/auth-service/internal/app/auth"
	"superset/auth-service/internal/delivery/http/middleware"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

// RoleHandler handles /api/v1/admin/roles endpoints.
type RoleHandler struct {
	svc *svcauth.RoleService
}

func NewRoleHandler(svc *svcauth.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

func (h *RoleHandler) List(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	roles, err := h.svc.ListRoles(c.Request.Context(), actor.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": roles})
}

func (h *RoleHandler) Create(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	var req domain.UpsertRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	role, err := h.svc.CreateRole(c.Request.Context(), actor.ID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": role})
}

func (h *RoleHandler) Update(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}

	var req domain.UpsertRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	role, err := h.svc.UpdateRole(c.Request.Context(), actor.ID, roleID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": role})
}

func (h *RoleHandler) Delete(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteRole(c.Request.Context(), actor.ID, roleID); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": true})
}

func (h *RoleHandler) ListPermissions(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}

	permissionViewIDs, err := h.svc.ListRolePermissions(c.Request.Context(), actor.ID, roleID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": domain.RolePermissionsPayload{RoleID: roleID, PermissionViewIDs: permissionViewIDs}})
}

func (h *RoleHandler) SetPermissions(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}

	var req domain.UpsertRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	permissionViewIDs, err := h.svc.SetRolePermissions(c.Request.Context(), actor.ID, roleID, req.PermissionViewIDs)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": domain.RolePermissionsPayload{RoleID: roleID, PermissionViewIDs: permissionViewIDs}})
}

func (h *RoleHandler) AddPermissions(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}

	var req domain.UpsertRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	permissionViewIDs, err := h.svc.AddRolePermissions(c.Request.Context(), actor.ID, roleID, req.PermissionViewIDs)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": domain.RolePermissionsPayload{RoleID: roleID, PermissionViewIDs: permissionViewIDs}})
}

func (h *RoleHandler) RemovePermission(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}

	permissionViewID, ok := parseRolePermissionViewID(c)
	if !ok {
		return
	}

	permissionViewIDs, err := h.svc.RemoveRolePermission(c.Request.Context(), actor.ID, roleID, permissionViewID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": domain.RolePermissionsPayload{RoleID: roleID, PermissionViewIDs: permissionViewIDs}})
}

func (h *RoleHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidRole), errors.Is(err, domain.ErrInvalidPermissionViewID):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrBuiltInRole):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrRoleHasUsers):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrRoleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func getActor(c *gin.Context) (domain.UserContext, bool) {
	value, ok := c.Get(middleware.UserContextKey)
	if !ok {
		return domain.UserContext{}, false
	}
	actor, ok := value.(domain.UserContext)
	return actor, ok
}

func parseRoleID(c *gin.Context) (uint, bool) {
	parsed, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return 0, false
	}
	return uint(parsed), true
}

func parseRolePermissionViewID(c *gin.Context) (uint, bool) {
	parsed, err := strconv.ParseUint(c.Param("pv_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission view id"})
		return 0, false
	}
	return uint(parsed), true
}
