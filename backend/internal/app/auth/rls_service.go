package auth

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	domain "superset/auth-service/internal/domain/auth"
)

type RLSService struct {
	repo domain.RLSFilterRepository
}

func NewRLSService(repo domain.RLSFilterRepository) *RLSService {
	return &RLSService{repo: repo}
}

func (s *RLSService) List(ctx context.Context, params domain.RLSFilterListParams) (*domain.RLSFilterListResult, error) {
	data, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("listing RLS filters: %w", err)
	}

	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if pages == 0 {
		pages = 1
	}

	return &domain.RLSFilterListResult{
		Total: total,
		Page:  page,
		Pages: pages,
		Data:  data,
	}, nil
}

func (s *RLSService) GetByID(ctx context.Context, id uint) (*domain.RLSFilterResponse, error) {
	filter, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting RLS filter: %w", err)
	}
	if filter == nil {
		return nil, domain.ErrNotFound
	}

	roles := make([]domain.Role, len(filter.Roles))
	for i, role := range filter.Roles {
		roles[i] = domain.Role{ID: role.ID, Name: role.Name}
	}
	tables := make([]domain.RLSFilterTableInfo, len(filter.Tables))
	for i, t := range filter.Tables {
		tables[i] = domain.RLSFilterTableInfo{
			DatasourceID:   t.DatasourceID,
			DatasourceType: t.DatasourceType,
			TableName:      t.Table,
			DatabaseName:   t.DbName,
		}
	}
	return &domain.RLSFilterResponse{
		ID:            filter.ID,
		Name:         filter.Name,
		FilterType:   string(filter.FilterType),
		Clause:      filter.Clause,
		GroupKey:   filter.GroupKey,
		Description: filter.Description,
		Roles:       roles,
		Tables:      tables,
		CreatedBy:   filter.CreatedByFK,
		CreatedOn:   filter.CreatedOn,
		ChangedOn:   filter.ChangedOn,
	}, nil
}

func (s *RLSService) ValidateClause(clause string) error {
	clause = strings.TrimSpace(clause)

	if clause == "" {
		return fmt.Errorf("clause cannot be empty")
	}

	if len(clause) > 5000 {
		return fmt.Errorf("clause exceeds maximum length of 5000 characters")
	}

	injections := []string{
		`(?i)(;|--|/\*|\*/)`,
		`(?i)(\bunion\b.*\bselect\b)`,
		`(?i)(\bdrop\b|\bdelete\b|\btruncate\b)`,
		`(?i)(\binsert\b|\bupdate\b)`,
		`(?i)exec\s*\(`,
		`(?i)execute\s*\(`,
	}
	for _, pattern := range injections {
		if matched, _ := regexp.MatchString(pattern, clause); matched {
			return fmt.Errorf("invalid SQL clause: contains disallowed pattern")
		}
	}

	openParens := strings.Count(clause, "(")
	closeParens := strings.Count(clause, ")")
	if openParens != closeParens {
		return fmt.Errorf("invalid SQL clause: unbalanced parentheses")
	}

	openBrackets := strings.Count(clause, "[")
	closeBrackets := strings.Count(clause, "]")
	if openBrackets != closeBrackets {
		return fmt.Errorf("invalid SQL clause: unbalanced brackets")
	}

	singleQuotes := strings.Count(clause, "'")
	if singleQuotes%2 != 0 {
		return fmt.Errorf("invalid SQL clause: unbalanced quotes")
	}

	words := strings.Fields(clause)
	if len(words) == 0 {
		return fmt.Errorf("clause cannot be empty")
	}

	return nil
}

func (s *RLSService) Create(ctx context.Context, actorUserID uint, req domain.CreateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	if err := s.ValidateClause(req.Clause); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, actorUserID, req)
}

func (s *RLSService) Update(ctx context.Context, actorUserID uint, id uint, req domain.UpdateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	if req.Clause != "" {
		if err := s.ValidateClause(req.Clause); err != nil {
			return nil, err
		}
	}

	return s.repo.Update(ctx, actorUserID, id, req)
}

func (s *RLSService) Delete(ctx context.Context, actorUserID uint, id uint) error {
	return s.repo.Delete(ctx, actorUserID, id)
}

func (s *RLSService) GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error) {
	return s.repo.GetRoleNamesByUser(ctx, userID)
}