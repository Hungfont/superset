package auth

import (
	"errors"
	"net/http"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

// RefreshHandler handles POST /api/v1/auth/refresh.
// The refresh token is read exclusively from the HttpOnly "refresh_token" cookie
// so it is never exposed to JavaScript.
type RefreshHandler struct {
	svc *svcauth.RefreshService
}

func NewRefreshHandler(svc *svcauth.RefreshService) *RefreshHandler {
	return &RefreshHandler{svc: svc}
}

func (h *RefreshHandler) Refresh(c *gin.Context) {
	token, err := c.Cookie("refresh_token")
	if err != nil || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenMissing.Error()})
		return
	}

	resp, err := h.svc.Refresh(c.Request.Context(), token)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Rotate the HttpOnly cookie with the new refresh token.
	c.SetCookie(
		"refresh_token",
		resp.RefreshToken,
		int((7 * 24 * time.Hour).Seconds()),
		"/",
		"",
		true, // secure
		true, // httpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
	})
}

func (h *RefreshHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrTokenInvalid),
		errors.Is(err, domain.ErrTokenReused),
		errors.Is(err, domain.ErrAccountInactive):
		// Clear the cookie on any auth failure so the client is not left
		// in a broken state holding a dead token.
		c.SetCookie("refresh_token", "", -1, "/", "", true, true)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
