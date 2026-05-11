package query

import (
	"crypto/rsa"
	"net/http"
	"strconv"
	"strings"
	"time"

	svcquery "superset/auth-service/internal/app/query"
	domain "superset/auth-service/internal/domain/auth"
	domainquery "superset/auth-service/internal/domain/query"
	"superset/auth-service/internal/delivery/http/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	executor      *svcquery.QueryExecutor
	asyncExecutor *svcquery.AsyncQueryExecutor
	pubKey        *rsa.PublicKey
	jwtRepo       domain.JWTRepository
	userRepo      domain.UserRepository
	queryRepo     domainquery.Repository
	roleRepo      domain.RoleRepository
}

func NewHandler(executor *svcquery.QueryExecutor) *Handler {
	return &Handler{executor: executor}
}

func NewHandlerWithAsync(executor *svcquery.QueryExecutor, asyncExecutor *svcquery.AsyncQueryExecutor, pubKey *rsa.PublicKey, jwtRepo domain.JWTRepository, userRepo domain.UserRepository, queryRepo domainquery.Repository, roleRepo domain.RoleRepository) *Handler {
	return &Handler{executor: executor, asyncExecutor: asyncExecutor, pubKey: pubKey, jwtRepo: jwtRepo, userRepo: userRepo, queryRepo: queryRepo, roleRepo: roleRepo}
}

// Use domain types via type aliases
type ExecuteRequest = domainquery.ExecuteRequest
type ExecuteResponse = domainquery.ExecuteResponse

func (h *Handler) Execute(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return
	}

	execReq := svcquery.ExecuteRequest{
		DatabaseID:   req.DatabaseID,
		SQL:          req.SQL,
		Limit:        req.Limit,
		Schema:       req.Schema,
		Catalog:      req.Catalog,
		TabName:      req.TabName,
		SqlEditorID:  req.SqlEditorID,
		ClientID:     req.ClientID,
		ForceRefresh: req.ForceRefresh,
		SelectAsCTA:  req.SelectAsCTA,
	}

	resp, err := h.executor.Execute(c.Request.Context(), execReq, userCtx)
	if err != nil {
		// QE-001 #6: Handle SQL errors as 400, timeouts as 408
		if qe, ok := err.(*svcquery.QueryError); ok {
			switch qe.Code {
			case 400:
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_sql", "message": qe.Message})
				return
			case 403:
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": qe.Message})
				return
			case 408:
				c.JSON(http.StatusRequestTimeout, gin.H{"error": "query_timeout", "message": qe.Message})
				return
			case 500:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "execution_error", "message": qe.Message})
				return
			}
		}
		// Default to 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": "execution_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Submit handles async query submission
type SubmitRequest domainquery.AsyncSubmitRequest

type SubmitResponse domainquery.AsyncSubmitResponse

func (h *Handler) Submit(c *gin.Context) {
	if h.asyncExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async_not_available"})
		return
	}

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return
	}

	asyncReq := domainquery.AsyncSubmitRequest{
		DatabaseID:   req.DatabaseID,
		SQL:          req.SQL,
		Limit:        req.Limit,
		Schema:       req.Schema,
		Catalog:      req.Catalog,
		TabName:      req.TabName,
		SqlEditorID:  req.SqlEditorID,
		ClientID:     req.ClientID,
		ForceRefresh: req.ForceRefresh,
		SelectAsCTA:  req.SelectAsCTA,
	}

	resp, err := h.asyncExecutor.Submit(c.Request.Context(), asyncReq, userCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "submit_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

// GetStatus handles getting query status
type StatusResponse domainquery.QueryStatusResponse

func (h *Handler) GetStatus(c *gin.Context) {
	if h.asyncExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async_not_available"})
		return
	}

	queryID := c.Param("id")
	if queryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "query id required"})
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return
	}

	resp, err := h.asyncExecutor.GetStatus(c.Request.Context(), queryID, userCtx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "query_not_found", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Cancel handles query cancellation with proper HTTP status codes.
// - 202: cancel initiated
// - 200: already stopped/completed
// - 403: forbidden (not owner)
// - 404: query not found
func (h *Handler) Cancel(c *gin.Context) {
	if h.asyncExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async_not_available"})
		return
	}

	queryID := c.Param("id")
	if queryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "query id required"})
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return
	}

	result, err := h.asyncExecutor.Cancel(c.Request.Context(), queryID, userCtx)
	if err != nil {
		switch err.Error() {
		case "forbidden":
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "You do not have permission to cancel this query"})
		case "query not found":
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Query not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cancel_error", "message": err.Error()})
		}
		return
	}

	// Map result action to HTTP status code
	switch result.Action {
	case "cancelling":
		c.JSON(http.StatusAccepted, gin.H{"status": "stopping", "query_id": queryID})
	default:
		// already_stopped or already_completed
		c.JSON(http.StatusOK, gin.H{"status": result.CurrentStatus, "query_id": queryID, "message": "Query already " + result.CurrentStatus})
	}
}

// GetResultByToken handles download link for large results via query param auth
func (h *Handler) GetResultByToken(c *gin.Context) {
	if h.asyncExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async_not_available"})
		return
	}

	tokenStr := c.Query("token")
	userCtx, statusCode := middleware.ValidateJWTFromQuery(tokenStr, h.pubKey, h.jwtRepo, h.userRepo)
	if statusCode != 0 {
		c.AbortWithStatus(statusCode)
		return
	}

	queryID := c.Param("id")
	if queryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "query id required"})
		return
	}

	resp, err := h.asyncExecutor.GetResultForUser(c.Request.Context(), queryID, *userCtx)
	if err != nil {
		if err.Error() == "forbidden" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "result_not_found", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetResult handles getting query result with ownership check (QE-007)
func (h *Handler) GetResult(c *gin.Context) {
	if h.asyncExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async_not_available"})
		return
	}

	queryID := c.Param("id")
	if queryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "query id required"})
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return
	}

	q, err := h.queryRepo.GetByID(c.Request.Context(), queryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if q.UserID != userCtx.ID {
		isAdmin, adminErr := h.roleRepo.IsAdmin(c.Request.Context(), userCtx.ID)
		if adminErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	if q.Status != "success" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "query not yet completed or failed"})
		return
	}

	resp, err := h.asyncExecutor.GetResult(c.Request.Context(), queryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "result_not_found", "message": err.Error()})
		return
	}

	// Check if result expired: query was successful and has ResultsKey but Redis returned empty
	if q.ResultsKey != "" {
		dataLen := 0
		if resp.Data != nil {
			if d, ok := resp.Data.([]interface{}); ok {
				dataLen = len(d)
			}
		}
		if dataLen == 0 && len(resp.Columns) == 0 {
			c.JSON(http.StatusGone, gin.H{"error": "result_expired", "message": "Result TTL expired. Rerun query."})
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ListHistory returns paginated query history for the current user (QE-007)
func (h *Handler) ListHistory(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	var req struct {
		Status      string `form:"status"`
		DatabaseID  uint   `form:"database_id"`
		SQLContains string `form:"sql_contains"`
		Page        int    `form:"page"`
		PageSize    int    `form:"page_size"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	isAdmin, err := h.roleRepo.IsAdmin(c.Request.Context(), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	filter := &domainquery.ListFilter{
		Status:     req.Status,
		DatabaseID: req.DatabaseID,
		SQLLike:    req.SQLContains,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}
	if !isAdmin {
		filter.UserID = userCtx.ID
	}

	items, total, err := h.queryRepo.ListHistory(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, domainquery.HistoryResponse{
		Queries:  items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
}

// DeleteHistory deletes old query records (admin only, QE-007)
func (h *Handler) DeleteHistory(c *gin.Context) {
	olderThanStr := c.Query("older_than")
	if olderThanStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "older_than query param required"})
		return
	}

	if !strings.HasSuffix(olderThanStr, "d") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "older_than must be in days, e.g., '30d'"})
		return
	}

	daysStr := strings.TrimSuffix(olderThanStr, "d")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid days value"})
		return
	}

	olderThan := time.Now().AddDate(0, 0, -days)
	deleted, err := h.queryRepo.DeleteOlderThan(c.Request.Context(), olderThan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, domainquery.DeleteHistoryResponse{Deleted: deleted})
}