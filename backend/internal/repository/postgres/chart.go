package postgres

import (
	"context"
	"errors"
	"fmt"

	"superset/auth-service/internal/domain/chart"

	"gorm.io/gorm"
)

type ChartRepository struct{ db *gorm.DB }

func NewChartRepository(db *gorm.DB) *ChartRepository { return &ChartRepository{db: db} }

func (r *ChartRepository) CreateSlice(ctx context.Context, s *chart.Slice) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *ChartRepository) CreateSliceUser(ctx context.Context, su *chart.SliceUser) error {
	return r.db.WithContext(ctx).Create(su).Error
}

func (r *ChartRepository) GetSliceByID(ctx context.Context, id uint) (*chart.Slice, error) {
	var s chart.Slice
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting slice: %w", err)
	}
	return &s, nil
}

func (r *ChartRepository) UpdateSlice(ctx context.Context, s *chart.Slice) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *ChartRepository) DeleteSlice(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&chart.Slice{}, id).Error
}

func (r *ChartRepository) ListSlices(ctx context.Context, f *chart.SliceListFilter) ([]*chart.Slice, int64, error) {
	var total int64

	base := r.db.WithContext(ctx).
		Table("slices").
		Joins("LEFT JOIN slice_user su ON su.slice_id = slices.id")

	if f.VisibilityAll {
		// Admin — no restriction
	} else {
		base = base.Distinct("slices.*")
		if len(f.PermissionNames) > 0 {
			base = base.Where("su.user_id = ? OR slices.perm IN (?)", f.VisibilityUserID, f.PermissionNames)
		} else {
			base = base.Where("su.user_id = ?", f.VisibilityUserID)
		}
	}

	if f.DatasourceID != 0 {
		base = base.Where("slices.datasource_id = ?", fmt.Sprintf("%d", f.DatasourceID))
	}
	if f.DatasourceType != "" {
		base = base.Where("slices.datasource_type = ?", f.DatasourceType)
	}
	if f.VizType != "" {
		base = base.Where("slices.viz_type = ?", f.VizType)
	}
	if f.OwnerID != 0 {
		base = base.Where("slices.created_by_fk = ?", f.OwnerID)
	}
	if f.Certified != nil {
		if *f.Certified {
			base = base.Where("slices.certified_by <> ''")
		} else {
			base = base.Where("slices.certified_by = ''")
		}
	}
	if f.Q != "" {
		base = base.Where("(slices.slice_name ILIKE ? OR slices.description ILIKE ?)", "%"+f.Q+"%", "%"+f.Q+"%")
	}

	countQ := r.db.WithContext(ctx).Table("(?) AS filtered", base)
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting slices: %w", err)
	}

	off := (f.Page - 1) * f.PageSize
	var slices []*chart.Slice
	if err := base.Order("slices.last_saved_at DESC").
		Offset(off).Limit(f.PageSize).
		Find(&slices).Error; err != nil {
		return nil, 0, fmt.Errorf("listing slices: %w", err)
	}

	return slices, total, nil
}

func (r *ChartRepository) GetSliceDetail(ctx context.Context, id uint) (*chart.Slice, error) {
	var s chart.Slice
	if err := r.db.WithContext(ctx).
		First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting slice detail: %w", err)
	}
	return &s, nil
}

func (r *ChartRepository) DashboardCount(ctx context.Context, sliceID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&chart.DashboardSlice{}).
		Where("slice_id = ?", sliceID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting dashboards for slice %d: %w", sliceID, err)
	}
	return count, nil
}

func (r *ChartRepository) CreateDashboard(ctx context.Context, d *chart.Dashboard) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *ChartRepository) GetDashboardByID(ctx context.Context, id uint) (*chart.Dashboard, error) {
	var d chart.Dashboard
	return &d, r.db.WithContext(ctx).First(&d, id).Error
}

func (r *ChartRepository) GetDashboardBySlug(ctx context.Context, slug string) (*chart.Dashboard, error) {
	var d chart.Dashboard
	return &d, r.db.WithContext(ctx).Where("slug = ?", slug).First(&d).Error
}

func (r *ChartRepository) UpdateDashboard(ctx context.Context, d *chart.Dashboard) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *ChartRepository) DeleteDashboard(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&chart.Dashboard{}, id).Error
}

func (r *ChartRepository) ListDashboards(ctx context.Context, f *chart.DashboardListFilter) ([]*chart.Dashboard, int64, error) {
	var ds []*chart.Dashboard
	var total int64
	q := r.db.WithContext(ctx).Model(&chart.Dashboard{})
	if f.Q != "" {
		q = q.Where("dashboard_title ILIKE ?", "%"+f.Q+"%")
	}
	q.Count(&total)
	off := (f.Page - 1) * f.PageSize
	q.Order("changed_on DESC").Offset(off).Limit(f.PageSize).Find(&ds)
	return ds, total, nil
}

func (r *ChartRepository) AddSliceToDashboard(ctx context.Context, did, sid uint) error {
	return r.db.WithContext(ctx).Create(&chart.DashboardSlice{DashboardID: did, SliceID: sid}).Error
}

func (r *ChartRepository) RemoveSliceFromDashboard(ctx context.Context, did, sid uint) error {
	return r.db.WithContext(ctx).Where("dashboard_id=? AND slice_id=?", did, sid).Delete(&chart.DashboardSlice{}).Error
}

func (r *ChartRepository) ListDashboardSlices(ctx context.Context, did uint) ([]*chart.DashboardSlice, error) {
	var items []*chart.DashboardSlice
	return items, r.db.WithContext(ctx).Where("dashboard_id=?", did).Find(&items).Error
}

var _ chart.Repository = (*ChartRepository)(nil)
