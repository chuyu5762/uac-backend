package service

import (
	"context"
	"testing"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

type mockUserRepository struct {
	users       map[string]*model.User
	usernameMap map[string]string
	emailMap    map[string]string
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:       make(map[string]*model.User),
		usernameMap: make(map[string]string),
		emailMap:    make(map[string]string),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *model.User) error {
	if _, exists := m.usernameMap[user.Username]; exists {
		return repository.ErrUserUsernameExists
	}
	if _, exists := m.emailMap[user.Email]; exists {
		return repository.ErrUserEmailExists
	}
	user.ID = "test-user-" + user.Username
	m.users[user.ID] = user
	m.usernameMap[user.Username] = user.ID
	m.emailMap[user.Email] = user.ID
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if id, exists := m.usernameMap[username]; exists {
		return m.users[id], nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if id, exists := m.emailMap[email]; exists {
		return m.users[id], nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepository) Update(ctx context.Context, user *model.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return repository.ErrUserNotFound
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	if user, exists := m.users[id]; exists {
		delete(m.usernameMap, user.Username)
		delete(m.emailMap, user.Email)
		delete(m.users, id)
		return nil
	}
	return repository.ErrUserNotFound
}

func (m *mockUserRepository) List(ctx context.Context, filter *repository.UserFilter, page *repository.Pagination) ([]*model.User, int64, error) {
	var result []*model.User
	for _, user := range m.users {
		result = append(result, user)
	}
	return result, int64(len(result)), nil
}

func (m *mockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	_, exists := m.usernameMap[username]
	return exists, nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, exists := m.emailMap[email]
	return exists, nil
}

type mockBindingRepository struct {
	bindings map[string]*model.UserOrgBinding
}

func newMockBindingRepository() *mockBindingRepository {
	return &mockBindingRepository{bindings: make(map[string]*model.UserOrgBinding)}
}

func (m *mockBindingRepository) Create(ctx context.Context, binding *model.UserOrgBinding) error {
	key := binding.UserID + "-" + binding.OrgID
	if _, exists := m.bindings[key]; exists {
		return repository.ErrBindingExists
	}
	binding.ID = "binding-" + key
	m.bindings[key] = binding
	return nil
}

func (m *mockBindingRepository) Delete(ctx context.Context, userID, orgID string) error {
	key := userID + "-" + orgID
	if _, exists := m.bindings[key]; !exists {
		return repository.ErrBindingNotFound
	}
	delete(m.bindings, key)
	return nil
}

func (m *mockBindingRepository) GetByUserAndOrg(ctx context.Context, userID, orgID string) (*model.UserOrgBinding, error) {
	key := userID + "-" + orgID
	if binding, exists := m.bindings[key]; exists {
		return binding, nil
	}
	return nil, repository.ErrBindingNotFound
}

func (m *mockBindingRepository) ListByUserID(ctx context.Context, userID string) ([]*model.UserOrgBinding, error) {
	var result []*model.UserOrgBinding
	for _, binding := range m.bindings {
		if binding.UserID == userID {
			result = append(result, binding)
		}
	}
	return result, nil
}

func (m *mockBindingRepository) ListByOrgID(ctx context.Context, orgID string, page *repository.Pagination) ([]*model.UserOrgBinding, int64, error) {
	var result []*model.UserOrgBinding
	for _, binding := range m.bindings {
		if binding.OrgID == orgID {
			result = append(result, binding)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockBindingRepository) Exists(ctx context.Context, userID, orgID string) (bool, error) {
	key := userID + "-" + orgID
	_, exists := m.bindings[key]
	return exists, nil
}

func TestUserService_Create(t *testing.T) {
	userRepo := newMockUserRepository()
	bindingRepo := newMockBindingRepository()
	svc := NewUserService(userRepo, bindingRepo, nil)
	ctx := context.Background()

	user := &model.User{Username: "testuser", Email: "test@example.com", DisplayName: "Test User"}
	err := svc.Create(ctx, user, "password123")
	if err != nil {
		t.Errorf("创建用户失败: %v", err)
	}
	if user.ID == "" {
		t.Error("期望生成用户 ID")
	}
}

func TestUserService_Authenticate(t *testing.T) {
	userRepo := newMockUserRepository()
	bindingRepo := newMockBindingRepository()
	svc := NewUserService(userRepo, bindingRepo, nil)
	ctx := context.Background()

	user := &model.User{Username: "authuser", Email: "auth@example.com"}
	_ = svc.Create(ctx, user, "password123")

	_, err := svc.Authenticate(ctx, "authuser", "password123")
	if err != nil {
		t.Errorf("认证失败: %v", err)
	}

	_, err = svc.Authenticate(ctx, "authuser", "wrongpassword")
	if err == nil {
		t.Error("错误密码应该认证失败")
	}
}

func TestUserService_BindOrganization(t *testing.T) {
	userRepo := newMockUserRepository()
	bindingRepo := newMockBindingRepository()
	orgRepo := newMockOrgRepository()
	svc := NewUserService(userRepo, bindingRepo, orgRepo)
	orgSvc := NewOrganizationService(orgRepo)
	ctx := context.Background()

	user := &model.User{Username: "binduser", Email: "bind@example.com"}
	_ = svc.Create(ctx, user, "password123")

	org := &model.Organization{Name: "测试组织", Slug: "bind-test-org"}
	_ = orgSvc.Create(ctx, org)

	err := svc.BindOrganization(ctx, user.ID, org.ID)
	if err != nil {
		t.Errorf("绑定组织失败: %v", err)
	}

	hasAccess, _ := svc.HasOrgAccess(ctx, user.ID, org.ID)
	if !hasAccess {
		t.Error("绑定后应该有访问权限")
	}

	err = svc.UnbindOrganization(ctx, user.ID, org.ID)
	if err != nil {
		t.Errorf("解绑组织失败: %v", err)
	}

	hasAccess, _ = svc.HasOrgAccess(ctx, user.ID, org.ID)
	if hasAccess {
		t.Error("解绑后不应该有访问权限")
	}
}
