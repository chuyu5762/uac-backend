// Package middleware 中间件
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

// RequirePermission 权限检查中间件
// 检查当前用户是否拥有指定的权限
func RequirePermission(rbacService service.RBACService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户 ID
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.CodeInvalidToken)
			c.Abort()
			return
		}

		// 检查权限
		hasPermission, err := rbacService.CheckPermission(c.Request.Context(), userID.(string), resource, action)
		if err != nil {
			response.Error(c, response.CodeServerError)
			c.Abort()
			return
		}

		if !hasPermission {
			response.ErrorWithMsg(c, response.CodeForbidden, "没有权限执行此操作")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole 角色检查中间件
// 检查当前用户是否拥有指定的角色
func RequireRole(rbacService service.RBACService, roleCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.CodeInvalidToken)
			c.Abort()
			return
		}

		hasRole, err := rbacService.HasRole(c.Request.Context(), userID.(string), roleCode)
		if err != nil {
			response.Error(c, response.CodeServerError)
			c.Abort()
			return
		}

		if !hasRole {
			response.ErrorWithMsg(c, response.CodeForbidden, "没有权限执行此操作")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole 任一角色检查中间件
// 检查当前用户是否拥有任一指定角色
func RequireAnyRole(rbacService service.RBACService, roleCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.CodeInvalidToken)
			c.Abort()
			return
		}

		for _, roleCode := range roleCodes {
			hasRole, err := rbacService.HasRole(c.Request.Context(), userID.(string), roleCode)
			if err != nil {
				continue
			}
			if hasRole {
				c.Next()
				return
			}
		}

		response.ErrorWithMsg(c, response.CodeForbidden, "没有权限执行此操作")
		c.Abort()
	}
}

// LoadUserPermissions 加载用户权限到上下文
// 用于在需要时获取用户的所有权限
func LoadUserPermissions(rbacService service.RBACService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		permissions, err := rbacService.GetUserPermissions(c.Request.Context(), userID.(string))
		if err == nil {
			c.Set("permissions", permissions)
		}

		roles, err := rbacService.GetUserRoles(c.Request.Context(), userID.(string))
		if err == nil {
			roleCodes := make([]string, len(roles))
			for i, role := range roles {
				roleCodes[i] = role.Code
			}
			c.Set("roles", roleCodes)
		}

		c.Next()
	}
}
