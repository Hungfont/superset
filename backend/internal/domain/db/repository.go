package db

import "context"

// DatabaseRepository manages database connection records.
type DatabaseRepository interface {
	// IsAdmin reports whether the given user has Admin role.
	IsAdmin(ctx context.Context, userID uint) (bool, error)
	// DatabaseNameExists returns true when a database with the given name already exists.
	DatabaseNameExists(ctx context.Context, databaseName string) (bool, error)
	// GetDatabaseByID loads one database record by ID.
	GetDatabaseByID(ctx context.Context, databaseID uint) (*Database, error)
	// CreateDatabase inserts one database record.
	CreateDatabase(ctx context.Context, database *Database) error
}
