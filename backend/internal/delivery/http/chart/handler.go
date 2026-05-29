package chart

import (
	"context"
	"net/http"
	"strings"

	chartsvc "superset/auth-service/internal/app/chart"
	"superset/auth-service/internal/delivery/http/middleware"
	domainauth "superset/auth-service/internal/domain/auth"
	chartdomain "superset/auth-service/internal/domain/chart"

	"github.com/gin-gonic/gin"
)

type createChartService interface {
	CreateChart(ctx context.Context, actorID uint, input chartsvc.CreateChartInput) (*chartdomain.Slice, error)
}

type Handler struct{ svcCreate createChartService }

func NewHandler(svcCreate createChartService) *Handler { return &Handler{svcCreate: svcCreate} }

func (h *Handler) Create(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateChartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slice, err := h.svcCreate.CreateChart(c.Request.Context(), actor.ID, chartsvc.CreateChartInput{
		SliceName:            req.SliceName,
		VizType:              req.VizType,
		DatasourceID:         req.DatasourceID,
		DatasourceType:       req.DatasourceType,
		Params:               req.Params,
		QueryContext:         req.QueryContext,
		Description:          req.Description,
		CacheTimeout:         req.CacheTimeout,
		CertifiedBy:          req.CertifiedBy,
		CertificationDetails: req.CertificationDetails,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": slice})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "forbidden"):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case strings.Contains(msg, "invalid datasource_id") || strings.Contains(msg, "dataset not found"):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid datasource_id"})
	case strings.Contains(msg, "invalid params JSON"):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid params JSON"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func getActor(c *gin.Context) (domainauth.UserContext, bool) {
	v, ok := c.Get(middleware.UserContextKey)
	if !ok {
		return domainauth.UserContext{}, false
	}
	actor, ok := v.(domainauth.UserContext)
	return actor, ok
}
