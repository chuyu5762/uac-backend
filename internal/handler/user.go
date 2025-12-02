// Package handler HTTP 处理器
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler 创建用户管理处理器
func NewUserHandler(userSvc service.UserService) *UserHandler {
	return &UserHandler{userService: userSvc}
}

// ListUsers 获取用户列表
// GET /api/v1/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 解析过滤参数
	filter := &repository.UserFilter{
		Username: c.Query("username"),
		Email:    c.Query("email"),
		Status:   c.Query("status"),
	}

	pagination := &repository.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	users, total, err := h.userService.List(c.Request.Context(), filter, pagination)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	// 转换为响应格式（隐藏敏感字段）
	list := make([]gin.H, len(users))
	for i, user := range users {
		list[i] = gin.H{
			"id":             user.ID,
			"username":       user.Username,
			"email":          user.Email,
			"display_name":   user.DisplayName,
			"phone":          user.Phone,
			"status":         user.Status,
			"email_verified": user.EmailVerified,
			"phone_verified": user.PhoneVerified,
			"created_at":     user.CreatedAt,
			"updated_at":     user.UpdatedAt,
		}
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetUser 获取用户详情
// GET /api/v1/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeUserNotFound)
		return
	}

	response.Success(c, gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"display_name":   user.DisplayName,
		"phone":          user.Phone,
		"status":         user.Status,
		"email_verified": user.EmailVerified,
		"phone_verified": user.PhoneVerified,
		"created_at":     user.CreatedAt,
		"updated_at":     user.UpdatedAt,
	})
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=3"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
	Status      string `json:"status"`
}

// CreateUser 创建用户
// POST /api/v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	// 检查密码强度
	if !service.IsPasswordStrong(req.Password) {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "密码强度不足，需要至少8位，包含大写字母、小写字母和数字")
		return
	}

	user := &model.User{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
	}

	if err := h.userService.Create(c.Request.Context(), user, req.Password); err != nil {
		// 检查是否是重复用户
		if err.Error() == "用户名已存在" || err.Error() == "邮箱已存在" {
			response.ErrorWithMsg(c, response.CodeUserExists, err.Error())
			return
		}
		response.ErrorWithMsg(c, response.CodeServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"phone":        user.Phone,
		"status":       user.Status,
		"created_at":   user.CreatedAt,
	})
}

// UpdateUser 更新用户
// PUT /api/v1/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeUserNotFound)
		return
	}

	// 更新字段
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	if err := h.userService.Update(c.Request.Context(), user); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"phone":        user.Phone,
		"status":       user.Status,
	})
}

// DeleteUser 删除用户
// DELETE /api/v1/users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// 不能删除自己
	currentUserID, _ := c.Get("user_id")
	if currentUserID == id {
		response.ErrorWithMsg(c, response.CodeForbidden, "不能删除自己")
		return
	}

	if err := h.userService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword 修改密码
// POST /api/v1/auth/change-password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.CodeInvalidToken)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	// 检查密码强度
	if !service.IsPasswordStrong(req.NewPassword) {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "密码强度不足，需要至少8位，包含大写字母、小写字母和数字")
		return
	}

	if err := h.userService.ChangePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword); err != nil {
		if err == service.ErrPasswordIncorrect {
			response.ErrorWithMsg(c, response.CodeInvalidCredentials, "原密码错误")
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{"message": "密码修改成功"})
}

// UpdateCurrentUser 更新当前用户信息
// PUT /api/v1/auth/me
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.CodeInvalidToken)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		response.Error(c, response.CodeUserNotFound)
		return
	}

	// 只允许更新部分字段
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	// 不允许用户自己修改状态

	if err := h.userService.Update(c.Request.Context(), user); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"phone":        user.Phone,
		"status":       user.Status,
	})
}
