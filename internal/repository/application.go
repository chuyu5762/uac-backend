package repository

import (
	"context"
	"errors"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrAppNotFound       = errors.New("应用不存在")
	ErrAppClientIDExists = errors.New("Client ID 已存在")
)

// ApplicationRepository 应用数据访问接口
type ApplicationRepository interface {
	Create(ctx context.Context, app *model.Application) error
	GetByID(ctx context.Context, id string) (*model.Application, error)
	GetByClientID(ctx context.Context, clientID string) (*model.Application, error)
	Update(ctx context.Context, app *model.Application) error
	UpdateSecret(ctx context.Context, id string, secretHash string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *AppFilter, page *Pagination) ([]*model.Application, int64, error)
	ListByOrgID(ctx context.Context, orgID string, page *Pagination) ([]*model.Application, int64, error)
	ExistsByClientID(ctx context.Context, clientID string) (bool, error)
	CountByOrgID(ctx context.Context, orgID string) (int64, error)
}

// AppFilter 应用查询过滤器
type AppFilter struct {
	OrgID    string // 组织 ID
	Name     string // 名称（模糊匹配）
	Status   string // 状态
	Protocol string // 协议类型
}

// applicationRepository 应用数据访问实现
type applicationRepository struct {
	db *gorm.DB
}

// NewApplicationRepository 创建应用数据访问实例
func NewApplicationRepository(db *gorm.DB) ApplicationRepository {
	return &applicationRepository{db: db}
}

// Create 创建应用
func (r *applicationRepository) Create(ctx context.Context, app *model.Application) error {
	// 检查 ClientID 是否已存在
	exists, err := r.ExistsByClientID(ctx, app.ClientID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAppClientIDExists
	}

	return r.db.WithContext(ctx).Create(app).Error
}

// GetByID 根据 ID 获取应用
func (r *applicationRepository) GetByID(ctx context.Context, id string) (*model.Application, error) {
	var app model.Application
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}
	return &app, nil
}

// GetByClientID 根据 ClientID 获取应用
func (r *applicationRepository) GetByClientID(ctx context.Context, clientID string) (*model.Application, error) {
	var app model.Application
	err := r.db.WithContext(ctx).Where("client_id = ?", clientID).First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}
	return &app, nil
}

// Update 更新应用
func (r *applicationRepository) Update(ctx context.Context, app *model.Application) error {
	// 使用 GORM 的 Save 方法，自动处理字段映射，兼容 PostgreSQL 和 MySQL
	result := r.db.WithContext(ctx).Model(app).Select(
		"name",
		"description",
		"redirect_uris",
		"allowed_scopes",
		"protocol",
		"status",
		"client_secret_hash",
	).Updates(app)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAppNotFound
	}
	return nil
}

// UpdateSecret 仅更新 Client Secret（用于重置密钥）
func (r *applicationRepository) UpdateSecret(ctx context.Context, id string, secretHash string) error {
	result := r.db.WithContext(ctx).Model(&model.Application{}).
		Where("id = ?", id).
		Update("client_secret_hash", secretHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAppNotFound
	}
	return nil
}

// Delete 删除应用（软删除）
func (r *applicationRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Application{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAppNotFound
	}
	return nil
}

// List 查询应用列表
func (r *applicationRepository) List(ctx context.Context, filter *AppFilter, page *Pagination) ([]*model.Application, int64, error) {
	var apps []*model.Application
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Application{})

	// 应用过滤条件
	if filter != nil {
		if filter.OrgID != "" {
			query = query.Where("org_id = ?", filter.OrgID)
		}
		if filter.Name != "" {
			query = query.Where("name LIKE ?", "%"+filter.Name+"%")
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.Protocol != "" {
			query = query.Where("protocol = ?", filter.Protocol)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if page != nil && page.Page > 0 && page.PageSize > 0 {
		offset := (page.Page - 1) * page.PageSize
		query = query.Offset(offset).Limit(page.PageSize)
	}

	// 按创建时间倒序
	if err := query.Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}

// ListByOrgID 根据组织 ID 查询应用列表
func (r *applicationRepository) ListByOrgID(ctx context.Context, orgID string, page *Pagination) ([]*model.Application, int64, error) {
	return r.List(ctx, &AppFilter{OrgID: orgID}, page)
}

// ExistsByClientID 检查 ClientID 是否已存在
func (r *applicationRepository) ExistsByClientID(ctx context.Context, clientID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Application{}).Where("client_id = ?", clientID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountByOrgID 统计组织下的应用数量
func (r *applicationRepository) CountByOrgID(ctx context.Context, orgID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Application{}).Where("org_id = ?", orgID).Count(&count).Error
	return count, err
}
