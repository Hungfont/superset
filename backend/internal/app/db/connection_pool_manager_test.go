package auth_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/db"
)

type fakeSQLConnection struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration

	pingErr atomic.Value

	closeCalled atomic.Int32
}

func (f *fakeSQLConnection) SetMaxOpenConns(n int) {
	f.maxOpenConns = n
}

func (f *fakeSQLConnection) SetMaxIdleConns(n int) {
	f.maxIdleConns = n
}

func (f *fakeSQLConnection) SetConnMaxLifetime(d time.Duration) {
	f.connMaxLifetime = d
}

func (f *fakeSQLConnection) PingContext(_ context.Context) error {
	value := f.pingErr.Load()
	if value == nil {
		return nil
	}
	err, _ := value.(error)
	return err
}

func (f *fakeSQLConnection) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeSQLConnection) Close() error {
	f.closeCalled.Add(1)
	return nil
}

func (f *fakeSQLConnection) setPingError(err error) {
	f.pingErr.Store(err)
}

type fakeSQLOpener struct {
	mu          sync.Mutex
	openCalls   int
	connections []*fakeSQLConnection
}

func (f *fakeSQLOpener) Open(_ string, _ string) (svcauth.SQLConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.openCalls++
	connection := &fakeSQLConnection{}
	f.connections = append(f.connections, connection)
	return connection, nil
}

func TestConnectionPoolManager_GetUsesSingleflightForConcurrentCalls(t *testing.T) {
	opener := &fakeSQLOpener{}
	manager := svcauth.NewConnectionPoolManager(opener, svcauth.ConnectionPoolManagerConfig{HealthInterval: time.Hour})
	t.Cleanup(func() {
		_ = manager.Shutdown(context.Background())
	})

	const workerCount = 25
	results := make([]svcauth.SQLConnection, workerCount)
	errorsByWorker := make([]error, workerCount)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			connection, err := manager.Get(context.Background(), 42, "postgresql://alice:secret@localhost:5432/analytics")
			results[index] = connection
			errorsByWorker[index] = err
		}(i)
	}
	wg.Wait()

	for _, err := range errorsByWorker {
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	}

	if opener.openCalls != 1 {
		t.Fatalf("expected opener called once, got %d", opener.openCalls)
	}

	first := results[0]
	for i := 1; i < len(results); i++ {
		if results[i] != first {
			t.Fatal("expected all workers to receive the same connection instance")
		}
	}
}

func TestConnectionPoolManager_GetAppliesConnectionLimits(t *testing.T) {
	opener := &fakeSQLOpener{}
	manager := svcauth.NewConnectionPoolManager(opener, svcauth.ConnectionPoolManagerConfig{HealthInterval: time.Hour})
	t.Cleanup(func() {
		_ = manager.Shutdown(context.Background())
	})

	_, err := manager.Get(context.Background(), 7, "postgresql://alice:secret@localhost:5432/analytics")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(opener.connections) != 1 {
		t.Fatalf("expected one connection, got %d", len(opener.connections))
	}

	connection := opener.connections[0]
	if connection.maxOpenConns != 10 {
		t.Fatalf("expected max open conns 10, got %d", connection.maxOpenConns)
	}
	if connection.maxIdleConns != 3 {
		t.Fatalf("expected max idle conns 3, got %d", connection.maxIdleConns)
	}
	if connection.connMaxLifetime != 30*time.Minute {
		t.Fatalf("expected conn max lifetime 30m, got %s", connection.connMaxLifetime)
	}
}

func TestConnectionPoolManager_CloseRemovesAndClosesPool(t *testing.T) {
	opener := &fakeSQLOpener{}
	manager := svcauth.NewConnectionPoolManager(opener, svcauth.ConnectionPoolManagerConfig{HealthInterval: time.Hour})
	t.Cleanup(func() {
		_ = manager.Shutdown(context.Background())
	})

	_, err := manager.Get(context.Background(), 8, "postgresql://alice:secret@localhost:5432/analytics")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	connection := opener.connections[0]
	if err := manager.Close(context.Background(), 8); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if connection.closeCalled.Load() != 1 {
		t.Fatalf("expected one close call, got %d", connection.closeCalled.Load())
	}

	_, err = manager.Get(context.Background(), 8, "postgresql://alice:secret@localhost:5432/analytics")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if opener.openCalls != 2 {
		t.Fatalf("expected opener called twice after close+get, got %d", opener.openCalls)
	}
}

func TestConnectionPoolManager_HealthMonitorEvictsUnhealthyPools(t *testing.T) {
	opener := &fakeSQLOpener{}
	manager := svcauth.NewConnectionPoolManager(opener, svcauth.ConnectionPoolManagerConfig{
		HealthInterval: 10 * time.Millisecond,
		PingTimeout:    10 * time.Millisecond,
	})
	t.Cleanup(func() {
		_ = manager.Shutdown(context.Background())
	})

	_, err := manager.Get(context.Background(), 9, "postgresql://alice:secret@localhost:5432/analytics")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	connection := opener.connections[0]
	connection.setPingError(errors.New("connection unavailable"))

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if connection.closeCalled.Load() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if connection.closeCalled.Load() == 0 {
		t.Fatal("expected health monitor to evict and close unhealthy connection")
	}
}

func TestConnectionPoolManager_ShutdownClosesAllPools(t *testing.T) {
	opener := &fakeSQLOpener{}
	manager := svcauth.NewConnectionPoolManager(opener, svcauth.ConnectionPoolManagerConfig{HealthInterval: time.Hour})

	_, err := manager.Get(context.Background(), 100, "postgresql://alice:secret@localhost:5432/analytics")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	_, err = manager.Get(context.Background(), 101, "postgresql://alice:secret@localhost:5432/analytics")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := manager.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if opener.connections[0].closeCalled.Load() != 1 {
		t.Fatalf("expected first connection closed once, got %d", opener.connections[0].closeCalled.Load())
	}
	if opener.connections[1].closeCalled.Load() != 1 {
		t.Fatalf("expected second connection closed once, got %d", opener.connections[1].closeCalled.Load())
	}
}
