package query

import (
	"context"

	dbpool "superset/auth-service/internal/app/db"
	domainquery "superset/auth-service/internal/domain/query"
)

// Estimator runs EXPLAIN on a database to estimate query cost.
type Estimator interface {
	Estimate(ctx context.Context, sql string, conn dbpool.SQLConnection) (*domainquery.EstimateResult, error)
}

// NewEstimator returns the appropriate estimator for the given database driver.
func NewEstimator(driver string) Estimator {
	switch driver {
	case "postgresql":
		return &PostgresEstimator{}
	default:
		return &UnsupportedEstimator{driver: driver}
	}
}
