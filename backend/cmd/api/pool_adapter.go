package main

import (
	"context"
	"database/sql"

	svcdataset "superset/auth-service/internal/app/dataset"
	dbapp "superset/auth-service/internal/app/db"
)

// poolToDatasetAdapter adapts db.ConnectionPoolManager to dataset.SQLConnectionPool.
type poolToDatasetAdapter struct {
	pool *dbapp.ConnectionPoolManager
}

func (a *poolToDatasetAdapter) Get(ctx context.Context, databaseID uint, uri string) (svcdataset.SQLConn, error) {
	conn, err := a.pool.Get(ctx, databaseID, uri)
	if err != nil {
		return nil, err
	}
	return &connAdapter{conn: conn}, nil
}

// connAdapter adapts db.SQLConnection to dataset.SQLConn.
type connAdapter struct {
	conn dbapp.SQLConnection
}

func (a *connAdapter) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.conn.QueryContext(ctx, query, args...)
}

func (a *connAdapter) Close() error {
	return a.conn.Close()
}
