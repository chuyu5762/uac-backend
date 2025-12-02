// Package service 业务逻辑层
package service

import (
	"context"
	"errors"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

var (
	ErrRoleNotFound        = errors.New("角色不存在")
	ErrRoleCodeExists      = errors.New("角色代码已存在")
	ErrPermissionNotFound  = errors.New("权限不存在")
	ErrPermissionExists    = errors.New("权限已存在")
	ErrSystemRole          = errors.New("系统内置角色不能删除")
	ErrSystemPermission    = errors.New("系统内置权限不能删除")
	ErrRoleAlreadyAssigned = errors.New("用户已拥有该角色")
)

// RBACService RBAC 服务接口
type RBACService interface {
	// 角色管理
	CreateRole(ctx context.Context, role *model.Role) error
	GetRole(ctx context.Context, id string) (*model.Role, error)
	GetRoleByCode(ctx context.Context, code string) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id string) error
	ListRoles(ctx context.Context, orgID string, page *repository.Pagination) ([]*model.Role, int64, error)

	// 权限管理
	CreatePermission(ctx context.Context, perm *model.Permission) error
	GetPermission(ctx context.Context, id string) (*model.Permission, error)
	DeletePermission(ctx context.Context, id string) error
	ListPermissions(ctx context.Context, orgID string) ([]*model.Permission, error)

	// 角色权限关联
	AddPermissionsToRole(ctx context.Context, roleID string, permissionIDs []string) error
	RemovePermissionsFromRole(ctx context.Context, roleID string, permissionIDs []string) error
	GetRolePermissions(ctx context.Context, roleID string) ([]model.Permission, error)

	// 用户角色
	AssignRole(ctx context.Context, userID, roleID string) error
	AssignRoleByCode(ctx context.Context, userID, roleCode string) error
	RevokeRole(ctx context.Context, userID, roleID string) error
	GetUserRoles(ctx context.Context, userID string) ([]*model.Role, error)
	HasRole(ctx context.Context, userID, roleCode string) (bool, error)

	// 权限检查
	CheckPermission(ctx context.Context, userID, resource, action string) (bool, error)
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)

	// 初始化
	InitDefaultRolesAndPermissions(ctx context.Context) error
}

type rbacService struct {
	roleRepo     repository.RoleRepository
	permRepo     repository.PermissionRepository
	userRoleRepo repository.UserRoleRepository
}

// NewRBACService 创建 RBAC 服务
func NewRBACService(roleRepo repository.RoleRepository, permRepo repository.PermissionRepository, userRoleRepo repository.UserRoleRepository) RBACService {
	return &rbacService{
		roleRepo:     roleRepo,
		permRepo:     permRepo,
		userRoleRepo: userRoleRepo,
	}
}

// 角色管理

func (s *rbacService) CreateRole(ctx context.Context, role *model.Role) error {
	// 检查角色代码是否已存在
	existing, err := s.roleRepo.GetByCode(ctx, role.Code)
	if err == nil && existing != nil {
		return ErrRoleCodeExists
	}

	if role.Status == "" {
		role.Status = model.StatusActive
	}

	return s.roleRepo.Create(ctx, role)
}

func (s *rbacService) GetRole(ctx context.Context, id string) (*model.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

func (s *rbacService) GetRoleByCode(ctx context.Context, code string) (*model.Role, error) {
	role, err := s.roleRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

func (s *rbacService) UpdateRole(ctx context.Context, role *model.Role) error {
	existing, err := s.roleRepo.GetByID(ctx, role.ID)
	if err != nil {
		return ErrRoleNotFound
	}

	// 系统角色不能修改代码
	if existing.IsSystem && role.Code != existing.Code {
		return ErrSystemRole
	}

	return s.roleRepo.Update(ctx, role)
}

func (s *rbacService) DeleteRole(ctx context.Context, id string) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return ErrRoleNotFound
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	return s.roleRepo.Delete(ctx, id)
}

func (s *rbacService) ListRoles(ctx context.Context, orgID string, page *repository.Pagination) ([]*model.Role, int64, error) {
	return s.roleRepo.List(ctx, orgID, page)
}

// 权限管理

func (s *rbacService) CreatePermission(ctx context.Context, perm *model.Permission) error {
	// 自动生成权限代码
	if perm.Code == "" {
		perm.Code = model.BuildPermissionCode(perm.Resource, perm.Action)
	}

	// 检查权限是否已存在
	existing, err := s.permRepo.GetByCode(ctx, perm.Code)
	if err == nil && existing != nil {
		return ErrPermissionExists
	}

	return s.permRepo.Create(ctx, perm)
}

func (s *rbacService) GetPermission(ctx context.Context, id string) (*model.Permission, error) {
	perm, err := s.permRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrPermissionNotFound
	}
	return perm, nil
}

func (s *rbacService) DeletePermission(ctx context.Context, id string) error {
	perm, err := s.permRepo.GetByID(ctx, id)
	if err != nil {
		return ErrPermissionNotFound
	}

	if perm.IsSystem {
		return ErrSystemPermission
	}

	return s.permRepo.Delete(ctx, id)
}

func (s *rbacService) ListPermissions(ctx context.Context, orgID string) ([]*model.Permission, error) {
	return s.permRepo.List(ctx, orgID)
}

// 角色权限关联

func (s *rbacService) AddPermissionsToRole(ctx context.Context, roleID string, permissionIDs []string) error {
	return s.roleRepo.AddPermissions(ctx, roleID, permissionIDs)
}

func (s *rbacService) RemovePermissionsFromRole(ctx context.Context, roleID string, permissionIDs []string) error {
	return s.roleRepo.RemovePermissions(ctx, roleID, permissionIDs)
}

func (s *rbacService) GetRolePermissions(ctx context.Context, roleID string) ([]model.Permission, error) {
	return s.roleRepo.GetPermissions(ctx, roleID)
}

// 用户角色

func (s *rbacService) AssignRole(ctx context.Context, userID, roleID string) error {
	// 检查角色是否存在
	_, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return ErrRoleNotFound
	}

	return s.userRoleRepo.Assign(ctx, userID, roleID)
}

func (s *rbacService) AssignRoleByCode(ctx context.Context, userID, roleCode string) error {
	// 根据角色代码获取角色
	role, err := s.roleRepo.GetByCode(ctx, roleCode)
	if err != nil {
		return ErrRoleNotFound
	}

	return s.userRoleRepo.Assign(ctx, userID, role.ID)
}

func (s *rbacService) RevokeRole(ctx context.Context, userID, roleID string) error {
	return s.userRoleRepo.Revoke(ctx, userID, roleID)
}

func (s *rbacService) GetUserRoles(ctx context.Context, userID string) ([]*model.Role, error) {
	return s.userRoleRepo.GetUserRoles(ctx, userID)
}

func (s *rbacService) HasRole(ctx context.Context, userID, roleCode string) (bool, error) {
	return s.userRoleRepo.HasRole(ctx, userID, roleCode)
}

// 权限检查

func (s *rbacService) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	// 获取用户所有角色
	roles, err := s.userRoleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}

	targetCode := model.BuildPermissionCode(resource, action)
	allCode := model.BuildPermissionCode(resource, model.ActionAll)

	for _, role := range roles {
		// 超级管理员拥有所有权限
		if role.Code == model.RoleSuperAdmin {
			return true, nil
		}

		for _, perm := range role.Permissions {
			// 精确匹配或通配符匹配
			if perm.Code == targetCode || perm.Code == allCode {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *rbacService) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	roles, err := s.userRoleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	permSet := make(map[string]bool)
	for _, role := range roles {
		// 超级管理员返回特殊标记
		if role.Code == model.RoleSuperAdmin {
			return []string{"*:*"}, nil
		}

		for _, perm := range role.Permissions {
			permSet[perm.Code] = true
		}
	}

	permissions := make([]string, 0, len(permSet))
	for code := range permSet {
		permissions = append(permissions, code)
	}
	return permissions, nil
}

// 初始化默认角色和权限

func (s *rbacService) InitDefaultRolesAndPermissions(ctx context.Context) error {
	// 创建默认权限
	defaultPerms := model.DefaultSystemPermissions()
	for _, perm := range defaultPerms {
		existing, _ := s.permRepo.GetByCode(ctx, perm.Code)
		if existing == nil {
			if err := s.permRepo.Create(ctx, &perm); err != nil {
				return err
			}
		}
	}

	// 获取所有权限 ID
	allPerms, err := s.permRepo.List(ctx, "")
	if err != nil {
		return err
	}
	allPermIDs := make([]string, len(allPerms))
	for i, p := range allPerms {
		allPermIDs[i] = p.ID
	}

	// 创建默认角色
	defaultRoles := model.DefaultSystemRoles()
	for _, role := range defaultRoles {
		existing, _ := s.roleRepo.GetByCode(ctx, role.Code)
		if existing == nil {
			if err := s.roleRepo.Create(ctx, &role); err != nil {
				return err
			}

			// 超级管理员拥有所有权限
			if role.Code == model.RoleSuperAdmin {
				createdRole, _ := s.roleRepo.GetByCode(ctx, role.Code)
				if createdRole != nil {
					_ = s.roleRepo.AddPermissions(ctx, createdRole.ID, allPermIDs)
				}
			}
		}
	}

	return nil
}
