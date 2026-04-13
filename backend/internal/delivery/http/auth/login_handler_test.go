package auth_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
	httpauth "superset/auth-service/internal/delivery/http/auth"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// --- fakes (scoped to login tests) ---

type fakeLoginRepo struct {
	user    *domain.User
	updated bool
}

func (f *fakeLoginRepo) FindByUsernameOrEmail(_ context.Context, _ string) (*domain.User, error) {
	return f.user, nil
}

func (f *fakeLoginRepo) UpdateLastLogin(_ context.Context, _ uint, _ int, _ time.Time) error {
	f.updated = true
	return nil
}

type fakeRateRepo struct {
	loginCount  int64
	failedCount int64
	lockExpiry  time.Time
}

func (f *fakeRateRepo) IncrLoginAttempt(_ context.Context, _ string) (int64, error) {
	f.loginCount++
	return f.loginCount, nil
}
func (f *fakeRateRepo) IncrFailedLogin(_ context.Context, _ string) (int64, error) {
	f.failedCount++
	return f.failedCount, nil
}
func (f *fakeRateRepo) ResetFailedLogin(_ context.Context, _ string) error { return nil }
func (f *fakeRateRepo) GetFailedLoginCount(_ context.Context, _ string) (int64, error) {
	return f.failedCount, nil
}
func (f *fakeRateRepo) SetLockout(_ context.Context, _ string) (time.Time, error) {
	f.lockExpiry = time.Now().Add(15 * time.Minute)
	return f.lockExpiry, nil
}
func (f *fakeRateRepo) GetLockoutExpiry(_ context.Context, _ string) (time.Time, error) {
	return f.lockExpiry, nil
}

type fakeRefreshRepo struct{}

func (f *fakeRefreshRepo) Store(_ context.Context, _ string, _ uint) error { return nil }
func (f *fakeRefreshRepo) GetUserID(_ context.Context, _ string) (uint, bool, error) {
	return 0, false, nil
}
func (f *fakeRefreshRepo) Delete(_ context.Context, _ string) (bool, error) { return true, nil }
func (f *fakeRefreshRepo) DeleteAllForUser(_ context.Context, _ uint) error  { return nil }

// --- helpers ---

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return key
}

func newLoginRouter(loginRepo domain.LoginRepository, rateRepo domain.RateLimitRepository, key *rsa.PrivateKey) *gin.Engine {
	svc := svcauth.NewLoginService(loginRepo, rateRepo, &fakeRefreshRepo{}, key)
	h := httpauth.NewLoginHandler(svc)
	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	return r
}

func postLogin(router *gin.Engine, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	router.ServeHTTP(w, req)
	return w
}

func activeTestUser() *domain.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("StrongP@ss1!"), bcrypt.MinCost)
	return &domain.User{
		ID:       1,
		Username: "johndoe",
		Email:    "john@example.com",
		Password: string(hash),
		Active:   true,
	}
}

// --- tests ---

func TestLoginHandler_Success_Returns200(t *testing.T) {
	key := newTestKey(t)
	router := newLoginRouter(&fakeLoginRepo{user: activeTestUser()}, &fakeRateRepo{}, key)
	w := postLogin(router, map[string]string{
		"username": "johndoe",
		"password": "StrongP@ss1!",
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] == "" {
		t.Error("expected access_token in response")
	}
	if resp["refresh_token"] == "" {
		t.Error("expected refresh_token in response")
	}

	// Refresh token should be in HttpOnly cookie
	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Error("expected Set-Cookie header for refresh_token")
	}
}

func TestLoginHandler_BadCredentials_Returns401(t *testing.T) {
	key := newTestKey(t)
	router := newLoginRouter(&fakeLoginRepo{user: activeTestUser()}, &fakeRateRepo{}, key)
	w := postLogin(router, map[string]string{
		"username": "johndoe",
		"password": "wrongpassword",
	})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLoginHandler_InactiveAccount_Returns403(t *testing.T) {
	key := newTestKey(t)
	u := activeTestUser()
	u.Active = false
	router := newLoginRouter(&fakeLoginRepo{user: u}, &fakeRateRepo{}, key)
	w := postLogin(router, map[string]string{
		"username": "johndoe",
		"password": "StrongP@ss1!",
	})

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestLoginHandler_LockedAccount_Returns423(t *testing.T) {
	key := newTestKey(t)
	router := newLoginRouter(
		&fakeLoginRepo{user: activeTestUser()},
		&fakeRateRepo{failedCount: 5, lockExpiry: time.Now().Add(10 * time.Minute)},
		key,
	)
	w := postLogin(router, map[string]string{
		"username": "johndoe",
		"password": "StrongP@ss1!",
	})

	if w.Code != http.StatusLocked {
		t.Errorf("expected 423, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["locked_until"] == "" {
		t.Error("expected locked_until in 423 response")
	}
}

func TestLoginHandler_RateLimit_Returns429(t *testing.T) {
	key := newTestKey(t)
	router := newLoginRouter(
		&fakeLoginRepo{user: activeTestUser()},
		&fakeRateRepo{loginCount: 20},
		key,
	)
	w := postLogin(router, map[string]string{
		"username": "johndoe",
		"password": "StrongP@ss1!",
	})

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

func TestLoginHandler_MissingFields_Returns422(t *testing.T) {
	key := newTestKey(t)
	router := newLoginRouter(&fakeLoginRepo{}, &fakeRateRepo{}, key)
	w := postLogin(router, map[string]string{"username": "johndoe"})

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}
