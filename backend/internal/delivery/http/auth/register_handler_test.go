package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
	httpauth "superset/auth-service/internal/delivery/http/auth"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// --- fakes ---

type fakeRepo struct {
	emailExists    bool
	usernameExists bool
}

func (f *fakeRepo) EmailExists(_ context.Context, _ string) (bool, error) {
	return f.emailExists, nil
}
func (f *fakeRepo) UsernameExists(_ context.Context, _ string) (bool, error) {
	return f.usernameExists, nil
}
func (f *fakeRepo) Create(_ context.Context, _ *domain.RegisterUser) error { return nil }

type fakeMailer struct{}

func (fakeMailer) SendVerification(_, _ string) error { return nil }

// --- helper ---

func newRouter(repo domain.RegisterUserRepository) *gin.Engine {
	svc := svcauth.NewRegisterService(repo, fakeMailer{}, "http://localhost:3000")
	h := httpauth.NewRegisterHandler(svc)
	r := gin.New()
	r.POST("/api/v1/auth/register", h.Register)
	return r
}

func postRegister(router *gin.Engine, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

// --- tests ---

func TestRegisterHandler_Success(t *testing.T) {
	router := newRouter(&fakeRepo{})
	w := postRegister(router, map[string]string{
		"first_name": "John",
		"last_name":  "Doe",
		"username":   "johndoe",
		"email":      "john@example.com",
		"password":   "StrongP@ss1!",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_WeakPassword_Returns400(t *testing.T) {
	router := newRouter(&fakeRepo{})
	w := postRegister(router, map[string]string{
		"first_name": "John",
		"last_name":  "Doe",
		"username":   "johndoe",
		"email":      "john@example.com",
		"password":   "weak",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegisterHandler_DuplicateEmail_Returns409(t *testing.T) {
	router := newRouter(&fakeRepo{emailExists: true})
	w := postRegister(router, map[string]string{
		"first_name": "John",
		"last_name":  "Doe",
		"username":   "johndoe",
		"email":      "john@example.com",
		"password":   "StrongP@ss1!",
	})

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestRegisterHandler_MissingField_Returns422(t *testing.T) {
	router := newRouter(&fakeRepo{})
	w := postRegister(router, map[string]string{
		"email": "john@example.com",
		// missing first_name, last_name, username, password
	})

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}
