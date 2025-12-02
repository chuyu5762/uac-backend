package service

import (
	"context"
	"testing"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRoleRepository 角色仓库 Mock
type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, role *model.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id string) (*model.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Role), args.Error(1)
}

func (m *MockRoleRepository) GetByCode(ctx context.Context, code string) (*model.Role, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Role), args.Error(1)
}

func (m *MockRoleRepository) Update(ctx context.Context, role *model.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepository) List(ctx context.Context, orgID string, page *repository.Pagination) ([]*model.Role, int64, error) {
	args := m.Called(ctx, orgID, page)
	return args.Get(0).([]*model.Role), args.Get(1).(int64), args.Error(2)
}

func (m *MockRoleRepository) AddPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	args := m.Called(ctx, roleID, permissionIDs)
	return args.Error(0)
}

func (m *MockRoleRepository) RemovePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	args := m.Called(ctx, roleID, permissionIDs)
	return args.Error(0)
}

func (m *MockRoleRepository) GetPermissions(ctx context.Context, roleID string) ([]model.Permission, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]model.Permission), args.Error(1)
}

// MockPermissionRepository 权限仓库 Mock
type MockPermissionRepository struct {
	mock.Mock
}

func (m *MockPermissionRepository) Create(ctx context.Context, perm *model.Permission) error {
	args := m.Called(ctx, perm)
	return args.Error(0)
}

func (m *MockPermissionRepository) GetByID(ctx context.Context, id string) (*model.Permission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Permission), args.Error(1)
}

func (m *MockPermissionRepository) GetByCode(ctx context.Context, code string) (*model.Permission, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Permission), args.Error(1)
}

func (m *MockPermissionRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPermissionRepository) List(ctx context.Context, orgID string) ([]*model.Permission, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]*model.Permission), args.Error(1)
}

func (m *MockPermissionRepository) BatchCreate(ctx context.Context, perms []model.Permission) error {
	args := m.Called(ctx, perms)
	return args.Error(0)
}

// MockUserRoleRepository 用户角色仓库 Mock
type MockUserRoleRepository struct {
	mock.Mock
}

func (m *MockUserRoleRepository) Assign(ctx context.Context, userID, roleID string) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) Revoke(ctx context.Context, userID, roleID string) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) GetUserRoles(ctx context.Context, userID string) ([]*model.Role, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*model.Role), args.Error(1)
}

func (m *MockUserRoleRepository) GetRoleUsers(ctx context.Context, roleID string, page *repository.Pagination) ([]*model.User, int64, error) {
	args := m.Called(ctx, roleID, page)
	return args.Get(0).([]*model.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRoleRepository) HasRole(ctx context.Context, userID, roleCode string) (bool, error) {
	args := m.Called(ctx, userID, roleCode)
	return args.Bool(0), args.Error(1)
}

// 测试用例

func TestRBACService_CreateRole(t *testing.T) {
	ctx := context.Background()
	roleRepo := new(MockRoleRepository)
	permRepo := new(MockPermissionRepository)
	userRoleRepo := new(MockUserRoleRepository)

	svc := NewRBACService(roleRepo, permRepo, userRoleRepo)

	role := &model.Role{
		Name: "测试角色",
		Code: "test_role",
	}

	// 角色不存在，创建成功
	roleRepo.On("GetByCode", ctx, "test_role").Return(nil, ErrRoleNotFound).Once()
	roleRepo.On("Create", ctx, role).Return(nil).Once()

	err := svc.CreateRole(ctx, role)
	assert.NoError(t, err)
	roleRepo.AssertExpectations(t)
}

func TestRBACService_CreateRole_CodeExists(t *testing.T) {
	ctx := context.Background()
	roleRepo := new(MockRoleRepository)
	permRepo := new(MockPermissionRepository)
	userRoleRepo := new(MockUserRoleRepository)

	svc := NewRBACService(roleRepo, permRepo, userRoleRepo)

	existingRole := &model.Role{
		Name: "已存在角色",
		Code: "existing_role",
	}

	role := &model.Role{
		Name: "新角色",
		Code: "existing_role",
	}

	roleRepo.On("GetByCode", ctx, "existing_role").Return(existingRole, nil).Once()

	err := svc.CreateRole(ctx, role)
	assert.Equal(t, ErrRoleCodeExists, err)
	roleRepo.AssertExpectations(t)
}

func TestRBACService_DeleteRole_SystemRole(t *testing.T) {
	ctx := context.Background()
	roleRepo := new(MockRoleRepository)
	permRepo := new(MockPermissionRepository)
	userRoleRepo := new(MockUserRoleRepository)

	svc := NewRBACService(roleRepo, permRepo, userRoleRepo)

	systemRole := &model.Role{
		BaseModel: model.BaseModel{ID: "role-1"},
		Name:      "超级管理员",
		Code:      model.RoleSuperAdmin,
		IsSystem:  true,
	}

	roleRepo.On("GetByID", ctx, "role-1").Return(systemRole, nil).Once()

	err := svc.DeleteRole(ctx, "role-1")
	assert.Equal(t, ErrSystemRole, err)
	roleRepo.AssertExpectations(t)
}

func TestRBACService_CheckPermission_SuperAdmin(t *testing.T) {
	ctx := context.Background()
	roleRepo := new(MockRoleRepository)
	permRepo := new(MockPermissionRepository)
	userRoleRepo := new(MockUserRoleRepository)

	svc := NewRBACService(roleRepo, permRepo, userRoleRepo)

	superAdminRole := &model.Role{
		Code: model.RoleSuperAdmin,
	}

	userRoleRepo.On("GetUserRoles", ctx, "user-1").Return([]*model.Role{superAdminRole}, nil).Once()

	hasPermission, err := svc.CheckPermission(ctx, "user-1", "user", "delete")
	assert.NoError(t, err)
	assert.True(t, hasPermission)
	userRoleRepo.AssertExpectations(t)
}

func TestRBACService_CheckPermission_WithPermission(t *testing.T) {
	ctx := context.Background()
	roleRepo := new(MockRoleRepository)
	permRepo := new(MockPermissionRepository)
	userRoleRepo := new(MockUserRoleRepository)

	svc := NewRBACService(roleRepo, permRepo, userRoleRepo)

	role := &model.Role{
		Code: "org_admin",
		Permissions: []model.Permission{
			{Code: "user:read"},
			{Code: "user:write"},
		},
	}

	userRoleRepo.On("GetUserRoles", ctx, "user-1").Return([]*model.Role{role}, nil).Once()

	hasPermission, err := svc.CheckPermission(ctx, "user-1", "user", "read")
	assert.NoError(t, err)
	assert.True(t, hasPermission)
	userRoleRepo.AssertExpectations(t)
}

func TestRBACService_CheckPermission_NoPermission(t *testing.T) {
	ctx := context.Background()
	roleRepo := new(MockRoleRepository)
	permRepo := new(MockPermissionRepository)
	userRoleRepo := new(MockUserRoleRepository)

	svc := NewRBACService(roleRepo, permRepo, userRoleRepo)

	role := &model.Role{
		Code: "user",
		Permissions: []model.Permission{
			{Code: "user:read"},
		},
	}

	userRoleRepo.On("GetUserRoles", ctx, "user-1").Return([]*model.Role{role}, nil).Once()

	hasPermission, err := svc.CheckPermission(ctx, "user-1", "user", "delete")
	assert.NoError(t, err)
	assert.False(t, hasPermission)
	userRoleRepo.AssertExpectations(t)
}

func TestRBACService_AssignRole(t *testing.T) {
	ctx := context.Background()
	roleRepo := new(MockRoleRepository)
	permRepo := new(MockPermissionRepository)
	userRoleRepo := new(MockUserRoleRepository)

	svc := NewRBACService(roleRepo, permRepo, userRoleRepo)

	role := &model.Role{
		BaseModel: model.BaseModel{ID: "role-1"},
		Code:      "user",
	}

	roleRepo.On("GetByID", ctx, "role-1").Return(role, nil).Once()
	userRoleRepo.On("Assign", ctx, "user-1", "role-1").Return(nil).Once()

	err := svc.AssignRole(ctx, "user-1", "role-1")
	assert.NoError(t, err)
	roleRepo.AssertExpectations(t)
	userRoleRepo.AssertExpectations(t)
}
