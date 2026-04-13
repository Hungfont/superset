package auth

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// LogoutHandler handles POST /api/v1/auth/logout.
// It always returns 204 and clears the refresh cookie.
type LogoutHandler struct {
	svc    *svcauth.LogoutService
	pubKey *rsa.PublicKey
}

func NewLogoutHandler(svc *svcauth.LogoutService, pubKey *rsa.PublicKey) *LogoutHandler {
	return &LogoutHandler{svc: svc, pubKey: pubKey}
}

func (h *LogoutHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie("refresh_token")
	logoutAll := parseLogoutAll(c.Query("all"))

	req := domain.LogoutRequest{
		RefreshToken: refreshToken,
		LogoutAll:    logoutAll,
	}
	req = h.enrichFromAccessToken(c, req)

	h.svc.Logout(c.Request.Context(), req)

	c.SetCookie("refresh_token", "", -1, "/", "", true, true)
	c.Status(http.StatusNoContent)
}

func parseLogoutAll(raw string) bool {
	if raw == "" {
		return false
	}
	v, err := strconv.ParseBool(raw)
	return err == nil && v
}

func (h *LogoutHandler) enrichFromAccessToken(c *gin.Context, req domain.LogoutRequest) domain.LogoutRequest {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return req
	}
	raw := strings.TrimPrefix(header, "Bearer ")

	token, err := jwt.Parse(raw, h.keyFunc, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil || !token.Valid {
		return req
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return req
	}

	if jti, ok := claims["jti"].(string); ok {
		req.JTI = jti
	}
	if sub, ok := claims["sub"].(string); ok {
		if uid, err := strconv.ParseUint(sub, 10, 64); err == nil {
			req.UserID = uint(uid)
		}
	}
	if expUnix, ok := claims["exp"].(float64); ok {
		req.AccessTokenExpiresAt = time.Unix(int64(expUnix), 0).UTC()
	}

	return req
}

func (h *LogoutHandler) keyFunc(token *jwt.Token) (any, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	return h.pubKey, nil
}
