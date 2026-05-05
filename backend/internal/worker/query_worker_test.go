package worker

import (
	"testing"
	"time"

	redismock "github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"

	svcquery "superset/auth-service/internal/app/query"
)

// TestQueryExecutorCreation tests that query executor can be created
func TestQueryExecutorCreation(t *testing.T) {
	queryExec := svcquery.NewQueryExecutor(nil, nil, nil, nil, nil, nil, nil)
	assert.NotNil(t, queryExec)

	// Create worker with nil dependencies - verifies constructor
	worker := NewQueryWorker(nil, queryExec, nil, DefaultQueryWorkerConfig())
	assert.NotNil(t, worker)
	assert.NotNil(t, worker.config)
}

// TestDefaultQueryWorkerConfig tests the default config
func TestDefaultQueryWorkerConfig(t *testing.T) {
	config := DefaultQueryWorkerConfig()
	assert.Equal(t, 5, config.WorkerCount)
	assert.Len(t, config.QueueKeys, 3)
	// Verify minimum poll interval (1 second)
	assert.GreaterOrEqual(t, config.PollInterval.Milliseconds(), int64(1000))
}

func TestQueryWorker_ProcessNext_UsesBRPop(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	queueKey := "queue:query:critical"
	mock.ExpectBRPop(1*time.Second, queueKey).RedisNil()

	worker := NewQueryWorker(rdb, nil, nil, QueryWorkerConfig{
		PollInterval: 1 * time.Second,
		WorkerCount:  1,
		QueueKeys:    []string{queueKey},
	})

	worker.processNext()
	assert.NoError(t, mock.ExpectationsWereMet())
}
