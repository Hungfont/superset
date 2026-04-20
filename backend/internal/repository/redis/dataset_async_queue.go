package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const datasetAsyncColumnsTaskType = "dataset:sync_columns"
const redisQueuePrefix = "queue:"
const defaultDatasetAsyncQueueBuffer = 128

var (
	errInvalidAsyncDatasetID = errors.New("dataset id must be greater than zero")
	errAsyncQueueClosed      = errors.New("dataset async queue is closed")
)

type asyncQueueWriter interface {
	Push(ctx context.Context, taskType string, payload []byte) error
}

type redisAsyncQueueWriter struct {
	client *redis.Client
}

type datasetAsyncQueue struct {
	writer   asyncQueueWriter
	taskType string
	newID    func() string

	jobCh chan []byte
	workerWG sync.WaitGroup

	mu        sync.RWMutex
	isClosed  bool
	closeOnce sync.Once
}

type datasetAsyncColumnsPayload struct {
	DatasetID uint   `json:"dataset_id"`
	JobID    string `json:"job_id"`
}

func NewDatasetAsyncQueue(client *redis.Client) *datasetAsyncQueue {
	return newDatasetAsyncQueue(redisAsyncQueueWriter{client: client}, datasetAsyncColumnsTaskType, defaultDatasetAsyncQueueBuffer)
}

func newDatasetAsyncQueue(writer asyncQueueWriter, taskType string, bufferSize int) *datasetAsyncQueue {
	resolvedTaskType := strings.TrimSpace(taskType)
	if resolvedTaskType == "" {
		resolvedTaskType = datasetAsyncColumnsTaskType
	}
	if bufferSize <= 0 {
		bufferSize = defaultDatasetAsyncQueueBuffer
	}

	queue := &datasetAsyncQueue{
		writer: writer,
		taskType: resolvedTaskType,
		newID:  uuid.NewString,
		jobCh:  make(chan []byte, bufferSize),
	}
	queue.startWorker()
	return queue
}

func (q *datasetAsyncQueue) EnqueueSyncColumns(ctx context.Context, datasetID uint) (string, error) {
	if datasetID == 0 {
		return "", errInvalidAsyncDatasetID
	}
	if q.writer == nil {
		return "", errors.New("async queue writer is nil")
	}

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	jobID := q.newID()
	payload, err := json.Marshal(datasetAsyncColumnsPayload{
		DatasetID: datasetID,
		JobID:    jobID,
	})
	if err != nil {
		return "", fmt.Errorf("marshalling async dataset sync payload: %w", err)
	}

	q.mu.RLock()
	if q.isClosed {
		q.mu.RUnlock()
		return "", errAsyncQueueClosed
	}
	jobCh := q.jobCh
	q.mu.RUnlock()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case jobCh <- payload:
		return jobID, nil
	default:
		return "", errors.New("async queue buffer full")
	}
}

func (q *datasetAsyncQueue) Shutdown(ctx context.Context) error {
	q.closeOnce.Do(func() {
		q.mu.Lock()
		q.isClosed = true
		close(q.jobCh)
		q.mu.Unlock()
	})

	waitCh := make(chan struct{})
	go func() {
		defer close(waitCh)
		q.workerWG.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitCh:
		return nil
	}
}

func (q *datasetAsyncQueue) startWorker() {
	q.workerWG.Add(1)
	go func() {
		defer q.workerWG.Done()
		for payload := range q.jobCh {
			_ = q.writer.Push(context.Background(), q.taskType, payload)
		}
	}()
}

func (w redisAsyncQueueWriter) Push(ctx context.Context, taskType string, payload []byte) error {
	if w.client == nil {
		return errors.New("redis client is nil")
	}
	resolvedTaskType := strings.TrimSpace(taskType)
	if resolvedTaskType == "" {
		return errors.New("task type is required")
	}
	queueKey := redisQueuePrefix + resolvedTaskType
	if err := w.client.RPush(ctx, queueKey, payload).Err(); err != nil {
		return fmt.Errorf("pushing async dataset sync job to redis: %w", err)
	}
	return nil
}
