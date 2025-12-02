// Package model 定义数据模型
package model

// Role 角色模型
type Role struct {
	BaseModel
	OrgID       string `gorm:"type:char(36);index" json:"org_id"`             // 所属组织 ID，空表示系统级角色
	Name        string `gorm:"type:varchar(100);not null" json:"name"`        // 角色名称
	Code        string `gorm:"type:varchar(50);uniqueIndex" json:"code"`      // 角色代码，如 super_admin, org_admin, user
	Description string `gorm:"type:varchar(500)" json:"description"`          // 角色描述
	IsSystem    bool   `gorm:"default:false" json:"is_system"`                // 是否系统内置角色
	Status      string `gorm:"type:varchar(20);default:active" json:"status"` // 状态

	// 关联
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// TableName 指定表名
func (Role) TableName() string {
	return "roles"
}

// IsActive 检查角色是否启用
func (r *Role) IsActive() bool {
	return r.Status == StatusActive
}

// Permission 权限模型
type Permission struct {
	BaseModel
	OrgID       string `gorm:"type:char(36);index" json:"org_id"`          // 所属组织 ID，空表示系统级权限
	Resource    string `gorm:"type:varchar(100);not null" json:"resource"` // 资源，如 user, role, app
	Action      string `gorm:"type:varchar(50);not null" json:"action"`    // 操作，如 read, write, delete
	Code        string `gorm:"type:varchar(150);uniqueIndex" json:"code"`  // 权限代码，格式：resource:action
	Description string `gorm:"type:varchar(500)" json:"description"`       // 权限描述
	IsSystem    bool   `gorm:"default:false" json:"is_system"`             // 是否系统内置权限
}

// TableName 指定表名
func (Permission) TableName() string {
	return "permissions"
}

// UserRole 用户角色关联模型
type UserRole struct {
	BaseModel
	UserID string `gorm:"type:char(36);index;not null" json:"user_id"` // 用户 ID
	RoleID string `gorm:"type:char(36);index;not null" json:"role_id"` // 角色 ID

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role *Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// TableName 指定表名
func (UserRole) TableName() string {
	return "user_roles"
}

// RolePermission 角色权限关联模型（GORM 自动创建，这里显式定义以便查询）
type RolePermission struct {
	RoleID       string `gorm:"type:char(36);primaryKey" json:"role_id"`
	PermissionID string `gorm:"type:char(36);primaryKey" json:"permission_id"`
}

// TableName 指定表名
func (RolePermission) TableName() string {
	return "role_permissions"
}

// 系统内置角色代码
const (
	RoleSuperAdmin = "super_admin" // 超级管理员
	RoleOrgAdmin   = "org_admin"   // 组织管理员
	RoleUser       = "user"        // 普通用户
)

// 系统内置权限资源
const (
	ResourceUser = "user" // 用户资源
	ResourceRole = "role" // 角色资源
	ResourceOrg  = "org"  // 组织资源
	ResourceApp  = "app"  // 应用资源
)

// 系统内置权限操作
const (
	ActionRead   = "read"   // 读取
	ActionWrite  = "write"  // 写入（创建/更新）
	ActionDelete = "delete" // 删除
	ActionAll    = "*"      // 所有操作
)

// BuildPermissionCode 构建权限代码
func BuildPermissionCode(resource, action string) string {
	return resource + ":" + action
}

// DefaultSystemPermissions 系统默认权限列表
func DefaultSystemPermissions() []Permission {
	resources := []string{ResourceUser, ResourceRole, ResourceOrg, ResourceApp}
	actions := []string{ActionRead, ActionWrite, ActionDelete}

	var permissions []Permission
	for _, resource := range resources {
		for _, action := range actions {
			permissions = append(permissions, Permission{
				Resource:    resource,
				Action:      action,
				Code:        BuildPermissionCode(resource, action),
				Description: resource + " " + action + " 权限",
				IsSystem:    true,
			})
		}
	}
	return permissions
}

// DefaultSystemRoles 系统默认角色列表
func DefaultSystemRoles() []Role {
	return []Role{
		{
			Name:        "超级管理员",
			Code:        RoleSuperAdmin,
			Description: "拥有系统所有权限",
			IsSystem:    true,
			Status:      StatusActive,
		},
		{
			Name:        "组织管理员",
			Code:        RoleOrgAdmin,
			Description: "管理组织内的用户和应用",
			IsSystem:    true,
			Status:      StatusActive,
		},
		{
			Name:        "普通用户",
			Code:        RoleUser,
			Description: "基本访问权限",
			IsSystem:    true,
			Status:      StatusActive,
		},
	}
}
