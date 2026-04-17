package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	svcdb "superset/auth-service/internal/app/db"
	"superset/auth-service/internal/delivery/http/middleware"
	domainauth "superset/auth-service/internal/domain/auth"
	domain "superset/auth-service/internal/domain/db"

	"github.com/gin-gonic/gin"
)

// DatabaseHandler handles /api/v1/admin/databases endpoints.
type DatabaseHandler struct {
	svc *svcdb.DatabaseService
}

func NewDatabaseHandler(svc *svcdb.DatabaseService) *DatabaseHandler {
	return &DatabaseHandler{svc: svc}
}

func (h *DatabaseHandler) List(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	pageSize, err := parsePositiveInt(c.Query("page_size"), 10)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	result, listErr := h.svc.ListDatabases(c.Request.Context(), actor.ID, domain.DatabaseListQuery{
		SearchQ:  c.Query("q"),
		Backend:  c.Query("backend"),
		Page:     page,
		PageSize: pageSize,
	})
	if listErr != nil {
		h.handleError(c, listErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result.Items,
		"pagination": gin.H{
			"total":     result.Total,
			"page":      result.Page,
			"page_size": result.PageSize,
		},
	})
}

func (h *DatabaseHandler) Get(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	databaseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || databaseID == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	result, getErr := h.svc.GetDatabase(c.Request.Context(), actor.ID, uint(databaseID))
	if getErr != nil {
		h.handleError(c, getErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *DatabaseHandler) Create(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	var req domain.CreateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	created, err := h.svc.CreateDatabase(c.Request.Context(), actor.ID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *DatabaseHandler) Update(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	databaseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || databaseID == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	var req domain.UpdateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	updated, updateErr := h.svc.UpdateDatabase(c.Request.Context(), actor.ID, uint(databaseID), req)
	if updateErr != nil {
		h.handleError(c, updateErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *DatabaseHandler) Delete(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	databaseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || databaseID == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	if err := h.svc.DeleteDatabase(c.Request.Context(), actor.ID, uint(databaseID)); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *DatabaseHandler) TestConnection(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	var req domain.TestDatabaseConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	rateLimitKey := fmt.Sprintf("database-test:user:%d:ip:%s", actor.ID, c.ClientIP())
	result, err := h.svc.TestConnection(c.Request.Context(), actor.ID, req, rateLimitKey)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *DatabaseHandler) TestConnectionByID(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	databaseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || databaseID == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	rateLimitKey := fmt.Sprintf("database-test:user:%d:ip:%s", actor.ID, c.ClientIP())
	result, testErr := h.svc.TestConnectionByID(c.Request.Context(), actor.ID, uint(databaseID), rateLimitKey)
	if testErr != nil {
		h.handleError(c, testErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *DatabaseHandler) ListSchemas(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	databaseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || databaseID == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	forceRefresh, err := parseBoolQuery(c.Query("force_refresh"), false)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	rateLimitKey := fmt.Sprintf("database-schema-refresh:user:%d:db:%d:ip:%s", actor.ID, databaseID, c.ClientIP())
	result, listErr := h.svc.ListSchemas(c.Request.Context(), actor.ID, uint(databaseID), forceRefresh, rateLimitKey)
	if listErr != nil {
		h.handleError(c, listErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *DatabaseHandler) ListTables(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	databaseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || databaseID == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	pageSize, err := parsePositiveInt(c.Query("page_size"), 50)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	forceRefresh, err := parseBoolQuery(c.Query("force_refresh"), false)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	req := domain.ListDatabaseTablesRequest{
		Schema:   strings.TrimSpace(c.Query("schema")),
		Page:     page,
		PageSize: pageSize,
	}
	if req.Schema == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	rateLimitKey := fmt.Sprintf("database-schema-refresh:user:%d:db:%d:ip:%s", actor.ID, databaseID, c.ClientIP())
	result, listErr := h.svc.ListTables(c.Request.Context(), actor.ID, uint(databaseID), req, forceRefresh, rateLimitKey)
	if listErr != nil {
		h.handleError(c, listErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result.Items,
		"pagination": gin.H{
			"total":     result.Total,
			"page":      result.Page,
			"page_size": result.PageSize,
		},
	})
}

func (h *DatabaseHandler) ListColumns(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
		return
	}

	databaseID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || databaseID == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	forceRefresh, err := parseBoolQuery(c.Query("force_refresh"), false)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	req := domain.ListDatabaseColumnsRequest{
		Schema: strings.TrimSpace(c.Query("schema")),
		Table:  strings.TrimSpace(c.Query("table")),
	}
	if req.Schema == "" || req.Table == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": domain.ErrInvalidDatabase.Error()})
		return
	}

	rateLimitKey := fmt.Sprintf("database-schema-refresh:user:%d:db:%d:ip:%s", actor.ID, databaseID, c.ClientIP())
	result, listErr := h.svc.ListColumns(c.Request.Context(), actor.ID, uint(databaseID), req, forceRefresh, rateLimitKey)
	if listErr != nil {
		h.handleError(c, listErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *DatabaseHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrRateLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many test attempts. wait 60 seconds."})
	case errors.Is(err, domain.ErrDatabaseUnreachable):
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDatabaseTimeout):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDatabaseNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDatabaseNameExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDatabaseInUse):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrUnknownDatabaseDriver), errors.Is(err, domain.ErrInvalidDatabase), errors.Is(err, domain.ErrInvalidDatabaseURI), errors.Is(err, domain.ErrDatabaseConnectionTestFailed):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDatabaseCredentialEncryption):
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

func parsePositiveInt(raw string, defaultValue int) (int, error) {
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, domain.ErrInvalidDatabase
	}

	return value, nil
}

func parseBoolQuery(raw string, defaultValue bool) (bool, error) {
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, domain.ErrInvalidDatabase
	}

	return value, nil
}
