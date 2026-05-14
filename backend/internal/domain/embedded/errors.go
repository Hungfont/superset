package embedded

import pkgerrors "superset/auth-service/internal/pkg/autherrors"

var (
	ErrCssTemplateNotFound          = pkgerrors.ErrCssTemplateNotFound
	ErrEmbeddedDashboardNotFound    = pkgerrors.ErrEmbeddedDashboardNotFound
	ErrKeyValueNotFound             = pkgerrors.ErrKeyValueNotFound
	ErrForbidden                    = pkgerrors.ErrForbidden
)
