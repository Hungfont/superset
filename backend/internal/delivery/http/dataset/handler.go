package dataset

import (
	"context"
	"errors"
	"net/http"

	"superset/auth-service/internal/delivery/http/middleware"
	domainauth "superset/auth-service/internal/domain/auth"
	domain "superset/auth-service/internal/domain/dataset"

	"github.com/gin-gonic/gin"
)

type createPhysicalDatasetService interface {
	CreatePhysicalDataset(ctx context.Context, actorUserID uint, req domain.CreatePhysicalDatasetRequest) (*domain.CreatePhysicalDatasetResponse, error)
}

type createVirtualDatasetService interface {
	CreateVirtualDataset(ctx context.Context, actorUserID uint, req domain.CreateVirtualDatasetRequest) (*domain.CreateVirtualDatasetResponse, error)
}

// Handler handles /api/v1/datasets endpoints.
type Handler struct {
	svcPhysical createPhysicalDatasetService
	svcVirtual  createVirtualDatasetService
}

func NewHandler(svcPhysical createPhysicalDatasetService, svcVirtual createVirtualDatasetService) *Handler {
	return &Handler{svcPhysical: svcPhysical, svcVirtual: svcVirtual}
}

func (h *Handler) CreatePhysicalDataset(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	var req domain.CreatePhysicalDatasetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	created, err := h.svcPhysical.CreatePhysicalDataset(c.Request.Context(), actor.ID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *Handler) CreateVirtualDataset(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	var req domain.CreateVirtualDatasetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	created, err := h.svcVirtual.CreateVirtualDataset(c.Request.Context(), actor.ID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDatasetDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidDataset), errors.Is(err, domain.ErrInvalidDatabase):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrSQLNotSelect), errors.Is(err, domain.ErrSQLSemicolon), errors.Is(err, domain.ErrSQLSemanticError):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func getActor(c *gin.Context) (domainauth.UserContext, bool) {
	value, ok := c.Get(middleware.UserContextKey)
	if !ok {
		return domainauth.UserContext{}, false
	}

	actor, ok := value.(domainauth.UserContext)
	if !ok {
		return domainauth.UserContext{}, false
	}

	return actor, true
}
