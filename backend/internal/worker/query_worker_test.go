package worker

import (
	"testing"

	svcquery "superset/auth-service/internal/app/query"
	"github.com/stretchr/testify/assert"
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