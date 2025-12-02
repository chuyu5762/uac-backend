// Package handler HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	userService  service.UserService
	authService  service.AuthService
	tokenService service.TokenService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(userSvc service.UserService, authSvc service.AuthService, tokenSvc service.TokenService) *AuthHandler {
	return &AuthHandler{
		userService:  userSvc,
		authService:  authSvc,
		tokenService: tokenSvc,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"` // 用户名或邮箱
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse 令牌响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // 秒
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	// 检查密码强度
	if !service.IsPasswordStrong(req.Password) {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "密码强度不足，需要至少8位，包含大写字母、小写字母和数字")
		return
	}

	// 创建用户
	user := &model.User{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
		Status:      model.StatusActive,
	}

	if err := h.userService.Create(c.Request.Context(), user, req.Password); err != nil {
		if err.Error() == "用户名已存在" {
			response.Error(c, response.CodeUserExists)
			return
		}
		if err.Error() == "邮箱已存在" {
			response.Error(c, response.CodeEmailExists)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"message":  "注册成功",
	})
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	// 必须提供用户名或邮箱
	if req.Username == "" && req.Email == "" {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "请提供用户名或邮箱")
		return
	}

	var user *model.User
	var err error

	// 根据用户名或邮箱认证
	if req.Email != "" {
		user, err = h.authService.AuthenticateByEmail(c.Request.Context(), req.Email, req.Password)
	} else {
		user, err = h.authService.Authenticate(c.Request.Context(), req.Username, req.Password)
	}

	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			response.Error(c, response.CodeInvalidCredentials)
		case service.ErrAccountLocked:
			response.Error(c, response.CodeAccountLocked)
		case service.ErrAccountDisabled:
			response.Error(c, response.CodeForbidden)
		default:
			response.Error(c, response.CodeServerError)
		}
		return
	}

	// 生成令牌
	claims := &service.TokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Scopes:   []string{"openid", "profile", "email"},
	}

	accessToken, err := h.tokenService.GenerateAccessToken(c.Request.Context(), claims)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken(c.Request.Context(), claims)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 分钟
	})
}

// RefreshToken 刷新令牌
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeInvalidRequest)
		return
	}

	// 验证刷新令牌
	claims, err := h.tokenService.ValidateToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, response.CodeInvalidRefreshToken)
		return
	}

	if claims.Type != "refresh" {
		response.Error(c, response.CodeInvalidRefreshToken)
		return
	}

	// 撤销旧的刷新令牌（轮换）
	h.tokenService.RevokeToken(c.Request.Context(), req.RefreshToken)

	// 生成新令牌
	newClaims := &service.TokenClaims{
		UserID:   claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Scopes:   claims.Scopes,
	}

	accessToken, _ := h.tokenService.GenerateAccessToken(c.Request.Context(), newClaims)
	refreshToken, _ := h.tokenService.GenerateRefreshToken(c.Request.Context(), newClaims)

	response.Success(c, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
	})
}

// Logout 用户登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从 Authorization 头获取令牌
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
		h.tokenService.RevokeToken(c.Request.Context(), token)
	}

	response.Success(c, gin.H{"message": "登出成功"})
}

// GetCurrentUser 获取当前用户信息
// GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// 从上下文获取用户 ID（由认证中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.CodeInvalidToken)
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), userID.(string))
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
	})
}
