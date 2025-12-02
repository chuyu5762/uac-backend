// Package repository 数据访问层
package repository

import (
	"context"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"gorm.io/gorm"
)

// RoleRepository 角色仓库接口
type RoleRepository interface {
	Create(ctx context.Context, role *model.Role) error
	GetByID(ctx context.Context, id string) (*model.Role, error)
	GetByCode(ctx context.Context, code string) (*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, orgID string, page *Pagination) ([]*model.Role, int64, error)
	AddPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	RemovePermissions(ctx context.Context, roleID string, permissionIDs []string) error
	GetPermissions(ctx context.Context, roleID string) ([]model.Permission, error)
}

// PermissionRepository 权限仓库接口
type PermissionRepository interface {
	Create(ctx context.Context, perm *model.Permission) error
	GetByID(ctx context.Context, id string) (*model.Permission, error)
	GetByCode(ctx context.Context, code string) (*model.Permission, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, orgID string) ([]*model.Permission, error)
	BatchCreate(ctx context.Context, perms []model.Permission) error
}

// UserRoleRepository 用户角色仓库接口
type UserRoleRepository interface {
	Assign(ctx context.Context, userID, roleID string) error
	Revoke(ctx context.Context, userID, roleID string) error
	GetUserRoles(ctx context.Context, userID string) ([]*model.Role, error)
	GetRoleUsers(ctx context.Context, roleID string, page *Pagination) ([]*model.User, int64, error)
	HasRole(ctx context.Context, userID, roleCode string) (bool, error)
}

// roleRepository 角色仓库实现
type roleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 创建角色仓库
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *roleRepository) GetByID(ctx context.Context, id string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) GetByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "code = ?", code).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *roleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Role{}, "id = ?", id).Error
}

func (r *roleRepository) List(ctx context.Context, orgID string, page *Pagination) ([]*model.Role, int64, error) {
	var roles []*model.Role
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Role{})
	if orgID != "" {
		query = query.Where("org_id = ? OR org_id = ''", orgID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page != nil {
		query = query.Offset((page.Page - 1) * page.PageSize).Limit(page.PageSize)
	}

	if err := query.Preload("Permissions").Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

func (r *roleRepository) AddPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		return err
	}

	var permissions []model.Permission
	if err := r.db.WithContext(ctx).Find(&permissions, "id IN ?", permissionIDs).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&role).Association("Permissions").Append(permissions)
}

func (r *roleRepository) RemovePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		return err
	}

	var permissions []model.Permission
	if err := r.db.WithContext(ctx).Find(&permissions, "id IN ?", permissionIDs).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&role).Association("Permissions").Delete(permissions)
}

func (r *roleRepository) GetPermissions(ctx context.Context, roleID string) ([]model.Permission, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", roleID).Error; err != nil {
		return nil, err
	}
	return role.Permissions, nil
}

// permissionRepository 权限仓库实现
type permissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository 创建权限仓库
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(ctx context.Context, perm *model.Permission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

func (r *permissionRepository) GetByID(ctx context.Context, id string) (*model.Permission, error) {
	var perm model.Permission
	if err := r.db.WithContext(ctx).First(&perm, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *permissionRepository) GetByCode(ctx context.Context, code string) (*model.Permission, error) {
	var perm model.Permission
	if err := r.db.WithContext(ctx).First(&perm, "code = ?", code).Error; err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *permissionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Permission{}, "id = ?", id).Error
}

func (r *permissionRepository) List(ctx context.Context, orgID string) ([]*model.Permission, error) {
	var perms []*model.Permission
	query := r.db.WithContext(ctx).Model(&model.Permission{})
	if orgID != "" {
		query = query.Where("org_id = ? OR org_id = ''", orgID)
	}
	if err := query.Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *permissionRepository) BatchCreate(ctx context.Context, perms []model.Permission) error {
	return r.db.WithContext(ctx).CreateInBatches(perms, 100).Error
}

// userRoleRepository 用户角色仓库实现
type userRoleRepository struct {
	db *gorm.DB
}

// NewUserRoleRepository 创建用户角色仓库
func NewUserRoleRepository(db *gorm.DB) UserRoleRepository {
	return &userRoleRepository{db: db}
}

func (r *userRoleRepository) Assign(ctx context.Context, userID, roleID string) error {
	userRole := &model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.WithContext(ctx).Create(userRole).Error
}

func (r *userRoleRepository) Revoke(ctx context.Context, userID, roleID string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserRole{}).Error
}

func (r *userRoleRepository) GetUserRoles(ctx context.Context, userID string) ([]*model.Role, error) {
	var userRoles []model.UserRole
	if err := r.db.WithContext(ctx).Preload("Role.Permissions").Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}

	roles := make([]*model.Role, 0, len(userRoles))
	for _, ur := range userRoles {
		if ur.Role != nil {
			roles = append(roles, ur.Role)
		}
	}
	return roles, nil
}

func (r *userRoleRepository) GetRoleUsers(ctx context.Context, roleID string, page *Pagination) ([]*model.User, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.UserRole{}).Where("role_id = ?", roleID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var userRoles []model.UserRole
	query := r.db.WithContext(ctx).Preload("User").Where("role_id = ?", roleID)
	if page != nil {
		query = query.Offset((page.Page - 1) * page.PageSize).Limit(page.PageSize)
	}
	if err := query.Find(&userRoles).Error; err != nil {
		return nil, 0, err
	}

	users := make([]*model.User, 0, len(userRoles))
	for _, ur := range userRoles {
		if ur.User != nil {
			users = append(users, ur.User)
		}
	}
	return users, total, nil
}

func (r *userRoleRepository) HasRole(ctx context.Context, userID, roleCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserRole{}).
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.code = ?", userID, roleCode).
		Count(&count).Error
	return count > 0, err
}
