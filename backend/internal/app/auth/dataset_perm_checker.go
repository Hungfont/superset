package auth

import (
	"context"
	"fmt"
	"strings"

	datasetdomain "superset/auth-service/internal/domain/dataset"
)

type DatasetPermChecker struct {
	getRoleNames func(ctx context.Context, userID uint) ([]string, error)
}

func NewDatasetPermChecker(fn func(ctx context.Context, userID uint) ([]string, error)) *DatasetPermChecker {
	return &DatasetPermChecker{getRoleNames: fn}
}

func (c *DatasetPermChecker) CanReadDataset(ctx context.Context, userID uint, d *datasetdomain.Dataset) (bool, error) {
	roles, err := c.getRoleNames(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("loading role names: %w", err)
	}
	for _, r := range roles {
		v := strings.ToLower(strings.TrimSpace(r))
		if v == "admin" || v == "alpha" {
			return true, nil
		}
	}
	return d.CreatedByFK == userID, nil
}
