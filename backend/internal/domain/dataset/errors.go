package dataset

import (
	pkgerrors "superset/auth-service/internal/pkg/autherrors"
)

var (
	ErrTokenInvalid       = pkgerrors.ErrTokenInvalid
	ErrForbidden          = pkgerrors.ErrForbidden
	ErrInvalidDatabase    = pkgerrors.ErrInvalidDatabase
	ErrInvalidDataset     = pkgerrors.ErrInvalidDataset
	ErrDatasetDuplicate   = pkgerrors.ErrDatasetDuplicate
	ErrDatasetSyncEnqueue = pkgerrors.ErrDatasetSyncEnqueue
	ErrInvalidSQL         = pkgerrors.ErrInvalidSQL
	ErrSQLNotSelect       = pkgerrors.ErrSQLNotSelect
	ErrSQLSemicolon       = pkgerrors.ErrSQLSemicolon
	ErrSQLSemanticError   = pkgerrors.ErrSQLSemanticError
	ErrInvalidMainDttmCol = pkgerrors.ErrInvalidMainDttmCol
	ErrColumnNotFound     = pkgerrors.ErrColumnNotFound
	ErrInvalidExpression  = pkgerrors.ErrInvalidExpression
	ErrInvalidDateFormat  = pkgerrors.ErrInvalidDateFormat
)
