// Package service 认证服务
package service

import (
	"context"
	"errors"
	"time"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

// 认证相关错误
var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrAccountLocked      = errors.New("账户已锁定，请稍后再试")
	ErrAccountDisabled    = errors.New("账户已禁用")
	ErrUserNotFound       = errors.New("用户不存在")
)

// AuthService 认证服务接口
type AuthService interface {
	// Authenticate 验证用户凭据
	Authenticate(ctx context.Context, username, password string) (*model.User, error)
	// AuthenticateByEmail 通过邮箱验证用户凭据
	AuthenticateByEmail(ctx context.Context, email, password string) (*model.User, error)
	// ChangePassword 修改密码
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	// ResetPassword 重置密码（管理员操作）
	ResetPassword(ctx context.Context, userID, newPassword string) error
	// UnlockAccount 解锁账户
	UnlockAccount(ctx context.Context, userID string) error
}

// authService 认证服务实现
type authService struct {
	userRepo repository.UserRepository
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo: userRepo}
}

// Authenticate 验证用户凭据
func (s *authService) Authenticate(ctx context.Context, username, password string) (*model.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.validateAndAuthenticate(ctx, user, password)
}

// AuthenticateByEmail 通过邮箱验证用户凭据
func (s *authService) AuthenticateByEmail(ctx context.Context, email, password string) (*model.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.validateAndAuthenticate(ctx, user, password)
}

// validateAndAuthenticate 验证用户并执行认证
func (s *authService) validateAndAuthenticate(ctx context.Context, user *model.User, password string) (*model.User, error) {
	// 检查账户是否被锁定
	if user.IsLocked() {
		return nil, ErrAccountLocked
	}

	// 检查账户是否被禁用
	if !user.IsActive() {
		return nil, ErrAccountDisabled
	}

	// 验证密码
	if !user.VerifyPassword(password) {
		// 增加失败次数
		user.IncrementFailedLogin()
		_ = s.userRepo.Update(ctx, user)
		return nil, ErrInvalidCredentials
	}

	// 登录成功，重置失败次数
	if user.FailedLoginCount > 0 {
		user.ResetFailedLogin()
		_ = s.userRepo.Update(ctx, user)
	}

	return user, nil
}

// ChangePassword 修改密码
func (s *authService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证旧密码
	if !user.VerifyPassword(oldPassword) {
		return ErrInvalidCredentials
	}

	// 设置新密码
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	return s.userRepo.Update(ctx, user)
}

// ResetPassword 重置密码（管理员操作）
func (s *authService) ResetPassword(ctx context.Context, userID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 设置新密码
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	// 重置登录失败次数和锁定状态
	user.ResetFailedLogin()

	return s.userRepo.Update(ctx, user)
}

// UnlockAccount 解锁账户
func (s *authService) UnlockAccount(ctx context.Context, userID string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	user.LockedUntil = nil
	user.FailedLoginCount = 0

	return s.userRepo.Update(ctx, user)
}

// IsPasswordStrong 检查密码强度
// 密码要求：最小 8 位，包含大写字母、小写字母、数字
func IsPasswordStrong(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}

	return hasUpper && hasLower && hasDigit
}

// LockDuration 账户锁定时长
const LockDuration = 15 * time.Minute

// MaxFailedAttempts 最大失败尝试次数
const MaxFailedAttempts = 5
