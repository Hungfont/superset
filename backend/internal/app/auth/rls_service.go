package auth

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	domain "superset/auth-service/internal/domain/auth"
	dbapp "superset/auth-service/internal/app/db"
	dbdomain "superset/auth-service/internal/domain/db"
	"superset/auth-service/internal/pkg/crypto"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/xwb1989/sqlparser"
)

type RLSService struct {
	repo        domain.RLSFilterRepository
	rdb         *redis.Client
	dbRepo      dbdomain.DatabaseRepository
	poolManager dbapp.DatabaseConnectionPool
	encKey      []byte
}

func NewRLSService(
	repo domain.RLSFilterRepository,
	rdb *redis.Client,
	dbRepo dbdomain.DatabaseRepository,
	poolManager dbapp.DatabaseConnectionPool,
	encryptionKey string,
) *RLSService {
	parsedKey, _ := crypto.ParseEncryptionKey(encryptionKey) // validated at startup
	return &RLSService{
		repo:        repo,
		rdb:         rdb,
		dbRepo:      dbRepo,
		poolManager: poolManager,
		encKey:      parsedKey,
	}
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

	// Security: block DML / multi-statement injection patterns
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
			return fmt.Errorf("Invalid SQL clause: contains disallowed pattern")
		}
	}

	// Validate by wrapping in a SELECT ... WHERE <clause> so sqlparser can parse
	// it as a valid expression. sqlparser.Parse is the available API in this version.
	wrapped := "SELECT 1 WHERE " + clause
	_, err := sqlparser.Parse(wrapped)
	if err != nil {
		return fmt.Errorf("Invalid SQL clause: %s", err.Error())
	}

	return nil
}

func (s *RLSService) Create(ctx context.Context, actorUserID uint, ipAddress string, req domain.CreateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	if err := s.ValidateClause(req.Clause); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, actorUserID, ipAddress, req)
}

func (s *RLSService) Update(ctx context.Context, actorUserID uint, ipAddress string, id uint, req domain.UpdateRLSFilterRequest) (*domain.RLSFilterResponse, error) {
	if req.Clause != "" {
		if err := s.ValidateClause(req.Clause); err != nil {
			return nil, err
		}
	}

	return s.repo.Update(ctx, actorUserID, ipAddress, id, req)
}

func (s *RLSService) Delete(ctx context.Context, actorUserID uint, ipAddress string, id uint) error {
	return s.repo.Delete(ctx, actorUserID, ipAddress, id)
}

func (s *RLSService) GetRoleNamesByUser(ctx context.Context, userID uint) ([]string, error) {
	return s.repo.GetRoleNamesByUser(ctx, userID)
}

// Validate performs two-phase clause validation.
// Returns (result, httpStatus, error).
func (s *RLSService) Validate(ctx context.Context, uc domain.UserContext, req domain.ValidateRequest) (domain.ValidateResult, int, error) {
	// Phase 0: Rate limit
	key := "rls:rate:validate:" + strconv.Itoa(int(uc.ID))
	cnt, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return domain.ValidateResult{}, 500, fmt.Errorf("rate limit check: %w", err)
	}
	if cnt == 1 {
		s.rdb.Expire(ctx, key, 60*time.Second)
	}
	if cnt > 60 {
		return domain.ValidateResult{}, 429, fmt.Errorf("rate limit exceeded")
	}

	// Phase 1: Syntax — replace template vars with placeholders first for valid SQL
	syntaxClause := strings.NewReplacer(
		"{{current_user_id}}", "1",
		"{{current_username}}", "'test'",
	).Replace(req.Clause)
	wrapped := "SELECT 1 WHERE " + syntaxClause
	if _, err := sqlparser.Parse(wrapped); err != nil {
		pos := extractSQLPosition(err)
		return domain.ValidateResult{
			IsValid:       false,
			Phase:         "syntax",
			Error:         err.Error(),
			ErrorPosition: pos,
		}, 200, nil
	}

	// Gate: Phase 2 requires database_id + test_user_id + table_name
	if req.DatabaseID == nil || req.TestUserID == nil || req.TableName == "" {
		return domain.ValidateResult{
			IsValid:        true,
			Phase:          "syntax",
			RenderedClause: req.Clause,
		}, 200, nil
	}

	// Phase 2: Render template vars with actual values
	rendered := strings.NewReplacer(
		"{{current_user_id}}", strconv.Itoa(*req.TestUserID),
		"{{current_username}}", req.TestUsername,
	).Replace(req.Clause)

	// Build probe SQL with identifier quoting
	probeSQL := fmt.Sprintf(
		"SELECT 1 FROM %s.%s WHERE (%s) LIMIT 0",
		pgx.Identifier{req.Schema}.Sanitize(),
		pgx.Identifier{req.TableName}.Sanitize(),
		rendered,
	)

	// Get DB connection + decrypt URI
	dbRecord, err := s.dbRepo.GetDatabaseByID(ctx, *req.DatabaseID)
	if err != nil {
		return domain.ValidateResult{}, 500, fmt.Errorf("database lookup: %w", err)
	}

	decryptedURI, err := crypto.DecryptSQLAlchemyURIPassword(dbRecord.SQLAlchemyURI, s.encKey)
	if err != nil {
		return domain.ValidateResult{}, 500, fmt.Errorf("decrypting database URI: %w", err)
	}

	conn, err := s.poolManager.Get(ctx, *req.DatabaseID, decryptedURI)
	if err != nil {
		return domain.ValidateResult{}, 500, fmt.Errorf("connection pool unavailable: %w", err)
	}

	// 5s timeout for probe
	ctx5s, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := conn.ExecContext(ctx5s, probeSQL); err != nil {
		return domain.ValidateResult{
			IsValid: false,
			Phase:   "runtime",
			Error:   err.Error(),
		}, 200, nil
	}

	return domain.ValidateResult{
		IsValid:        true,
		Phase:          "runtime",
		RenderedClause: rendered,
	}, 200, nil
}

// extractSQLPosition parses the position from a sqlparser error message.
// Example input: "syntax error at position 14" → &14
func extractSQLPosition(err error) *int {
	re := regexp.MustCompile(`position (\d+)`)
	matches := re.FindStringSubmatch(err.Error())
	if len(matches) >= 2 {
		pos, parseErr := strconv.Atoi(matches[1])
		if parseErr == nil {
			return &pos
		}
	}
	return nil
}