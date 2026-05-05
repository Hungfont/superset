package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	svcquery "superset/auth-service/internal/app/query"
	"superset/auth-service/internal/domain/query"
)

// QueryWorkerConfig configuration for query worker
type QueryWorkerConfig struct {
	PollInterval time.Duration
	WorkerCount  int
	QueueKeys    []string
}

// DefaultQueryWorkerConfig returns default configuration
func DefaultQueryWorkerConfig() QueryWorkerConfig {
	return QueryWorkerConfig{
		PollInterval: 1 * time.Second,
		WorkerCount:  5,
		QueueKeys:    []string{"queue:query:critical", "queue:query:async", "queue:query:low"},
	}
}

// QueryWorker processes async queries from Redis queue
type QueryWorker struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	redisClient   *redis.Client
	queryExecutor *svcquery.QueryExecutor
	asyncExecutor *svcquery.AsyncQueryExecutor

	config QueryWorkerConfig
}

// NewQueryWorker creates a new query worker
func NewQueryWorker(
	redisClient *redis.Client,
	queryExecutor *svcquery.QueryExecutor,
	asyncExecutor *svcquery.AsyncQueryExecutor,
	config QueryWorkerConfig,
) *QueryWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &QueryWorker{
		ctx:           ctx,
		cancel:        cancel,
		redisClient:   redisClient,
		queryExecutor: queryExecutor,
		asyncExecutor: asyncExecutor,
		config:        config,
	}
}

// Start starts the query worker with multiple goroutines
func (w *QueryWorker) Start() {
	log.Printf("[query_worker] starting with %d workers", w.config.WorkerCount)

	for i := 0; i < w.config.WorkerCount; i++ {
		w.wg.Add(1)
		go func(workerID int) {
			defer w.wg.Done()
			log.Printf("[query_worker] worker %d started", workerID)
			w.run(workerID)
		}(i)
	}
}

// Stop stops the query worker
func (w *QueryWorker) Stop() error {
	log.Println("[query_worker] shutting down...")
	w.cancel()
	w.wg.Wait()
	log.Println("[query_worker] stopped")
	return nil
}

// run runs the worker loop
func (w *QueryWorker) run(workerID int) {
	for {
		select {
		case <-w.ctx.Done():
			log.Printf("[query_worker] worker %d stopped", workerID)
			return
		default:
			w.processNext()
		}
	}
}

// processNext processes the next query from the queue
func (w *QueryWorker) processNext() {
	// Try to pop from critical queue first, then default, then low
	for _, queueKey := range w.config.QueueKeys {
		result, err := w.redisClient.BRPop(w.ctx, w.config.PollInterval, queueKey).Result()
		if err == redis.Nil {
			continue // This queue is empty, try next
		}
		if err != nil {
			if w.ctx.Err() != nil {
				return
			}
			log.Printf("[query_worker] error popping from queue %s: %v", queueKey, err)
			continue
		}

		if len(result) < 2 {
			log.Println("[query_worker] unexpected response format")
			continue
		}

		payloadJSON := result[1]
		var task query.QueryTask
		if err := json.Unmarshal([]byte(payloadJSON), &task); err != nil {
			log.Printf("[query_worker] error unmarshaling task: %v", err)
			continue
		}

		w.executeTask(&task)
	}
}

// executeTask executes a single query task
func (w *QueryWorker) executeTask(task *query.QueryTask) {
	log.Printf("[query_worker] executing query %s: %s", task.QueryID, task.SQL)
	err := w.asyncExecutor.ExecuteTask(w.ctx, task)
	if err != nil {
		log.Printf("[query_worker] error processing query: %v", err)
	}
}
