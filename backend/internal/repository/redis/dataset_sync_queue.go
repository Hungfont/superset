package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultDatasetSyncQueueName = redisQueuePrefix + datasetAsyncColumnsTaskType

var errInvalidSyncDatasetID = errors.New("dataset id must be greater than zero")

type syncQueueWriter interface {
	Push(ctx context.Context, queueName string, payload []byte) error
}

type redisSyncQueueWriter struct {
	client *redis.Client
}

type datasetSyncQueue struct {
	writer    syncQueueWriter
	queueName string
	newID     func() string
}

type datasetSyncColumnsPayload struct {
	DatasetID uint `json:"dataset_id"`
}

func NewDatasetSyncQueue(client *redis.Client) *datasetSyncQueue {
	return newDatasetSyncQueue(redisSyncQueueWriter{client: client}, defaultDatasetSyncQueueName)
}

func newDatasetSyncQueue(writer syncQueueWriter, queueName string) *datasetSyncQueue {
	resolvedQueueName := strings.TrimSpace(queueName)
	if resolvedQueueName == "" {
		resolvedQueueName = defaultDatasetSyncQueueName
	}

	return &datasetSyncQueue{
		writer:    writer,
		queueName: resolvedQueueName,
		newID:     uuid.NewString,
	}
}

func (q *datasetSyncQueue) EnqueueSyncColumns(ctx context.Context, datasetID uint) (string, error) {
	if datasetID == 0 {
		return "", errInvalidSyncDatasetID
	}
	if q.writer == nil {
		return "", errors.New("sync queue writer is nil")
	}

	jobID := q.newID()
	payload, err := json.Marshal(datasetSyncColumnsPayload{DatasetID: datasetID})
	if err != nil {
		return "", fmt.Errorf("marshalling sync dataset payload: %w", err)
	}
	if err := q.writer.Push(ctx, q.queueName, payload); err != nil {
		return "", fmt.Errorf("enqueue sync columns job: %w", err)
	}

	return jobID, nil
}

func (w redisSyncQueueWriter) Push(ctx context.Context, queueName string, payload []byte) error {
	if w.client == nil {
		return errors.New("redis client is nil")
	}

	resolvedQueueName := strings.TrimSpace(queueName)
	if resolvedQueueName == "" {
		return errors.New("queue name is required")
	}
	if err := w.client.RPush(ctx, resolvedQueueName, payload).Err(); err != nil {
		return fmt.Errorf("pushing sync dataset job to redis: %w", err)
	}

	return nil
}
