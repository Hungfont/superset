package redis

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeAsyncQueueWriter struct {
	mu      sync.Mutex
	err     error
	payload [][]byte
}

func (f *fakeAsyncQueueWriter) Push(_ context.Context, _ string, payload []byte) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payload = append(f.payload, append([]byte(nil), payload...))
	return nil
}

func TestDatasetAsyncQueueEnqueueSyncColumnsSuccess(t *testing.T) {
	writer := &fakeAsyncQueueWriter{}
	queue := newDatasetAsyncQueue(writer, datasetAsyncColumnsTaskType, 1)
	t.Cleanup(func() {
		_ = queue.Shutdown(context.Background())
	})
	queue.newID = func() string { return "job-1" }

	jobID, err := queue.EnqueueSyncColumns(context.Background(), 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if jobID != "job-1" {
		t.Fatalf("expected job-1, got %s", jobID)
	}

	time.Sleep(50 * time.Millisecond)

	writer.mu.Lock()
	defer writer.mu.Unlock()
	if len(writer.payload) != 1 {
		t.Fatalf("expected one payload, got %d", len(writer.payload))
	}

	var body datasetAsyncColumnsPayload
	if err := json.Unmarshal(writer.payload[0], &body); err != nil {
		t.Fatalf("expected valid payload json, got %v", err)
	}
	if body.DatasetID != 42 {
		t.Fatalf("expected dataset_id 42, got %d", body.DatasetID)
	}
	if body.JobID != "job-1" {
		t.Fatalf("expected job_id job-1, got %s", body.JobID)
	}
}

func TestDatasetAsyncQueueEnqueueSyncColumnsFillsBuffer(t *testing.T) {
	writer := &fakeAsyncQueueWriter{}
	queue := newDatasetAsyncQueue(writer, datasetAsyncColumnsTaskType, 2)
	t.Cleanup(func() {
		_ = queue.Shutdown(context.Background())
	})

	jobID1, err := queue.EnqueueSyncColumns(context.Background(), 7)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if jobID1 == "" {
		t.Fatal("expected jobID")
	}

	jobID2, err := queue.EnqueueSyncColumns(context.Background(), 8)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if jobID2 == "" {
		t.Fatal("expected jobID")
	}
	if jobID1 == jobID2 {
		t.Fatal("expected different jobIDs")
	}
}

func TestDatasetAsyncQueueEnqueueSyncColumnsRejectsZeroDatasetID(t *testing.T) {
	writer := &fakeAsyncQueueWriter{}
	queue := newDatasetAsyncQueue(writer, datasetAsyncColumnsTaskType, 1)
	t.Cleanup(func() {
		_ = queue.Shutdown(context.Background())
	})

	_, err := queue.EnqueueSyncColumns(context.Background(), 0)
	if !errors.Is(err, errInvalidAsyncDatasetID) {
		t.Fatalf("expected errInvalidAsyncDatasetID, got %v", err)
	}
}

func TestDatasetAsyncQueueReturnsContextErrorWhenCanceledBeforeSend(t *testing.T) {
	writer := &fakeAsyncQueueWriter{}
	queue := newDatasetAsyncQueue(writer, datasetAsyncColumnsTaskType, 1)
	t.Cleanup(func() {
		_ = queue.Shutdown(context.Background())
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := queue.EnqueueSyncColumns(ctx, 9)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestDatasetAsyncQueueEnqueueAfterShutdownReturnsClosed(t *testing.T) {
	writer := &fakeAsyncQueueWriter{}
	queue := newDatasetAsyncQueue(writer, datasetAsyncColumnsTaskType, 1)
	if err := queue.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected nil shutdown error, got %v", err)
	}

	_, err := queue.EnqueueSyncColumns(context.Background(), 10)
	if !errors.Is(err, errAsyncQueueClosed) {
		t.Fatalf("expected errAsyncQueueClosed, got %v", err)
	}
}

func TestDatasetAsyncQueueShutdownHonorsContextTimeout(t *testing.T) {
	writer := &fakeAsyncQueueWriter{}
	queue := newDatasetAsyncQueue(writer, datasetAsyncColumnsTaskType, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := queue.Shutdown(ctx); err != nil {
		t.Fatalf("expected nil shutdown error, got %v", err)
	}
}
