package auth

import (
	"errors"
	"net/http"

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

func (h *DatabaseHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDatabaseNameExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidDatabase), errors.Is(err, domain.ErrInvalidDatabaseURI), errors.Is(err, domain.ErrDatabaseConnectionTestFailed):
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
