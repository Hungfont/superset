package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	domain "superset/auth-service/internal/domain/auth"
	dbdomain "superset/auth-service/internal/domain/db"
	dbapp "superset/auth-service/internal/app/db"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mock helpers
type mockDBRepo struct {
	dbdomain.DatabaseRepository // embed for unimplemented methods
	db                          *dbdomain.Database
	err                         error
}

func (m *mockDBRepo) GetDatabaseByID(_ context.Context, id uint) (*dbdomain.Database, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.db, nil
}

type mockPool struct {
	err error
}

func (m *mockPool) Get(_ context.Context, _ uint, _ string) (dbapp.SQLConnection, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &mockConn{}, nil
}
func (m *mockPool) GetPinned(_ context.Context, _ uint, _ string) (*sql.Conn, error) { return nil, nil }
func (m *mockPool) Close(_ context.Context, _ uint) error { return nil }
func (m *mockPool) Shutdown(_ context.Context) error { return nil }

type mockConn struct{}
func (m *mockConn) SetMaxOpenConns(int) {}
func (m *mockConn) SetMaxIdleConns(int) {}
func (m *mockConn) SetConnMaxLifetime(time.Duration) {}
func (m *mockConn) PingContext(_ context.Context) error { return nil }
func (m *mockConn) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) { return nil, nil }
func (m *mockConn) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	if strings.Contains(query, "nonexistent_col") {
		return nil, errors.New("column \"nonexistent_col\" does not exist")
	}
	return nil, nil
}
func (m *mockConn) Close() error { return nil }

func newTestRLSService() *RLSService {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	rdb.FlushDB(context.Background())
	return &RLSService{
		rdb: rdb,
		dbRepo: &mockDBRepo{
			db: &dbdomain.Database{
				SQLAlchemyURI: "postgresql://user:pass@localhost:5432/test",
			},
		},
		poolManager: &mockPool{},
	}
}

func TestValidate_SyntaxOnly_Valid(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{ID: 1}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause: "org_id = 42",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, result.IsValid)
	assert.Equal(t, "syntax", result.Phase)
	assert.Equal(t, "org_id = 42", result.RenderedClause)
}

func TestValidate_SyntaxOnly_Invalid(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{ID: 2}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause: "org_id = AND",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.False(t, result.IsValid)
	assert.Equal(t, "syntax", result.Phase)
	assert.NotEmpty(t, result.Error)
	assert.NotNil(t, result.ErrorPosition)
}

func TestValidate_SyntaxWithTemplate(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{ID: 3}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause:       "org_id = {{current_user_id}}",
		TestUserID:   intPtr(42),
		TestUsername: "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, result.IsValid)
	assert.Equal(t, "syntax", result.Phase)
	// Phase 1 returns raw clause (template vars not rendered yet)
	assert.Equal(t, "org_id = {{current_user_id}}", result.RenderedClause)
}

func TestValidate_Runtime_ColumnNotFound(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{ID: 4}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause:       "nonexistent_col = 1",
		DatabaseID:   uintPtr(1),
		TableName:    "orders",
		Schema:       "public",
		TestUserID:   intPtr(42),
		TestUsername: "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.False(t, result.IsValid)
	assert.Equal(t, "runtime", result.Phase)
	assert.Contains(t, result.Error, "column \"nonexistent_col\" does not exist")
}

func TestValidate_Runtime_ValidProbe(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{ID: 5}

	result, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
		Clause:       "org_id = {{current_user_id}}",
		DatabaseID:   uintPtr(1),
		TableName:    "orders",
		Schema:       "public",
		TestUserID:   intPtr(42),
		TestUsername: "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, result.IsValid)
	assert.Equal(t, "runtime", result.Phase)
	assert.Equal(t, "org_id = 42", result.RenderedClause)
}

func TestValidate_RateLimit(t *testing.T) {
	svc := newTestRLSService()
	uc := domain.UserContext{ID: 99}

	for i := 0; i < 61; i++ {
		_, status, err := svc.Validate(context.Background(), uc, domain.ValidateRequest{
			Clause: "org_id = 1",
		})
		if i < 60 {
			require.NoError(t, err, "call %d should succeed", i)
			assert.Equal(t, 200, status, "call %d", i)
		} else {
			require.Error(t, err)
			assert.Equal(t, 429, status)
			assert.Contains(t, err.Error(), "rate limit")
		}
	}
}

func intPtr(i int) *int { return &i }
func uintPtr(i uint) *uint { return &i }
