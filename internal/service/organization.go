// Package service 业务逻辑层
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

// 错误定义
var (
	ErrOrgNameEmpty = errors.New("组织名称不能为空")
	ErrOrgIDEmpty   = errors.New("组织 ID 不能为空")
)

// OrganizationService 组织服务接口
type OrganizationService interface {
	Create(ctx context.Context, org *model.Organization) error
	GetByID(ctx context.Context, id string) (*model.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*model.Organization, error)
	Update(ctx context.Context, org *model.Organization) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *repository.OrgFilter, page *repository.Pagination) ([]*model.Organization, int64, error)
	UpdateBranding(ctx context.Context, id string, branding *model.Branding) error
}

// organizationService 组织服务实现
type organizationService struct {
	repo repository.OrganizationRepository
}

// NewOrganizationService 创建组织服务实例
func NewOrganizationService(repo repository.OrganizationRepository) OrganizationService {
	return &organizationService{repo: repo}
}

// generateSlug 生成组织标识（使用短 UUID）
func generateSlug() string {
	// 使用 UUID 的前 8 位作为 slug
	id := uuid.New().String()
	return strings.ReplaceAll(id[:8], "-", "")
}

// Create 创建组织
func (s *organizationService) Create(ctx context.Context, org *model.Organization) error {
	// 校验名称
	org.Name = strings.TrimSpace(org.Name)
	if org.Name == "" {
		return ErrOrgNameEmpty
	}

	// 自动生成 slug
	if org.Slug == "" {
		org.Slug = generateSlug()
	}

	// 设置默认状态
	if org.Status == "" {
		org.Status = model.StatusActive
	}

	return s.repo.Create(ctx, org)
}

// GetByID 根据 ID 获取组织
func (s *organizationService) GetByID(ctx context.Context, id string) (*model.Organization, error) {
	if id == "" {
		return nil, ErrOrgIDEmpty
	}
	return s.repo.GetByID(ctx, id)
}

// GetBySlug 根据 Slug 获取组织
func (s *organizationService) GetBySlug(ctx context.Context, slug string) (*model.Organization, error) {
	if slug == "" {
		return nil, errors.New("组织标识不能为空")
	}
	return s.repo.GetBySlug(ctx, strings.ToLower(slug))
}

// Update 更新组织
func (s *organizationService) Update(ctx context.Context, org *model.Organization) error {
	if org.ID == "" {
		return ErrOrgIDEmpty
	}

	// 校验名称
	if org.Name == "" {
		return ErrOrgNameEmpty
	}

	return s.repo.Update(ctx, org)
}

// Delete 删除组织
func (s *organizationService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrOrgIDEmpty
	}
	// TODO: 检查组织下是否有应用，有则不允许删除或级联删除
	return s.repo.Delete(ctx, id)
}

// List 查询组织列表
func (s *organizationService) List(ctx context.Context, filter *repository.OrgFilter, page *repository.Pagination) ([]*model.Organization, int64, error) {
	// 设置默认分页
	if page == nil {
		page = &repository.Pagination{Page: 1, PageSize: 20}
	}
	if page.Page < 1 {
		page.Page = 1
	}
	if page.PageSize < 1 || page.PageSize > 100 {
		page.PageSize = 20
	}

	return s.repo.List(ctx, filter, page)
}

// UpdateBranding 更新组织品牌配置
func (s *organizationService) UpdateBranding(ctx context.Context, id string, branding *model.Branding) error {
	if id == "" {
		return ErrOrgIDEmpty
	}

	// 获取现有组织
	org, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 更新品牌配置
	org.Branding = *branding
	return s.repo.Update(ctx, org)
}
