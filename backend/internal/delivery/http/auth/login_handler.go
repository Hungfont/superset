package auth

import (
	"errors"
	"net/http"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

// LoginHandler handles POST /api/v1/auth/login.
type LoginHandler struct {
	svc *svcauth.LoginService
}

func NewLoginHandler(svc *svcauth.LoginService) *LoginHandler {
	return &LoginHandler{svc: svc}
}

func (h *LoginHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	ip := c.ClientIP()
	resp, err := h.svc.Login(c.Request.Context(), ip, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Set refresh token as HttpOnly, Secure, SameSite=Strict cookie (7 days).
	c.SetCookie(
		"refresh_token",
		resp.RefreshToken,
		int((7 * 24 * time.Hour).Seconds()),
		"/",
		"",
		true,  // secure
		true,  // httpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  resp.AccessToken,
		"refresh_token": resp.RefreshToken,
	})
}

func (h *LoginHandler) handleError(c *gin.Context, err error) {
	var lockErr svcauth.ErrLocked
	switch {
	case errors.As(err, &lockErr):
		c.JSON(http.StatusLocked, gin.H{
			"error":        "account locked",
			"locked_until": lockErr.Until.UTC().Format(time.RFC3339),
		})
	case errors.Is(err, domain.ErrAccountInactive):
		c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
	case errors.Is(err, domain.ErrRateLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, try again later"})
	case errors.Is(err, domain.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
