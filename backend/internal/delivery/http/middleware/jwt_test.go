package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"superset/auth-service/internal/delivery/http/middleware"
	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeJWTRepo struct {
	blacklisted map[string]bool
	cache       map[uint]*domain.UserContext
}

func newFakeJWTRepo() *fakeJWTRepo {
	return &fakeJWTRepo{
		blacklisted: make(map[string]bool),
		cache:       make(map[uint]*domain.UserContext),
	}
}

func (f *fakeJWTRepo) IsBlacklisted(_ context.Context, jti string) (bool, error) {
	return f.blacklisted[jti], nil
}

func (f *fakeJWTRepo) BlacklistJTI(_ context.Context, _ string, _ time.Duration) error {
	return nil
}

func (f *fakeJWTRepo) GetCachedUser(_ context.Context, userID uint) (*domain.UserContext, error) {
	u := f.cache[userID]
	return u, nil
}

func (f *fakeJWTRepo) SetCachedUser(_ context.Context, userID uint, u *domain.UserContext) error {
	f.cache[userID] = u
	return nil
}

type fakeUserRepo struct {
	user *domain.User
}

func (f *fakeUserRepo) FindByID(_ context.Context, _ uint) (*domain.User, error) {
	return f.user, nil
}

// --- helpers ---

func generateKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return priv, &priv.PublicKey
}

func makeToken(t *testing.T, priv *rsa.PrivateKey, userID uint, jti string, expOffset time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   fmt.Sprintf("%d", userID),
		"email": "user@example.com",
		"uname": "testuser",
		"jti":   jti,
		"iat":   now.Unix(),
		"exp":   now.Add(expOffset).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(priv)
	require.NoError(t, err)
	return signed
}

func setupRouter(pubKey *rsa.PublicKey, jwtRepo domain.JWTRepository, userRepo domain.UserRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTMiddleware(pubKey, jwtRepo, userRepo))
	r.GET("/protected", func(c *gin.Context) {
		u := c.MustGet(middleware.UserContextKey).(domain.UserContext)
		c.JSON(http.StatusOK, gin.H{"user_id": u.ID})
	})
	return r
}

// --- tests ---

func TestJWTMiddleware_ValidToken_InjectsContext(t *testing.T) {
	priv, pub := generateKeyPair(t)
	jti := uuid.NewString()
	token := makeToken(t, priv, 42, jti, 15*time.Minute)

	jwtRepo := newFakeJWTRepo()
	userRepo := &fakeUserRepo{user: &domain.User{ID: 42, Username: "testuser", Email: "user@example.com", Active: true}}

	r := setupRouter(pub, jwtRepo, userRepo)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTMiddleware_MissingToken_Returns401(t *testing.T) {
	_, pub := generateKeyPair(t)
	r := setupRouter(pub, newFakeJWTRepo(), &fakeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_ExpiredToken_Returns401(t *testing.T) {
	priv, pub := generateKeyPair(t)
	token := makeToken(t, priv, 1, uuid.NewString(), -1*time.Minute) // already expired

	r := setupRouter(pub, newFakeJWTRepo(), &fakeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_TamperedToken_Returns401(t *testing.T) {
	priv, _ := generateKeyPair(t)
	_, otherPub := generateKeyPair(t) // different key pair
	token := makeToken(t, priv, 1, uuid.NewString(), 15*time.Minute)

	r := setupRouter(otherPub, newFakeJWTRepo(), &fakeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_RevokedToken_Returns401(t *testing.T) {
	priv, pub := generateKeyPair(t)
	jti := uuid.NewString()
	token := makeToken(t, priv, 1, jti, 15*time.Minute)

	jwtRepo := newFakeJWTRepo()
	jwtRepo.blacklisted[jti] = true

	r := setupRouter(pub, jwtRepo, &fakeUserRepo{user: &domain.User{ID: 1, Active: true}})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_DeactivatedUser_Returns403(t *testing.T) {
	priv, pub := generateKeyPair(t)
	token := makeToken(t, priv, 99, uuid.NewString(), 15*time.Minute)

	jwtRepo := newFakeJWTRepo()
	// Cache the user as inactive
	jwtRepo.cache[99] = &domain.UserContext{ID: 99, Username: "inactive", Email: "x@x.com", Active: false}

	r := setupRouter(pub, jwtRepo, &fakeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestJWTMiddleware_CacheMiss_LoadsFromDB(t *testing.T) {
	priv, pub := generateKeyPair(t)
	token := makeToken(t, priv, 7, uuid.NewString(), 15*time.Minute)

	jwtRepo := newFakeJWTRepo() // empty cache
	userRepo := &fakeUserRepo{user: &domain.User{ID: 7, Username: "dbuser", Email: "db@example.com", Active: true}}

	r := setupRouter(pub, jwtRepo, userRepo)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Verify user was repopulated into cache
	cached, _ := jwtRepo.GetCachedUser(context.Background(), 7)
	require.NotNil(t, cached)
	assert.Equal(t, uint(7), cached.ID)
}
