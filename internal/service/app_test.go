package service

import (
	"context"
	"testing"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

type mockAppRepository struct {
	apps      map[string]*model.Application
	clientMap map[string]string
}

func newMockAppRepository() *mockAppRepository {
	return &mockAppRepository{
		apps:      make(map[string]*model.Application),
		clientMap: make(map[string]string),
	}
}

func (m *mockAppRepository) Create(ctx context.Context, app *model.Application) error {
	if _, exists := m.clientMap[app.ClientID]; exists {
		return repository.ErrAppClientIDExists
	}
	app.ID = "test-app-" + app.ClientID[:8]
	m.apps[app.ID] = app
	m.clientMap[app.ClientID] = app.ID
	return nil
}

func (m *mockAppRepository) GetByID(ctx context.Context, id string) (*model.Application, error) {
	if app, exists := m.apps[id]; exists {
		return app, nil
	}
	return nil, repository.ErrAppNotFound
}

func (m *mockAppRepository) GetByClientID(ctx context.Context, clientID string) (*model.Application, error) {
	if id, exists := m.clientMap[clientID]; exists {
		return m.apps[id], nil
	}
	return nil, repository.ErrAppNotFound
}

func (m *mockAppRepository) Update(ctx context.Context, app *model.Application) error {
	if _, exists := m.apps[app.ID]; !exists {
		return repository.ErrAppNotFound
	}
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepository) Delete(ctx context.Context, id string) error {
	if app, exists := m.apps[id]; exists {
		delete(m.clientMap, app.ClientID)
		delete(m.apps, id)
		return nil
	}
	return repository.ErrAppNotFound
}

func (m *mockAppRepository) List(ctx context.Context, filter *repository.AppFilter, page *repository.Pagination) ([]*model.Application, int64, error) {
	var result []*model.Application
	for _, app := range m.apps {
		if filter != nil && filter.OrgID != "" && (app.OrgID == nil || *app.OrgID != filter.OrgID) {
			continue
		}
		result = append(result, app)
	}
	return result, int64(len(result)), nil
}

func (m *mockAppRepository) ListByOrgID(ctx context.Context, orgID string, page *repository.Pagination) ([]*model.Application, int64, error) {
	return m.List(ctx, &repository.AppFilter{OrgID: orgID}, page)
}

func (m *mockAppRepository) ExistsByClientID(ctx context.Context, clientID string) (bool, error) {
	_, exists := m.clientMap[clientID]
	return exists, nil
}

func (m *mockAppRepository) CountByOrgID(ctx context.Context, orgID string) (int64, error) {
	var count int64
	for _, app := range m.apps {
		if app.OrgID != nil && *app.OrgID == orgID {
			count++
		}
	}
	return count, nil
}

func (m *mockAppRepository) UpdateSecret(ctx context.Context, id string, secretHash string) error {
	if app, exists := m.apps[id]; exists {
		app.ClientSecretHash = secretHash
		return nil
	}
	return repository.ErrAppNotFound
}

func TestAppService_Create(t *testing.T) {
	appRepo := newMockAppRepository()
	orgRepo := newMockOrgRepository()
	svc := NewApplicationService(appRepo, orgRepo)
	ctx := context.Background()

	org := &model.Organization{Name: "测试组织", Slug: "test-org-app"}
	orgSvc := NewOrganizationService(orgRepo)
	_ = orgSvc.Create(ctx, org)

	app := &model.Application{Name: "测试应用", OrgID: &org.ID}
	secret, err := svc.Create(ctx, app)
	if err != nil {
		t.Errorf("创建应用失败: %v", err)
	}
	if secret == "" {
		t.Error("期望返回 Client Secret")
	}
	if app.ClientID == "" {
		t.Error("期望生成 Client ID")
	}
}

func TestAppService_ResetSecret(t *testing.T) {
	appRepo := newMockAppRepository()
	orgRepo := newMockOrgRepository()
	svc := NewApplicationService(appRepo, orgRepo)
	ctx := context.Background()

	org := &model.Organization{Name: "测试组织", Slug: "test-org-reset"}
	orgSvc := NewOrganizationService(orgRepo)
	_ = orgSvc.Create(ctx, org)

	app := &model.Application{Name: "测试应用", OrgID: &org.ID}
	oldSecret, _ := svc.Create(ctx, app)

	newSecret, err := svc.ResetSecret(ctx, app.ID)
	if err != nil {
		t.Errorf("重置 Secret 失败: %v", err)
	}
	if newSecret == oldSecret {
		t.Error("新旧 Secret 相同")
	}

	_, err = svc.ValidateClientCredentials(ctx, app.ClientID, oldSecret)
	if err == nil {
		t.Error("旧 Secret 应该失效")
	}

	_, err = svc.ValidateClientCredentials(ctx, app.ClientID, newSecret)
	if err != nil {
		t.Errorf("新 Secret 验证失败: %v", err)
	}
}
