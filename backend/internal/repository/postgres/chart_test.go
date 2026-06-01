package postgres_test

import (
	"context"
	"testing"
	"time"

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

func TestGetSliceDetail_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`SELECT.*FROM "slices"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "slice_name", "viz_type", "datasource_id", "datasource_type",
			"datasource_name", "perm", "last_saved_at",
		}).AddRow(1, "Test", "bar", "3", "table", "sales", "[sales](id:1)", time.Now()))

	slice, err := repo.GetSliceDetail(context.Background(), 1)
	assert.NoError(t, err)
	assert.NotNil(t, slice)
	assert.Equal(t, uint(1), slice.ID)
}

func TestGetSliceDetail_NotFound(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`SELECT.*FROM "slices"`).
		WillReturnError(gorm.ErrRecordNotFound)

	slice, err := repo.GetSliceDetail(context.Background(), 999)
	assert.NoError(t, err)
	assert.Nil(t, slice)
}

func TestDashboardCount_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "dashboard_slices"`).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.DashboardCount(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestListSlices_AdminVisibility(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := postgres.NewChartRepository(gormDB)

	mock.ExpectQuery(`SELECT count\(\*\).*FROM "slices"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`SELECT.*FROM "slices".*ORDER BY.*last_saved_at DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "slice_name", "viz_type", "perm", "last_saved_at"}).
			AddRow(1, "Test", "bar", "[sales](id:1)", time.Now()))

	slices, total, err := repo.ListSlices(context.Background(), &chart.SliceListFilter{
		VisibilityAll: true, Page: 1, PageSize: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, slices, 1)
}
