package query

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	dbpool "superset/auth-service/internal/app/db"
	"superset/auth-service/internal/domain/auth"
	"superset/auth-service/internal/domain/dataset"
	domdb "superset/auth-service/internal/domain/db"
	"superset/auth-service/internal/domain/query"

	"github.com/redis/go-redis/v9"
)

// Type aliases for domain types
type ExecuteRequest = query.ExecuteRequest
type ExecuteResponse = query.ExecuteResponse

type RoleNameProvider interface {
	GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error)
}

type QueryExecutorRunner interface {
	Execute(ctx context.Context, req ExecuteRequest, userCtx auth.UserContext) (*ExecuteResponse, error)
}

const (
	// Queue keys for async query processing
	// FIX: Use correct queue keys matching worker config (queue:query:default, not queue:query:async)
	queryQueueDefault   = "queue:query:default"  // Added for default/standard queue
	queryQueueCritical = "queue:query:critical" // For Admin priority
	queryQueueLow      = "queue:query:low"     // For Gamma/background
	queryQueueKey     = "queue:query:default" // Legacy - use queryQueueDefault

	// Status event channels
	queryStatusChannel = "query:status:"

	// Cancel flag key
	queryCancelKey = "query:cancel:"

	// Query result key prefix
	queryResultKey = "query:result:"

	// QE-004 #5: Retry configuration (exponential: 5s -> 25s with MaxRetry=3)
	MaxRetry        = 3
	RetryInterval   = 5 * time.Second
	RetryMultiplier = 5

	// QE-004 #6: Worker pool sizes
	WorkerPoolCritical = 10
	WorkerPoolDefault  = 20
	WorkerPoolLow      = 5

	// QE-005: Max inline result size (1MB)
	MaxInlineResultBytes = 1024 * 1024

	// QE-006: PID key for pg_cancel_backend
	queryPIDKey = "query:pid:"

	// QE-006: Cancel flag TTL (5 min — down from 30 min)
	CancelFlagTTL = 5 * time.Minute
)

// CancelResult describes the outcome of a cancel request.
type CancelResult struct {
	Action        string `json:"action"`
	CurrentStatus string `json:"current_status"`
}

// AsyncQueryExecutor handles async query execution
type AsyncQueryExecutor struct {
	rdb          *redis.Client
	queryRepo    query.Repository
	rlsRepo      RoleNameProvider
	datasetRepo  dataset.Repository
	queryCache   QueryExecutorRunner
	workerPool   *WorkerPool
	waitForRetry func(ctx context.Context, attempt int) error
	connPool     dbpool.DatabaseConnectionPool
	databaseRepo domdb.DatabaseRepository
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
	rlsRepo RoleNameProvider,
	datasetRepo dataset.Repository,
	queryCache QueryExecutorRunner,
	connPool dbpool.DatabaseConnectionPool,
	databaseRepo domdb.DatabaseRepository,
) *AsyncQueryExecutor {
	return &AsyncQueryExecutor{
		rdb:          rdb,
		queryRepo:    queryRepo,
		rlsRepo:      rlsRepo,
		datasetRepo:  datasetRepo,
		queryCache:   queryCache,
		workerPool:   NewWorkerPool(),
		waitForRetry: defaultWaitForRetry,
		connPool:     connPool,
		databaseRepo: databaseRepo,
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

	// Phase 4: Client ID dedup — check for existing pending/running query
	if req.ClientID != "" {
		existing, lookupErr := e.queryRepo.GetByClientID(ctx, req.ClientID)
		if lookupErr == nil && existing != nil {
			switch existing.Status {
			case "pending", "running":
				log.Printf("[async_executor] dedup: query %s already %s for client_id %s", existing.ID, existing.Status, req.ClientID)
				return &query.AsyncSubmitResponse{
					QueryID: existing.ID,
					Status:  existing.Status,
					Queue:   queueKeyToName(queueKey),
				}, nil
			case "success":
				// Return existing cached result info
				return &query.AsyncSubmitResponse{
					QueryID: existing.ID,
					Status:  existing.Status,
					Queue:   queueKeyToName(queueKey),
				}, nil
			}
		}
	}

	// Create query record with all metadata fields
	now := time.Now()
	q := &query.Query{
		ID:              queryID,
		ClientID:        req.ClientID,
		DatabaseID:      req.DatabaseID,
		UserID:          userCtx.ID,
		TabName:         req.TabName,
		SqlEditorID:     req.SqlEditorID,
		Schema:          req.Schema,
		Catalog:         req.Catalog,
		SQL:             req.SQL,
		SelectAsCTAUsed: req.SelectAsCTA,
		Progress:        "queued",
		Status:          "pending",
		StartTime:       &now,
		ChangedOn:       &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Save query to database
	if err := e.queryRepo.Create(ctx, q); err != nil {
		return nil, fmt.Errorf("creating query record: %w", err)
	}

	// Create task payload with all metadata
	task := query.QueryTask{
		QueryID:      queryID,
		DatabaseID:   req.DatabaseID,
		SQL:          req.SQL,
		Limit:        req.Limit,
		Schema:       req.Schema,
		Catalog:      req.Catalog,
		TabName:      req.TabName,
		SqlEditorID:  req.SqlEditorID,
		ClientID:     req.ClientID,
		ForceRefresh: req.ForceRefresh,
		SelectAsCTA:  req.SelectAsCTA,
		UserID:       userCtx.ID,
		Username:     userCtx.Username,
		Roles:        roles,
	}

	// Enqueue task using Redis LPush
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("marshaling task: %w", err)
	}

	log.Printf("[async_executor] enqueueing query %s to queue %s", queryID, queueKey)
	_, err = e.rdb.LPush(ctx, queueKey, taskJSON).Result()
	if err != nil {
		failed := *q
		failed.Status = "failed"
		failed.ErrorMessage = fmt.Sprintf("enqueueing task: %v", err)
		now := time.Now()
		failed.EndTime = &now
		failed.UpdatedAt = now
		_ = e.queryRepo.Update(ctx, &failed)
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
		QueryID:  queryID,
		Status:   q.Status,
		Progress: q.Progress,
		Rows:     q.Rows,
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
		// FIX G-1: Only set timeout if StartTime is valid (not zero time) and status is pending/running
		// Check StartTime is not a zero/empty time (year > 2020 indicates valid recent time)
		isValidStartTime := !q.StartTime.IsZero() && q.StartTime.Year() > 2020
		if isValidStartTime && (q.Status == "pending" || q.Status == "running") {
			timeoutDuration := 30 * time.Second
			timeoutAt := q.StartTime.Add(timeoutDuration)
			response.TimeoutAt = timeoutAt
		}
	}

	return response, nil
}

// Cancel cancels a running query with idempotent and race-safe semantics.
// Returns CancelResult describing what action was taken.
func (e *AsyncQueryExecutor) Cancel(ctx context.Context, queryID string, userCtx auth.UserContext) (*CancelResult, error) {
	q, err := e.queryRepo.GetByID(ctx, queryID)
	if err != nil {
		return nil, fmt.Errorf("getting query: %w", err)
	}
	if q == nil {
		return nil, fmt.Errorf("query not found")
	}

	// Check ownership
	if q.UserID != userCtx.ID {
		roles, err := e.rlsRepo.GetRoleNamesByUser(ctx, userCtx.ID)
		if err != nil || !isAdminRole(roles) {
			return nil, fmt.Errorf("forbidden")
		}
	}

	// Idempotent: return early if already finished
	switch q.Status {
	case "stopped":
		return &CancelResult{Action: "already_stopped", CurrentStatus: "stopped"}, nil
	case "success", "failed":
		return &CancelResult{Action: "already_completed", CurrentStatus: q.Status}, nil
	case "pending", "running":
		// proceed
	default:
		return &CancelResult{Action: "already_completed", CurrentStatus: q.Status}, nil
	}

	// Set cancel flag in Redis
	if e.rdb != nil {
		e.rdb.Set(ctx, queryCancelKey+queryID, "1", CancelFlagTTL)
	}

	// DB-level cancellation: read PID from extra_json first, fallback Redis
	if e.connPool != nil && e.databaseRepo != nil {
		pid := 0
		// Try extra_json first
		if q.ExtraJSON != "" {
			var extra struct {
				BackendPID int `json:"backend_pid"`
			}
			if err := json.Unmarshal([]byte(q.ExtraJSON), &extra); err == nil && extra.BackendPID > 0 {
				pid = extra.BackendPID
			}
		}
		// Fallback to Redis
		if pid == 0 && e.rdb != nil {
			pidStr, pidErr := e.rdb.Get(ctx, queryPIDKey+queryID).Result()
			if pidErr == nil && pidStr != "" {
				pid, _ = strconv.Atoi(pidStr)
			}
		}
		if pid > 0 {
			dbInfo, dbErr := e.databaseRepo.GetDatabaseByID(ctx, q.DatabaseID)
			if dbErr == nil && dbInfo != nil {
				cancelConn, connErr := e.connPool.Get(ctx, q.DatabaseID, dbInfo.SQLAlchemyURI)
				if connErr == nil {
					cancelConn.ExecContext(ctx, "SELECT pg_cancel_backend($1)", pid)
				}
			}
		}
	}

	// Conditional update: only update if status is still pending or running
	now := time.Now()
	ok, err := e.queryRepo.UpdateStatusConditional(ctx, queryID, "stopped", []string{"pending", "running"}, map[string]interface{}{
		"progress":      "stopped",
		"error_message": "Cancelled by user",
		"end_time":      now,
		"changed_on":    now,
		"updated_at":    now,
	})
	if err != nil {
		return nil, fmt.Errorf("updating query status: %w", err)
	}
	if !ok {
		// Worker already updated status before us — keep worker's result
		log.Printf("[async_executor] cancel race: query %s already updated by worker", queryID)
		q2, _ := e.queryRepo.GetByID(ctx, queryID)
		if q2 != nil {
			return &CancelResult{Action: "already_completed", CurrentStatus: q2.Status}, nil
		}
	}

	return &CancelResult{Action: "cancelling", CurrentStatus: "stopped"}, nil
}

// ExecuteTask executes a task directly (used by worker)
func (e *AsyncQueryExecutor) ExecuteTask(ctx context.Context, task *query.QueryTask) error {
	queueKey := resolveQueueForTask(task)
	return e.executeQuery(ctx, task, queueKey)
}

// executeQuery executes a query task with retry logic
func (e *AsyncQueryExecutor) executeQuery(ctx context.Context, task *query.QueryTask, queueKey string) error {
	queryID := task.QueryID
	if err := ctx.Err(); err != nil {
		return err
	}

	// Update status to running
	q, err := e.queryRepo.GetByID(ctx, queryID)
	if err != nil {
		log.Printf("[query_worker] error getting query %s: %v", queryID, err)
		return err
	}
	if q == nil {
		return fmt.Errorf("query not found")
	}

	cancelled, err := e.isCancelled(ctx, queryID)
	if err != nil {
		log.Printf("[query_worker] cancel check failed for query %s: %v", queryID, err)
	} else if cancelled {
		return e.handleQueryCancelled(ctx, queryID)
	}

	startTime := time.Now()
	// Conditional update: only set running if still pending (cancel might have fired)
	runningOK, err := e.queryRepo.UpdateStatusConditional(ctx, queryID, "running", []string{"pending"}, map[string]interface{}{
		"progress":           "running",
		"start_running_time": startTime,
		"start_time":         startTime,
		"changed_on":         startTime,
		"updated_at":         time.Now(),
	})
	if err != nil {
		log.Printf("[query_worker] error updating query %s: %v", queryID, err)
		return err
	}
	if !runningOK {
		// Query was cancelled before it could start running
		log.Printf("[query_worker] query %s was cancelled before execution", queryID)
		return e.handleQueryCancelled(ctx, queryID)
	}

	// Build running copy for downstream use (all-retries-failed path)
	running := *q
	running.Status = "running"
	running.Progress = "running"
	running.StartRunningTime = &startTime
	running.StartTime = &startTime

	// Publish status: running
	e.publishProgress(ctx, queryID, "running")

	// Execute the query using the sync executor
	execReq := ExecuteRequest{
		DatabaseID:   task.DatabaseID,
		SQL:          task.SQL,
		Limit:        task.Limit,
		Schema:       task.Schema,
		Catalog:      task.Catalog,
		TabName:      task.TabName,
		SqlEditorID:  task.SqlEditorID,
		ClientID:     task.ClientID,
		ForceRefresh: task.ForceRefresh,
		SelectAsCTA:  task.SelectAsCTA,
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
		if err := ctx.Err(); err != nil {
			return err
		}

		cancelled, err := e.isCancelled(ctx, queryID)
		if err != nil {
			log.Printf("[query_worker] cancel check failed for query %s: %v", queryID, err)
		} else if cancelled {
			return e.handleQueryCancelled(ctx, queryID)
		}

		if attempt > 0 {
			if err := e.waitForRetry(ctx, attempt); err != nil {
				return err
			}
		}

		resp, err := e.executeWithWorkerSlot(queueKey, func() (*ExecuteResponse, error) {
			execCtx := WithProgressCallback(ctx, func(fetchedRows, totalRows int) {
				pct := 50
				if totalRows > 0 {
					pct = 50 + int(float64(fetchedRows)/float64(totalRows)*50.0)
					if pct > 99 {
						pct = 99
					}
				}
				e.publishProgress(ctx, queryID, "fetching", pct)
			})
			return e.queryCache.Execute(execCtx, execReq, userCtx)
		})
		if err == nil {
			// Phase 2: extra_json with PID + queue + attempt
			e.enrichExtraJSON(ctx, queryID, queueKey, attempt+1)
			return e.handleQuerySuccess(ctx, queryID, resp)
		}
		lastErr = err
		log.Printf("[query_worker] attempt %d failed for query %s: %v", attempt+1, queryID, err)
	}

	// All retries failed - QE-004 #5
	ok, _ := e.queryRepo.UpdateStatusConditional(ctx, queryID, "failed", []string{"running"}, map[string]interface{}{
		"progress":      "failed",
		"error_message": fmt.Sprintf("failed after %d attempts: %v", MaxRetry, lastErr),
		"end_time":      time.Now(),
		"changed_on":    time.Now(),
		"updated_at":    time.Now(),
	})
	if ok {
		e.publishProgress(ctx, queryID, "failed")
		e.publishError(ctx, queryID, fmt.Sprintf("failed after %d attempts: %v", MaxRetry, lastErr))
	}
	return lastErr
}

func (e *AsyncQueryExecutor) executeWithWorkerSlot(queueKey string, fn func() (*ExecuteResponse, error)) (*ExecuteResponse, error) {
	if !e.workerPool.acquire(queueKey) {
		return nil, fmt.Errorf("no worker available")
	}
	defer e.workerPool.release(queueKey)
	return fn()
}

func (e *AsyncQueryExecutor) isCancelled(ctx context.Context, queryID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return true, err
	}
	if e.rdb == nil {
		return false, nil
	}
	cancelled, err := e.rdb.Exists(ctx, queryCancelKey+queryID).Result()
	if err != nil {
		return false, err
	}
	return cancelled > 0, nil
}

// enrichExtraJSON reads the PID from Redis (set by sync executor) and merges it with
// queue/attempt metadata into the query record's extra_json field.
func (e *AsyncQueryExecutor) enrichExtraJSON(ctx context.Context, queryID string, queueKey string, attempt int) {
	if queryID == "" {
		return
	}

	// Read PID from Redis (set by sync executor's executeAndRespond)
	pid := 0
	if e.rdb != nil {
		pidStr, _ := e.rdb.Get(ctx, queryPIDKey+queryID).Result()
		if pidStr != "" {
			pid, _ = strconv.Atoi(pidStr)
		}
	}
	queue := queueKeyToName(queueKey)

	extraJSON := buildExtraJSON(pid, queue, attempt)
	e.queryRepo.UpdateStatusConditional(ctx, queryID, "running", []string{"pending", "running"}, map[string]interface{}{
		"extra_json": extraJSON,
		"changed_on": time.Now(),
	})
}

func (e *AsyncQueryExecutor) handleQueryCancelled(ctx context.Context, queryID string) error {
	ok, err := e.queryRepo.UpdateStatusConditional(ctx, queryID, "stopped", []string{"running"}, map[string]interface{}{
		"progress":      "stopped",
		"error_message": "Cancelled by user",
		"end_time":      time.Now(),
		"changed_on":    time.Now(),
		"updated_at":    time.Now(),
	})
	if err != nil {
		return err
	}
	if !ok {
		// Cancel handler already updated status — nothing to do
		return nil
	}
	e.publishError(ctx, queryID, "Query cancelled")
	return nil
}

func defaultWaitForRetry(ctx context.Context, attempt int) error {
	return waitWithContext(ctx, backoffForAttempt(attempt))
}

func backoffForAttempt(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	backoff := RetryInterval
	for i := 1; i < attempt; i++ {
		backoff *= RetryMultiplier
	}
	return backoff
}

func waitWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (e *AsyncQueryExecutor) handleQuerySuccess(ctx context.Context, queryID string, resp *ExecuteResponse) error {
	// Check cancel flag
	cancelled, err := e.isCancelled(ctx, queryID)
	if err != nil {
		log.Printf("[query_worker] cancel check failed for query %s: %v", queryID, err)
	} else if cancelled {
		return e.handleQueryCancelled(ctx, queryID)
	}

	// Store result in Redis (up to 10MB)
	var resultsKey string
	var resultSize int
	var respJSON []byte
	if e.rdb != nil && resp.Data != nil {
		respJSON, err = json.Marshal(resp)
		if err == nil {
			resultSize = len(respJSON)
			if resultSize <= 10*MaxInlineResultBytes {
				resultsKey = queryResultKey + queryID
				e.rdb.Set(ctx, resultsKey, respJSON, 24*time.Hour)
			}
		}
	}

	endTime := time.Now()
	rowCount := 0
	if resp.Data != nil {
		if data, ok := resp.Data.([]interface{}); ok {
			rowCount = len(data)
		}
	}

	// Conditional update: only update if status is still running
	ok, err := e.queryRepo.UpdateStatusConditional(ctx, queryID, "success", []string{"running"}, map[string]interface{}{
		"progress":     "done",
		"end_time":     endTime,
		"rows":         rowCount,
		"results_key":  resultsKey,
		"executed_sql": resp.Query.ExecutedSQL,
		"changed_on":   endTime,
		"updated_at":   time.Now(),
	})
	if err != nil {
		log.Printf("[query_worker] error updating query %s: %v", queryID, err)
	}
	if !ok {
		// Query was cancelled — discard result
		log.Printf("[query_worker] query %s was cancelled, discarding result", queryID)
		if e.rdb != nil {
			e.rdb.Del(ctx, queryResultKey+queryID)
		}
		return nil
	}

	e.publishProgress(ctx, queryID, "done")

	switch {
	case resultSize > 0 && resultSize <= MaxInlineResultBytes:
		e.publishInlineResult(ctx, queryID, resp)
	case resultSize > MaxInlineResultBytes:
		e.publishResultReady(ctx, queryID)
	default:
		emptyResp := &ExecuteResponse{
			Data:    []interface{}{},
			Columns: resp.Columns,
		}
		e.publishInlineResult(ctx, queryID, emptyResp)
	}
	return nil
}

// publishProgress publishes a progress event via Redis pub/sub with an optional
// dynamic percent. If percent < 0, it is derived from the progress stage name.
func (e *AsyncQueryExecutor) publishProgress(ctx context.Context, queryID, progress string, optPercent ...int) {
	if e.rdb == nil {
		return
	}

	percent := -1
	if len(optPercent) > 0 {
		percent = optPercent[0]
	}
	if percent < 0 {
		switch progress {
		case "queued":
			percent = 10
		case "running":
			percent = 50
		case "fetching":
			percent = 80
		case "done":
			percent = 100
		default:
			percent = 0
		}
	}

	event := map[string]interface{}{
		"type":     "progress",
		"query_id": queryID,
		"progress": progress,
		"percent":  percent,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("[query_worker] error marshaling progress event: %v", err)
		return
	}

	if err := e.rdb.Publish(ctx, queryStatusChannel+queryID, eventJSON).Err(); err != nil {
		log.Printf("[query_worker] error publishing progress event: %v", err)
	}
}

// publishInlineResult publishes a "done" event with inline result data (≤1MB)
func (e *AsyncQueryExecutor) publishInlineResult(ctx context.Context, queryID string, result *ExecuteResponse) {
	if e.rdb == nil {
		return
	}

	// Format data per spec: { "rows": [...], "columns": [...] }
	data := map[string]interface{}{
		"rows":    result.Data,
		"columns": result.Columns,
	}

	event := map[string]interface{}{
		"type":     "done",
		"query_id": queryID,
		"data":     data,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("[query_worker] error marshaling done event: %v", err)
		return
	}

	if err := e.rdb.Publish(ctx, queryStatusChannel+queryID, eventJSON).Err(); err != nil {
		log.Printf("[query_worker] error publishing done event: %v", err)
	}
}

// publishResultReady publishes a "result_ready" event for large results (>1MB)
func (e *AsyncQueryExecutor) publishResultReady(ctx context.Context, queryID string) {
	if e.rdb == nil {
		return
	}

	event := map[string]interface{}{
		"type":         "result_ready",
		"query_id":     queryID,
		"download_url": "/api/v1/query/" + queryID + "/result/download",
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("[query_worker] error marshaling result_ready event: %v", err)
		return
	}

	if err := e.rdb.Publish(ctx, queryStatusChannel+queryID, eventJSON).Err(); err != nil {
		log.Printf("[query_worker] error publishing result_ready event: %v", err)
	}
}

// publishError publishes an error event via Redis pub/sub
func (e *AsyncQueryExecutor) publishError(ctx context.Context, queryID, message string) {
	if e.rdb == nil {
		return
	}

	event := map[string]interface{}{
		"type":     "error",
		"query_id": queryID,
		"message":  message,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("[query_worker] error marshaling error event: %v", err)
		return
	}

	if err := e.rdb.Publish(ctx, queryStatusChannel+queryID, eventJSON).Err(); err != nil {
		log.Printf("[query_worker] error publishing error event: %v", err)
	}
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
			return queryQueueDefault // FIX: was queryQueueKey (wrong queue "queue:query:async")
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

// isAlphaRole checks if user has Alpha role
func isAlphaRole(roles []string) bool {
	for _, role := range roles {
		if role == "Alpha" {
			return true
		}
	}
	return false
}

// resolveQueueForTask resolves the queue key based on user roles
// Per QE-004 spec: Admin→critical, Alpha→default, Gamma→low
func resolveQueueForTask(task *query.QueryTask) string {
	if len(task.Roles) > 0 {
		if isAdminRole(task.Roles) {
			return queryQueueCritical
		} else if isAlphaRole(task.Roles) {
			return queryQueueDefault // Fixed G-3: was hardcoded to "queue:query:async"
		}
	}
	// Default to low queue for Gamma and unknown roles
	return queryQueueLow
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

// GetResult gets the result of a completed query with optional pagination.
func (e *AsyncQueryExecutor) GetResult(ctx context.Context, queryID string, offset, limit int) (*ExecuteResponse, error) {
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
				return paginateResult(&result, offset, limit), nil
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

// paginateResult slices the result data to the requested page.
func paginateResult(resp *ExecuteResponse, offset, limit int) *ExecuteResponse {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 1000
	}

	data, ok := resp.Data.([]interface{})
	if !ok || len(data) == 0 {
		return resp
	}

	total := len(data)
	if offset >= total {
		return &ExecuteResponse{
			Data:      []interface{}{},
			Columns:   resp.Columns,
			FromCache: resp.FromCache,
			Query:     resp.Query,
		}
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// Copy to avoid aliasing the underlying Redis-cached slice
	page := make([]interface{}, end-offset)
	copy(page, data[offset:end])

	return &ExecuteResponse{
		Data:      page,
		Columns:   resp.Columns,
		FromCache: resp.FromCache,
		Query:     resp.Query,
	}
}

// GetResultForUser retrieves result with ownership check (for download link auth)
func (e *AsyncQueryExecutor) GetResultForUser(ctx context.Context, queryID string, userCtx auth.UserContext) (*ExecuteResponse, error) {
	q, err := e.queryRepo.GetByID(ctx, queryID)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, fmt.Errorf("query not found")
	}

	if q.UserID != userCtx.ID {
		roles, err := e.rlsRepo.GetRoleNamesByUser(ctx, userCtx.ID)
		if err != nil || !isAdminRole(roles) {
			return nil, fmt.Errorf("forbidden")
		}
	}

	if q.Status != "success" {
		return nil, fmt.Errorf("query not completed")
	}

	return e.GetResult(ctx, queryID, 0, 0) // 0 means no pagination
}
