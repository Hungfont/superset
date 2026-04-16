package db

import pkgerrors "superset/auth-service/internal/pkg/autherrors"

var (
	ErrTokenInvalid                 = pkgerrors.ErrTokenInvalid
	ErrForbidden                    = pkgerrors.ErrForbidden
	ErrInvalidDatabase              = pkgerrors.ErrInvalidDatabase
	ErrInvalidDatabaseURI           = pkgerrors.ErrInvalidDatabaseURI
	ErrDatabaseNameExists           = pkgerrors.ErrDatabaseNameExists
	ErrDatabaseConnectionTestFailed = pkgerrors.ErrDatabaseConnectionTestFailed
	ErrDatabaseCredentialEncryption = pkgerrors.ErrDatabaseCredentialEncryption
)
