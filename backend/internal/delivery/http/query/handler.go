package query

import (
	"net/http"

	svcquery "superset/auth-service/internal/app/query"
	domain "superset/auth-service/internal/domain/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	executor *svcquery.QueryExecutor
}

func NewHandler(executor *svcquery.QueryExecutor) *Handler {
	return &Handler{executor: executor}
}

type ExecuteRequest struct {
	DatabaseID uint   `json:"database_id" binding:"required"`
	SQL       string `json:"sql" binding:"required"`
	Limit     *int   `json:"limit"`
	Schema    string `json:"schema"`
	ClientID  string `json:"client_id"`
}

type ExecuteResponse struct {
	Data       interface{} `json:"data"`
	Columns   []string   `json:"columns"`
	FromCache bool      `json:"from_cache"`
	Query     struct {
		ExecutedSQL string `json:"executed_sql"`
		RLSApplied  bool   `json:"rls_applied"`
	} `json:"query"`
}

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
		DatabaseID: req.DatabaseID,
		SQL:       req.SQL,
		Limit:     req.Limit,
		Schema:    req.Schema,
	}

	resp, err := h.executor.Execute(c.Request.Context(), execReq, userCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "execution_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}