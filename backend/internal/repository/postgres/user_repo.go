package postgres

import (
	"context"
	"errors"
	"fmt"

	domain "superset/auth-service/internal/domain/auth"

	"gorm.io/gorm"
)

type userRepo struct {
	db *gorm.DB
}

type userRoleAssignmentRow struct {
	UserID uint `gorm:"column:user_id"`
	RoleID uint `gorm:"column:role_id"`
}

func (userRoleAssignmentRow) TableName() string { return "ab_user_role" }

// NewUserRepository returns a UserRepository backed by PostgreSQL.
func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepo{db: db}
}

// NewUserAdminRepository returns a UserAdminRepository backed by PostgreSQL.
func NewUserAdminRepository(db *gorm.DB) domain.UserAdminRepository {
	return &userRepo{db: db}
}

func (r *userRepo) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	return &user, nil
}

func (r *userRepo) IsAdmin(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("ab_user_role ur").
		Joins("JOIN ab_role ro ON ro.id = ur.role_id").
		Where("ur.user_id = ? AND LOWER(ro.name) = ?", userID, "admin").
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking admin role: %w", err)
	}
	return count > 0, nil
}

func (r *userRepo) ListUsers(ctx context.Context) ([]domain.UserListItem, error) {
	var users []domain.UserListItem
	err := r.db.WithContext(ctx).
		Table("ab_user").
		Select("id, first_name, last_name, username, email, active, login_count, last_login").
		Order("id ASC").
		Scan(&users).Error
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	roleIDsByUser, err := r.listRoleIDsByUsers(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("listing roles for users: %w", err)
	}

	for i := range users {
		users[i].RoleIDs = roleIDsByUser[users[i].ID]
	}

	return users, nil
}

func (r *userRepo) GetUserByID(ctx context.Context, userID uint) (*domain.UserDetail, error) {
	var user domain.UserDetail
	err := r.db.WithContext(ctx).
		Table("ab_user").
		Select("id, first_name, last_name, username, email, active, login_count, last_login, created_on, changed_on").
		Where("id = ?", userID).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting user by id: %w", err)
	}

	roleIDs, err := r.listRoleIDsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user roles: %w", err)
	}
	user.RoleIDs = roleIDs

	return &user, nil
}

func (r *userRepo) CreateUser(ctx context.Context, req domain.CreateUserRequest) (uint, error) {
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	user := domain.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		Active:    active,
	}

	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return 0, fmt.Errorf("creating user: %w", err)
	}
	return user.ID, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, userID uint, req domain.UpdateUserRequest) error {
	updates := map[string]any{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"username":   req.Username,
		"email":      req.Email,
		"active":     req.Active,
	}

	res := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("updating user: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *userRepo) DeactivateUser(ctx context.Context, userID uint) error {
	res := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Update("active", false)
	if res.Error != nil {
		return fmt.Errorf("deactivating user: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *userRepo) CountExistingRoles(ctx context.Context, roleIDs []uint) (int64, error) {
	uniqueIDs := uniqueUintIDs(roleIDs)
	if len(uniqueIDs) == 0 {
		return 0, nil
	}

	var count int64
	err := r.db.WithContext(ctx).
		Table("ab_role").
		Where("id IN ?", uniqueIDs).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting roles: %w", err)
	}

	return count, nil
}

func (r *userRepo) ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	uniqueRoleIDs := uniqueUintIDs(roleIDs)
	rows := make([]userRoleAssignmentRow, 0, len(uniqueRoleIDs))
	for _, roleID := range uniqueRoleIDs {
		rows = append(rows, userRoleAssignmentRow{UserID: userID, RoleID: roleID})
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ab_user_role").Where("user_id = ?", userID).Delete(&userRoleAssignmentRow{}).Error; err != nil {
			return fmt.Errorf("deleting existing user roles: %w", err)
		}

		if len(rows) == 0 {
			return nil
		}

		if err := tx.Table("ab_user_role").CreateInBatches(rows, 100).Error; err != nil {
			return fmt.Errorf("creating user roles: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("replacing user roles: %w", err)
	}

	return nil
}

func (r *userRepo) listRoleIDsByUser(ctx context.Context, userID uint) ([]uint, error) {
	roleIDs := make([]uint, 0)
	err := r.db.WithContext(ctx).
		Table("ab_user_role").
		Where("user_id = ?", userID).
		Order("role_id ASC").
		Pluck("role_id", &roleIDs).Error
	if err != nil {
		return nil, fmt.Errorf("listing role ids by user: %w", err)
	}
	return roleIDs, nil
}

func (r *userRepo) listRoleIDsByUsers(ctx context.Context, userIDs []uint) (map[uint][]uint, error) {
	result := make(map[uint][]uint, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	var rows []userRoleAssignmentRow
	err := r.db.WithContext(ctx).
		Table("ab_user_role").
		Where("user_id IN ?", userIDs).
		Order("user_id ASC, role_id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("listing role ids by users: %w", err)
	}

	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], row.RoleID)
	}

	return result, nil
}
