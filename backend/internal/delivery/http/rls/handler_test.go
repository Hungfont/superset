package rls

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockValidateService struct {
	result domain.ValidateResult
	status int
	err    error
}

func (m *mockValidateService) List(ctx context.Context, params domain.RLSFilterListParams) (*domain.RLSFilterListResult, error) {
	return nil, nil
}
func (m *mockValidateService) GetByID(ctx context.Context, id uint) (*domain.RLSFilterResponse, error) {
	return nil, nil
}
func (m *mockValidateService) Create(ctx context.Context, actorUserID uint, ipAddress string, req domain.CreateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	return nil, nil
}
func (m *mockValidateService) Update(ctx context.Context, actorUserID uint, ipAddress string, id uint, req domain.UpdateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	return nil, nil
}
func (m *mockValidateService) Delete(ctx context.Context, actorUserID uint, ipAddress string, id uint) error {
	return nil
}
func (m *mockValidateService) Validate(ctx context.Context, uc domain.UserContext, req domain.ValidateRequest) (domain.ValidateResult, int, error) {
	return m.result, m.status, m.err
}

func setupValidateTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &Handler{svc: &mockValidateService{
		result: domain.ValidateResult{IsValid: true, Phase: "syntax", RenderedClause: "org_id = 42"},
		status: 200,
	}}
	r := gin.New()
	r.POST("/api/v1/rls/validate", func(c *gin.Context) {
		c.Set("user", domain.UserContext{ID: 1})
		h.Validate(c)
	})
	return r
}

func TestValidateHandler_SyntaxValid(t *testing.T) {
	r := setupValidateTest()

	body, _ := json.Marshal(domain.ValidateRequest{Clause: "org_id = 42"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rls/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result domain.ValidateResult
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.True(t, result.IsValid)
	assert.Equal(t, "syntax", result.Phase)
}

func TestValidateHandler_MalformedBody(t *testing.T) {
	r := setupValidateTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rls/validate", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidateHandler_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{svc: &mockValidateService{
		result: domain.ValidateResult{IsValid: true, Phase: "syntax"},
		status: 200,
	}}
	r := gin.New()
	r.POST("/api/v1/rls/validate", h.Validate)

	body, _ := json.Marshal(domain.ValidateRequest{Clause: "org_id = 42"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rls/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}
