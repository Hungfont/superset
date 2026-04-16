package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	domain "superset/auth-service/internal/domain/db"
)

// DatabaseConnectionTester validates a database connection before persistence.
type DatabaseConnectionTester interface {
	TestConnection(ctx context.Context, sqlalchemyURI string) error
}

// DatabaseAuditLogger emits asynchronous audit events.
type DatabaseAuditLogger interface {
	LogDatabaseCreated(ctx context.Context, databaseID uint)
}

type defaultDatabaseConnectionTester struct{}

type noopDatabaseAuditLogger struct{}

func (defaultDatabaseConnectionTester) TestConnection(_ context.Context, sqlalchemyURI string) error {
	if _, err := parseSQLAlchemyURI(sqlalchemyURI); err != nil {
		return err
	}
	return nil
}

func (noopDatabaseAuditLogger) LogDatabaseCreated(_ context.Context, _ uint) {}

// DatabaseService handles admin database connection management.
type DatabaseService struct {
	repo          domain.DatabaseRepository
	tester        DatabaseConnectionTester
	auditLogger   DatabaseAuditLogger
	encryptionKey []byte
}

func NewDatabaseService(repo domain.DatabaseRepository, tester DatabaseConnectionTester, auditLogger DatabaseAuditLogger, encryptionKey string) (*DatabaseService, error) {
	parsedKey, err := parseDatabaseEncryptionKey(encryptionKey)
	if err != nil {
		return nil, err
	}

	resolvedTester := tester
	if resolvedTester == nil {
		resolvedTester = defaultDatabaseConnectionTester{}
	}

	resolvedAuditLogger := auditLogger
	if resolvedAuditLogger == nil {
		resolvedAuditLogger = noopDatabaseAuditLogger{}
	}

	return &DatabaseService{
		repo:          repo,
		tester:        resolvedTester,
		auditLogger:   resolvedAuditLogger,
		encryptionKey: parsedKey,
	}, nil
}

func (s *DatabaseService) CreateDatabase(ctx context.Context, actorUserID uint, req domain.CreateDatabaseRequest) (*domain.DatabaseDetail, error) {
	if err := s.ensureAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	normalizedReq, strictTest, err := normalizeCreateDatabaseRequest(req)
	if err != nil {
		return nil, err
	}

	exists, err := s.repo.DatabaseNameExists(ctx, normalizedReq.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("checking duplicate database name: %w", err)
	}
	if exists {
		return nil, domain.ErrDatabaseNameExists
	}

	encryptedURI, err := encryptSQLAlchemyURIPassword(normalizedReq.SQLAlchemyURI, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	if strictTest {
		if err := s.tester.TestConnection(ctx, normalizedReq.SQLAlchemyURI); err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrDatabaseConnectionTestFailed, err)
		}
	}

	database := domain.Database{
		DatabaseName:    normalizedReq.DatabaseName,
		SQLAlchemyURI:   encryptedURI,
		AllowDML:        normalizedReq.AllowDML,
		ExposeInSQLLab:  normalizedReq.ExposeInSQLLab,
		AllowRunAsync:   normalizedReq.AllowRunAsync,
		AllowFileUpload: normalizedReq.AllowFileUpload,
	}

	if err := s.repo.CreateDatabase(ctx, &database); err != nil {
		if errors.Is(err, domain.ErrDatabaseNameExists) {
			return nil, domain.ErrDatabaseNameExists
		}
		return nil, fmt.Errorf("creating database: %w", err)
	}

	maskedURI, err := maskSQLAlchemyURI(normalizedReq.SQLAlchemyURI)
	if err != nil {
		return nil, err
	}

	go s.auditLogger.LogDatabaseCreated(context.Background(), database.ID)

	return &domain.DatabaseDetail{
		ID:              database.ID,
		DatabaseName:    database.DatabaseName,
		SQLAlchemyURI:   maskedURI,
		AllowDML:        database.AllowDML,
		ExposeInSQLLab:  database.ExposeInSQLLab,
		AllowRunAsync:   database.AllowRunAsync,
		AllowFileUpload: database.AllowFileUpload,
	}, nil
}

func (s *DatabaseService) ensureAdmin(ctx context.Context, actorUserID uint) error {
	isAdmin, err := s.repo.IsAdmin(ctx, actorUserID)
	if err != nil {
		return fmt.Errorf("checking admin role: %w", err)
	}
	if !isAdmin {
		return domain.ErrForbidden
	}
	return nil
}

func normalizeCreateDatabaseRequest(req domain.CreateDatabaseRequest) (domain.CreateDatabaseRequest, bool, error) {
	databaseName := strings.TrimSpace(req.DatabaseName)
	sqlalchemyURI := strings.TrimSpace(req.SQLAlchemyURI)
	if databaseName == "" || sqlalchemyURI == "" {
		return domain.CreateDatabaseRequest{}, false, domain.ErrInvalidDatabase
	}

	if _, err := parseSQLAlchemyURI(sqlalchemyURI); err != nil {
		return domain.CreateDatabaseRequest{}, false, err
	}

	strictTest := true
	if req.StrictTest != nil {
		strictTest = *req.StrictTest
	}

	return domain.CreateDatabaseRequest{
		DatabaseName:    databaseName,
		SQLAlchemyURI:   sqlalchemyURI,
		AllowDML:        req.AllowDML,
		ExposeInSQLLab:  req.ExposeInSQLLab,
		AllowRunAsync:   req.AllowRunAsync,
		AllowFileUpload: req.AllowFileUpload,
		StrictTest:      req.StrictTest,
	}, strictTest, nil
}

func parseDatabaseEncryptionKey(rawKey string) ([]byte, error) {
	trimmed := strings.TrimSpace(rawKey)
	if trimmed == "" {
		return nil, domain.ErrDatabaseCredentialEncryption
	}

	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	if len(trimmed) == 32 {
		return []byte(trimmed), nil
	}

	return nil, domain.ErrDatabaseCredentialEncryption
}

func parseSQLAlchemyURI(sqlalchemyURI string) (*url.URL, error) {
	parsedURI, err := url.Parse(sqlalchemyURI)
	if err != nil {
		return nil, domain.ErrInvalidDatabaseURI
	}
	if parsedURI.Scheme == "" || parsedURI.Host == "" {
		return nil, domain.ErrInvalidDatabaseURI
	}
	return parsedURI, nil
}

func encryptSQLAlchemyURIPassword(sqlalchemyURI string, encryptionKey []byte) (string, error) {
	parsedURI, err := parseSQLAlchemyURI(sqlalchemyURI)
	if err != nil {
		return "", err
	}

	if parsedURI.User == nil {
		return parsedURI.String(), nil
	}

	username := parsedURI.User.Username()
	password, hasPassword := parsedURI.User.Password()
	if !hasPassword || password == "" {
		return parsedURI.String(), nil
	}

	encryptedPassword, err := encryptField(password, encryptionKey)
	if err != nil {
		return "", domain.ErrDatabaseCredentialEncryption
	}

	parsedURI.User = url.UserPassword(username, encryptedPassword)
	return parsedURI.String(), nil
}

func maskSQLAlchemyURI(sqlalchemyURI string) (string, error) {
	parsedURI, err := parseSQLAlchemyURI(sqlalchemyURI)
	if err != nil {
		return "", err
	}

	if parsedURI.User == nil {
		return parsedURI.String(), nil
	}

	username := parsedURI.User.Username()
	_, hasPassword := parsedURI.User.Password()
	if !hasPassword {
		return parsedURI.String(), nil
	}

	parsedURI.User = url.UserPassword(username, "***")
	maskedURI := parsedURI.String()
	return strings.Replace(maskedURI, "%2A%2A%2A", "***", 1), nil
}

func encryptField(plainText string, encryptionKey []byte) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nil, nonce, []byte(plainText), nil)
	combined := append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(combined), nil
}
