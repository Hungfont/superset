package chart

import pkgerrors "superset/auth-service/internal/pkg/autherrors"

var (
	ErrChartNotFound     = pkgerrors.ErrChartNotFound
	ErrDashboardNotFound = pkgerrors.ErrDashboardNotFound
	ErrForbidden         = pkgerrors.ErrForbidden
)
