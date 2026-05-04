package query

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"superset/auth-service/internal/domain/auth"
	"superset/auth-service/internal/domain/dataset"
	"superset/auth-service/internal/domain/query"

	"github.com/redis/go-redis/v9"
)

// Type aliases for domain types
type ExecuteRequest = query.ExecuteRequest
type ExecuteResponse = query.ExecuteResponse

const (
	// Queue keys for async query processing
	queryQueueKey      = "queue:query:async"
	queryQueueCritical = "queue:query:critical"
	queryQueueLow      = "queue:query:low"

	// Status event channels
	queryStatusChannel = "query:status:"

	// Cancel flag key
	queryCancelKey = "query:cancel:"

	// Query result key prefix
	queryResultKey = "query:result:"

	// QE-004 #5: Retry configuration
	MaxRetry      = 3
	RetryInterval = 5 * time.Second

	// QE-004 #6: Worker pool sizes
	WorkerPoolCritical = 10
	WorkerPoolDefault  = 20
	WorkerPoolLow      = 5
)

// AsyncQueryExecutor handles async query execution
type AsyncQueryExecutor struct {
	rdb         *redis.Client
	queryRepo   query.Repository
	rlsRepo     auth.RLSFilterRepository
	datasetRepo dataset.Repository
	queryCache  *QueryExecutor
	workerPool  *WorkerPool
}

// WorkerPool manages concurrent workers per queue
type WorkerPool struct {
	critical chan struct{}
	defaultQ chan struct{}
	low      chan struct{}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		critical: make(chan struct{}, WorkerPoolCritical),
		defaultQ: make(chan struct{}, WorkerPoolDefault),
		low:      make(chan struct{}, WorkerPoolLow),
	}
}

// acquire acquires a worker slot from the pool
func (wp *WorkerPool) acquire(queue string) bool {
	var slot chan struct{}
	switch queue {
	case queryQueueCritical:
		slot = wp.critical
	case queryQueueLow:
		slot = wp.low
	default:
		slot = wp.defaultQ
	}
	select {
	case slot <- struct{}{}:
		return true
	default:
		return false
	}
}

// release releases a worker slot back to the pool
func (wp *WorkerPool) release(queue string) {
	var slot chan struct{}
	switch queue {
	case queryQueueCritical:
		slot = wp.critical
	case queryQueueLow:
		slot = wp.low
	default:
		slot = wp.defaultQ
	}
	<-slot
}

// NewAsyncQueryExecutor creates a new async query executor
func NewAsyncQueryExecutor(
	rdb *redis.Client,
	queryRepo query.Repository,
	rlsRepo auth.RLSFilterRepository,
	datasetRepo dataset.Repository,
	queryCache *QueryExecutor,
) *AsyncQueryExecutor {
	return &AsyncQueryExecutor{
		rdb:         rdb,
		queryRepo:   queryRepo,
		rlsRepo:     rlsRepo,
		datasetRepo: datasetRepo,
		queryCache:  queryCache,
		workerPool:  NewWorkerPool(),
	}
}

// Submit submits a query for async execution
func (e *AsyncQueryExecutor) Submit(ctx context.Context, req query.AsyncSubmitRequest, userCtx auth.UserContext) (*query.AsyncSubmitResponse, error) {
	if e.rdb == nil {
		return nil, fmt.Errorf("redis client not configured")
	}
	
	log.Printf("[async_executor] Submit: database_id=%d, sql=%s", req.DatabaseID, req.SQL)
	
	queryID := "q-" + generateQueryID()
	if req.ClientID != "" {
		queryID = "q-" + req.ClientID[:8]
	}

	// Determine queue based on user role (fetch roles from repo)
	roles, err := e.rlsRepo.GetRoleNamesByUser(ctx, userCtx.ID)
	if err != nil {
		roles = []string{}
	}
	queueKey := resolveQueue(roles)

	// Create query record
	q := &query.Query{
		ID:         queryID,
		ClientID:   req.ClientID,
		DatabaseID: req.DatabaseID,
		UserID:     userCtx.ID,
		SQL:        req.SQL,
		Status:     "pending",
		Schema:     req.Schema,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Save query to database
	if err := e.queryRepo.Create(ctx, q); err != nil {
		return nil, fmt.Errorf("creating query record: %w", err)
	}

	// Create task payload
	task := query.QueryTask{
		QueryID:      queryID,
		DatabaseID:   req.DatabaseID,
		SQL:          req.SQL,
		Limit:        req.Limit,
		Schema:       req.Schema,
		ClientID:     req.ClientID,
		ForceRefresh: req.ForceRefresh,
		UserID:       userCtx.ID,
		Username:     userCtx.Username,
	}

	// Enqueue task using Redis LPush
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("marshaling task: %w", err)
	}

	log.Printf("[async_executor] enqueueing query %s to queue %s", queryID, queueKey)
	_, err = e.rdb.LPush(ctx, queueKey, taskJSON).Result()
	if err != nil {
		// Try to delete the query record if enqueue failed
		_ = e.queryRepo.Update(ctx, q)
		return nil, fmt.Errorf("enqueueing task: %w", err)
	}

	log.Printf("[async_executor] successfully enqueued query %s", queryID)

	return &query.AsyncSubmitResponse{
		QueryID: queryID,
		Status:  "pending",
		Queue:   queueKeyToName(queueKey),
	}, nil
}

// GetStatus gets the status of an async query
func (e *AsyncQueryExecutor) GetStatus(ctx context.Context, queryID string, userCtx auth.UserContext) (*query.QueryStatusResponse, error) {
	q, err := e.queryRepo.GetByID(ctx, queryID)
	if err != nil {
		return nil, fmt.Errorf("getting query: %w", err)
	}

	if q == nil {
		return nil, fmt.Errorf("query not found")
	}

	// Check ownership
	if q.UserID != userCtx.ID {
		// Check if user is Admin
		roles, err := e.rlsRepo.GetRoleNamesByUser(ctx, userCtx.ID)
		if err != nil || !isAdminRole(roles) {
			return nil, fmt.Errorf("forbidden")
		}
	}

	response := &query.QueryStatusResponse{
		QueryID: queryID,
		Status:  q.Status,
		Rows:    q.Rows,
	}

	if q.StartTime != nil {
		response.StartTime = *q.StartTime
	}
	if q.EndTime != nil {
		response.EndTime = *q.EndTime
	}
	if q.ResultsKey != "" {
		response.ResultsKey = q.ResultsKey
	}
	if q.ErrorMessage != "" {
		response.Error = q.ErrorMessage
	}

	// Calculate elapsed time
	if q.StartTime != nil {
		endTime := time.Now()
		if q.EndTime != nil {
			endTime = *q.EndTime
		}
		response.ElapsedMs = endTime.Sub(*q.StartTime).Milliseconds()

		// Add timeout_at for async queries (30s from start_time)
		if q.Status == "pending" || q.Status == "running" {
			timeoutDuration := 30 * time.Second
			timeoutAt := q.StartTime.Add(timeoutDuration)
			response.TimeoutAt = timeoutAt
		}
	}

	return response, nil
}

// Cancel cancels a running query
func (e *AsyncQueryExecutor) Cancel(ctx context.Context, queryID string, userCtx auth.UserContext) error {
	q, err := e.queryRepo.GetByID(ctx, queryID)
	if err != nil {
		return fmt.Errorf("getting query: %w", err)
	}

	if q == nil {
		return fmt.Errorf("query not found")
	}

	// Check ownership
	if q.UserID != userCtx.ID {
		roles, err := e.rlsRepo.GetRoleNamesByUser(ctx, userCtx.ID)
		if err != nil || !isAdminRole(roles) {
			return fmt.Errorf("forbidden")
		}
	}

	// Only can cancel pending or running queries
	if q.Status != "pending" && q.Status != "running" {
		return fmt.Errorf("query cannot be cancelled")
	}

	// Set cancel flag in Redis
	if e.rdb != nil {
		e.rdb.Set(ctx, queryCancelKey+queryID, "1", 30*time.Minute)
	}

	// Update query status
	q.Status = "stopped"
	q.ErrorMessage = "Cancelled by user"
	now := time.Now()
	q.EndTime = &now
	if err := e.queryRepo.Update(ctx, q); err != nil {
		return fmt.Errorf("updating query: %w", err)
	}

	return nil
}

// ProcessNext processes the next query in the queue
func (e *AsyncQueryExecutor) ProcessNext(ctx context.Context) error {
	// QE-004 #6: Check worker pool availability
	for _, queueKey := range []string{queryQueueCritical, queryQueueKey, queryQueueLow} {
		if e.workerPool.acquire(queueKey) {
			defer e.workerPool.release(queueKey)
			return e.processFromQueue(ctx, queueKey)
		}
	}

	// Fallback if no worker pool
	return e.processFromQueue(ctx, queryQueueKey)
}

// ExecuteTask executes a task directly (used by worker)
func (e *AsyncQueryExecutor) ExecuteTask(ctx context.Context, task *query.QueryTask) error {
	// Acquire worker slot from pool
	queueKey := resolveQueueForTask(task)
	if !e.workerPool.acquire(queueKey) {
		return fmt.Errorf("no worker available")
	}
	defer e.workerPool.release(queueKey)
	return e.executeQuery(ctx, task)
}

// resolveQueueForTask resolves the queue key for a task
func resolveQueueForTask(task *query.QueryTask) string {
	// For now, use default queue - in production would check user roles
	return queryQueueKey
}

func (e *AsyncQueryExecutor) processFromQueue(ctx context.Context, queueKey string) error {
	task, err := e.rdb.RPop(ctx, queueKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("popping from queue: %w", err)
	}

	var queryTask query.QueryTask
	if err := json.Unmarshal([]byte(task), &queryTask); err != nil {
		log.Printf("[query_worker] error unmarshaling task: %v", err)
		return nil
	}

	return e.executeQuery(ctx, &queryTask)
}

// executeQuery executes a query task with retry logic
func (e *AsyncQueryExecutor) executeQuery(ctx context.Context, task *query.QueryTask) error {
	queryID := task.QueryID

	// Update status to running
	q, err := e.queryRepo.GetByID(ctx, queryID)
	if err != nil {
		log.Printf("[query_worker] error getting query %s: %v", queryID, err)
		return err
	}

	startTime := time.Now()
	q.Status = "running"
	q.StartTime = &startTime
	q.UpdatedAt = time.Now()
	if err := e.queryRepo.Update(ctx, q); err != nil {
		log.Printf("[query_worker] error updating query %s: %v", queryID, err)
		return err
	}

	// Publish status: running
	e.publishStatus(ctx, queryID, "running", nil)

	// Execute the query using the sync executor
	execReq := ExecuteRequest{
		DatabaseID:   task.DatabaseID,
		SQL:          task.SQL,
		Limit:        task.Limit,
		Schema:       task.Schema,
		ForceRefresh: task.ForceRefresh,
	}

	// Create user context from task
	userCtx := auth.UserContext{
		ID:       task.UserID,
		Username: task.Username,
		Active:   true,
	}

	// QE-004 #5: Retry logic
	var lastErr error
	for attempt := 0; attempt < MaxRetry; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 5s -> 25s -> 125s
			backoff := RetryInterval * time.Duration(5*attempt)
			time.Sleep(backoff)
		}

		resp, err := e.queryCache.Execute(ctx, execReq, userCtx)
		if err == nil {
			// Success
			return e.handleQuerySuccess(ctx, q, queryID, resp)
		}
		lastErr = err
		log.Printf("[query_worker] attempt %d failed for query %s: %v", attempt+1, queryID, err)
	}

	// All retries failed - QE-004 #5
	q.Status = "failed"
	q.ErrorMessage = fmt.Sprintf("failed after %d attempts: %v", MaxRetry, lastErr)
	now := time.Now()
	q.EndTime = &now
	_ = e.queryRepo.Update(ctx, q)
	e.publishStatus(ctx, queryID, "failed", nil)
	return lastErr
}

func (e *AsyncQueryExecutor) handleQuerySuccess(ctx context.Context, q *query.Query, queryID string, resp *ExecuteResponse) error {
	// Check cancel flag
	if e.rdb != nil {
		cancelled, _ := e.rdb.Exists(ctx, queryCancelKey+queryID).Result()
		if cancelled > 0 {
			q.Status = "stopped"
			q.ErrorMessage = "Cancelled by user"
			now := time.Now()
			q.EndTime = &now
			_ = e.queryRepo.Update(ctx, q)
			e.publishStatus(ctx, queryID, "stopped", nil)
			return nil
		}
	}

	// Store result in Redis if needed
	var resultsKey string
	if e.rdb != nil && resp.Data != nil {
		respJSON, err := json.Marshal(resp)
		if err == nil && len(respJSON) <= 1024*1024 {
			resultsKey = queryResultKey + queryID
			e.rdb.Set(ctx, resultsKey, respJSON, 24*time.Hour)
		}
	}

	endTime := time.Now()
	rowCount := 0
	if resp.Data != nil {
		if data, ok := resp.Data.([]interface{}); ok {
			rowCount = len(data)
		}
	}

	q.Status = "success"
	q.EndTime = &endTime
	q.Rows = rowCount
	q.ResultsKey = resultsKey
	q.ExecutedSQL = resp.Query.ExecutedSQL
	q.UpdatedAt = time.Now()
	if err := e.queryRepo.Update(ctx, q); err != nil {
		log.Printf("[query_worker] error updating query %s: %v", queryID, err)
	}

	e.publishStatus(ctx, queryID, "success", resp)
	return nil
}

// publishStatus publishes a status event via Redis pub/sub
func (e *AsyncQueryExecutor) publishStatus(ctx context.Context, queryID, status string, result *ExecuteResponse) {
	if e.rdb == nil {
		log.Printf("[async_executor] publishStatus: redis is nil, skipping publish")
		return
	}

	var event map[string]interface{}
	if result != nil {
		event = map[string]interface{}{
			"type":     "done",
			"query_id": queryID,
			"status":   status,
			"data":     result.Data,
			"columns":  result.Columns,
		}
	} else {
		event = map[string]interface{}{
			"type":     "status",
			"query_id": queryID,
			"status":   status,
		}
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("[query_worker] error marshaling event: %v", err)
		return
	}

	e.rdb.Publish(ctx, queryStatusChannel+queryID, eventJSON)
}

// resolveQueue determines the queue based on user role
func resolveQueue(roles []string) string {
	for _, role := range roles {
		if role == "Admin" {
			return queryQueueCritical
		}
	}
	for _, role := range roles {
		if role == "Alpha" {
			return queryQueueKey
		}
	}
	return queryQueueLow
}

// isAdminRole checks if user has Admin role
func isAdminRole(roles []string) bool {
	for _, role := range roles {
		if role == "Admin" {
			return true
		}
	}
	return false
}

// queueKeyToName converts a queue key to a human-readable name
func queueKeyToName(queueKey string) string {
	switch queueKey {
	case queryQueueCritical:
		return "critical"
	case queryQueueLow:
		return "low"
	default:
		return "default"
	}
}

// generateQueryID generates a short query ID
func generateQueryID() string {
	// Use simple random string
	return fmt.Sprintf("%08x", time.Now().UnixNano())
}

// GetResult gets the result of a completed query
func (e *AsyncQueryExecutor) GetResult(ctx context.Context, queryID string) (*ExecuteResponse, error) {
	q, err := e.queryRepo.GetByID(ctx, queryID)
	if err != nil {
		return nil, err
	}

	if q == nil {
		return nil, fmt.Errorf("query not found")
	}

	if q.Status != "success" {
		return nil, fmt.Errorf("query not completed")
	}

	// Try to get from Redis first
	if e.rdb != nil && q.ResultsKey != "" {
		resultJSON, err := e.rdb.Get(ctx, q.ResultsKey).Bytes()
		if err == nil {
			var result ExecuteResponse
			if err := json.Unmarshal(resultJSON, &result); err == nil {
				return &result, nil
			}
		}
	}

	// Return empty response with metadata
	return &ExecuteResponse{
		Data:      []interface{}{},
		Columns:   []query.ColumnInfo{},
		FromCache: false,
	}, nil
}
