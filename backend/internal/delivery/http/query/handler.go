package query

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	dbpool "superset/auth-service/internal/app/db"
	svcquery "superset/auth-service/internal/app/query"
	domain "superset/auth-service/internal/domain/auth"
	domdb "superset/auth-service/internal/domain/db"
	domainquery "superset/auth-service/internal/domain/query"
	"superset/auth-service/internal/delivery/http/middleware"
	"superset/auth-service/internal/pkg/crypto"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	executor      *svcquery.QueryExecutor
	asyncExecutor *svcquery.AsyncQueryExecutor
	pubKey        *rsa.PublicKey
	jwtRepo       domain.JWTRepository
	userRepo      domain.UserRepository
	queryRepo     domainquery.Repository
	roleRepo      domain.RoleRepository
	rdb           *redis.Client
	dbRepo        domdb.DatabaseRepository
	poolMgr       dbpool.DatabaseConnectionPool
}

func NewHandler(executor *svcquery.QueryExecutor) *Handler {
	return &Handler{executor: executor}
}

func NewHandlerWithAsync(executor *svcquery.QueryExecutor, asyncExecutor *svcquery.AsyncQueryExecutor, pubKey *rsa.PublicKey, jwtRepo domain.JWTRepository, userRepo domain.UserRepository, queryRepo domainquery.Repository, roleRepo domain.RoleRepository) *Handler {
	return &Handler{executor: executor, asyncExecutor: asyncExecutor, pubKey: pubKey, jwtRepo: jwtRepo, userRepo: userRepo, queryRepo: queryRepo, roleRepo: roleRepo}
}

func NewHandlerWithAsyncAndPool(executor *svcquery.QueryExecutor, asyncExecutor *svcquery.AsyncQueryExecutor, pubKey *rsa.PublicKey, jwtRepo domain.JWTRepository, userRepo domain.UserRepository, queryRepo domainquery.Repository, roleRepo domain.RoleRepository, rdb *redis.Client, dbRepo domdb.DatabaseRepository, poolMgr dbpool.DatabaseConnectionPool) *Handler {
	return &Handler{executor: executor, asyncExecutor: asyncExecutor, pubKey: pubKey, jwtRepo: jwtRepo, userRepo: userRepo, queryRepo: queryRepo, roleRepo: roleRepo, rdb: rdb, dbRepo: dbRepo, poolMgr: poolMgr}
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

// GetResult handles getting query result with ownership check and optional pagination.
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

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	if err != nil || limit <= 0 || limit > 100000 {
		limit = 1000
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

	resp, err := h.asyncExecutor.GetResult(c.Request.Context(), queryID, offset, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "result_not_found", "message": err.Error()})
		return
	}

	// Check if result expired
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

type EstimateRequest = domainquery.EstimateRequest

func (h *Handler) Estimate(c *gin.Context) {
	var req EstimateRequest
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

	// Rate limit: 30 requests per 60 seconds per user
	if h.rdb != nil {
		key := "rate:estimate:" + strconv.FormatUint(uint64(userCtx.ID), 10)
		count, err := h.rdb.Incr(c.Request.Context(), key).Result()
		if err == nil {
			if count == 1 {
				h.rdb.Expire(c.Request.Context(), key, 60*time.Second)
			}
			if count > 30 {
				ttl, _ := h.rdb.TTL(c.Request.Context(), key).Result()
				retryAfter := int(ttl.Seconds())
				if retryAfter <= 0 {
					retryAfter = 60
				}
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited", "retry_after": retryAfter})
				return
			}
		}
	}

	// Look up database to get SQLAlchemyURI
	db, err := h.dbRepo.GetDatabaseByID(c.Request.Context(), req.DatabaseID)
	if err != nil || db == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "database not found"})
		return
	}

	// Detect driver from SQLAlchemyURI scheme
	parsedURI, err := crypto.ParseSQLAlchemyURI(db.SQLAlchemyURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid database URI"})
		return
	}
	driver := parsedURI.Scheme
	if driver == "postgres" {
		driver = "postgresql"
	}

	// Get connection from pool
	conn, err := h.poolMgr.Get(c.Request.Context(), req.DatabaseID, db.SQLAlchemyURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "failed to get database connection"})
		return
	}

	// Run estimate
	estimator := svcquery.NewEstimator(driver)
	result, err := estimator.Estimate(c.Request.Context(), req.SQL, conn)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid_sql", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Download handles GET /api/v1/query/:id/download?format=csv|xlsx|json
func (h *Handler) Download(c *gin.Context) {
	if h.asyncExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async_not_available"})
		return
	}

	queryID := c.Param("id")
	format := c.Query("format")

	if !svcquery.IsValidFormat(format) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid_format", "message": "Format must be csv, xlsx, or json"})
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

	// Rate limit: 10 downloads per hour per user
	if h.rdb != nil {
		rateKey := fmt.Sprintf("rate:download:%d", userCtx.ID)
		count, err := h.rdb.Incr(c.Request.Context(), rateKey).Result()
		if err == nil {
			if count == 1 {
				h.rdb.Expire(c.Request.Context(), rateKey, 1*time.Hour)
			}
			if count > 10 {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited", "message": "Download limit reached. Try again later."})
				return
			}
		}
	}

	resp, err := h.asyncExecutor.GetResultForUser(c.Request.Context(), queryID, userCtx)
	if err != nil {
		switch err.Error() {
		case "forbidden":
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to download this query"})
		case "query not found":
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Query not found"})
		case "query not completed":
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "not_ready", "message": "Query has not completed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "download_error", "message": err.Error()})
		}
		return
	}

	// 410: query record has ResultsKey set but Redis data expired.
	if resp.Columns == nil || len(resp.Columns) == 0 {
		if h.queryRepo != nil {
			q, qErr := h.queryRepo.GetByID(c.Request.Context(), queryID)
			if qErr == nil && q != nil && q.ResultsKey != "" {
				c.JSON(http.StatusGone, gin.H{"error": "expired", "message": "Result expired. Re-run the query to download."})
				return
			}
		}
	}

	ext := format
	mime := "application/octet-stream"
	switch format {
	case "csv":
		mime = "text/csv; charset=utf-8"
	case "json":
		mime = "application/json; charset=utf-8"
	case "xlsx":
		mime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		ext = "xlsx"
	}

	filename := fmt.Sprintf("query_%s_%d.%s", queryID, time.Now().Unix(), ext)
	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Status(http.StatusOK)

	_ = svcquery.Export(c.Writer, format, resp)
}