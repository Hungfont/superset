package chart

import (
	"context"
	"fmt"
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

type listChartsService interface {
	ListCharts(ctx context.Context, actorID uint, input chartsvc.ListChartsInput) (*chartsvc.ChartListResult, error)
}

type getChartService interface {
	GetChart(ctx context.Context, actorID uint, id uint) (*chartsvc.ChartDetail, error)
}

type Handler struct {
	svcCreate createChartService
	svcList   listChartsService
	svcGet    getChartService
}

func NewHandler(svcCreate createChartService, svcList listChartsService, svcGet getChartService) *Handler {
	return &Handler{svcCreate: svcCreate, svcList: svcList, svcGet: svcGet}
}

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

func (h *Handler) List(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var query ChartListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svcList.ListCharts(c.Request.Context(), actor.ID, chartsvc.ListChartsInput{
		Q:            query.Q,
		VizType:      query.VizType,
		DatasourceID: query.DatasourceID,
		Owner:        query.Owner,
		Certified:    query.Certified,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) Get(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil || id == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	detail, err := h.svcGet.GetChart(c.Request.Context(), actor.ID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": detail})
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
	case strings.Contains(msg, "chart not found"):
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
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
