package service

import (
	"context"
	"testing"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

// mockOrgRepository 模拟组织仓库
type mockOrgRepository struct {
	orgs     map[string]*model.Organization
	slugMap  map[string]string // slug -> id
	createFn func(ctx context.Context, org *model.Organization) error
	getFn    func(ctx context.Context, id string) (*model.Organization, error)
}

func newMockOrgRepository() *mockOrgRepository {
	return &mockOrgRepository{
		orgs:    make(map[string]*model.Organization),
		slugMap: make(map[string]string),
	}
}

func (m *mockOrgRepository) Create(ctx context.Context, org *model.Organization) error {
	if m.createFn != nil {
		return m.createFn(ctx, org)
	}
	if _, exists := m.slugMap[org.Slug]; exists {
		return repository.ErrOrgSlugExists
	}
	org.ID = "test-id-" + org.Slug
	m.orgs[org.ID] = org
	m.slugMap[org.Slug] = org.ID
	return nil
}

func (m *mockOrgRepository) GetByID(ctx context.Context, id string) (*model.Organization, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	if org, exists := m.orgs[id]; exists {
		return org, nil
	}
	return nil, repository.ErrOrgNotFound
}

func (m *mockOrgRepository) GetBySlug(ctx context.Context, slug string) (*model.Organization, error) {
	if id, exists := m.slugMap[slug]; exists {
		return m.orgs[id], nil
	}
	return nil, repository.ErrOrgNotFound
}

func (m *mockOrgRepository) Update(ctx context.Context, org *model.Organization) error {
	if _, exists := m.orgs[org.ID]; !exists {
		return repository.ErrOrgNotFound
	}
	m.orgs[org.ID] = org
	return nil
}

func (m *mockOrgRepository) Delete(ctx context.Context, id string) error {
	if org, exists := m.orgs[id]; exists {
		delete(m.slugMap, org.Slug)
		delete(m.orgs, id)
		return nil
	}
	return repository.ErrOrgNotFound
}

func (m *mockOrgRepository) List(ctx context.Context, filter *repository.OrgFilter, page *repository.Pagination) ([]*model.Organization, int64, error) {
	var result []*model.Organization
	for _, org := range m.orgs {
		if filter != nil {
			if filter.Status != "" && org.Status != filter.Status {
				continue
			}
		}
		result = append(result, org)
	}
	return result, int64(len(result)), nil
}

func (m *mockOrgRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	_, exists := m.slugMap[slug]
	return exists, nil
}

// 测试用例

func TestOrganizationService_Create(t *testing.T) {
	repo := newMockOrgRepository()
	svc := NewOrganizationService(repo)
	ctx := context.Background()

	tests := []struct {
		name    string
		org     *model.Organization
		wantErr error
	}{
		{
			name: "创建成功（指定 slug）",
			org: &model.Organization{
				Name: "测试组织",
				Slug: "test-org",
			},
			wantErr: nil,
		},
		{
			name: "创建成功（自动生成 slug）",
			org: &model.Organization{
				Name: "测试组织2",
				Slug: "", // slug 为空时自动生成
			},
			wantErr: nil,
		},
		{
			name: "名称为空",
			org: &model.Organization{
				Name: "",
				Slug: "test-org-2",
			},
			wantErr: ErrOrgNameEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Create(ctx, tt.org)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("期望错误 %v，但没有错误", tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("期望错误 %v，实际错误 %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("不期望错误，但得到 %v", err)
			}
		})
	}
}

func TestOrganizationService_GetByID(t *testing.T) {
	repo := newMockOrgRepository()
	svc := NewOrganizationService(repo)
	ctx := context.Background()

	// 先创建一个组织
	org := &model.Organization{Name: "测试组织", Slug: "test-org"}
	_ = svc.Create(ctx, org)

	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{
			name:    "查询成功",
			id:      org.ID,
			wantErr: nil,
		},
		{
			name:    "ID 为空",
			id:      "",
			wantErr: ErrOrgIDEmpty,
		},
		{
			name:    "组织不存在",
			id:      "non-existent-id",
			wantErr: repository.ErrOrgNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GetByID(ctx, tt.id)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("期望错误 %v，但没有错误", tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("期望错误 %v，实际错误 %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到 %v", err)
				}
				if result == nil {
					t.Error("期望返回组织，但得到 nil")
				}
			}
		})
	}
}

func TestOrganizationService_Update(t *testing.T) {
	repo := newMockOrgRepository()
	svc := NewOrganizationService(repo)
	ctx := context.Background()

	// 先创建一个组织
	org := &model.Organization{Name: "测试组织", Slug: "test-org"}
	_ = svc.Create(ctx, org)

	tests := []struct {
		name    string
		org     *model.Organization
		wantErr error
	}{
		{
			name: "更新成功",
			org: &model.Organization{
				BaseModel: model.BaseModel{ID: org.ID},
				Name:      "更新后的名称",
			},
			wantErr: nil,
		},
		{
			name: "ID 为空",
			org: &model.Organization{
				Name: "测试",
			},
			wantErr: ErrOrgIDEmpty,
		},
		{
			name: "名称为空",
			org: &model.Organization{
				BaseModel: model.BaseModel{ID: org.ID},
				Name:      "",
			},
			wantErr: ErrOrgNameEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Update(ctx, tt.org)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("期望错误 %v，但没有错误", tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("期望错误 %v，实际错误 %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("不期望错误，但得到 %v", err)
			}
		})
	}
}

func TestOrganizationService_Delete(t *testing.T) {
	repo := newMockOrgRepository()
	svc := NewOrganizationService(repo)
	ctx := context.Background()

	// 先创建一个组织
	org := &model.Organization{Name: "测试组织", Slug: "test-org"}
	_ = svc.Create(ctx, org)

	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{
			name:    "删除成功",
			id:      org.ID,
			wantErr: nil,
		},
		{
			name:    "ID 为空",
			id:      "",
			wantErr: ErrOrgIDEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Delete(ctx, tt.id)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("期望错误 %v，但没有错误", tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("期望错误 %v，实际错误 %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("不期望错误，但得到 %v", err)
			}
		})
	}
}

func TestOrganizationService_UpdateBranding(t *testing.T) {
	repo := newMockOrgRepository()
	svc := NewOrganizationService(repo)
	ctx := context.Background()

	// 先创建一个组织
	org := &model.Organization{Name: "测试组织", Slug: "test-org"}
	_ = svc.Create(ctx, org)

	branding := &model.Branding{
		LogoURL:      "https://example.com/logo.png",
		PrimaryColor: "#FF5500",
	}

	err := svc.UpdateBranding(ctx, org.ID, branding)
	if err != nil {
		t.Errorf("更新品牌配置失败: %v", err)
	}

	// 验证更新结果
	updated, _ := svc.GetByID(ctx, org.ID)
	if updated.Branding.LogoURL != branding.LogoURL {
		t.Errorf("Logo URL 不匹配，期望 %s，实际 %s", branding.LogoURL, updated.Branding.LogoURL)
	}
}
