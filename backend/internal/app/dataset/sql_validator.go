package dataset

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/dataset"
	dbdomain "superset/auth-service/internal/domain/db"
)

// SQLConnectionPool provides database connections for SQL validation.
type SQLConnectionPool interface {
	Get(ctx context.Context, databaseID uint, sqlalchemyURI string) (SQLConn, error)
}

// SQLConn is the minimum contract for executing validation queries.
type SQLConn interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	Close() error
}

// databaseURIResolver resolves SQLAlchemy URIs for database IDs.
type databaseURIResolver interface {
	GetDatabaseByID(ctx context.Context, id uint) (*dbdomain.Database, error)
}

type defaultSQLValidator struct {
	pool       SQLConnectionPool
	dbResolver databaseURIResolver
}

func NewSQLValidator(pool SQLConnectionPool, dbResolver databaseURIResolver) SQLValidator {
	return &defaultSQLValidator{pool: pool, dbResolver: dbResolver}
}

func (v *defaultSQLValidator) ValidateSQL(ctx context.Context, databaseID uint, sql string) ([]domain.Column, error) {
	if v.pool == nil || v.dbResolver == nil {
		return nil, nil
	}

	dbRec, err := v.dbResolver.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("getting database %d for sql validation: %w", databaseID, err)
	}
	if dbRec == nil {
		return nil, fmt.Errorf("database %d not found for sql validation", databaseID)
	}

	conn, err := v.pool.Get(ctx, databaseID, dbRec.SQLAlchemyURI)
	if err != nil {
		return nil, fmt.Errorf("getting connection for sql validation: %w", err)
	}
	defer conn.Close()

	query := fmt.Sprintf("SELECT * FROM (%s) AS _validate LIMIT 0", strings.TrimSpace(sql))

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sql validation query failed: %w", err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns from validation result: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("getting column types from validation result: %w", err)
	}

	columns := make([]domain.Column, 0, len(columnNames))
	for i, colName := range columnNames {
		colType := ""
		if i < len(columnTypes) && columnTypes[i] != nil {
			colType = columnTypes[i].DatabaseTypeName()
		}

		isDttm := false
		colTypeLower := strings.ToLower(colType)
		if strings.Contains(colTypeLower, "timestamp") || strings.Contains(colTypeLower, "date") || strings.Contains(colTypeLower, "time") {
			isDttm = true
		}

		columns = append(columns, domain.Column{
			ColumnName: colName,
			Type:       colType,
			IsDateTime: isDttm,
			IsActive:   true,
		})
	}

	return columns, nil
}
