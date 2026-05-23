package sqllab

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	domain "superset/auth-service/internal/domain/auth"
	domdb "superset/auth-service/internal/domain/db"
	domainquery "superset/auth-service/internal/domain/query"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	sqllabRepo   domainquery.SQLLabRepository
	databaseRepo domdb.DatabaseRepository
}

func NewHandler(sqllabRepo domainquery.SQLLabRepository, databaseRepo domdb.DatabaseRepository) *Handler {
	return &Handler{sqllabRepo: sqllabRepo, databaseRepo: databaseRepo}
}

func (h *Handler) CreateTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	var req domainquery.CreateTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	scope, err := h.resolveVisibilityScope(c.Request.Context(), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to resolve visibility"})
		return
	}

	if _, err := h.databaseRepo.GetVisibleDatabaseByID(c.Request.Context(), req.DbID, scope, userCtx.ID); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "db_not_visible", "message": "Database not accessible"})
		return
	}

	label, err := h.generateLabel(c.Request.Context(), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to generate tab label"})
		return
	}

	tab := &domainquery.TabState{
		UserID:      userCtx.ID,
		DbID:        req.DbID,
		Schema:      req.Schema,
		Catalog:     req.Catalog,
		SQL:         req.SQL,
		QueryLimit:  req.QueryLimit,
		Label:       label,
		Active:      true,
		CreatedByFK: userCtx.ID,
		ChangedByFK: userCtx.ID,
		CreatedOn:   time.Now(),
		ChangedOn:   time.Now(),
	}

	if err := h.sqllabRepo.Create(c.Request.Context(), tab); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_failed", "message": "Failed to create tab"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": tab.ID, "label": tab.Label, "active": true})
}

func (h *Handler) ListTabs(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	includeClosed := c.Query("include_closed") == "true"

	tabs, err := h.sqllabRepo.ListByUser(c.Request.Context(), userCtx.ID, includeClosed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to list tabs"})
		return
	}

	resp := make([]domainquery.TabResponse, 0, len(tabs))
	for _, t := range tabs {
		resp = append(resp, tabToResponse(t))
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	tab, err := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve tab"})
		return
	}
	if tab == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
		return
	}

	c.JSON(http.StatusOK, tabToResponse(tab))
}

func (h *Handler) UpdateTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	tab, err := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve tab"})
		return
	}
	if tab == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to update this tab"})
		return
	}

	var req domainquery.UpdateTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	const maxSQLBytes = 64 * 1024
	if req.SQL != nil && len(*req.SQL) > maxSQLBytes {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "sql_too_large", "message": "SQL exceeds 64KB limit"})
		return
	}

	if req.Label != nil {
		tab.Label = *req.Label
	}
	if req.SQL != nil {
		tab.SQL = *req.SQL
	}
	if req.Schema != nil {
		tab.Schema = *req.Schema
	}
	if req.Catalog != nil {
		tab.Catalog = *req.Catalog
	}
	if req.QueryLimit != nil {
		tab.QueryLimit = *req.QueryLimit
	}
	if req.DbID != nil {
		tab.DbID = *req.DbID
	}
	if req.LatestQueryID != nil {
		tab.LatestQueryID = req.LatestQueryID
	}
	if req.HideLeftBar != nil {
		tab.HideLeftBar = *req.HideLeftBar
	}
	if req.ExtraJSON != nil {
		tab.ExtraJSON = *req.ExtraJSON
	}

	tab.ChangedOn = time.Now()
	tab.ChangedByFK = userCtx.ID

	if err := h.sqllabRepo.Update(c.Request.Context(), tab); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update_failed", "message": "Failed to update tab"})
		return
	}

	c.JSON(http.StatusOK, tabToResponse(tab))
}

func (h *Handler) CloseTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	err = h.sqllabRepo.CloseTab(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		tab, getErr := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
		if getErr != nil || tab == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to close this tab"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"closed": true})
}

func (h *Handler) CloseAllTabs(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	var req domainquery.CloseAllTabsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = domainquery.CloseAllTabsRequest{}
	}

	closed, err := h.sqllabRepo.CloseAllTabs(c.Request.Context(), userCtx.ID, req.ExceptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to close tabs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"closed": closed})
}

func (h *Handler) ReopenTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	err = h.sqllabRepo.ReopenTab(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		tab, getErr := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
		if getErr != nil || tab == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to reopen this tab"})
		return
	}

	tab, _ := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
	label := ""
	if tab != nil {
		label = tab.Label
	}

	c.JSON(http.StatusOK, gin.H{"reopened": true, "id": id, "label": label})
}

func (h *Handler) HardDeleteTab(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid tab id"})
		return
	}

	err = h.sqllabRepo.HardDelete(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		tab, getErr := h.sqllabRepo.GetByID(c.Request.Context(), uint(id), userCtx.ID)
		if getErr != nil || tab == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to delete this tab"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) resolveVisibilityScope(ctx context.Context, userID uint) (domdb.DatabaseVisibilityScope, error) {
	roleNames, err := h.databaseRepo.GetRoleNamesByUser(ctx, userID)
	if err != nil {
		return "", err
	}
	for _, name := range roleNames {
		n := strings.ToLower(strings.TrimSpace(name))
		if n == "admin" {
			return domdb.DatabaseVisibilityAdmin, nil
		}
	}
	for _, name := range roleNames {
		n := strings.ToLower(strings.TrimSpace(name))
		if n == "alpha" {
			return domdb.DatabaseVisibilityAlpha, nil
		}
	}
	return domdb.DatabaseVisibilityGamma, nil
}

func (h *Handler) generateLabel(ctx context.Context, userID uint) (string, error) {
	tabs, err := h.sqllabRepo.ListByUser(ctx, userID, false)
	if err != nil {
		return "", err
	}

	maxN := 0
	prefix := "Untitled Query "
	for _, t := range tabs {
		if strings.HasPrefix(t.Label, prefix) {
			if n, nErr := strconv.Atoi(t.Label[len(prefix):]); nErr == nil && n > maxN {
				maxN = n
			}
		}
	}
	return prefix + strconv.Itoa(maxN+1), nil
}

func getUserContext(c *gin.Context) (domain.UserContext, bool) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return domain.UserContext{}, false
	}
	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return domain.UserContext{}, false
	}
	return userCtx, true
}

func tabToResponse(t *domainquery.TabState) domainquery.TabResponse {
	resp := domainquery.TabResponse{
		ID:            t.ID,
		Label:         t.Label,
		DbID:          t.DbID,
		Schema:        t.Schema,
		Catalog:       t.Catalog,
		SQL:           t.SQL,
		Active:        t.Active,
		QueryLimit:    t.QueryLimit,
		LatestQueryID: t.LatestQueryID,
		HideLeftBar:   t.HideLeftBar,
		CreatedOn:     t.CreatedOn.Format("2006-01-02T15:04:05Z07:00"),
	}
	if t.ExtraJSON != "" {
		var extra struct {
			LatestQueryStatus string `json:"latest_query_status"`
		}
		if json.Unmarshal([]byte(t.ExtraJSON), &extra) == nil && extra.LatestQueryStatus != "" {
			resp.LatestQueryStatus = extra.LatestQueryStatus
		}
	}
	return resp
}
