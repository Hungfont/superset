package dataset

import (
	"context"
	"errors"
	"fmt"
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

type listDatasetsService interface {
	ListDatasets(ctx context.Context, actorUserID uint, query domain.DatasetListQuery) (*domain.DatasetListResult, error)
}

type getDatasetService interface {
	GetDatasetDetail(ctx context.Context, actorUserID uint, id uint) (*domain.DatasetDetail, error)
}

type updateDatasetService interface {
	UpdateDatasetMetadata(ctx context.Context, actorUserID uint, id uint, req domain.UpdateDatasetMetadataRequest) (*domain.UpdateDatasetMetadataResponse, error)
}

type updateColumnService interface {
	UpdateColumn(ctx context.Context, actorUserID uint, datasetID uint, columnID uint, req domain.UpdateColumnRequest) (*domain.UpdateColumnResponse, error)
	BulkUpdateColumns(ctx context.Context, actorUserID uint, datasetID uint, req domain.BulkUpdateColumnRequest) (*domain.BulkUpdateColumnResponse, error)
}

// Handler handles /api/v1/datasets endpoints.
type Handler struct {
	svcPhysical  createPhysicalDatasetService
	svcVirtual   createVirtualDatasetService
	svcList      listDatasetsService
	svcGet       getDatasetService
	svcUpdate    updateDatasetService
	svcUpdateCol updateColumnService
}

func NewHandler(svcPhysical createPhysicalDatasetService, svcVirtual createVirtualDatasetService, svcList listDatasetsService, svcGet getDatasetService, svcUpdate updateDatasetService, svcUpdateCol updateColumnService) *Handler {
	return &Handler{svcPhysical: svcPhysical, svcVirtual: svcVirtual, svcList: svcList, svcGet: svcGet, svcUpdate: svcUpdate, svcUpdateCol: svcUpdateCol}
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

func (h *Handler) ListDatasets(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	var query domain.DatasetListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svcList.ListDatasets(c.Request.Context(), actor.ID, query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetDataset(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil || id == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrDatasetNotFound.Error()})
		return
	}

	detail, err := h.svcGet.GetDatasetDetail(c.Request.Context(), actor.ID, id)
	if err != nil {
		if errors.Is(err, domain.ErrDatasetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func (h *Handler) UpdateDataset(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil || id == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrDatasetNotFound.Error()})
		return
	}

	var req domain.UpdateDatasetMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.svcUpdate.UpdateDatasetMetadata(c.Request.Context(), actor.ID, id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) UpdateColumn(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	datasetIDParam := c.Param("id")
	var datasetID uint
	if _, err := fmt.Sscanf(datasetIDParam, "%d", &datasetID); err != nil || datasetID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrDatasetNotFound.Error()})
		return
	}

	colIDParam := c.Param("col_id")
	var columnID uint
	if _, err := fmt.Sscanf(colIDParam, "%d", &columnID); err != nil || columnID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrColumnNotFound.Error()})
		return
	}

	var req domain.UpdateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.svcUpdateCol.UpdateColumn(c.Request.Context(), actor.ID, datasetID, columnID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) BulkUpdateColumns(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	datasetIDParam := c.Param("id")
	var datasetID uint
	if _, err := fmt.Sscanf(datasetIDParam, "%d", &datasetID); err != nil || datasetID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrDatasetNotFound.Error()})
		return
	}

	var req domain.BulkUpdateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svcUpdateCol.BulkUpdateColumns(c.Request.Context(), actor.ID, datasetID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
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
	case errors.Is(err, domain.ErrInvalidMainDttmCol):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrColumnNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidExpression), errors.Is(err, domain.ErrInvalidDateFormat):
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
