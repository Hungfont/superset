package db

import pkgerrors "superset/auth-service/internal/pkg/autherrors"

var (
	ErrTokenInvalid                 = pkgerrors.ErrTokenInvalid
	ErrForbidden                    = pkgerrors.ErrForbidden
	ErrRateLimited                  = pkgerrors.ErrRateLimited
	ErrInvalidDatabase              = pkgerrors.ErrInvalidDatabase
	ErrInvalidDatabaseURI           = pkgerrors.ErrInvalidDatabaseURI
	ErrDatabaseNotFound             = pkgerrors.ErrDatabaseNotFound
	ErrDatabaseNameExists           = pkgerrors.ErrDatabaseNameExists
	ErrDatabaseInUse                = pkgerrors.ErrDatabaseInUse
	ErrDatabaseUnreachable          = pkgerrors.ErrDatabaseUnreachable
	ErrDatabaseTimeout              = pkgerrors.ErrDatabaseTimeout
	ErrDatabaseConnectionTestFailed = pkgerrors.ErrDatabaseConnectionTestFailed
	ErrDatabaseCredentialEncryption = pkgerrors.ErrDatabaseCredentialEncryption
	ErrUnknownDatabaseDriver        = pkgerrors.ErrUnknownDatabaseDriver
)
