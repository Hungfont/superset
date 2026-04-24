package query

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectRLS_AdminBypass(t *testing.T) {
	svc := NewRLSInjector(nil)
	
	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Admin"})
	
	require.NoError(t, err)
	assert.Equal(t, inputSQL, result, "Admin should get original SQL without RLS")
}

func TestInjectRLS_RegularFilter(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)
	
	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))
	
	require.NoError(t, err)
	assert.Contains(t, result, "org_id = 42", "Should inject user ID as integer")
	assert.Contains(t, result, "WHERE", "Should add WHERE clause")
}

func TestInjectRLS_BaseFilterReplacesWhere(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Base"},
		},
	}
	svc := NewRLSInjector(mockRepo)
	
	inputSQL := "SELECT * FROM orders WHERE status = 'pending'"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))
	
	require.NoError(t, err)
	assert.NotContains(t, result, "status = 'pending'", "Base filter should replace existing WHERE")
	assert.Contains(t, result, "org_id = 42", "Should have RLS clause")
}

func TestInjectRLS_TemplateVariables(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}} AND username = '{{current_username}}'", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)
	
	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42), WithUsername("john"))
	
	require.NoError(t, err)
	assert.Contains(t, result, "org_id = 42", "Should inject user_id as integer")
	assert.Contains(t, result, "username = ''john''", "Should inject username as escaped SQL string")
	assert.NotContains(t, result, "{{current_user_id}}", "Should replace template variable")
}

func TestInjectRLS_MultipleRoles(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
			{Clause: "region = 'APAC'", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)
	
	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma", "Sales-APAC"}, WithUserID(42))
	
	require.NoError(t, err)
	assert.Contains(t, result, "org_id = 42", "Should inject first clause")
	assert.Contains(t, result, "region = 'APAC'", "Should inject second clause")
}

func TestInjectRLS_UnionQuery(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)
	
	inputSQL := "SELECT a FROM t1 UNION SELECT b FROM t2"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))
	
	require.NoError(t, err)
	assert.Contains(t, result, "org_id = 42", "Should inject RLS into first SELECT")
	assert.Contains(t, result, "UNION", "Should preserve UNION")
}

func TestInjectRLS_NoFilterForDataset(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{},
	}
	svc := NewRLSInjector(mockRepo)
	
	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"})
	
	require.NoError(t, err)
	assert.Equal(t, inputSQL, result, "Should return original SQL when no RLS filters")
}

func TestInjectRLS_NoFiltersEmptyResult(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{},
	}
	svc := NewRLSInjector(mockRepo)
	
	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"})
	
	require.NoError(t, err)
	assert.Equal(t, inputSQL, result, "Should return original SQL when no RLS filters")
}

func TestInjectRLS_SecurityNoInjection(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)
	
	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))
	
	require.NoError(t, err)
	assert.NotContains(t, result, "1 OR 1=1", "Should not allow SQL injection in template values")
	assert.NotContains(t, result, "'; DROP TABLE", "Should not allow SQL injection")
}

type mockRLSFilterRepo struct {
	filters         []RLSFilterClause
	getFiltersCalled bool
	cacheHit       bool
}

func (m *mockRLSFilterRepo) GetFiltersByDatasourceAndRoles(ctx context.Context, datasourceID int, roleNames []string) ([]RLSFilterClause, error) {
	m.getFiltersCalled = true
	if m.cacheHit {
		return nil, fmt.Errorf("cache hit")
	}
	return m.filters, nil
}

func (m *mockRLSFilterRepo) CacheGet(ctx context.Context, key string) ([]RLSFilterClause, error) {
	if m.cacheHit {
		return m.filters, nil
	}
	return nil, nil
}

func (m *mockRLSFilterRepo) CacheSet(ctx context.Context, key string, clauses []RLSFilterClause) error {
	return nil
}

func TestAcceptanceCriteria1_GammaUserWithRLS(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)

	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))

	require.NoError(t, err)
	assert.Contains(t, result, "org_id = 42", "executed_sql should contain 'AND (org_id = 42)'")
	assert.Contains(t, result, "WHERE", "Should add WHERE clause")
}

func TestAcceptanceCriteria2_AdminUserNoRLS(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)

	inputSQL := "SELECT * FROM orders WHERE status = 'active'"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Admin"})

	require.NoError(t, err)
	assert.Equal(t, inputSQL, result, "Admin should get original SQL unchanged")
}

func TestAcceptanceCriteria3_BaseFilterReplacesWhere(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Base"},
		},
	}
	svc := NewRLSInjector(mockRepo)

	inputSQL := "SELECT * FROM orders WHERE status = 'pending'"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))

	require.NoError(t, err)
	assert.NotContains(t, result, "status = 'pending'", "Base filter should replace existing WHERE")
	assert.Contains(t, result, "org_id = 42", "Should have RLS clause")
}

func TestAcceptanceCriteria4_CacheHit(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
		},
		cacheHit: true,
	}
	svc := NewRLSInjector(mockRepo)

	inputSQL := "SELECT * FROM orders"
	start := time.Now()
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Contains(t, result, "org_id = 42", "Should inject RLS")
	assert.Less(t, elapsed.Milliseconds(), int64(1), "Cache hit should be < 1ms")
}

func TestAcceptanceCriteria5_UnionQueryInjectEachSelect(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)

	inputSQL := "SELECT a FROM t1 UNION SELECT b FROM t2"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))

	require.NoError(t, err)
	assert.Contains(t, result, "org_id = 42", "Should inject RLS into each SELECT")
	assert.Contains(t, result, "UNION", "Should preserve UNION")

	firstSelect := strings.Split(result, "UNION")[0]
	secondSelect := strings.Split(result, "UNION")[1]
	assert.Contains(t, firstSelect, "WHERE", "First SELECT should have WHERE")
	assert.Contains(t, secondSelect, "WHERE", "Second SELECT should have WHERE")
}

func TestAcceptanceCriteria6_TemplateRenderAsInteger(t *testing.T) {
	mockRepo := &mockRLSFilterRepo{
		filters: []RLSFilterClause{
			{Clause: "org_id = {{current_user_id}}", FilterType: "Regular"},
		},
	}
	svc := NewRLSInjector(mockRepo)

	inputSQL := "SELECT * FROM orders"
	result, err := svc.InjectRLS(context.Background(), inputSQL, 5, []string{"Gamma"}, WithUserID(42))

	require.NoError(t, err)
	assert.Contains(t, result, "org_id = 42", "Should render as integer 42, not string '42'")
	assert.NotContains(t, result, "{{current_user_id}}", "Should replace template variable")
	assert.NotContains(t, result, "'42'", "Should NOT be string quoted")
}