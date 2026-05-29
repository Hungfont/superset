package chart

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	chartdomain "superset/auth-service/internal/domain/chart"
	datasetdomain "superset/auth-service/internal/domain/dataset"
	pkgerrors "superset/auth-service/internal/pkg/autherrors"
)

type DatasetRepo interface {
	GetDatasetByID(ctx context.Context, id uint) (*datasetdomain.Dataset, error)
}

type DatasetPermChecker interface {
	CanReadDataset(ctx context.Context, userID uint, dataset *datasetdomain.Dataset) (bool, error)
}

type noopPermChecker struct{}

func (noopPermChecker) CanReadDataset(_ context.Context, _ uint, _ *datasetdomain.Dataset) (bool, error) {
	return true, nil
}

type CreateChartInput struct {
	SliceName            string
	VizType              string
	DatasourceID         string
	DatasourceType       string
	Params               string
	QueryContext         string
	Description          string
	CacheTimeout         int
	CertifiedBy          string
	CertificationDetails string
}

type Service struct {
	chartRepo   chartdomain.Repository
	datasetRepo DatasetRepo
	permChecker DatasetPermChecker
}

func NewService(chartRepo chartdomain.Repository, datasetRepo DatasetRepo, permChecker DatasetPermChecker) *Service {
	if permChecker == nil {
		permChecker = noopPermChecker{}
	}
	return &Service{chartRepo: chartRepo, datasetRepo: datasetRepo, permChecker: permChecker}
}

func (s *Service) CreateChart(ctx context.Context, actorID uint, input CreateChartInput) (*chartdomain.Slice, error) {
	datasourceID, err := strconv.ParseUint(input.DatasourceID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid datasource_id", pkgerrors.ErrInvalidDataset)
	}

	dataset, err := s.datasetRepo.GetDatasetByID(ctx, uint(datasourceID))
	if err != nil {
		return nil, fmt.Errorf("loading dataset: %w", err)
	}
	if dataset == nil || dataset.ID == 0 {
		return nil, pkgerrors.ErrInvalidDataset
	}

	canRead, err := s.permChecker.CanReadDataset(ctx, actorID, dataset)
	if err != nil {
		return nil, fmt.Errorf("checking dataset permission: %w", err)
	}
	if !canRead {
		return nil, pkgerrors.ErrForbidden
	}

	if input.Params != "" && !json.Valid([]byte(input.Params)) {
		return nil, fmt.Errorf("invalid params JSON")
	}

	now := time.Now()
	slice := &chartdomain.Slice{
		SliceName:            input.SliceName,
		VizType:              input.VizType,
		DatasourceID:         input.DatasourceID,
		DatasourceType:       input.DatasourceType,
		DatasourceName:       dataset.Name,
		Params:               input.Params,
		QueryContext:         input.QueryContext,
		Description:          input.Description,
		CacheTimeout:         input.CacheTimeout,
		Perm:                 dataset.Perm,
		SchemaPerm:           dataset.SchemaPerm,
		CertifiedBy:          input.CertifiedBy,
		CertificationDetails: input.CertificationDetails,
		LastSavedAt:          now,
		LastSavedByFK:        actorID,
		CreatedByFK:          actorID,
		ChangedByFK:          actorID,
	}

	if err := s.chartRepo.CreateSlice(ctx, slice); err != nil {
		return nil, fmt.Errorf("creating slice: %w", err)
	}
	if err := s.chartRepo.CreateSliceUser(ctx, &chartdomain.SliceUser{SliceID: slice.ID, UserID: actorID}); err != nil {
		return nil, fmt.Errorf("creating slice_user: %w", err)
	}
	return slice, nil
}
