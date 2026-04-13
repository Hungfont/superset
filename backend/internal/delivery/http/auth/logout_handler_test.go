package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	httpauth "superset/auth-service/internal/delivery/http/auth"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type fakeJWTRepoForLogoutHandler struct {
	jti string
	ttl time.Duration
}

func (f *fakeJWTRepoForLogoutHandler) IsBlacklisted(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (f *fakeJWTRepoForLogoutHandler) GetCachedUser(_ context.Context, _ uint) (*domain.UserContext, error) {
	return nil, nil
}
func (f *fakeJWTRepoForLogoutHandler) SetCachedUser(_ context.Context, _ uint, _ *domain.UserContext) error {
	return nil
}
func (f *fakeJWTRepoForLogoutHandler) BlacklistJTI(_ context.Context, jti string, ttl time.Duration) error {
	f.jti = jti
	f.ttl = ttl
	return nil
}

type fakeRefreshRepoForLogoutHandler struct {
	deletedToken    string
	deleteAllUserID uint
}

func (f *fakeRefreshRepoForLogoutHandler) Store(_ context.Context, _ string, _ uint) error {
	return nil
}
func (f *fakeRefreshRepoForLogoutHandler) GetUserID(_ context.Context, _ string) (uint, bool, error) {
	return 0, false, nil
}
func (f *fakeRefreshRepoForLogoutHandler) Delete(_ context.Context, token string) (bool, error) {
	f.deletedToken = token
	return true, nil
}
func (f *fakeRefreshRepoForLogoutHandler) DeleteAllForUser(_ context.Context, userID uint) error {
	f.deleteAllUserID = userID
	return nil
}

func newLogoutTestKey(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return priv, &priv.PublicKey
}

func makeSignedAccessToken(t *testing.T, key *rsa.PrivateKey, userID uint, jti string, expOffset time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": fmt.Sprintf("%d", userID),
		"jti": jti,
		"exp": now.Add(expOffset).Unix(),
		"iat": now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}
	return raw
}

func newLogoutRouter(pubKey *rsa.PublicKey, jwtRepo domain.JWTRepository, refreshRepo domain.RefreshRepository) *gin.Engine {
	svc := svcauth.NewLogoutService(jwtRepo, refreshRepo)
	h := httpauth.NewLogoutHandler(svc, pubKey)
	r := gin.New()
	r.POST("/api/v1/auth/logout", h.Logout)
	return r
}

func postLogout(router *gin.Engine, query string, bearer string, refreshTokenCookie string) *httptest.ResponseRecorder {
	url := "/api/v1/auth/logout"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(""))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if refreshTokenCookie != "" {
		req.Header.Set("Cookie", "refresh_token="+refreshTokenCookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestLogoutHandler_DefaultFlow_Returns204AndRevokesCurrentSession(t *testing.T) {
	priv, pub := newLogoutTestKey(t)
	jwtRepo := &fakeJWTRepoForLogoutHandler{}
	refreshRepo := &fakeRefreshRepoForLogoutHandler{}
	router := newLogoutRouter(pub, jwtRepo, refreshRepo)

	bearer := makeSignedAccessToken(t, priv, 42, "jti-abc", 10*time.Minute)
	w := postLogout(router, "", bearer, "refresh-abc")

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if jwtRepo.jti != "jti-abc" {
		t.Fatalf("expected jti blacklist, got %q", jwtRepo.jti)
	}
	if refreshRepo.deletedToken != "refresh-abc" {
		t.Fatalf("expected current refresh token delete, got %q", refreshRepo.deletedToken)
	}
	if refreshRepo.deleteAllUserID != 0 {
		t.Fatalf("did not expect all-session revocation, got userID=%d", refreshRepo.deleteAllUserID)
	}
	cookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "refresh_token=") || !strings.Contains(cookie, "Max-Age=0") {
		t.Fatalf("expected cleared refresh cookie, got %q", cookie)
	}
}

func TestLogoutHandler_AllDevices_Returns204AndRevokesAllSessions(t *testing.T) {
	priv, pub := newLogoutTestKey(t)
	jwtRepo := &fakeJWTRepoForLogoutHandler{}
	refreshRepo := &fakeRefreshRepoForLogoutHandler{}
	router := newLogoutRouter(pub, jwtRepo, refreshRepo)

	bearer := makeSignedAccessToken(t, priv, 7, "jti-all", 15*time.Minute)
	w := postLogout(router, "all=true", bearer, "refresh-current")

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if refreshRepo.deleteAllUserID != 7 {
		t.Fatalf("expected all-session revocation for user 7, got %d", refreshRepo.deleteAllUserID)
	}
	if refreshRepo.deletedToken != "" {
		t.Fatalf("did not expect single-session delete on all=true, got %q", refreshRepo.deletedToken)
	}
}

func TestLogoutHandler_MissingCookie_StillReturns204(t *testing.T) {
	priv, pub := newLogoutTestKey(t)
	jwtRepo := &fakeJWTRepoForLogoutHandler{}
	refreshRepo := &fakeRefreshRepoForLogoutHandler{}
	router := newLogoutRouter(pub, jwtRepo, refreshRepo)

	bearer := makeSignedAccessToken(t, priv, 9, "jti-9", 5*time.Minute)
	w := postLogout(router, "", bearer, "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if refreshRepo.deletedToken != "" {
		t.Fatalf("did not expect delete when cookie missing, got %q", refreshRepo.deletedToken)
	}
}

func TestLogoutHandler_InvalidAccessToken_StillReturns204(t *testing.T) {
	_, pub := newLogoutTestKey(t)
	jwtRepo := &fakeJWTRepoForLogoutHandler{}
	refreshRepo := &fakeRefreshRepoForLogoutHandler{}
	router := newLogoutRouter(pub, jwtRepo, refreshRepo)

	w := postLogout(router, "all=true", "not-a-valid-jwt", "refresh-abc")

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if jwtRepo.jti != "" {
		t.Fatalf("did not expect blacklist with invalid access token, got %q", jwtRepo.jti)
	}
	if refreshRepo.deleteAllUserID != 0 {
		t.Fatalf("did not expect all-session delete with invalid access token, got %d", refreshRepo.deleteAllUserID)
	}
}
