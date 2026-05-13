package query

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresEstimatorParsesExplainJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	explainJSON := `[{"Plan":{"Node Type":"Seq Scan","Startup Cost":0.00,"Total Cost":1250.50,"Plan Rows":50000,"Plan Width":100}}]`
	rows := sqlmock.NewRows([]string{"QUERY PLAN"}).AddRow(explainJSON)
	mock.ExpectQuery(`EXPLAIN \(FORMAT JSON\) SELECT`).WillReturnRows(rows)

	e := &PostgresEstimator{}
	result, err := e.Estimate(context.Background(), "SELECT * FROM orders", db)
	require.NoError(t, err)
	assert.True(t, result.Supported)
	assert.Equal(t, "postgresql", result.Driver)
	assert.Equal(t, float64(1250.50), result.TotalCost)
	assert.Equal(t, int64(50000), result.EstimatedRows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEstimatorHandlesExplainError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`EXPLAIN \(FORMAT JSON\)`).WillReturnError(sql.ErrConnDone)

	e := &PostgresEstimator{}
	result, err := e.Estimate(context.Background(), "SELECT * FROM orders", db)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEstimatorHandlesInvalidJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"QUERY PLAN"}).AddRow(`not json`)
	mock.ExpectQuery(`EXPLAIN \(FORMAT JSON\)`).WillReturnRows(rows)

	e := &PostgresEstimator{}
	result, err := e.Estimate(context.Background(), "SELECT 1", db)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
