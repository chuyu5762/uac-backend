package repository

import (
	"context"
	"errors"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound       = errors.New("用户不存在")
	ErrUserUsernameExists = errors.New("用户名已存在")
	ErrUserEmailExists    = errors.New("邮箱已存在")
	ErrBindingNotFound    = errors.New("绑定关系不存在")
	ErrBindingExists      = errors.New("绑定关系已存在")
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *UserFilter, page *Pagination) ([]*model.User, int64, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type UserOrgBindingRepository interface {
	Create(ctx context.Context, binding *model.UserOrgBinding) error
	Delete(ctx context.Context, userID, orgID string) error
	GetByUserAndOrg(ctx context.Context, userID, orgID string) (*model.UserOrgBinding, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.UserOrgBinding, error)
	ListByOrgID(ctx context.Context, orgID string, page *Pagination) ([]*model.UserOrgBinding, int64, error)
	Exists(ctx context.Context, userID, orgID string) (bool, error)
}

type UserFilter struct {
	Username string
	Email    string
	Status   string
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	exists, _ := r.ExistsByUsername(ctx, user.Username)
	if exists {
		return ErrUserUsernameExists
	}
	exists, _ = r.ExistsByEmail(ctx, user.Email)
	if exists {
		return ErrUserEmailExists
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, filter *UserFilter, page *Pagination) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64
	query := r.db.WithContext(ctx).Model(&model.User{})
	if filter != nil {
		if filter.Username != "" {
			query = query.Where("username LIKE ?", "%"+filter.Username+"%")
		}
		if filter.Email != "" {
			query = query.Where("email LIKE ?", "%"+filter.Email+"%")
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page != nil && page.Page > 0 && page.PageSize > 0 {
		offset := (page.Page - 1) * page.PageSize
		query = query.Offset(offset).Limit(page.PageSize)
	}
	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// UserOrgBinding Repository

type userOrgBindingRepository struct {
	db *gorm.DB
}

func NewUserOrgBindingRepository(db *gorm.DB) UserOrgBindingRepository {
	return &userOrgBindingRepository{db: db}
}

func (r *userOrgBindingRepository) Create(ctx context.Context, binding *model.UserOrgBinding) error {
	exists, _ := r.Exists(ctx, binding.UserID, binding.OrgID)
	if exists {
		return ErrBindingExists
	}
	return r.db.WithContext(ctx).Create(binding).Error
}

func (r *userOrgBindingRepository) Delete(ctx context.Context, userID, orgID string) error {
	result := r.db.WithContext(ctx).Where("user_id = ? AND org_id = ?", userID, orgID).Delete(&model.UserOrgBinding{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrBindingNotFound
	}
	return nil
}

func (r *userOrgBindingRepository) GetByUserAndOrg(ctx context.Context, userID, orgID string) (*model.UserOrgBinding, error) {
	var binding model.UserOrgBinding
	err := r.db.WithContext(ctx).Where("user_id = ? AND org_id = ?", userID, orgID).First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

func (r *userOrgBindingRepository) ListByUserID(ctx context.Context, userID string) ([]*model.UserOrgBinding, error) {
	var bindings []*model.UserOrgBinding
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Preload("Organization").Find(&bindings).Error
	return bindings, err
}

func (r *userOrgBindingRepository) ListByOrgID(ctx context.Context, orgID string, page *Pagination) ([]*model.UserOrgBinding, int64, error) {
	var bindings []*model.UserOrgBinding
	var total int64
	query := r.db.WithContext(ctx).Model(&model.UserOrgBinding{}).Where("org_id = ?", orgID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page != nil && page.Page > 0 && page.PageSize > 0 {
		offset := (page.Page - 1) * page.PageSize
		query = query.Offset(offset).Limit(page.PageSize)
	}
	if err := query.Preload("User").Find(&bindings).Error; err != nil {
		return nil, 0, err
	}
	return bindings, total, nil
}

func (r *userOrgBindingRepository) Exists(ctx context.Context, userID, orgID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserOrgBinding{}).Where("user_id = ? AND org_id = ?", userID, orgID).Count(&count).Error
	return count > 0, err
}
