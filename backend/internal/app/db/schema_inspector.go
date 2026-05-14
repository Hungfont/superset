package auth

import (
	"context"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/db"
)

const (
	databaseTablesDefaultPage     = 1
	databaseTablesDefaultPageSize = 50
	databaseTablesMaxPageSize     = 200
)

// SchemaInspector abstracts schema discovery per SQL driver.
type SchemaInspector interface {
	ListSchemas(ctx context.Context, conn SQLConnection) ([]string, error)
	ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int) ([]domain.DatabaseTable, int64, error)
	ListColumns(ctx context.Context, conn SQLConnection, schema string, table string) ([]domain.DatabaseColumn, error)
}

func selectSchemaInspector(driverName string) SchemaInspector {
	switch driverName {
	case "pgx":
		return postgresSchemaInspector{}
	case "mysql":
		return mysqlSchemaInspector{}
	case "bigquery":
		return bigquerySchemaInspector{}
	case "snowflake":
		return snowflakeSchemaInspector{}
	default:
		return postgresSchemaInspector{}
	}
}

type postgresSchemaInspector struct{}

func NewDefaultSchemaInspector() SchemaInspector {
	return postgresSchemaInspector{}
}

func newDefaultSchemaInspector() SchemaInspector {
	return postgresSchemaInspector{}
}

func (postgresSchemaInspector) ListSchemas(ctx context.Context, conn SQLConnection) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog')
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing schemas: %w", err)
	}
	defer rows.Close()

	schemas := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scanning schema row: %w", scanErr)
		}
		schemas = append(schemas, name)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating schema rows: %w", rowsErr)
	}

	return schemas, nil
}

func (postgresSchemaInspector) ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int) ([]domain.DatabaseTable, int64, error) {
	normalizedSchema := strings.TrimSpace(schema)
	if normalizedSchema == "" {
		return nil, 0, domain.ErrInvalidDatabase
	}

	normalizedPage, normalizedPageSize := normalizeTablesPagination(page, pageSize)
	offset := (normalizedPage - 1) * normalizedPageSize

	countRows, err := conn.QueryContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
	`, normalizedSchema)
	if err != nil {
		return nil, 0, fmt.Errorf("counting tables: %w", err)
	}
	defer countRows.Close()

	var total int64
	if countRows.Next() {
		if scanErr := countRows.Scan(&total); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table count: %w", scanErr)
		}
	}
	if countRowsErr := countRows.Err(); countRowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table count rows: %w", countRowsErr)
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
		LIMIT $2 OFFSET $3
	`, normalizedSchema, normalizedPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	tables := make([]domain.DatabaseTable, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
		}
		tables = append(tables, domain.DatabaseTable{Name: name})
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table rows: %w", rowsErr)
	}

	return tables, total, nil
}

func (postgresSchemaInspector) ListColumns(ctx context.Context, conn SQLConnection, schema string, table string) ([]domain.DatabaseColumn, error) {
	normalizedSchema := strings.TrimSpace(schema)
	normalizedTable := strings.TrimSpace(table)
	if normalizedSchema == "" || normalizedTable == "" {
		return nil, domain.ErrInvalidDatabase
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT
			column_name,
			data_type,
			is_nullable,
			COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = $2
		ORDER BY ordinal_position
	`, normalizedSchema, normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()

	columns := make([]domain.DatabaseColumn, 0)
	for rows.Next() {
		var (
			name         string
			dataType     string
			nullable     string
			defaultValue string
		)

		if scanErr := rows.Scan(&name, &dataType, &nullable, &defaultValue); scanErr != nil {
			return nil, fmt.Errorf("scanning column row: %w", scanErr)
		}

		columns = append(columns, domain.DatabaseColumn{
			Name:         name,
			DataType:     dataType,
			IsNullable:   strings.EqualFold(nullable, "YES"),
			DefaultValue: defaultValue,
			IsDttm:       isDateTimeColumnType(dataType),
		})
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating column rows: %w", rowsErr)
	}

	return columns, nil
}

type mysqlSchemaInspector struct{}

func (mysqlSchemaInspector) ListSchemas(ctx context.Context, conn SQLConnection) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing schemas: %w", err)
	}
	defer rows.Close()

	schemas := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scanning schema row: %w", scanErr)
		}
		schemas = append(schemas, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating schema rows: %w", rowsErr)
	}
	return schemas, nil
}

func (mysqlSchemaInspector) ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int) ([]domain.DatabaseTable, int64, error) {
	normalizedSchema := strings.TrimSpace(schema)
	if normalizedSchema == "" {
		return nil, 0, domain.ErrInvalidDatabase
	}

	normalizedPage, normalizedPageSize := normalizeTablesPagination(page, pageSize)
	offset := (normalizedPage - 1) * normalizedPageSize

	countRows, err := conn.QueryContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
	`, normalizedSchema)
	if err != nil {
		return nil, 0, fmt.Errorf("counting tables: %w", err)
	}
	defer countRows.Close()

	var total int64
	if countRows.Next() {
		if scanErr := countRows.Scan(&total); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table count: %w", scanErr)
		}
	}
	if countRowsErr := countRows.Err(); countRowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table count rows: %w", countRowsErr)
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
		LIMIT ? OFFSET ?
	`, normalizedSchema, normalizedPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	tables := make([]domain.DatabaseTable, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
		}
		tables = append(tables, domain.DatabaseTable{Name: name})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table rows: %w", rowsErr)
	}
	return tables, total, nil
}

func (mysqlSchemaInspector) ListColumns(ctx context.Context, conn SQLConnection, schema string, table string) ([]domain.DatabaseColumn, error) {
	normalizedSchema := strings.TrimSpace(schema)
	normalizedTable := strings.TrimSpace(table)
	if normalizedSchema == "" || normalizedTable == "" {
		return nil, domain.ErrInvalidDatabase
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT
			column_name,
			data_type,
			is_nullable,
			COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = ?
		  AND table_name = ?
		ORDER BY ordinal_position
	`, normalizedSchema, normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()

	columns := make([]domain.DatabaseColumn, 0)
	for rows.Next() {
		var (
			name         string
			dataType     string
			nullable     string
			defaultValue string
		)
		if scanErr := rows.Scan(&name, &dataType, &nullable, &defaultValue); scanErr != nil {
			return nil, fmt.Errorf("scanning column row: %w", scanErr)
		}
		columns = append(columns, domain.DatabaseColumn{
			Name:         name,
			DataType:     dataType,
			IsNullable:   strings.EqualFold(nullable, "YES"),
			DefaultValue: defaultValue,
			IsDttm:       isMySQLDateTimeColumnType(dataType),
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating column rows: %w", rowsErr)
	}
	return columns, nil
}

type bigquerySchemaInspector struct{}

func (bigquerySchemaInspector) ListSchemas(ctx context.Context, conn SQLConnection) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT schema_name
		FROM INFORMATION_SCHEMA.SCHEMATA
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing schemas: %w", err)
	}
	defer rows.Close()

	schemas := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scanning schema row: %w", scanErr)
		}
		schemas = append(schemas, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating schema rows: %w", rowsErr)
	}
	return schemas, nil
}

func (bigquerySchemaInspector) ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int) ([]domain.DatabaseTable, int64, error) {
	normalizedSchema := strings.TrimSpace(schema)
	if normalizedSchema == "" {
		return nil, 0, domain.ErrInvalidDatabase
	}
	normalizedPage, normalizedPageSize := normalizeTablesPagination(page, pageSize)
	offset := (normalizedPage - 1) * normalizedPageSize

	countRows, err := conn.QueryContext(ctx,
		"SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE table_schema = @schema AND table_type = 'BASE TABLE'",
		normalizedSchema)
	if err != nil {
		return nil, 0, fmt.Errorf("counting tables: %w", err)
	}
	defer countRows.Close()
	var total int64
	if countRows.Next() {
		if scanErr := countRows.Scan(&total); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table count: %w", scanErr)
		}
	}
	if countRowsErr := countRows.Err(); countRowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table count rows: %w", countRowsErr)
	}

	rows, err := conn.QueryContext(ctx,
		"SELECT table_name FROM INFORMATION_SCHEMA.TABLES WHERE table_schema = @schema AND table_type = 'BASE TABLE' ORDER BY table_name LIMIT @limit OFFSET @offset",
		normalizedSchema, normalizedPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	tables := make([]domain.DatabaseTable, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
		}
		tables = append(tables, domain.DatabaseTable{Name: name})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table rows: %w", rowsErr)
	}

	return tables, total, nil
}

func (bigquerySchemaInspector) ListColumns(ctx context.Context, conn SQLConnection, schema string, table string) ([]domain.DatabaseColumn, error) {
	normalizedSchema := strings.TrimSpace(schema)
	normalizedTable := strings.TrimSpace(table)
	if normalizedSchema == "" || normalizedTable == "" {
		return nil, domain.ErrInvalidDatabase
	}

	rows, err := conn.QueryContext(ctx,
		"SELECT column_name, data_type, is_nullable, COALESCE(column_default, '') FROM INFORMATION_SCHEMA.COLUMNS WHERE table_schema = @schema AND table_name = @table ORDER BY ordinal_position",
		normalizedSchema, normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()

	columns := make([]domain.DatabaseColumn, 0)
	for rows.Next() {
		var name, dataType, nullable, defaultValue string
		if scanErr := rows.Scan(&name, &dataType, &nullable, &defaultValue); scanErr != nil {
			return nil, fmt.Errorf("scanning column row: %w", scanErr)
		}
		columns = append(columns, domain.DatabaseColumn{
			Name: name, DataType: dataType,
			IsNullable:   strings.EqualFold(nullable, "YES"),
			DefaultValue: defaultValue,
			IsDttm:       isDateTimeColumnType(dataType),
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating column rows: %w", rowsErr)
	}
	return columns, nil
}

type snowflakeSchemaInspector struct{}

func (snowflakeSchemaInspector) ListSchemas(ctx context.Context, conn SQLConnection) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('INFORMATION_SCHEMA')
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing schemas: %w", err)
	}
	defer rows.Close()
	schemas := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scanning schema row: %w", scanErr)
		}
		schemas = append(schemas, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating schema rows: %w", rowsErr)
	}
	return schemas, nil
}

func (snowflakeSchemaInspector) ListTables(ctx context.Context, conn SQLConnection, schema string, page int, pageSize int) ([]domain.DatabaseTable, int64, error) {
	normalizedSchema := strings.TrimSpace(schema)
	if normalizedSchema == "" {
		return nil, 0, domain.ErrInvalidDatabase
	}
	normalizedPage, normalizedPageSize := normalizeTablesPagination(page, pageSize)
	offset := (normalizedPage - 1) * normalizedPageSize

	countRows, err := conn.QueryContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
	`, normalizedSchema)
	if err != nil {
		return nil, 0, fmt.Errorf("counting tables: %w", err)
	}
	defer countRows.Close()
	var total int64
	if countRows.Next() {
		if scanErr := countRows.Scan(&total); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table count: %w", scanErr)
		}
	}
	if countRowsErr := countRows.Err(); countRowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table count rows: %w", countRowsErr)
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
		LIMIT ? OFFSET ?
	`, normalizedSchema, normalizedPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()
	tables := make([]domain.DatabaseTable, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, 0, fmt.Errorf("scanning table row: %w", scanErr)
		}
		tables = append(tables, domain.DatabaseTable{Name: name})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterating table rows: %w", rowsErr)
	}
	return tables, total, nil
}

func (snowflakeSchemaInspector) ListColumns(ctx context.Context, conn SQLConnection, schema string, table string) ([]domain.DatabaseColumn, error) {
	normalizedSchema := strings.TrimSpace(schema)
	normalizedTable := strings.TrimSpace(table)
	if normalizedSchema == "" || normalizedTable == "" {
		return nil, domain.ErrInvalidDatabase
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`, normalizedSchema, normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()
	columns := make([]domain.DatabaseColumn, 0)
	for rows.Next() {
		var name, dataType, nullable, defaultValue string
		if scanErr := rows.Scan(&name, &dataType, &nullable, &defaultValue); scanErr != nil {
			return nil, fmt.Errorf("scanning column row: %w", scanErr)
		}
		columns = append(columns, domain.DatabaseColumn{
			Name: name, DataType: dataType,
			IsNullable:   strings.EqualFold(nullable, "YES"),
			DefaultValue: defaultValue,
			IsDttm:       isSnowflakeDateTimeColumnType(dataType),
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating column rows: %w", rowsErr)
	}
	return columns, nil
}

func normalizeTablesPagination(page int, pageSize int) (int, int) {
	normalizedPage := page
	if normalizedPage < 1 {
		normalizedPage = databaseTablesDefaultPage
	}

	normalizedPageSize := pageSize
	if normalizedPageSize < 1 {
		normalizedPageSize = databaseTablesDefaultPageSize
	}
	if normalizedPageSize > databaseTablesMaxPageSize {
		normalizedPageSize = databaseTablesMaxPageSize
	}

	return normalizedPage, normalizedPageSize
}

func isDateTimeColumnType(dataType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(dataType))
	switch normalized {
	case "date",
		"datetime",
		"time",
		"time without time zone",
		"time with time zone",
		"timestamp",
		"timestamp without time zone",
		"timestamp with time zone",
		"timestamptz":
		return true
	default:
		return false
	}
}

func isMySQLDateTimeColumnType(dataType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(dataType))
	switch normalized {
	case "date", "datetime", "time", "timestamp", "year":
		return true
	default:
		return false
	}
}

func isSnowflakeDateTimeColumnType(dataType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(dataType))
	switch normalized {
	case "date", "time", "datetime", "timestamp", "timestamp_ltz", "timestamp_ntz", "timestamp_tz":
		return true
	default:
		return false
	}
}
