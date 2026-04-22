package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	"superset/auth-service/internal/pkg/crypto"

	domain "superset/auth-service/internal/domain/db"

	"golang.org/x/sync/singleflight"
)

const (
	poolManagerDefaultMaxOpenConns    = 10
	poolManagerDefaultMaxIdleConns    = 3
	poolManagerDefaultConnMaxLifetime = 30 * time.Minute
	poolManagerDefaultHealthInterval  = 60 * time.Second
	poolManagerDefaultPingTimeout     = 5 * time.Second
)

// SQLConnection represents the minimum contract required by the pool manager.
type SQLConnection interface {
	SetMaxOpenConns(n int)
	SetMaxIdleConns(n int)
	SetConnMaxLifetime(d time.Duration)
	PingContext(ctx context.Context) error
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	Close() error
}

// SQLConnectionOpener creates SQL connections from driver + DSN.
type SQLConnectionOpener interface {
	Open(driverName string, dsn string) (SQLConnection, error)
}

type defaultSQLConnectionOpener struct{}

func (defaultSQLConnectionOpener) Open(driverName string, dsn string) (SQLConnection, error) {
	return sql.Open(driverName, dsn)
}

// ConnectionPoolManagerConfig controls pool limits and monitor settings.
type ConnectionPoolManagerConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	HealthInterval  time.Duration
	PingTimeout     time.Duration
}

// DatabaseConnectionPool manages SQL connection pools keyed by database ID.
type DatabaseConnectionPool interface {
	Get(ctx context.Context, databaseID uint, sqlalchemyURI string) (SQLConnection, error)
	Close(ctx context.Context, databaseID uint) error
	Shutdown(ctx context.Context) error
}

// ConnectionPoolManager provides lazy init, singleflight protection, and health checks.
type ConnectionPoolManager struct {
	opener         SQLConnectionOpener
	encryptionKey  []byte
	maxOpenConns   int
	maxIdleConns   int
	connMaxLife    time.Duration
	healthInterval time.Duration
	pingTimeout    time.Duration

	pools sync.Map
	sf    singleflight.Group

	stopOnce sync.Once
	stopCh   chan struct{}
}

func NewConnectionPoolManager(opener SQLConnectionOpener, config ConnectionPoolManagerConfig, encryptionKey string) (*ConnectionPoolManager, error) {
	resolvedOpener := opener
	if resolvedOpener == nil {
		resolvedOpener = defaultSQLConnectionOpener{}
	}

	parsedKey, err := crypto.ParseEncryptionKey(encryptionKey)
	if err != nil {
		return nil, domain.ErrDatabaseCredentialEncryption
	}

	manager := &ConnectionPoolManager{
		opener:         resolvedOpener,
		encryptionKey:  parsedKey,
		maxOpenConns:   resolveInt(config.MaxOpenConns, poolManagerDefaultMaxOpenConns),
		maxIdleConns:   resolveInt(config.MaxIdleConns, poolManagerDefaultMaxIdleConns),
		connMaxLife:    resolveDuration(config.ConnMaxLifetime, poolManagerDefaultConnMaxLifetime),
		healthInterval: resolveDuration(config.HealthInterval, poolManagerDefaultHealthInterval),
		pingTimeout:    resolveDuration(config.PingTimeout, poolManagerDefaultPingTimeout),
		stopCh:         make(chan struct{}),
	}

	go manager.healthMonitor()

	return manager, nil
}

func (m *ConnectionPoolManager) Get(ctx context.Context, databaseID uint, sqlalchemyURI string) (SQLConnection, error) {
	if databaseID == 0 {
		return nil, fmt.Errorf("invalid database id")
	}

	if existing, ok := m.pools.Load(databaseID); ok {
		return existing.(SQLConnection), nil
	}

	key := strconv.FormatUint(uint64(databaseID), 10)
	loaded, err, _ := m.sf.Do(key, func() (interface{}, error) {
		if existing, ok := m.pools.Load(databaseID); ok {
			return existing.(SQLConnection), nil
		}

		decryptedURI, decryptErr := crypto.DecryptSQLAlchemyURIPassword(sqlalchemyURI, m.encryptionKey)
		if decryptErr != nil {
			return nil, decryptErr
		}

		parsedURI, parseErr := crypto.ParseSQLAlchemyURI(decryptedURI)
		if parseErr != nil {
			return nil, parseErr
		}

		driverName, _, resolveErr := resolveSQLDriver(parsedURI.Scheme)
		if resolveErr != nil {
			return nil, resolveErr
		}

		connection, openErr := m.opener.Open(driverName, decryptedURI)
		if openErr != nil {
			return nil, openErr
		}

		connection.SetMaxOpenConns(m.maxOpenConns)
		connection.SetMaxIdleConns(m.maxIdleConns)
		connection.SetConnMaxLifetime(m.connMaxLife)

		pingCtx, cancel := context.WithTimeout(ctx, m.pingTimeout)
		defer cancel()
		if pingErr := connection.PingContext(pingCtx); pingErr != nil {
			_ = connection.Close()
			return nil, pingErr
		}

		m.pools.Store(databaseID, connection)
		return connection, nil
	})
	if err != nil {
		return nil, err
	}

	return loaded.(SQLConnection), nil
}

func (m *ConnectionPoolManager) Close(ctx context.Context, databaseID uint) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	value, ok := m.pools.LoadAndDelete(databaseID)
	if !ok {
		return nil
	}

	connection := value.(SQLConnection)
	if err := connection.Close(); err != nil {
		return fmt.Errorf("closing pool for database %d: %w", databaseID, err)
	}

	return nil
}

func (m *ConnectionPoolManager) Shutdown(ctx context.Context) error {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})

	var firstErr error
	m.pools.Range(func(key interface{}, _ interface{}) bool {
		if err := ctx.Err(); err != nil {
			firstErr = err
			return false
		}

		databaseID, ok := key.(uint)
		if !ok {
			return true
		}

		if err := m.Close(ctx, databaseID); err != nil && firstErr == nil {
			firstErr = err
		}

		return true
	})

	return firstErr
}

func (m *ConnectionPoolManager) healthMonitor() {
	ticker := time.NewTicker(m.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.runHealthCheck()
		case <-m.stopCh:
			return
		}
	}
}

func (m *ConnectionPoolManager) runHealthCheck() {
	m.pools.Range(func(key interface{}, value interface{}) bool {
		databaseID, ok := key.(uint)
		if !ok {
			return true
		}

		connection, ok := value.(SQLConnection)
		if !ok {
			return true
		}

		ctx, cancel := context.WithTimeout(context.Background(), m.pingTimeout)
		pingErr := connection.PingContext(ctx)
		cancel()
		if pingErr != nil {
			_ = m.Close(context.Background(), databaseID)
		}

		return true
	})
}

func resolveInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func resolveDuration(value time.Duration, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}
