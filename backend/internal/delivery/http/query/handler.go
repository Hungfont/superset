package query

import (
	"crypto/rsa"
	"net/http"

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
}

func NewHandler(executor *svcquery.QueryExecutor) *Handler {
	return &Handler{executor: executor}
}

func NewHandlerWithAsync(executor *svcquery.QueryExecutor, asyncExecutor *svcquery.AsyncQueryExecutor, pubKey *rsa.PublicKey, jwtRepo domain.JWTRepository, userRepo domain.UserRepository) *Handler {
	return &Handler{executor: executor, asyncExecutor: asyncExecutor, pubKey: pubKey, jwtRepo: jwtRepo, userRepo: userRepo}
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

// GetResult handles getting query result
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

	resp, err := h.asyncExecutor.GetResult(c.Request.Context(), queryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "result_not_found", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}