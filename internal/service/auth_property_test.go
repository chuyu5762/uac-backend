package service

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/pu-ac-cn/uac-backend/internal/model"
)

// Property 10: 密码哈希验证
// *For any* 有效密码，设置后验证应成功，其他密码验证应失败
// **Feature: unified-auth-center, Property 10: 密码哈希验证**
// **Validates: Requirements 6.1**
func TestProperty_PasswordHashVerification(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// 生成符合强度要求的密码
	strongPasswordGen := gen.SliceOfN(8, gen.AlphaNumChar()).Map(func(chars []rune) string {
		// 确保包含大写、小写、数字
		result := make([]rune, len(chars))
		copy(result, chars)
		if len(result) >= 3 {
			result[0] = 'A' // 大写
			result[1] = 'a' // 小写
			result[2] = '1' // 数字
		}
		return string(result)
	})

	properties.Property("设置密码后验证成功", prop.ForAll(
		func(password string) bool {
			user := &model.User{}
			err := user.SetPassword(password)
			if err != nil {
				return false
			}

			// 正确密码应验证成功
			if !user.VerifyPassword(password) {
				t.Log("正确密码验证失败")
				return false
			}

			// 错误密码应验证失败
			if user.VerifyPassword(password + "wrong") {
				t.Log("错误密码验证成功")
				return false
			}

			return true
		},
		strongPasswordGen,
	))

	properties.TestingRun(t)
}

// Property 11: 登录失败锁定
// *For any* 用户，连续 5 次登录失败后账户应被锁定
// **Feature: unified-auth-center, Property 11: 登录失败锁定**
// **Validates: Requirements 6.5**
func TestProperty_LoginFailureLocking(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// 生成 3-20 个字母的用户名
	usernameGen := gen.SliceOfN(10, gen.AlphaLowerChar()).Map(func(chars []rune) string {
		return "user" + string(chars)
	})

	properties.Property("连续5次失败后锁定", prop.ForAll(
		func(username string) bool {
			userRepo := newMockUserRepository()
			svc := NewAuthService(userRepo)
			ctx := context.Background()

			// 创建用户
			user := &model.User{
				Username:    username,
				Email:       username + "@test.com",
				DisplayName: username,
				Status:      model.StatusActive,
			}
			user.SetPassword("Correct123")
			userRepo.Create(ctx, user)

			// 连续 5 次错误密码
			for i := 0; i < MaxFailedAttempts; i++ {
				_, err := svc.Authenticate(ctx, username, "WrongPass1")
				if err != ErrInvalidCredentials {
					t.Logf("第 %d 次尝试期望 ErrInvalidCredentials", i+1)
					return false
				}
			}

			// 第 6 次即使密码正确也应该锁定
			_, err := svc.Authenticate(ctx, username, "Correct123")
			if err != ErrAccountLocked {
				t.Log("账户应该被锁定")
				return false
			}

			return true
		},
		usernameGen,
	))

	properties.TestingRun(t)
}

// Property: 密码强度验证
// *For any* 密码，只有满足强度要求的密码才能通过检查
// **Feature: unified-auth-center, Property: 密码强度验证**
// **Validates: Requirements 6.1**
func TestProperty_PasswordStrength(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// 生成随机字符串
	anyStringGen := gen.AnyString()

	properties.Property("密码强度检查一致性", prop.ForAll(
		func(password string) bool {
			result := IsPasswordStrong(password)

			// 手动验证
			if len(password) < 8 {
				return result == false
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

			expected := hasUpper && hasLower && hasDigit
			return result == expected
		},
		anyStringGen,
	))

	properties.TestingRun(t)
}
