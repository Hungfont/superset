package redis

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

type fakeSyncQueueWriter struct {
	mu        sync.Mutex
	err       error
	queueName string
	payloads  [][]byte
}

func (f *fakeSyncQueueWriter) Push(_ context.Context, queueName string, payload []byte) error {
	if f.err != nil {
		return f.err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.queueName = queueName
	f.payloads = append(f.payloads, append([]byte(nil), payload...))
	return nil
}

func TestDatasetSyncQueueEnqueueSyncColumnsSuccess(t *testing.T) {
	writer := &fakeSyncQueueWriter{}
	queue := newDatasetSyncQueue(writer, defaultDatasetSyncQueueName)
	queue.newID = func() string { return "job-sync-1" }

	jobID, err := queue.EnqueueSyncColumns(context.Background(), 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if jobID != "job-sync-1" {
		t.Fatalf("expected job-sync-1, got %s", jobID)
	}

	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.queueName != defaultDatasetSyncQueueName {
		t.Fatalf("expected queue name %s, got %s", defaultDatasetSyncQueueName, writer.queueName)
	}
	if len(writer.payloads) != 1 {
		t.Fatalf("expected one payload, got %d", len(writer.payloads))
	}

	var payload datasetSyncColumnsPayload
	if err := json.Unmarshal(writer.payloads[0], &payload); err != nil {
		t.Fatalf("expected valid payload json, got %v", err)
	}
	if payload.DatasetID != 42 {
		t.Fatalf("expected dataset_id 42, got %d", payload.DatasetID)
	}
}

func TestDatasetSyncQueueEnqueueSyncColumnsReturnsWriterError(t *testing.T) {
	writer := &fakeSyncQueueWriter{err: errors.New("redis down")}
	queue := newDatasetSyncQueue(writer, defaultDatasetSyncQueueName)

	_, err := queue.EnqueueSyncColumns(context.Background(), 7)
	if err == nil {
		t.Fatal("expected writer error")
	}
}

func TestDatasetSyncQueueEnqueueSyncColumnsRejectsZeroDatasetID(t *testing.T) {
	writer := &fakeSyncQueueWriter{}
	queue := newDatasetSyncQueue(writer, defaultDatasetSyncQueueName)

	_, err := queue.EnqueueSyncColumns(context.Background(), 0)
	if !errors.Is(err, errInvalidSyncDatasetID) {
		t.Fatalf("expected errInvalidSyncDatasetID, got %v", err)
	}
}

func TestDatasetSyncQueueDefaultsQueueNameWhenEmpty(t *testing.T) {
	queue := newDatasetSyncQueue(&fakeSyncQueueWriter{}, " ")

	if queue.queueName != defaultDatasetSyncQueueName {
		t.Fatalf("expected queue name %s, got %s", defaultDatasetSyncQueueName, queue.queueName)
	}
}

