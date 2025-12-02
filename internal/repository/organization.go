// Package repository 数据访问层
package repository

import (
	"context"
	"errors"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrOrgNotFound   = errors.New("组织不存在")
	ErrOrgSlugExists = errors.New("组织标识已存在")
	ErrOrgHasApps    = errors.New("组织下存在应用，无法删除")
)

// OrganizationRepository 组织数据访问接口
type OrganizationRepository interface {
	Create(ctx context.Context, org *model.Organization) error
	GetByID(ctx context.Context, id string) (*model.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*model.Organization, error)
	Update(ctx context.Context, org *model.Organization) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *OrgFilter, page *Pagination) ([]*model.Organization, int64, error)
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

// OrgFilter 组织查询过滤器
type OrgFilter struct {
	TenantID string // 租户 ID
	Name     string // 名称（模糊匹配）
	Status   string // 状态
}

// Pagination 分页参数
type Pagination struct {
	Page     int // 页码，从 1 开始
	PageSize int // 每页数量
}

// organizationRepository 组织数据访问实现
type organizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository 创建组织数据访问实例
func NewOrganizationRepository(db *gorm.DB) OrganizationRepository {
	return &organizationRepository{db: db}
}

// Create 创建组织
func (r *organizationRepository) Create(ctx context.Context, org *model.Organization) error {
	// 检查 slug 是否已存在
	exists, err := r.ExistsBySlug(ctx, org.Slug)
	if err != nil {
		return err
	}
	if exists {
		return ErrOrgSlugExists
	}

	return r.db.WithContext(ctx).Create(org).Error
}

// GetByID 根据 ID 获取组织
func (r *organizationRepository) GetByID(ctx context.Context, id string) (*model.Organization, error) {
	var org model.Organization
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&org).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return &org, nil
}

// GetBySlug 根据 Slug 获取组织
func (r *organizationRepository) GetBySlug(ctx context.Context, slug string) (*model.Organization, error) {
	var org model.Organization
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&org).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return &org, nil
}

// Update 更新组织
func (r *organizationRepository) Update(ctx context.Context, org *model.Organization) error {
	result := r.db.WithContext(ctx).Model(org).Updates(map[string]interface{}{
		"name":        org.Name,
		"description": org.Description,
		"branding":    org.Branding,
		"status":      org.Status,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrgNotFound
	}
	return nil
}

// Delete 删除组织（软删除）
func (r *organizationRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Organization{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrgNotFound
	}
	return nil
}

// List 查询组织列表
func (r *organizationRepository) List(ctx context.Context, filter *OrgFilter, page *Pagination) ([]*model.Organization, int64, error) {
	var orgs []*model.Organization
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Organization{})

	// 应用过滤条件
	if filter != nil {
		if filter.TenantID != "" {
			query = query.Where("tenant_id = ?", filter.TenantID)
		}
		if filter.Name != "" {
			query = query.Where("name LIKE ?", "%"+filter.Name+"%")
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
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
	if err := query.Order("created_at DESC").Find(&orgs).Error; err != nil {
		return nil, 0, err
	}

	return orgs, total, nil
}

// ExistsBySlug 检查 Slug 是否已存在
func (r *organizationRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Organization{}).Where("slug = ?", slug).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
