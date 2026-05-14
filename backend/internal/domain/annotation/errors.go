package annotation

import pkgerrors "superset/auth-service/internal/pkg/autherrors"

var (
	ErrAnnotationLayerNotFound = pkgerrors.ErrAnnotationLayerNotFound
	ErrAnnotationNotFound      = pkgerrors.ErrAnnotationNotFound
	ErrForbidden               = pkgerrors.ErrForbidden
)
