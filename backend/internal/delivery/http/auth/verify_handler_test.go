package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
	httpauth "superset/auth-service/internal/delivery/http/auth"

	"github.com/gin-gonic/gin"
)

// --- fakes ---

type fakeVerifyRepo struct {
	reg         *domain.RegisterUser
	findErr     error
	activateErr error
}

func (f *fakeVerifyRepo) FindByHash(_ context.Context, _ string) (*domain.RegisterUser, error) {
	return f.reg, f.findErr
}

func (f *fakeVerifyRepo) Activate(_ context.Context, _ *domain.RegisterUser) error {
	return f.activateErr
}

func validPendingReg() *domain.RegisterUser {
	return &domain.RegisterUser{
		ID:               1,
		FirstName:        "Jane",
		Username:         "jane",
		Email:            "jane@example.com",
		Password:         "hash",
		RegistrationHash: "abc123",
		CreatedAt:        time.Now().Add(-1 * time.Hour),
	}
}

func newVerifyRouter(repo domain.VerifyRepository) *gin.Engine {
	svc := svcauth.NewVerifyService(repo)
	h := httpauth.NewVerifyHandler(svc, "http://localhost:3000")
	r := gin.New()
	r.GET("/api/v1/auth/verify", h.Verify)
	return r
}

func getVerify(router *gin.Engine, hash string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	url := "/api/v1/auth/verify"
	if hash != "" {
		url += "?hash=" + hash
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	router.ServeHTTP(w, req)
	return w
}

// --- tests ---

func TestVerifyHandler_Success_Redirects(t *testing.T) {
	router := newVerifyRouter(&fakeVerifyRepo{reg: validPendingReg()})
	w := getVerify(router, "abc123")

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d: %s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if loc != "/login?activated=true" {
		t.Errorf("expected redirect to /login?activated=true, got %q", loc)
	}
}

func TestVerifyHandler_MissingHash_Returns400(t *testing.T) {
	router := newVerifyRouter(&fakeVerifyRepo{})
	w := getVerify(router, "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestVerifyHandler_InvalidHash_Returns404(t *testing.T) {
	router := newVerifyRouter(&fakeVerifyRepo{reg: nil})
	w := getVerify(router, "badHash")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestVerifyHandler_ExpiredHash_Returns410(t *testing.T) {
	reg := validPendingReg()
	reg.CreatedAt = time.Now().Add(-25 * time.Hour)
	router := newVerifyRouter(&fakeVerifyRepo{reg: reg})
	w := getVerify(router, "abc123")

	if w.Code != http.StatusGone {
		t.Errorf("expected 410, got %d", w.Code)
	}
}

func TestVerifyHandler_ActivateError_Returns500(t *testing.T) {
	router := newVerifyRouter(&fakeVerifyRepo{
		reg:         validPendingReg(),
		activateErr: errors.New("tx error"),
	})
	w := getVerify(router, "abc123")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
