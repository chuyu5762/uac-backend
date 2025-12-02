// Package handler HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

// RBACHandler RBAC 处理器
type RBACHandler struct {
	rbacService service.RBACService
}

// NewRBACHandler 创建 RBAC 处理器
func NewRBACHandler(rbacSvc service.RBACService) *RBACHandler {
	return &RBACHandler{rbacService: rbacSvc}
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Code        string   `json:"code" binding:"required"`
	Description string   `json:"description"`
	OrgID       string   `json:"org_id"`
	Permissions []string `json:"permissions"` // 权限 ID 列表
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// AssignRoleRequest 分配角色请求
type AssignRoleRequest struct {
	UserID string `json:"user_id" binding:"required"`
	RoleID string `json:"role_id" binding:"required"`
}

// CreateRole 创建角色
// POST /api/v1/roles
func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	role := &model.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		OrgID:       req.OrgID,
		Status:      model.StatusActive,
	}

	if err := h.rbacService.CreateRole(c.Request.Context(), role); err != nil {
		if err == service.ErrRoleCodeExists {
			response.ErrorWithMsg(c, response.CodeInvalidRequest, "角色代码已存在")
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	// 添加权限
	if len(req.Permissions) > 0 {
		_ = h.rbacService.AddPermissionsToRole(c.Request.Context(), role.ID, req.Permissions)
	}

	response.Success(c, role)
}

// GetRole 获取角色详情
// GET /api/v1/roles/:id
func (h *RBACHandler) GetRole(c *gin.Context) {
	id := c.Param("id")
	role, err := h.rbacService.GetRole(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeRoleNotFound)
		return
	}
	response.Success(c, role)
}

// UpdateRole 更新角色
// PUT /api/v1/roles/:id
func (h *RBACHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	role, err := h.rbacService.GetRole(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeRoleNotFound)
		return
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.Status != "" {
		role.Status = req.Status
	}

	if err := h.rbacService.UpdateRole(c.Request.Context(), role); err != nil {
		if err == service.ErrSystemRole {
			response.ErrorWithMsg(c, response.CodeForbidden, "系统角色不能修改")
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, role)
}

// DeleteRole 删除角色
// DELETE /api/v1/roles/:id
func (h *RBACHandler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	if err := h.rbacService.DeleteRole(c.Request.Context(), id); err != nil {
		if err == service.ErrRoleNotFound {
			response.Error(c, response.CodeRoleNotFound)
			return
		}
		if err == service.ErrSystemRole {
			response.ErrorWithMsg(c, response.CodeForbidden, "系统角色不能删除")
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}
	response.Success(c, gin.H{"message": "删除成功"})
}

// ListRoles 获取角色列表
// GET /api/v1/roles
func (h *RBACHandler) ListRoles(c *gin.Context) {
	orgID := c.Query("org_id")
	page := &repository.Pagination{
		Page:     1,
		PageSize: 20,
	}

	roles, total, err := h.rbacService.ListRoles(c.Request.Context(), orgID, page)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{
		"list":      roles,
		"total":     total,
		"page":      page.Page,
		"page_size": page.PageSize,
	})
}

// ListPermissions 获取权限列表
// GET /api/v1/permissions
func (h *RBACHandler) ListPermissions(c *gin.Context) {
	orgID := c.Query("org_id")
	permissions, err := h.rbacService.ListPermissions(c.Request.Context(), orgID)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}
	response.Success(c, permissions)
}

// GetPermission 获取权限详情
// GET /api/v1/permissions/:id
func (h *RBACHandler) GetPermission(c *gin.Context) {
	id := c.Param("id")
	perm, err := h.rbacService.GetPermission(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodePermissionNotFound)
		return
	}
	response.Success(c, perm)
}

// CreatePermissionRequest 创建权限请求
type CreatePermissionRequest struct {
	Code        string `json:"code"`
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
	Description string `json:"description"`
	OrgID       string `json:"org_id"`
}

// CreatePermission 创建权限
// POST /api/v1/permissions
func (h *RBACHandler) CreatePermission(c *gin.Context) {
	var req CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	perm := &model.Permission{
		Code:        req.Code,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
		OrgID:       req.OrgID,
	}

	if err := h.rbacService.CreatePermission(c.Request.Context(), perm); err != nil {
		if err == service.ErrPermissionExists {
			response.ErrorWithMsg(c, response.CodeInvalidRequest, "权限已存在")
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, perm)
}

// DeletePermission 删除权限
// DELETE /api/v1/permissions/:id
func (h *RBACHandler) DeletePermission(c *gin.Context) {
	id := c.Param("id")
	if err := h.rbacService.DeletePermission(c.Request.Context(), id); err != nil {
		if err == service.ErrPermissionNotFound {
			response.Error(c, response.CodePermissionNotFound)
			return
		}
		if err == service.ErrSystemPermission {
			response.ErrorWithMsg(c, response.CodeForbidden, "系统权限不能删除")
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}
	response.Success(c, gin.H{"message": "删除成功"})
}

// GetRolePermissions 获取角色权限
// GET /api/v1/roles/:id/permissions
func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	roleID := c.Param("id")
	permissions, err := h.rbacService.GetRolePermissions(c.Request.Context(), roleID)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}
	response.Success(c, permissions)
}

// AssignRole 分配角色给用户
// POST /api/v1/users/:user_id/roles
func (h *RBACHandler) AssignRole(c *gin.Context) {
	userID := c.Param("user_id")
	var req struct {
		RoleID string `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.rbacService.AssignRole(c.Request.Context(), userID, req.RoleID); err != nil {
		if err == service.ErrRoleNotFound {
			response.Error(c, response.CodeRoleNotFound)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{"message": "角色分配成功"})
}

// RevokeRole 撤销用户角色
// DELETE /api/v1/users/:user_id/roles/:role_id
func (h *RBACHandler) RevokeRole(c *gin.Context) {
	userID := c.Param("user_id")
	roleID := c.Param("role_id")

	if err := h.rbacService.RevokeRole(c.Request.Context(), userID, roleID); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{"message": "角色撤销成功"})
}

// GetUserRoles 获取用户角色
// GET /api/v1/users/:user_id/roles
func (h *RBACHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("user_id")
	roles, err := h.rbacService.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}
	response.Success(c, roles)
}

// GetCurrentUserPermissions 获取当前用户权限
// GET /api/v1/auth/permissions
func (h *RBACHandler) GetCurrentUserPermissions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.CodeInvalidToken)
		return
	}

	roles, err := h.rbacService.GetUserRoles(c.Request.Context(), userID.(string))
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	roleCodes := make([]string, len(roles))
	accessCodes := make([]string, 0)
	for i, role := range roles {
		roleCodes[i] = role.Code
		// 如果是超级管理员或组织管理员，添加 admin 权限码
		if role.Code == model.RoleSuperAdmin || role.Code == model.RoleOrgAdmin {
			accessCodes = append(accessCodes, "admin")
		}
	}

	// 去重
	accessMap := make(map[string]bool)
	var uniqueAccessCodes []string
	for _, code := range accessCodes {
		if !accessMap[code] {
			accessMap[code] = true
			uniqueAccessCodes = append(uniqueAccessCodes, code)
		}
	}

	response.Success(c, gin.H{
		"permissions": uniqueAccessCodes,
		"roles":       roleCodes,
	})
}

// AddPermissionsToRole 添加权限到角色
// POST /api/v1/roles/:id/permissions
func (h *RBACHandler) AddPermissionsToRole(c *gin.Context) {
	roleID := c.Param("id")
	var req struct {
		PermissionIDs []string `json:"permission_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.rbacService.AddPermissionsToRole(c.Request.Context(), roleID, req.PermissionIDs); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{"message": "权限添加成功"})
}

// RemovePermissionsFromRole 从角色移除权限
// DELETE /api/v1/roles/:id/permissions
func (h *RBACHandler) RemovePermissionsFromRole(c *gin.Context) {
	roleID := c.Param("id")
	var req struct {
		PermissionIDs []string `json:"permission_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.rbacService.RemovePermissionsFromRole(c.Request.Context(), roleID, req.PermissionIDs); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{"message": "权限移除成功"})
}
