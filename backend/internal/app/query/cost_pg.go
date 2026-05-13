package query

import (
	"context"
	"encoding/json"
	"fmt"

	dbpool "superset/auth-service/internal/app/db"
	domainquery "superset/auth-service/internal/domain/query"
)

type PostgresEstimator struct{}

type pgPlanNode struct {
	Plan pgPlanDetail `json:"Plan"`
}

type pgPlanDetail struct {
	TotalCost float64 `json:"Total Cost"`
	PlanRows  int64   `json:"Plan Rows"`
}

func (e *PostgresEstimator) Estimate(ctx context.Context, sqlStr string, conn dbpool.SQLConnection) (*domainquery.EstimateResult, error) {
	if conn == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	explainSQL := "EXPLAIN (FORMAT JSON) " + sqlStr
	rows, err := conn.QueryContext(ctx, explainSQL)
	if err != nil {
		return nil, fmt.Errorf("EXPLAIN failed: %w", err)
	}
	defer rows.Close()

	var planText string
	if rows.Next() {
		if err := rows.Scan(&planText); err != nil {
			return nil, fmt.Errorf("scanning EXPLAIN result: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading EXPLAIN result: %w", err)
	}

	var nodes []pgPlanNode
	if err := json.Unmarshal([]byte(planText), &nodes); err != nil {
		return nil, fmt.Errorf("parsing EXPLAIN JSON: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("empty EXPLAIN result")
	}

	return &domainquery.EstimateResult{
		Supported:     true,
		Driver:        "postgresql",
		TotalCost:     nodes[0].Plan.TotalCost,
		EstimatedRows: nodes[0].Plan.PlanRows,
	}, nil
}
