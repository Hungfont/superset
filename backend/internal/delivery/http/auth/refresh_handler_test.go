package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
	httpauth "superset/auth-service/internal/delivery/http/auth"

	"github.com/gin-gonic/gin"
)

// --- fakes for refresh handler ---

type fakeRefreshRepoForHandler struct {
	storedUserID uint
	found        bool
	deleted      bool
}

func (f *fakeRefreshRepoForHandler) Store(_ context.Context, _ string, _ uint) error { return nil }
func (f *fakeRefreshRepoForHandler) GetUserID(_ context.Context, _ string) (uint, bool, error) {
	return f.storedUserID, f.found, nil
}
func (f *fakeRefreshRepoForHandler) Delete(_ context.Context, _ string) (bool, error) {
	return f.deleted, nil
}
func (f *fakeRefreshRepoForHandler) DeleteAllForUser(_ context.Context, _ uint) error { return nil }

type fakeUserRepoForHandler struct {
	user *domain.User
}

func (f *fakeUserRepoForHandler) FindByID(_ context.Context, _ uint) (*domain.User, error) {
	return f.user, nil
}

// --- helpers ---

func newRefreshRouter(t *testing.T, refreshRepo domain.RefreshRepository, userRepo domain.UserRepository) *gin.Engine {
	t.Helper()
	key := newTestKey(t)
	svc := svcauth.NewRefreshService(refreshRepo, userRepo, key)
	h := httpauth.NewRefreshHandler(svc)
	r := gin.New()
	r.POST("/api/v1/auth/refresh", h.Refresh)
	return r
}

func postRefresh(router *gin.Engine, cookie string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(""))
	if cookie != "" {
		req.Header.Set("Cookie", "refresh_token="+cookie)
	}
	router.ServeHTTP(w, req)
	return w
}

func activeHandlerUser() *domain.User {
	return &domain.User{ID: 42, Username: "alice", Email: "alice@example.com", Active: true}
}

// --- tests ---

func TestRefreshHandler_HappyPath_Returns200WithNewAccessToken(t *testing.T) {
	refreshRepo := &fakeRefreshRepoForHandler{storedUserID: 42, found: true, deleted: true}
	userRepo := &fakeUserRepoForHandler{user: activeHandlerUser()}
	router := newRefreshRouter(t, refreshRepo, userRepo)

	w := postRefresh(router, "valid-refresh-token")

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] == "" {
		t.Error("expected access_token in response body")
	}

	// Rotated refresh token must be set as a new HttpOnly cookie.
	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Error("expected Set-Cookie header with rotated refresh_token")
	}
	if !strings.Contains(cookie, "HttpOnly") {
		t.Error("expected HttpOnly flag on refresh_token cookie")
	}
}

func TestRefreshHandler_MissingCookie_Returns401(t *testing.T) {
	refreshRepo := &fakeRefreshRepoForHandler{}
	userRepo := &fakeUserRepoForHandler{}
	router := newRefreshRouter(t, refreshRepo, userRepo)

	w := postRefresh(router, "") // no cookie

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRefreshHandler_UnknownToken_Returns401(t *testing.T) {
	refreshRepo := &fakeRefreshRepoForHandler{found: false}
	userRepo := &fakeUserRepoForHandler{}
	router := newRefreshRouter(t, refreshRepo, userRepo)

	w := postRefresh(router, "unknown-token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRefreshHandler_ReuseAttack_Returns401AndClearsCookie(t *testing.T) {
	// found=true but deleted=false simulates reuse (GET succeeded but DEL missed).
	refreshRepo := &fakeRefreshRepoForHandler{storedUserID: 42, found: true, deleted: false}
	userRepo := &fakeUserRepoForHandler{}
	router := newRefreshRouter(t, refreshRepo, userRepo)

	w := postRefresh(router, "reused-token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	// Cookie must be cleared. Go's net/http renders MaxAge<0 as "Max-Age=0" in the header.
	cookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "Max-Age=0") {
		t.Errorf("expected cookie to be cleared (Max-Age=0) on reuse attack, got: %s", cookie)
	}
}

func TestRefreshHandler_InactiveUser_Returns401AndClearsCookie(t *testing.T) {
	refreshRepo := &fakeRefreshRepoForHandler{storedUserID: 42, found: true, deleted: true}
	u := activeHandlerUser()
	u.Active = false
	userRepo := &fakeUserRepoForHandler{user: u}
	router := newRefreshRouter(t, refreshRepo, userRepo)

	w := postRefresh(router, "valid-token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
