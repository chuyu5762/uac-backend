package service

import (
	"context"
	"errors"
	"strings"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

var (
	ErrAppNameEmpty       = errors.New("应用名称不能为空")
	ErrAppOrgIDEmpty      = errors.New("组织 ID 不能为空")
	ErrAppIDEmpty         = errors.New("应用 ID 不能为空")
	ErrAppDisabled        = errors.New("应用已禁用")
	ErrAppInvalidProtocol = errors.New("无效的协议类型")
	ErrAppInvalidVersion  = errors.New("无效的 OAuth 版本")
)

type ApplicationService interface {
	Create(ctx context.Context, app *model.Application) (string, error)
	GetByID(ctx context.Context, id string) (*model.Application, error)
	GetByClientID(ctx context.Context, clientID string) (*model.Application, error)
	Update(ctx context.Context, app *model.Application) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *repository.AppFilter, page *repository.Pagination) ([]*model.Application, int64, error)
	ListByOrgID(ctx context.Context, orgID string, page *repository.Pagination) ([]*model.Application, int64, error)
	ResetSecret(ctx context.Context, id string) (string, error)
	ValidateRedirectURI(ctx context.Context, clientID, redirectURI string) error
	ValidateClientCredentials(ctx context.Context, clientID, clientSecret string) (*model.Application, error)
}

type appService struct {
	repo    repository.ApplicationRepository
	orgRepo repository.OrganizationRepository
}

func NewApplicationService(repo repository.ApplicationRepository, orgRepo repository.OrganizationRepository) ApplicationService {
	return &appService{repo: repo, orgRepo: orgRepo}
}

func (s *appService) Create(ctx context.Context, app *model.Application) (string, error) {
	if err := s.validateApp(app); err != nil {
		return "", err
	}
	// 只有当指定了组织 ID 时才验证组织是否存在（支持系统级应用）
	if s.orgRepo != nil && app.OrgID != nil && *app.OrgID != "" {
		if _, err := s.orgRepo.GetByID(ctx, *app.OrgID); err != nil {
			return "", errors.New("组织不存在")
		}
	} else {
		// 系统级应用：将空串标准化为 NULL
		if app.OrgID != nil && *app.OrgID == "" {
			app.OrgID = nil
		}
	}

	clientID, err := model.GenerateClientID()
	if err != nil {
		return "", errors.New("生成 Client ID 失败")
	}
	app.ClientID = clientID
	clientSecret, err := model.GenerateClientSecret()
	if err != nil {
		return "", errors.New("生成 Client Secret 失败")
	}
	if err := app.SetClientSecret(clientSecret); err != nil {
		return "", errors.New("加密 Client Secret 失败")
	}
	if app.Status == "" {
		app.Status = model.StatusActive
	}
	if app.OAuthVersion == "" {
		app.OAuthVersion = model.OAuthVersion21
	}
	if app.Protocol == "" {
		app.Protocol = model.ProtocolOAuth
	}
	if err := s.repo.Create(ctx, app); err != nil {
		return "", err
	}
	return clientSecret, nil
}

func (s *appService) GetByID(ctx context.Context, id string) (*model.Application, error) {
	if id == "" {
		return nil, ErrAppIDEmpty
	}
	return s.repo.GetByID(ctx, id)
}

func (s *appService) GetByClientID(ctx context.Context, clientID string) (*model.Application, error) {
	if clientID == "" {
		return nil, errors.New("Client ID 不能为空")
	}
	return s.repo.GetByClientID(ctx, clientID)
}

func (s *appService) Update(ctx context.Context, app *model.Application) error {
	if app.ID == "" {
		return ErrAppIDEmpty
	}
	if app.Name == "" {
		return ErrAppNameEmpty
	}
	// 标准化系统级应用的 OrgID
	if app.OrgID != nil && *app.OrgID == "" {
		app.OrgID = nil
	}
	return s.repo.Update(ctx, app)
}

func (s *appService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrAppIDEmpty
	}
	return s.repo.Delete(ctx, id)
}

func (s *appService) List(ctx context.Context, filter *repository.AppFilter, page *repository.Pagination) ([]*model.Application, int64, error) {
	if page == nil {
		page = &repository.Pagination{Page: 1, PageSize: 20}
	}
	return s.repo.List(ctx, filter, page)
}

func (s *appService) ListByOrgID(ctx context.Context, orgID string, page *repository.Pagination) ([]*model.Application, int64, error) {
	if orgID == "" {
		return nil, 0, ErrAppOrgIDEmpty
	}
	return s.repo.ListByOrgID(ctx, orgID, page)
}

func (s *appService) ResetSecret(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", ErrAppIDEmpty
	}
	// 先检查应用是否存在
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return "", err
	}
	// 生成新密钥
	newSecret, err := model.GenerateClientSecret()
	if err != nil {
		return "", errors.New("生成 Client Secret 失败")
	}
	// 创建临时对象用于生成哈希
	tempApp := &model.Application{}
	if err := tempApp.SetClientSecret(newSecret); err != nil {
		return "", errors.New("加密 Client Secret 失败")
	}
	// 仅更新密钥哈希字段
	if err := s.repo.UpdateSecret(ctx, id, tempApp.ClientSecretHash); err != nil {
		return "", err
	}
	return newSecret, nil
}

func (s *appService) ValidateRedirectURI(ctx context.Context, clientID, redirectURI string) error {
	app, err := s.repo.GetByClientID(ctx, clientID)
	if err != nil {
		return err
	}
	if !app.HasRedirectURI(redirectURI) {
		return errors.New("回调地址不在允许列表中")
	}
	return nil
}

func (s *appService) ValidateClientCredentials(ctx context.Context, clientID, clientSecret string) (*model.Application, error) {
	app, err := s.repo.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if !app.IsActive() {
		return nil, ErrAppDisabled
	}
	if !app.VerifyClientSecret(clientSecret) {
		return nil, errors.New("Client Secret 验证失败")
	}
	return app, nil
}

func (s *appService) validateApp(app *model.Application) error {
	if app == nil {
		return errors.New("应用信息不能为空")
	}
	app.Name = strings.TrimSpace(app.Name)
	if app.Name == "" {
		return ErrAppNameEmpty
	}
	// OrgID 可以为空，表示系统级应用
	return nil
}
