package query

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	redismock "github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"

	authdomain "superset/auth-service/internal/domain/auth"
	domainquery "superset/auth-service/internal/domain/query"
)

// Ensure queryRepoStub implements domain Repository interface
var _ domainquery.Repository = (*queryRepoStub)(nil)

type queryRepoStub struct {
	created *Query
	updated []*Query
	queries map[string]*Query
}

func newQueryRepoStub() *queryRepoStub {
	return &queryRepoStub{queries: map[string]*Query{}}
}

func (s *queryRepoStub) Create(_ context.Context, q *Query) error {
	copy := *q
	s.created = &copy
	s.queries[q.ID] = &copy
	return nil
}

func (s *queryRepoStub) GetByID(_ context.Context, id string) (*Query, error) {
	if q, ok := s.queries[id]; ok {
		return q, nil
	}
	return nil, nil
}

func (s *queryRepoStub) Update(_ context.Context, q *Query) error {
	copy := *q
	s.updated = append(s.updated, &copy)
	s.queries[q.ID] = &copy
	return nil
}

func (s *queryRepoStub) List(_ context.Context, _ *ListFilter) ([]*Query, int64, error) {
	return nil, 0, nil
}

type roleNameProviderStub struct {
	roles []string
	err   error
}

func (s roleNameProviderStub) GetRoleNamesByUser(_ context.Context, _ uint) ([]string, error) {
	return s.roles, s.err
}

type executorStub struct {
	calls int
	resp  *ExecuteResponse
	err   error
}

func (s *executorStub) Execute(_ context.Context, _ ExecuteRequest, _ authdomain.UserContext) (*ExecuteResponse, error) {
	s.calls++
	return s.resp, s.err
}

func TestSubmit_EnqueueFailureSetsFailedStatus(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	mock.ExpectLPush(queryQueueKey, mock.Anything).SetErr(errors.New("enqueue failed"))

	repo := newQueryRepoStub()
	roles := roleNameProviderStub{roles: []string{"Alpha"}}
	executor := &executorStub{}
	asyncExecutor := NewAsyncQueryExecutor(rdb, repo, roles, nil, executor)

	_, err := asyncExecutor.Submit(context.Background(), AsyncSubmitRequest{
		DatabaseID: 1,
		SQL:        "select 1",
	}, authdomain.UserContext{ID: 1, Username: "tester"})

	assert.Error(t, err)
	if assert.Len(t, repo.updated, 1) {
		assert.Equal(t, "failed", repo.updated[0].Status)
		assert.Contains(t, repo.updated[0].ErrorMessage, "enqueueing task")
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteQuery_CancelledBeforeAttempt(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	queryID := "q-test"
	mock.ExpectExists(queryCancelKey + queryID).SetVal(1)
	mock.ExpectPublish(queryStatusChannel+queryID, mock.Anything).SetVal(1)

	repo := newQueryRepoStub()
	repo.queries[queryID] = &Query{ID: queryID, Status: "pending", DatabaseID: 1, UserID: 1, SQL: "select 1"}
	executor := &executorStub{err: errors.New("should not run")}
	asyncExecutor := NewAsyncQueryExecutor(rdb, repo, roleNameProviderStub{}, nil, executor)

	err := asyncExecutor.executeQuery(context.Background(), &QueryTask{
		QueryID:    queryID,
		DatabaseID: 1,
		SQL:        "select 1",
		UserID:     1,
		Username:   "tester",
	}, queryQueueKey)

	assert.NoError(t, err)
	assert.Equal(t, 0, executor.calls)
	if assert.NotEmpty(t, repo.updated) {
		assert.Equal(t, "stopped", repo.updated[len(repo.updated)-1].Status)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWaitWithContext_RespectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := waitWithContext(ctx, time.Minute)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), 200*time.Millisecond)
}

func TestExecuteQuery_ReleasesWorkerSlotBeforeBackoff(t *testing.T) {
	repo := newQueryRepoStub()
	queryID := "q-backoff"
	repo.queries[queryID] = &Query{ID: queryID, Status: "pending", DatabaseID: 1, UserID: 1, SQL: "select 1"}

	executor := &executorStub{err: errors.New("boom")}
	asyncExecutor := NewAsyncQueryExecutor(nil, repo, roleNameProviderStub{}, nil, executor)
	asyncExecutor.workerPool = &WorkerPool{
		critical: make(chan struct{}, 1),
		defaultQ: make(chan struct{}, 1),
		low:      make(chan struct{}, 1),
	}

	var poolLen int
	asyncExecutor.waitForRetry = func(_ context.Context, _ int) error {
		poolLen = len(asyncExecutor.workerPool.defaultQ)
		return context.Canceled
	}

	err := asyncExecutor.executeQuery(context.Background(), &QueryTask{
		QueryID:    queryID,
		DatabaseID: 1,
		SQL:        "select 1",
		UserID:     1,
		Username:   "tester",
	}, queryQueueKey)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, poolLen)
}

func TestBackoffForAttempt_Exponential(t *testing.T) {
	assert.Equal(t, RetryInterval, backoffForAttempt(1))
	assert.Equal(t, RetryInterval*RetryMultiplier, backoffForAttempt(2))
}
