package tag

import pkgerrors "superset/auth-service/internal/pkg/autherrors"

var (
	ErrTagNotFound = pkgerrors.ErrTagNotFound
	ErrForbidden   = pkgerrors.ErrForbidden
)
