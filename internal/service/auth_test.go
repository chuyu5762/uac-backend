package service

import (
	"context"
	"testing"

	"github.com/pu-ac-cn/uac-backend/internal/model"
)

// TestAuthService_Authenticate 测试用户认证
func TestAuthService_Authenticate(t *testing.T) {
	userRepo := newMockUserRepository()
	svc := NewAuthService(userRepo)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "测试用户",
		Status:      model.StatusActive,
	}
	user.SetPassword("Test1234")
	userRepo.Create(ctx, user)

	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{
			name:     "正确凭据",
			username: "testuser",
			password: "Test1234",
			wantErr:  nil,
		},
		{
			name:     "错误密码",
			username: "testuser",
			password: "wrongpassword",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "用户不存在",
			username: "nonexistent",
			password: "Test1234",
			wantErr:  ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Authenticate(ctx, tt.username, tt.password)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("期望错误 %v, 实际 %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误, 实际 %v", err)
				}
				if result == nil {
					t.Error("期望返回用户, 实际 nil")
				}
			}
		})
	}
}

// TestAuthService_AccountLocking 测试账户锁定
func TestAuthService_AccountLocking(t *testing.T) {
	userRepo := newMockUserRepository()
	svc := NewAuthService(userRepo)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:    "locktest",
		Email:       "lock@example.com",
		DisplayName: "锁定测试",
		Status:      model.StatusActive,
	}
	user.SetPassword("Test1234")
	userRepo.Create(ctx, user)

	// 连续 5 次错误密码
	for i := 0; i < 5; i++ {
		_, err := svc.Authenticate(ctx, "locktest", "wrongpassword")
		if err != ErrInvalidCredentials {
			t.Errorf("第 %d 次尝试期望 ErrInvalidCredentials, 实际 %v", i+1, err)
		}
	}

	// 第 6 次应该返回账户锁定
	_, err := svc.Authenticate(ctx, "locktest", "Test1234")
	if err != ErrAccountLocked {
		t.Errorf("期望 ErrAccountLocked, 实际 %v", err)
	}
}

// TestAuthService_ChangePassword 测试修改密码
func TestAuthService_ChangePassword(t *testing.T) {
	userRepo := newMockUserRepository()
	svc := NewAuthService(userRepo)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:    "pwdchange",
		Email:       "pwd@example.com",
		DisplayName: "密码修改测试",
		Status:      model.StatusActive,
	}
	user.SetPassword("OldPass123")
	userRepo.Create(ctx, user)

	// 使用错误的旧密码
	err := svc.ChangePassword(ctx, user.ID, "wrongold", "NewPass456")
	if err != ErrInvalidCredentials {
		t.Errorf("期望 ErrInvalidCredentials, 实际 %v", err)
	}

	// 使用正确的旧密码
	err = svc.ChangePassword(ctx, user.ID, "OldPass123", "NewPass456")
	if err != nil {
		t.Errorf("不期望错误, 实际 %v", err)
	}

	// 验证新密码可用
	_, err = svc.Authenticate(ctx, "pwdchange", "NewPass456")
	if err != nil {
		t.Errorf("新密码认证失败: %v", err)
	}

	// 验证旧密码不可用
	_, err = svc.Authenticate(ctx, "pwdchange", "OldPass123")
	if err != ErrInvalidCredentials {
		t.Errorf("旧密码应该失效")
	}
}

// TestAuthService_UnlockAccount 测试解锁账户
func TestAuthService_UnlockAccount(t *testing.T) {
	userRepo := newMockUserRepository()
	svc := NewAuthService(userRepo)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:    "unlocktest",
		Email:       "unlock@example.com",
		DisplayName: "解锁测试",
		Status:      model.StatusActive,
	}
	user.SetPassword("Test1234")
	userRepo.Create(ctx, user)

	// 锁定账户
	for i := 0; i < 5; i++ {
		svc.Authenticate(ctx, "unlocktest", "wrong")
	}

	// 确认已锁定
	_, err := svc.Authenticate(ctx, "unlocktest", "Test1234")
	if err != ErrAccountLocked {
		t.Errorf("期望 ErrAccountLocked, 实际 %v", err)
	}

	// 解锁账户
	err = svc.UnlockAccount(ctx, user.ID)
	if err != nil {
		t.Errorf("解锁失败: %v", err)
	}

	// 验证可以登录
	_, err = svc.Authenticate(ctx, "unlocktest", "Test1234")
	if err != nil {
		t.Errorf("解锁后登录失败: %v", err)
	}
}

// TestIsPasswordStrong 测试密码强度检查
func TestIsPasswordStrong(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{"Test1234", true},
		{"Abc12345", true},
		{"PASSWORD1", false}, // 无小写
		{"password1", false}, // 无大写
		{"Password", false},  // 无数字
		{"Test123", false},   // 太短
		{"Ab1", false},       // 太短
		{"", false},          // 空
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			got := IsPasswordStrong(tt.password)
			if got != tt.want {
				t.Errorf("IsPasswordStrong(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}
