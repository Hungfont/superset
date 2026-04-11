package auth

import (
	"errors"
	"net/http"

	svcauth "superset/auth-service/internal/app/auth"

	"github.com/gin-gonic/gin"
)

// VerifyHandler handles GET /api/v1/auth/verify?hash=.
type VerifyHandler struct {
	svc     *svcauth.VerifyService
	baseURL string
}

func NewVerifyHandler(svc *svcauth.VerifyService, baseURL string) *VerifyHandler {
	return &VerifyHandler{svc: svc, baseURL: baseURL}
}

func (h *VerifyHandler) Verify(c *gin.Context) {
	hash := c.Query("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hash query parameter is required"})
		return
	}

	if err := h.svc.Verify(c.Request.Context(), hash); err != nil {
		switch {
		case errors.Is(err, svcauth.ErrInvalidHash):
			c.JSON(http.StatusNotFound, gin.H{"error": "invalid or already used verification link"})
		case errors.Is(err, svcauth.ErrExpiredHash):
			c.JSON(http.StatusGone, gin.H{"error": "verification link has expired"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.Redirect(http.StatusFound, "/login?activated=true")
}
