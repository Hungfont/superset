package rls

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

type service interface {
	List(ctx context.Context, params domain.RLSFilterListParams) (*domain.RLSFilterListResult, error)
	GetByID(ctx context.Context, id uint) (*domain.RLSFilterResponse, error)
	Create(ctx context.Context, actorUserID uint, req domain.CreateRLSFilterRequest) (*domain.RLSFilterResponse, error)
	Update(ctx context.Context, actorUserID uint, id uint, req domain.UpdateRLSFilterRequest) (*domain.RLSFilterResponse, error)
	Delete(ctx context.Context, actorUserID uint, id uint) error
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c *gin.Context) {
	var params domain.RLSFilterListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Get(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	result, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Create(c *gin.Context) {
	var req domain.CreateRLSFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.svc.Create(c.Request.Context(), actor.ID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "Invalid SQL clause") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *Handler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	var req domain.UpdateRLSFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.svc.Update(c.Request.Context(), actor.ID, uint(id), req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
			return
		}
		if strings.Contains(err.Error(), "Invalid SQL clause") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err = h.svc.Delete(c.Request.Context(), actor.ID, uint(id))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func getActor(c *gin.Context) (*domain.UserContext, bool) {
	v, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	actor, ok := v.(domain.UserContext)
	return &actor, ok
}