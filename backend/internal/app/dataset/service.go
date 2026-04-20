package dataset

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	domain "superset/auth-service/internal/domain/dataset"
	dbdomain "superset/auth-service/internal/domain/db"

	"github.com/google/uuid"
)

var (
	ErrSyncQueueRequired = errors.New("dataset sync queue is required")

	datasetListDefaultPage     = 1
	datasetListDefaultPageSize = 10
	datasetListMaxPageSize     = 100
)

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

var (
	selectPattern    = regexp.MustCompile(`(?i)^\s*SELECT\s`)
	semicolonPattern = regexp.MustCompile(`;`)
)

func (s *Service) CreateVirtualDataset(ctx context.Context, actorUserID uint, req domain.CreateVirtualDatasetRequest) (*domain.CreateVirtualDatasetResponse, error) {
	normalizedReq, err := normalizeVirtualCreateRequest(req)
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

	sql := strings.TrimSpace(normalizedReq.SQL)
	if !selectPattern.MatchString(sql) {
		return nil, domain.ErrSQLNotSelect
	}
	if semicolonPattern.MatchString(sql) {
		return nil, domain.ErrSQLSemicolon
	}

	exists, err := s.repo.ExistsVirtualDataset(ctx, normalizedReq.DatabaseID, normalizedReq.TableName)
	if err != nil {
		return nil, fmt.Errorf("checking virtual dataset duplicate: %w", err)
	}
	if exists {
		return nil, domain.ErrDatasetDuplicate
	}

	created := domain.Dataset{
		Name:        normalizedReq.TableName,
		DatabaseID:  normalizedReq.DatabaseID,
		SQL:        sql,
		Perm:       buildPhysicalDatasetPerm(database.DatabaseName, normalizedReq.TableName),
		CreatedByFK: actorUserID,
		ChangedByFK: actorUserID,
	}

	if err := s.repo.CreateVirtualDataset(ctx, &created); err != nil {
		if errors.Is(err, domain.ErrDatasetDuplicate) {
			return nil, domain.ErrDatasetDuplicate
		}
		return nil, fmt.Errorf("creating virtual dataset: %w", err)
	}

	if _, err := s.queue.EnqueueSyncColumns(ctx, created.ID); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrDatasetSyncEnqueue, err)
	}

	return &domain.CreateVirtualDatasetResponse{
		ID:             created.ID,
		TableName:       created.Name,
		BackgroundSync: true,
	}, nil
}

func normalizeVirtualCreateRequest(req domain.CreateVirtualDatasetRequest) (domain.CreateVirtualDatasetRequest, error) {
	databaseID := req.DatabaseID
	tableName := strings.TrimSpace(req.TableName)
	sql := strings.TrimSpace(req.SQL)

	if databaseID == 0 || tableName == "" || sql == "" {
		return domain.CreateVirtualDatasetRequest{}, domain.ErrInvalidDataset
	}

	return domain.CreateVirtualDatasetRequest{
		DatabaseID:  databaseID,
		TableName:  tableName,
		SQL:       sql,
		ValidateSQL: req.ValidateSQL,
	}, nil
}

func (s *Service) ListDatasets(ctx context.Context, actorUserID uint, query domain.DatasetListQuery) (*domain.DatasetListResult, error) {
	normalized := normalizeDatasetListQuery(query)

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	result, err := s.repo.ListDatasets(ctx, actorUserID, domain.DatasetListFilters{
		SearchQ:         normalized.Q,
		DatabaseID:       normalized.DatabaseID,
		Schema:           normalized.Schema,
		Type:             normalized.Type,
		Owner:            normalized.Owner,
		VisibilityScope:  visibilityScope,
		ActorUserID:      actorUserID,
		Page:            normalized.Page,
		PageSize:         normalized.PageSize,
		Offset:          (normalized.Page - 1) * normalized.PageSize,
		Limit:           normalized.PageSize,
		OrderBy:         normalized.OrderBy,
	})
	if err != nil {
		return nil, fmt.Errorf("listing datasets: %w", err)
	}

	return result, nil
}

func (s *Service) GetDatasetDetail(ctx context.Context, actorUserID uint, id uint) (*domain.DatasetDetail, error) {
	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	detail, err := s.repo.GetDatasetDetail(ctx, id)
	if err != nil {
		return nil, err
	}

	if detail == nil || detail.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	canView, err := s.canViewDataset(ctx, actorUserID, detail, visibilityScope)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, domain.ErrDatasetNotFound
	}

	return detail, nil
}

func (s *Service) resolveVisibilityScope(ctx context.Context, actorUserID uint) (domain.DatasetVisibilityScope, error) {
	roleNames, err := s.databaseRepo.GetRoleNamesByUser(ctx, actorUserID)
	if err != nil {
		return "", fmt.Errorf("loading actor role names: %w", err)
	}

	for _, roleName := range roleNames {
		value := strings.ToLower(strings.TrimSpace(roleName))
		if value == "admin" {
			return domain.VisibilityScopeAdmin, nil
		}
	}

	for _, roleName := range roleNames {
		value := strings.ToLower(strings.TrimSpace(roleName))
		if value == "alpha" {
			return domain.VisibilityScopeAlpha, nil
		}
	}

	return domain.VisibilityScopeGamma, nil
}

func (s *Service) canViewDataset(ctx context.Context, actorUserID uint, detail *domain.DatasetDetail, scope domain.DatasetVisibilityScope) (bool, error) {
	switch scope {
	case domain.VisibilityScopeAdmin, domain.VisibilityScopeAlpha:
		return true, nil
	case domain.VisibilityScopeGamma:
		return detail.CreatedByFK == actorUserID, nil
	default:
		return detail.CreatedByFK == actorUserID, nil
	}
}

func normalizeDatasetListQuery(query domain.DatasetListQuery) domain.DatasetListQuery {
	page := query.Page
	if page < 1 {
		page = datasetListDefaultPage
	}

	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = datasetListDefaultPageSize
	}
	if pageSize > datasetListMaxPageSize {
		pageSize = datasetListMaxPageSize
	}

	orderBy := strings.TrimSpace(query.OrderBy)
	if orderBy == "" {
		orderBy = "changed_on desc"
	}

	return domain.DatasetListQuery{
		Q:         strings.TrimSpace(query.Q),
		DatabaseID: query.DatabaseID,
		Schema:    strings.TrimSpace(query.Schema),
		Type:      strings.TrimSpace(query.Type),
		Owner:     query.Owner,
		Page:      page,
		PageSize:  pageSize,
		OrderBy:   orderBy,
	}
}

func (s *Service) UpdateDatasetMetadata(ctx context.Context, actorUserID uint, id uint, req domain.UpdateDatasetMetadataRequest) (*domain.UpdateDatasetMetadataResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canEdit, err := s.canEditDataset(ctx, actorUserID, dataset, visibilityScope, req.IsFeatured)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, domain.ErrForbidden
	}

	if req.MainDttmCol != "" {
		column, err := s.repo.GetColumnByName(ctx, id, req.MainDttmCol)
		if err != nil {
			return nil, fmt.Errorf("validating main_dttm_col: %w", err)
		}
		if column == nil || !column.IsDateTime {
			return nil, domain.ErrInvalidMainDttmCol
		}
	}

	backgroundSync := false
	if req.SQL != "" {
		sql := strings.TrimSpace(req.SQL)
		if !selectPattern.MatchString(sql) {
			return nil, domain.ErrSQLNotSelect
		}
		if semicolonPattern.MatchString(sql) {
			return nil, domain.ErrSQLSemicolon
		}
		backgroundSync = true
	}

	if err := s.repo.UpdateDatasetMetadata(ctx, id, req); err != nil {
		return nil, fmt.Errorf("updating dataset metadata: %w", err)
	}

	if backgroundSync {
		if _, err := s.queue.EnqueueSyncColumns(ctx, id); err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrDatasetSyncEnqueue, err)
		}
	}

	return &domain.UpdateDatasetMetadataResponse{
		ID:             id,
		TableName:      dataset.Name,
		BackgroundSync: backgroundSync,
	}, nil
}

func (s *Service) canEditDataset(ctx context.Context, actorUserID uint, dataset *domain.Dataset, scope domain.DatasetVisibilityScope, setFeatured bool) (bool, error) {
	if setFeatured {
		roleNames, err := s.databaseRepo.GetRoleNamesByUser(ctx, actorUserID)
		if err != nil {
			return false, fmt.Errorf("loading actor role names: %w", err)
		}
		isAdmin := false
		for _, roleName := range roleNames {
			if strings.ToLower(strings.TrimSpace(roleName)) == "admin" {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			return false, nil
		}
	}

	switch scope {
	case domain.VisibilityScopeAdmin, domain.VisibilityScopeAlpha:
		return true, nil
	case domain.VisibilityScopeGamma:
		return dataset.CreatedByFK == actorUserID, nil
	default:
		return dataset.CreatedByFK == actorUserID, nil
	}
}

func (s *Service) UpdateColumn(ctx context.Context, actorUserID uint, datasetID uint, columnID uint, req domain.UpdateColumnRequest) (*domain.UpdateColumnResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canEdit, err := s.canEditDataset(ctx, actorUserID, dataset, visibilityScope, false)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, domain.ErrForbidden
	}

	column, err := s.repo.GetColumnByID(ctx, columnID)
	if err != nil {
		return nil, fmt.Errorf("getting column: %w", err)
	}
	if column == nil || column.ID == 0 {
		return nil, domain.ErrColumnNotFound
	}

	if column.TableID != datasetID {
		return nil, domain.ErrColumnNotFound
	}

	if err := s.validateColumnRequest(req); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateColumn(ctx, columnID, req); err != nil {
		return nil, fmt.Errorf("updating column: %w", err)
	}

	return &domain.UpdateColumnResponse{ID: columnID}, nil
}

func (s *Service) BulkUpdateColumns(ctx context.Context, actorUserID uint, datasetID uint, req domain.BulkUpdateColumnRequest) (*domain.BulkUpdateColumnResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canEdit, err := s.canEditDataset(ctx, actorUserID, dataset, visibilityScope, false)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, domain.ErrForbidden
	}

	for i := range req.Columns {
		if err := s.validateColumnRequest(req.Columns[i]); err != nil {
			return nil, err
		}
	}

	updatedIDs := make([]domain.UpdateColumnRequest, 0, len(req.Columns))
	for _, col := range req.Columns {
		column, err := s.repo.GetColumnByID(ctx, col.ID)
		if err != nil {
			return nil, fmt.Errorf("getting column %d: %w", col.ID, err)
		}
		if column == nil || column.ID == 0 {
			return nil, domain.ErrColumnNotFound
		}
		if column.TableID != datasetID {
			return nil, domain.ErrColumnNotFound
		}
		updatedIDs = append(updatedIDs, col)
	}

	if err := s.repo.BulkUpdateColumns(ctx, updatedIDs); err != nil {
		return nil, fmt.Errorf("bulk updating columns: %w", err)
	}

	return &domain.BulkUpdateColumnResponse{UpdatedCount: len(updatedIDs)}, nil
}

func (s *Service) validateColumnRequest(req domain.UpdateColumnRequest) error {
	if req.Expression != "" {
		if !isValidSQLExpression(req.Expression) {
			return domain.ErrInvalidExpression
		}
	}

	if req.PythonDateFormat != "" {
		if !isValidPythonDateFormat(req.PythonDateFormat) {
			return domain.ErrInvalidDateFormat
		}
	}

	return nil
}

var pythonDateFormatPattern = regexp.MustCompile(`^(\s*%[YymdHMScDbBApzZf_j]\s*)+$`)

func isValidPythonDateFormat(format string) bool {
	if format == "" {
		return true
	}
	return pythonDateFormatPattern.MatchString(format)
}

var sqlExprKeywords = regexp.MustCompile(`(?i)^(\s*[[:alnum:]_]+\s*|COUNT|SUM|AVG|MIN|MAX|COALESCE|IFNULL|NULLIF|CASE|WHEN|THEN|ELSE|END|\+|\-|\*|\/|\\|<|>|=|!|AND|OR|NOT|IN|LIKE|BETWEEN|IS|NULL|\(|\)|\.)+$`)

func isValidSQLExpression(expr string) bool {
	if expr == "" {
		return true
	}
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return false
	}
	return sqlExprKeywords.MatchString(trimmed)
}

var aggregateFuncs = map[string]bool{
	"sum":   true,
	"count": true,
	"avg":   true,
	"max":   true,
	"min":   true,
	"stddev": true,
	"variance": true,
}

func containsAggregateFunction(expr string) bool {
	lower := strings.ToLower(expr)
	for funcName := range aggregateFuncs {
		if strings.Contains(lower, funcName+"(") {
			return true
		}
	}
	return false
}

func (s *Service) GetMetrics(ctx context.Context, actorUserID uint, datasetID uint) ([]domain.SqlMetric, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	metrics, err := s.repo.GetMetricsByTableID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting metrics: %w", err)
	}

	return metrics, nil
}

func (s *Service) CreateMetric(ctx context.Context, actorUserID uint, datasetID uint, req domain.CreateMetricRequest) (*domain.CreateMetricResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canEdit, err := s.canEditDataset(ctx, actorUserID, dataset, visibilityScope, false)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, domain.ErrForbidden
	}

	normalizedName := normalizeMetricName(req.MetricName)
	if normalizedName == "" {
		return nil, domain.ErrInvalidDataset
	}

	exists, err := s.repo.MetricNameExists(ctx, datasetID, normalizedName, 0)
	if err != nil {
		return nil, fmt.Errorf("checking metric name: %w", err)
	}
	if exists {
		return nil, domain.ErrMetricDuplicate
	}

	if !containsAggregateFunction(req.Expression) {
		return nil, domain.ErrNoAggregateFunction
	}

	metric := domain.SqlMetric{
		TableID:              datasetID,
		MetricName:          normalizedName,
		VerboseName:         req.VerboseName,
		MetricType:          req.MetricType,
		Expression:          req.Expression,
		D3Format:            req.D3Format,
		WarningText:         req.WarningText,
		IsRestricted:       req.IsRestricted,
		CertifiedBy:         req.CertifiedBy,
		CertificationDetails: req.CertificationDetails,
	}

	if err := s.repo.CreateMetric(ctx, &metric); err != nil {
		if errors.Is(err, domain.ErrMetricDuplicate) {
			return nil, domain.ErrMetricDuplicate
		}
		return nil, fmt.Errorf("creating metric: %w", err)
	}

	return &domain.CreateMetricResponse{ID: metric.ID}, nil
}

func (s *Service) UpdateMetric(ctx context.Context, actorUserID uint, datasetID uint, metricID uint, req domain.UpdateMetricRequest) (*domain.UpdateMetricResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canEdit, err := s.canEditDataset(ctx, actorUserID, dataset, visibilityScope, false)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, domain.ErrForbidden
	}

	metric, err := s.repo.GetMetricByID(ctx, metricID)
	if err != nil {
		return nil, fmt.Errorf("getting metric: %w", err)
	}
	if metric == nil || metric.ID == 0 {
		return nil, domain.ErrMetricNotFound
	}

	if metric.TableID != datasetID {
		return nil, domain.ErrMetricNotFound
	}

	if req.MetricName != "" {
		normalizedName := normalizeMetricName(req.MetricName)
		if normalizedName == "" {
			return nil, domain.ErrInvalidDataset
		}

		exists, err := s.repo.MetricNameExists(ctx, datasetID, normalizedName, metricID)
		if err != nil {
			return nil, fmt.Errorf("checking metric name: %w", err)
		}
		if exists {
			return nil, domain.ErrMetricDuplicate
		}
	}

	if req.Expression != "" && !containsAggregateFunction(req.Expression) {
		return nil, domain.ErrNoAggregateFunction
	}

	if err := s.repo.UpdateMetric(ctx, metricID, req); err != nil {
		return nil, fmt.Errorf("updating metric: %w", err)
	}

	return &domain.UpdateMetricResponse{ID: metricID}, nil
}

func (s *Service) DeleteMetric(ctx context.Context, actorUserID uint, datasetID uint, metricID uint) (*domain.DeleteMetricResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canEdit, err := s.canEditDataset(ctx, actorUserID, dataset, visibilityScope, false)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, domain.ErrForbidden
	}

	metric, err := s.repo.GetMetricByID(ctx, metricID)
	if err != nil {
		return nil, fmt.Errorf("getting metric: %w", err)
	}
	if metric == nil || metric.ID == 0 {
		return nil, domain.ErrMetricNotFound
	}

	if metric.TableID != datasetID {
		return nil, domain.ErrMetricNotFound
	}

	warnings := []string{}

	if err := s.repo.DeleteMetric(ctx, metricID); err != nil {
		return nil, fmt.Errorf("deleting metric: %w", err)
	}

	return &domain.DeleteMetricResponse{Warnings: warnings}, nil
}

func (s *Service) BulkUpdateMetrics(ctx context.Context, actorUserID uint, datasetID uint, req domain.BulkUpdateMetricsRequest) (*domain.BulkUpdateMetricsResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canEdit, err := s.canEditDataset(ctx, actorUserID, dataset, visibilityScope, false)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, domain.ErrForbidden
	}

	seenNames := make(map[string]bool)
	metrics := make([]domain.SqlMetric, 0, len(req.Metrics))

	for _, m := range req.Metrics {
		normalizedName := normalizeMetricName(m.MetricName)
		if normalizedName == "" {
			return nil, domain.ErrInvalidDataset
		}

		if seenNames[normalizedName] {
			return nil, domain.ErrMetricDuplicate
		}
		seenNames[normalizedName] = true

		if !containsAggregateFunction(m.Expression) {
			return nil, domain.ErrNoAggregateFunction
		}

		var metricID uint
		if m.ID != nil {
			metricID = *m.ID
		}

		if metricID > 0 {
			exists, err := s.repo.MetricNameExists(ctx, datasetID, normalizedName, metricID)
			if err != nil {
				return nil, fmt.Errorf("checking metric name: %w", err)
			}
			if exists {
				return nil, domain.ErrMetricDuplicate
			}
		}

		metric := domain.SqlMetric{
			ID:                   metricID,
			TableID:              datasetID,
			MetricName:          normalizedName,
			VerboseName:         m.VerboseName,
			MetricType:          m.MetricType,
			Expression:          m.Expression,
			D3Format:            m.D3Format,
			WarningText:         m.WarningText,
			IsRestricted:       m.IsRestricted,
			CertifiedBy:         m.CertifiedBy,
			CertificationDetails: m.CertificationDetails,
		}
		metrics = append(metrics, metric)
	}

	if err := s.repo.BulkReplaceMetrics(ctx, datasetID, metrics); err != nil {
		return nil, fmt.Errorf("bulk updating metrics: %w", err)
	}

	return &domain.BulkUpdateMetricsResponse{UpdatedCount: len(metrics)}, nil
}

var metricNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func normalizeMetricName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) < 3 {
		return ""
	}
	if !metricNamePattern.MatchString(trimmed) {
		return ""
	}
	return trimmed
}

func (s *Service) DeleteDataset(ctx context.Context, actorUserID uint, id uint, req domain.DeleteDatasetRequest) (*domain.DeleteDatasetResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canDelete, err := s.canDeleteDataset(ctx, actorUserID, dataset, visibilityScope)
	if err != nil {
		return nil, err
	}
	if !canDelete {
		return nil, domain.ErrForbidden
	}

	chartCount, err := s.repo.CountChartsByDatasetID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("counting charts: %w", err)
	}

	isAdmin := false
	if visibilityScope == domain.VisibilityScopeAdmin {
		isAdmin = true
	}

	if chartCount > 0 && (!req.Force || !isAdmin) {
		return nil, domain.ErrDatasetReferencedByCharts
	}

	if err := s.repo.DeleteDataset(ctx, id); err != nil {
		return nil, fmt.Errorf("deleting dataset: %w", err)
	}

	return &domain.DeleteDatasetResponse{
		Deleted:          true,
		ChartsDeleted: int(chartCount),
	}, nil
}

func (s *Service) canDeleteDataset(ctx context.Context, actorUserID uint, dataset *domain.Dataset, scope domain.DatasetVisibilityScope) (bool, error) {
	switch scope {
	case domain.VisibilityScopeAdmin, domain.VisibilityScopeAlpha:
		return true, nil
	case domain.VisibilityScopeGamma:
		return dataset.CreatedByFK == actorUserID, nil
	default:
		return dataset.CreatedByFK == actorUserID, nil
	}
}

func (s *Service) RefreshDataset(ctx context.Context, actorUserID uint, datasetID uint) (*domain.RefreshDatasetResponse, error) {
	dataset, err := s.repo.GetDatasetByID(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, domain.ErrDatasetNotFound
	}

	visibilityScope, err := s.resolveVisibilityScope(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	canEdit, err := s.canEditDataset(ctx, actorUserID, dataset, visibilityScope, false)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, domain.ErrForbidden
	}

	jobID, err := s.queue.EnqueueSyncColumns(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrDatasetSyncEnqueue, err)
	}

	return &domain.RefreshDatasetResponse{
		JobID:          jobID,
		BackgroundSync: true,
	}, nil
}
