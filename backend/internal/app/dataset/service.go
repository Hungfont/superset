package dataset

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/dataset"
	dbdomain "superset/auth-service/internal/domain/db"

	"github.com/google/uuid"
)

var ErrSyncQueueRequired = errors.New("dataset sync queue is required")

// SyncQueue enqueues background column sync jobs.
type SyncQueue interface {
	EnqueueSyncColumns(ctx context.Context, datasetID uint) (string, error)
}
type noopSyncQueue struct{}

func (noopSyncQueue) EnqueueSyncColumns(_ context.Context, _ uint) (string, error) {
	return uuid.NewString(), nil
}

// databaseLookupRepository provides only db lookups required by dataset service.
type databaseLookupRepository interface {
	GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error)
	GetDatabaseByID(ctx context.Context, databaseID uint) (*dbdomain.Database, error)
}

// Service handles dataset lifecycle use cases.
type Service struct {
	repo         domain.Repository
	databaseRepo databaseLookupRepository
	queue        SyncQueue
}

func NewService(repo domain.Repository, databaseRepo databaseLookupRepository, queue SyncQueue) (*Service, error) {
	if queue == nil {
		queue = noopSyncQueue{}
	}

	return &Service{repo: repo, databaseRepo: databaseRepo, queue: queue}, nil
}

func (s *Service) CreatePhysicalDataset(ctx context.Context, actorUserID uint, req domain.CreatePhysicalDatasetRequest) (*domain.CreatePhysicalDatasetResponse, error) {
	normalizedReq, err := normalizeCreateRequest(req)
	if err != nil {
		return nil, err
	}

	allowed, err := s.allowPhysicalDatasetCreation(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, domain.ErrForbidden
	}

	database, err := s.databaseRepo.GetDatabaseByID(ctx, normalizedReq.DatabaseID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidDatabase) || errors.Is(err, dbdomain.ErrDatabaseNotFound) {
			return nil, domain.ErrInvalidDatabase
		}
		return nil, fmt.Errorf("loading database by id: %w", err)
	}
	if database == nil || strings.TrimSpace(database.DatabaseName) == "" {
		return nil, domain.ErrInvalidDatabase
	}

	exists, err := s.repo.ExistsPhysicalDataset(ctx, normalizedReq.DatabaseID, normalizedReq.Schema, normalizedReq.TableName)
	if err != nil {
		return nil, fmt.Errorf("checking dataset duplicate: %w", err)
	}
	if exists {
		return nil, domain.ErrDatasetDuplicate
	}

	created := domain.Dataset{
		Name:        normalizedReq.TableName,
		Schema:      normalizedReq.Schema,
		DatabaseID:  normalizedReq.DatabaseID,
		Perm:        buildPhysicalDatasetPerm(database.DatabaseName, normalizedReq.TableName),
		CreatedByFK: actorUserID,
		ChangedByFK: actorUserID,
	}

	if err := s.repo.CreatePhysicalDataset(ctx, &created); err != nil {
		if errors.Is(err, domain.ErrDatasetDuplicate) {
			return nil, domain.ErrDatasetDuplicate
		}
		return nil, fmt.Errorf("creating physical dataset: %w", err)
	}

	if _, err := s.queue.EnqueueSyncColumns(ctx, created.ID); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrDatasetSyncEnqueue, err)
	}

	return &domain.CreatePhysicalDatasetResponse{
		ID:             created.ID,
		TableName:      created.Name,
		BackgroundSync: true,
	}, nil
}

func (s *Service) allowPhysicalDatasetCreation(ctx context.Context, actorUserID uint) (bool, error) {
	roleNames, err := s.databaseRepo.GetRoleNamesByUser(ctx, actorUserID)
	if err != nil {
		return false, fmt.Errorf("loading actor role names: %w", err)
	}

	for _, roleName := range roleNames {
		value := strings.ToLower(strings.TrimSpace(roleName))
		if value == "admin" || value == "alpha" {
			return true, nil
		}
	}

	return false, nil
}

func normalizeCreateRequest(req domain.CreatePhysicalDatasetRequest) (domain.CreatePhysicalDatasetRequest, error) {
	databaseID := req.DatabaseID
	schema := strings.TrimSpace(req.Schema)
	tableName := strings.TrimSpace(req.TableName)

	if databaseID == 0 || tableName == "" {
		return domain.CreatePhysicalDatasetRequest{}, domain.ErrInvalidDataset
	}

	return domain.CreatePhysicalDatasetRequest{
		DatabaseID: databaseID,
		Schema:     schema,
		TableName:  tableName,
	}, nil
}

func buildPhysicalDatasetPerm(databaseName string, tableName string) string {
	return fmt.Sprintf("[can_read].[%s].[%s]", strings.TrimSpace(databaseName), strings.TrimSpace(tableName))
}
