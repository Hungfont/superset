package db

import "context"

// DatabaseRepository manages database connection records.
type DatabaseRepository interface {
	// IsAdmin reports whether the given user has Admin role.
	IsAdmin(ctx context.Context, userID uint) (bool, error)
	// DatabaseNameExists returns true when a database with the given name already exists.
	DatabaseNameExists(ctx context.Context, databaseName string) (bool, error)
	// CreateDatabase inserts one database record.
	CreateDatabase(ctx context.Context, database *Database) error
}
