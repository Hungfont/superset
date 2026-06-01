package postgres_test

import (
	"context"
	"testing"

	"superset/auth-service/internal/domain/chart"
	"superset/auth-service/internal/repository/postgres"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	dialector := gormpg.New(gormpg.Config{Conn: db, DriverName: "postgres"})
	gormDB, err := gorm.Open(dialector, &gorm.Config{SkipDefaultTransaction: true})
	require.NoError(t, err)
	return gormDB, mock
}

func TestCreateSlice_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`INSERT INTO "slices"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	slice := &chart.Slice{
		SliceName:      "Revenue by Month",
		VizType:        "bar",
		DatasourceID:   "3",
		DatasourceType: "table",
		DatasourceName: "sales",
		Perm:           "[sales](id:1)",
	}

	err := repo.CreateSlice(context.Background(), slice)
	assert.NoError(t, err)
	assert.Equal(t, uint(42), slice.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateSlice_DBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`INSERT INTO "slices"`).
		WillReturnError(gorm.ErrInvalidDB)

	err := repo.CreateSlice(context.Background(), &chart.Slice{SliceName: "Test"})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateSliceUser_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`INSERT INTO "slice_user"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(99))

	su := &chart.SliceUser{SliceID: 42, UserID: 10}
	err := repo.CreateSliceUser(context.Background(), su)
	assert.NoError(t, err)
	assert.Equal(t, uint(99), su.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateSliceUser_DBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`INSERT INTO "slice_user"`).
		WillReturnError(gorm.ErrInvalidDB)

	err := repo.CreateSliceUser(context.Background(), &chart.SliceUser{SliceID: 42, UserID: 10})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChartRepository_ImplementsInterface(t *testing.T) {
	// Compile-time assertion already in chart.go; runtime sanity check
	gormDB, _ := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)
	var _ chart.Repository = repo
	assert.NotNil(t, repo)
}
