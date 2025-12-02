// Package service 业务逻辑层
package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

// 错误定义
var (
	ErrOrgNameEmpty    = errors.New("组织名称不能为空")
	ErrOrgSlugEmpty    = errors.New("组织标识不能为空")
	ErrOrgSlugInvalid  = errors.New("组织标识只能包含小写字母、数字和连字符")
	ErrOrgSlugTooShort = errors.New("组织标识长度不能少于 2 个字符")
	ErrOrgSlugTooLong  = errors.New("组织标识长度不能超过 50 个字符")
	ErrOrgIDEmpty      = errors.New("组织 ID 不能为空")
)

// slugRegex 组织标识正则表达式
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

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

// Create 创建组织
func (s *organizationService) Create(ctx context.Context, org *model.Organization) error {
	// 参数校验
	if err := s.validateOrg(org); err != nil {
		return err
	}

	// 规范化 slug
	org.Slug = strings.ToLower(strings.TrimSpace(org.Slug))

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
		return nil, ErrOrgSlugEmpty
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

// validateOrg 校验组织数据
func (s *organizationService) validateOrg(org *model.Organization) error {
	if org == nil {
		return errors.New("组织信息不能为空")
	}

	// 校验名称
	org.Name = strings.TrimSpace(org.Name)
	if org.Name == "" {
		return ErrOrgNameEmpty
	}

	// 校验 slug
	org.Slug = strings.TrimSpace(org.Slug)
	if org.Slug == "" {
		return ErrOrgSlugEmpty
	}
	if len(org.Slug) < 2 {
		return ErrOrgSlugTooShort
	}
	if len(org.Slug) > 50 {
		return ErrOrgSlugTooLong
	}
	if !slugRegex.MatchString(org.Slug) {
		return ErrOrgSlugInvalid
	}

	return nil
}
