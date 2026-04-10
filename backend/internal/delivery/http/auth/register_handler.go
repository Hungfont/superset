package auth

import (
	"errors"
	"net/http"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
)

// RegisterHandler handles POST /api/v1/auth/register.
type RegisterHandler struct {
	svc *svcauth.RegisterService
}

func NewRegisterHandler(svc *svcauth.RegisterService) *RegisterHandler {
	return &RegisterHandler{svc: svc}
}

func (h *RegisterHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	_, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		var weakPwd svcauth.ErrWeakPassword
		switch {
		case errors.As(err, &weakPwd):
			c.JSON(http.StatusBadRequest, gin.H{"error": weakPwd.Reason})
		case errors.Is(err, svcauth.ErrDuplicateEmail):
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		case errors.Is(err, svcauth.ErrDuplicateUsername):
			c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Verification email sent"})
}
