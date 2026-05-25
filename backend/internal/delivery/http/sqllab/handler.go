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
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"closed": true})
}

func (h *Handler) CloseAllTabs(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	var exceptID *uint
	if exceptStr := c.Query("except_id"); exceptStr != "" {
		id, err := strconv.ParseUint(exceptStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid except_id"})
			return
		}
		uid := uint(id)
		exceptID = &uid
	}

	closed, err := h.sqllabRepo.CloseAllTabs(c.Request.Context(), userCtx.ID, exceptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to close tabs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"closed": closed})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Tab not found"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) CreateSavedQuery(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	var req domainquery.CreateSavedQueryRequest
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

	// Atomically check case-insensitive label uniqueness per spec
	exists, err := h.sqllabRepo.LabelExists(c.Request.Context(), userCtx.ID, req.Label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to check label uniqueness"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate_label", "message": "A saved query with this label already exists"})
		return
	}

	published := false
	if req.Published != nil {
		published = *req.Published
	}

	now := time.Now()
	sq := &domainquery.SavedQuery{
		DbID:        req.DbID,
		UserID:      userCtx.ID,
		Label:       req.Label,
		Schema:      req.Schema,
		Catalog:     req.Catalog,
		SQL:         req.SQL,
		Description: req.Description,
		Published:   published,
		CreatedByFK: userCtx.ID,
		ChangedByFK: userCtx.ID,
		CreatedOn:   now,
		ChangedOn:   now,
	}

	if err := h.sqllabRepo.CreateSavedQuery(c.Request.Context(), sq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_failed", "message": "Failed to save query"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": sq.ID, "label": sq.Label, "sql_tables": sq.SQLTables})
}

func (h *Handler) ListSavedQueries(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	var params domainquery.SavedQueryListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		params = domainquery.SavedQueryListParams{}
	}

	rows, total, err := h.sqllabRepo.ListSavedQueries(c.Request.Context(), userCtx.ID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to list saved queries"})
		return
	}

	resp := make([]domainquery.SavedQueryResponse, 0, len(rows))
	for _, sq := range rows {
		resp = append(resp, savedQueryToResponse(sq))
	}

	c.JSON(http.StatusOK, gin.H{
		"items": resp,
		"meta": gin.H{
			"total": total,
			"page":  params.Page,
			"limit": params.Limit,
		},
	})
}

func (h *Handler) UpdateSavedQuery(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid saved query id"})
		return
	}

	sq, err := h.sqllabRepo.GetSavedQuery(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve saved query"})
		return
	}
	if sq == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to update this saved query"})
		return
	}

	var req domainquery.UpdateSavedQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	if req.Label != nil {
		if !strings.EqualFold(*req.Label, sq.Label) {
			exists, checkErr := h.sqllabRepo.LabelExists(c.Request.Context(), userCtx.ID, *req.Label)
			if checkErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to check label uniqueness"})
				return
			}
			if exists {
				c.JSON(http.StatusConflict, gin.H{"error": "duplicate_label", "message": "A saved query with this label already exists"})
				return
			}
		}
		sq.Label = *req.Label
	}
	if req.SQL != nil {
		sq.SQL = *req.SQL
	}
	if req.Schema != nil {
		sq.Schema = *req.Schema
	}
	if req.Catalog != nil {
		sq.Catalog = *req.Catalog
	}
	if req.Description != nil {
		sq.Description = *req.Description
	}
	if req.Published != nil {
		sq.Published = *req.Published
	}
	if req.ExtraJSON != nil {
		sq.ExtraJSON = *req.ExtraJSON
	}

	sq.ChangedOn = time.Now()
	sq.ChangedByFK = userCtx.ID

	if err := h.sqllabRepo.UpdateSavedQuery(c.Request.Context(), sq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update_failed", "message": "Failed to update saved query"})
		return
	}

	c.JSON(http.StatusOK, savedQueryToResponse(sq))
}

func (h *Handler) DeleteSavedQuery(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid saved query id"})
		return
	}

	sq, err := h.sqllabRepo.GetSavedQuery(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to retrieve saved query"})
		return
	}
	if sq == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to delete this saved query"})
		return
	}

	if err := h.sqllabRepo.DeleteSavedQuery(c.Request.Context(), uint(id), userCtx.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete_failed", "message": "Failed to delete saved query"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ForkSavedQuery(c *gin.Context) {
	userCtx, ok := getUserContext(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "invalid saved query id"})
		return
	}

	fork, err := h.sqllabRepo.ForkSavedQuery(c.Request.Context(), uint(id), userCtx.ID)
	if err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to fork this saved query"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fork_failed", "message": "Failed to fork saved query"})
		return
	}

	c.JSON(http.StatusCreated, savedQueryToResponse(fork))
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

func savedQueryToResponse(sq *domainquery.SavedQuery) domainquery.SavedQueryResponse {
	return domainquery.SavedQueryResponse{
		ID:          sq.ID,
		Label:       sq.Label,
		DbID:        sq.DbID,
		Schema:      sq.Schema,
		Catalog:     sq.Catalog,
		SQL:         sq.SQL,
		Description: sq.Description,
		SQLTables:   sq.SQLTables,
		Published:   sq.Published,
		CreatedOn:   sq.CreatedOn.Format("2006-01-02T15:04:05Z07:00"),
		ChangedOn:   sq.ChangedOn.Format("2006-01-02T15:04:05Z07:00"),
	}
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
