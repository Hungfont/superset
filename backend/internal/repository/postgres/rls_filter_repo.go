package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	domain "superset/auth-service/internal/domain/auth"
	datasetdomain "superset/auth-service/internal/domain/dataset"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type rlsFilterRepo struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewRLSFilterRepository(db *gorm.DB, rdb *redis.Client) domain.RLSFilterRepository {
	return &rlsFilterRepo{db: db, rdb: rdb}
}

func (r *rlsFilterRepo) List(ctx context.Context, params domain.RLSFilterListParams) ([]domain.RLSFilterResponse, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	query := r.db.WithContext(ctx).Model(&domain.RLSFilter{})

	if params.Q != "" {
		query = query.Where("name ILIKE ?", "%"+params.Q+"%")
	}
	if params.FilterType != "" {
		query = query.Where("filter_type = ?", params.FilterType)
	}
	if params.RoleID > 0 {
		query = query.Joins("JOIN rls_filter_roles rfr ON rfr.rls_id = row_level_security_filters.id").
			Where("rfr.role_id = ?", params.RoleID)
	}
	if params.DatasourceID > 0 {
		query = query.Joins("JOIN rls_filter_tables rft ON rft.rls_id = row_level_security_filters.id").
			Where("rft.datasource_id = ?", params.DatasourceID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting RLS filters: %w", err)
	}

	var filters []domain.RLSFilter
	if err := query.Preload("Roles").Preload("Tables").
		Order("changed_on DESC").
		Offset(offset).Limit(params.PageSize).
		Find(&filters).Error; err != nil {
		return nil, 0, fmt.Errorf("listing RLS filters: %w", err)
	}

	responses := make([]domain.RLSFilterResponse, len(filters))
	for i, f := range filters {
		roles := make([]domain.Role, len(f.Roles))
		for j, role := range f.Roles {
			roles[j] = domain.Role{ID: role.ID, Name: role.Name}
		}

		tables := make([]domain.RLSFilterTableInfo, len(f.Tables))
		for j, t := range f.Tables {
			tables[j] = domain.RLSFilterTableInfo{
				DatasourceID:   t.DatasourceID,
				DatasourceType: t.DatasourceType,
				TableName:      t.Table,
				DatabaseName:   t.DbName,
			}
		}
		responses[i] = domain.RLSFilterResponse{
			ID:          f.ID,
			Name:        f.Name,
			FilterType:  string(f.FilterType),
			Clause:      f.Clause,
			GroupKey:    f.GroupKey,
			Description: f.Description,
			Roles:       roles,
			Tables:      tables,
			CreatedBy:   f.CreatedByFK,
			CreatedOn:   f.CreatedOn,
			ChangedOn:   f.ChangedOn,
		}
	}

	pages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))
	if pages == 0 {
		pages = 1
	}

	return responses, total, nil
}

func (r *rlsFilterRepo) GetByID(ctx context.Context, id uint) (*domain.RLSFilter, error) {
	var filter domain.RLSFilter
	if err := r.db.WithContext(ctx).
		Preload("Roles").
		Preload("Tables").
		First(&filter, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting RLS filter: %w", err)
	}
	return &filter, nil
}

func (r *rlsFilterRepo) Create(ctx context.Context, actorUserID uint, req domain.CreateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("starting transaction: %w", tx.Error)
	}

	filter := domain.RLSFilter{
		Name:        req.Name,
		FilterType:  domain.RLSFilterType(req.FilterType),
		Clause:      req.Clause,
		GroupKey:    req.GroupKey,
		Description: req.Description,
		CreatedByFK: actorUserID,
		ChangedByFK: actorUserID,
	}

	if err := tx.Session(&gorm.Session{FullSaveAssociations: true}).Create(&filter).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, fmt.Errorf("filter name already exists")
		}
		return nil, fmt.Errorf("creating RLS filter: %w", err)
	}

	var roleIDs []uint
	for _, rid := range req.RoleIDs {
		roleIDs = append(roleIDs, rid)
	}
	if len(roleIDs) > 0 {
		var roles []domain.Role
		if err := tx.Find(&roles, roleIDs).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("finding roles: %w", err)
		}
		filter.Roles = roles
	}

	var tableIDs []uint
	for _, tid := range req.TableIDs {
		tableIDs = append(tableIDs, tid)
	}
	if len(tableIDs) > 0 {
		var datasets []datasetdomain.Dataset
		if err := tx.Find(&datasets, tableIDs).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("finding datasets: %w", err)
		}
		for _, ds := range datasets {
			jt := domain.FromDataset(&ds, filter.ID)
			if err := tx.Create(&jt).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("creating filter table junction: %w", err)
			}
		}
	}

	if err := r.bustCache(ctx, tx, filter.ID); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("busting cache: %w", err)
	}

	oldJSON := "null"
	newJSON, _ := json.Marshal(filter)
	audit := domain.RLSAuditLog{
		FilterID:   filter.ID,
		FilterName: filter.Name,
		EventType:  domain.RLSAuditEventFilterCreated,
		OldValue:   oldJSON,
		NewValue:   string(newJSON),
		ChangedBy:  actorUserID,
		IPAddress:  "",
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("creating audit log: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	var result domain.RLSFilter
	if err := r.db.Preload("Roles").Preload("Tables").First(&result, filter.ID).Error; err != nil {
		return nil, fmt.Errorf("refetching filter: %w", err)
	}

	return toResponse(&result), nil
}

func (r *rlsFilterRepo) Update(ctx context.Context, actorUserID uint, id uint, req domain.UpdateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("starting transaction: %w", tx.Error)
	}

	var oldFilter domain.RLSFilter
	if err := tx.Preload("Roles").Preload("Tables").First(&oldFilter, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("filter not found")
		}
		return nil, fmt.Errorf("finding filter: %w", err)
	}

	updates := map[string]interface{}{
		"changed_by_fk": actorUserID,
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.FilterType != "" {
		updates["filter_type"] = req.FilterType
	}
	if req.Clause != "" {
		updates["clause"] = req.Clause
	}
	if req.GroupKey != "" || req.GroupKey == "" {
		updates["group_key"] = req.GroupKey
	}
	if req.Description != "" || req.Description == "" {
		updates["description"] = req.Description
	}

	if err := tx.Model(&domain.RLSFilter{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, fmt.Errorf("filter name already exists")
		}
		return nil, fmt.Errorf("updating RLS filter: %w", err)
	}

	if req.RoleIDs != nil {
		if err := tx.Where("rls_id = ?", id).Delete(&domain.RLSFilterRoleJunction{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("deleting old roles: %w", err)
		}
		for _, rid := range req.RoleIDs {
			jt := domain.RLSFilterRoleJunction{RLSID: id, RoleID: rid}
			if err := tx.Create(&jt).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("creating role junction: %w", err)
			}
		}
	}

	if req.TableIDs != nil {
		if err := tx.Where("rls_id = ?", id).Delete(&domain.RLSFilterTableJunction{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("deleting old tables: %w", err)
		}
		var datasets []datasetdomain.Dataset
		if err := tx.Find(&datasets, req.TableIDs).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("finding datasets: %w", err)
		}
		for _, ds := range datasets {
			jt := domain.FromDataset(&ds, id)
			if err := tx.Create(&jt).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("creating table junction: %w", err)
			}
		}
	}

	if err := r.bustCache(ctx, tx, id); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("busting cache: %w", err)
	}

	oldJSON, _ := json.Marshal(oldFilter)
	var newFilter domain.RLSFilter
	r.db.Preload("Roles").Preload("Tables").First(&newFilter, id)
	newJSON, _ := json.Marshal(newFilter)
	audit := domain.RLSAuditLog{
		FilterID:   id,
		FilterName: oldFilter.Name,
		EventType:  domain.RLSAuditEventFilterUpdated,
		OldValue:   string(oldJSON),
		NewValue:   string(newJSON),
		ChangedBy:  actorUserID,
		IPAddress:  "",
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("creating audit log: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	var result domain.RLSFilter
	if err := r.db.Preload("Roles").Preload("Tables").First(&result, id).Error; err != nil {
		return nil, fmt.Errorf("refetching filter: %w", err)
	}

	return toResponse(&result), nil
}

func (r *rlsFilterRepo) Delete(ctx context.Context, actorUserID uint, id uint) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("starting transaction: %w", tx.Error)
	}

	var filter domain.RLSFilter
	if err := tx.Preload("Roles").Preload("Tables").First(&filter, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("filter not found")
		}
		return fmt.Errorf("finding filter: %w", err)
	}

	oldJSON, _ := json.Marshal(filter)
	audit := domain.RLSAuditLog{
		FilterID:   id,
		FilterName: filter.Name,
		EventType:  domain.RLSAuditEventFilterDeleted,
		OldValue:   string(oldJSON),
		NewValue:   "null",
		ChangedBy:  actorUserID,
		IPAddress:  "",
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("creating audit log: %w", err)
	}

	if err := r.bustCache(ctx, tx, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("busting cache: %w", err)
	}

	if err := tx.Delete(&filter).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("deleting filter: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func (r *rlsFilterRepo) GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error) {
	var roles []string
	err := r.db.WithContext(ctx).
		Table("ab_user_role ur").
		Joins("JOIN ab_role ro ON ro.id = ur.role_id").
		Where("ur.user_id = ?", userID).
		Pluck("ro.name", &roles).Error
	if err != nil {
		return nil, fmt.Errorf("getting role names: %w", err)
	}
	return roles, nil
}

func (r *rlsFilterRepo) GetFiltersByDatasourceAndRoles(ctx context.Context, datasourceID uint, roleIDs []uint) ([]domain.RLSFilter, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	var filters []domain.RLSFilter
	err := r.db.WithContext(ctx).
		Table("row_level_security_filters rls").
		Joins("JOIN rls_filter_roles rfr ON rfr.rls_id = rls.id").
		Joins("JOIN rls_filter_tables rft ON rft.rls_id = rls.id").
		Where("rfr.role_id IN ?", roleIDs).
		Where("rft.datasource_id = ?", datasourceID).
		Preload("Roles").
		Preload("Tables").
		Find(&filters).Error
	if err != nil {
		return nil, fmt.Errorf("getting RLS filters: %w", err)
	}
	return filters, nil
}

func (r *rlsFilterRepo) bustCache(ctx context.Context, tx *gorm.DB, filterID uint) error {
	var dsIDs []uint
	if err := tx.Table("rls_filter_tables").Where("rls_id = ?", filterID).Pluck("datasource_id", &dsIDs).Error; err != nil {
		return err
	}
	for _, dsID := range dsIDs {
		pattern := fmt.Sprintf("rls:*:%d", dsID)
		iter := r.rdb.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			if err := r.rdb.Del(ctx, iter.Val()).Err(); err != nil {
				return err
			}
		}
		if err := iter.Err(); err != nil {
			return err
		}
	}
	return nil
}

func toResponse(f *domain.RLSFilter) *domain.RLSFilterResponse {
	roles := make([]domain.Role, len(f.Roles))
	for i, role := range f.Roles {
		roles[i] = domain.Role{ID: role.ID, Name: role.Name}
	}
	tables := make([]domain.RLSFilterTableInfo, len(f.Tables))
	for i, t := range f.Tables {
		tables[i] = domain.RLSFilterTableInfo{
			DatasourceID:   t.DatasourceID,
			DatasourceType: t.DatasourceType,
			TableName:      t.Table,
			DatabaseName:   t.DbName,
		}
	}
	return &domain.RLSFilterResponse{
		ID:          f.ID,
		Name:        f.Name,
		FilterType:  string(f.FilterType),
		Clause:      f.Clause,
		GroupKey:    f.GroupKey,
		Description: f.Description,
		Roles:       roles,
		Tables:      tables,
		CreatedBy:   f.CreatedByFK,
		CreatedOn:   f.CreatedOn,
		ChangedOn:   f.ChangedOn,
	}
}

func getActor(c *gin.Context) (*domain.UserContext, bool) {
	v, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	actor, ok := v.(domain.UserContext)
	return &actor, ok
}
