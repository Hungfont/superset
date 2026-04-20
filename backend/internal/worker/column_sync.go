package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	svcdb "superset/auth-service/internal/app/db"
	"superset/auth-service/internal/domain/dataset"

	"github.com/redis/go-redis/v9"
)

const (
	columnSyncQueueKey     = "queue:dataset:sync_columns"
	columnSyncPollInterval = 1 * time.Second
	columnSyncTimeout     = 5 * time.Minute
)

type ColumnSyncWorker struct {
	redisClient       *redis.Client
	datasetRepo     datasetRepoWithDB
	poolManager     *svcdb.ConnectionPoolManager
	schemaInspector svcdb.SchemaInspector
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         func()
}

type ColumnSyncPayload struct {
	DatasetID uint `json:"dataset_id"`
}

func NewColumnSyncWorker(
	redisClient *redis.Client,
	repo datasetRepoWithDB,
	poolManager *svcdb.ConnectionPoolManager,
	inspector svcdb.SchemaInspector,
) *ColumnSyncWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &ColumnSyncWorker{
		redisClient:       redisClient,
		datasetRepo:     repo,
		poolManager:     poolManager,
		schemaInspector: inspector,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (w *ColumnSyncWorker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		log.Println("[column_sync_worker] started")
		for {
			select {
			case <-w.ctx.Done():
				log.Println("[column_sync_worker] stopped")
				return
			default:
				w.processNext()
			}
		}
	}()
}

func (w *ColumnSyncWorker) Stop() error {
	log.Println("[column_sync_worker] shutting down...")
	w.cancel()
	w.wg.Wait()
	return nil
}

func (w *ColumnSyncWorker) processNext() {
	result, err := w.redisClient.BLPop(w.ctx, columnSyncPollInterval, columnSyncQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return
		}
		if w.ctx.Err() != nil {
			return
		}
		log.Printf("[column_sync_worker] error popping from queue: %v", err)
		return
	}

	if len(result) < 2 {
		log.Println("[column_sync_worker] unexpected response format")
		return
	}

	payloadJSON := result[1]
	var payload ColumnSyncPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		log.Printf("[column_sync_worker] error unmarshaling payload: %v", err)
		return
	}

	w.processSync(payload.DatasetID)
}

func (w *ColumnSyncWorker) processSync(datasetID uint) {
	log.Printf("[column_sync_worker] syncing columns for dataset %d", datasetID)

	ctx, cancel := context.WithTimeout(w.ctx, columnSyncTimeout)
	defer cancel()

	ds, err := w.datasetRepo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		log.Printf("[column_sync_worker] error getting dataset %d: %v", datasetID, err)
		return
	}
	if ds == nil || ds.ID == 0 {
		log.Printf("[column_sync_worker] dataset %d not found", datasetID)
		return
	}

	var columns []dataset.Column

	if ds.SQL != "" && strings.TrimSpace(ds.SQL) != "" {
		columns = w.getVirtualColumns(ctx, ds)
	} else {
		columns = w.getPhysicalColumns(ctx, ds)
	}

	if columns == nil {
		log.Printf("[column_sync_worker] no columns found for dataset %d", datasetID)
		return
	}

	if err := w.datasetRepo.RefreshDatasetColumns(ctx, datasetID, columns); err != nil {
		log.Printf("[column_sync_worker] error refreshing columns for dataset %d: %v", datasetID, err)
		return
	}

	log.Printf("[column_sync_worker] synced %d columns for dataset %d", len(columns), datasetID)
}

func (w *ColumnSyncWorker) getPhysicalColumns(ctx context.Context, ds *dataset.Dataset) []dataset.Column {
	dbRec, err := w.datasetRepo.GetDatabaseByID(ctx, ds.DatabaseID)
	if err != nil {
		log.Printf("[column_sync_worker] error getting database %d: %v", ds.DatabaseID, err)
		return nil
	}
	if dbRec == nil {
		log.Printf("[column_sync_worker] database %d not found", ds.DatabaseID)
		return nil
	}

	conn, err := w.poolManager.Get(ctx, ds.DatabaseID, dbRec.SQLAlchemyURI)
	if err != nil {
		log.Printf("[column_sync_worker] error getting connection for database %d: %v", ds.DatabaseID, err)
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	remoteColumns, err := w.schemaInspector.ListColumns(timeoutCtx, conn, ds.Schema, ds.Name)
	if err != nil {
		log.Printf("[column_sync_worker] error listing columns for %s.%s: %v", ds.Schema, ds.Name, err)
		return nil
	}

	columns := make([]dataset.Column, 0, len(remoteColumns))
	for _, col := range remoteColumns {
		isDttm := false
		colType := strings.ToLower(col.DataType)
		if strings.Contains(colType, "timestamp") || strings.Contains(colType, "date") || strings.Contains(colType, "time") {
			isDttm = true
		}

		columns = append(columns, dataset.Column{
			TableID:     ds.ID,
			ColumnName: col.Name,
			Type:       col.DataType,
			IsDateTime: isDttm,
		})
	}

	return columns
}

func (w *ColumnSyncWorker) getVirtualColumns(ctx context.Context, ds *dataset.Dataset) []dataset.Column {
	dbRec, err := w.datasetRepo.GetDatabaseByID(ctx, ds.DatabaseID)
	if err != nil {
		log.Printf("[column_sync_worker] error getting database %d: %v", ds.DatabaseID, err)
		return nil
	}
	if dbRec == nil {
		log.Printf("[column_sync_worker] database %d not found", ds.DatabaseID)
		return nil
	}

	conn, err := w.poolManager.Get(ctx, ds.DatabaseID, dbRec.SQLAlchemyURI)
	if err != nil {
		log.Printf("[column_sync_worker] error getting connection for database %d: %v", ds.DatabaseID, err)
		return nil
	}

	sql := strings.TrimSpace(ds.SQL)
	if sql == "" {
		return nil
	}

	query := fmt.Sprintf("SELECT * FROM (%s) AS _sync LIMIT 0", sql)

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	rows, err := conn.QueryContext(timeoutCtx, query)
	if err != nil {
		log.Printf("[column_sync_worker] error executing virtual query: %v", err)
		return nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Printf("[column_sync_worker] error getting columns: %v", err)
		return nil
	}

	types, err := rows.ColumnTypes()
	if err != nil {
		log.Printf("[column_sync_worker] error getting column types: %v", err)
		return nil
	}

	result := make([]dataset.Column, 0, len(columns))
	for i, colName := range columns {
		colType := ""
		if i < len(types) && types[i] != nil {
			colType = types[i].DatabaseTypeName()
		}

		isDttm := false
		colTypeLower := strings.ToLower(colType)
		if strings.Contains(colTypeLower, "timestamp") || strings.Contains(colTypeLower, "date") || strings.Contains(colTypeLower, "time") {
			isDttm = true
		}

		result = append(result, dataset.Column{
			TableID:     ds.ID,
			ColumnName: colName,
			Type:       colType,
			IsDateTime: isDttm,
		})
	}

	return result
}