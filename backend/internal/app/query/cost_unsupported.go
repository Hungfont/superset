package query

import (
	"context"

	dbpool "superset/auth-service/internal/app/db"
	domainquery "superset/auth-service/internal/domain/query"
)

type UnsupportedEstimator struct {
	driver string
}

func (e *UnsupportedEstimator) Estimate(ctx context.Context, sql string, conn dbpool.SQLConnection) (*domainquery.EstimateResult, error) {
	return &domainquery.EstimateResult{Supported: false}, nil
}
