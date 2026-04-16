package db

import "context"

// DatabaseRepository manages database connection records.
type DatabaseRepository interface {
	// IsAdmin reports whether the given user has Admin role.
	IsAdmin(ctx context.Context, userID uint) (bool, error)
	// GetRoleNamesByUser returns normalized role names for an actor.
	GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error)
	// DatabaseNameExists returns true when a database with the given name already exists.
	DatabaseNameExists(ctx context.Context, databaseName string) (bool, error)
	// ListDatabases returns paginated databases for a visibility scope.
	ListDatabases(ctx context.Context, filters DatabaseListFilters) (DatabaseListResult, error)
	// GetVisibleDatabaseByID returns a database only if visible for the given scope.
	GetVisibleDatabaseByID(ctx context.Context, databaseID uint, scope DatabaseVisibilityScope, actorUserID uint) (*DatabaseWithDatasetCount, error)
	// GetDatabaseByID loads one database record by ID.
	GetDatabaseByID(ctx context.Context, databaseID uint) (*Database, error)
	// CreateDatabase inserts one database record.
	CreateDatabase(ctx context.Context, database *Database) error
}
