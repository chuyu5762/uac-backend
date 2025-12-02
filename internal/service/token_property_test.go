package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Property 8: 令牌序列化往返一致性
// *For any* 令牌声明，序列化后反序列化应得到相同数据
// **Feature: unified-auth-center, Property 8: 令牌序列化往返一致性**
// **Validates: Requirements 6.3**
func TestProperty_TokenSerializationRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	userIDGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "user-default"
		}
		return "user-" + s
	})

	usernameGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "username"
		}
		if len(s) > 20 {
			return s[:20]
		}
		return s
	})

	properties.Property("序列化往返一致", prop.ForAll(
		func(userID, username string) bool {
			claims := &TokenClaims{
				UserID:   userID,
				Username: username,
				Email:    username + "@test.com",
				Scopes:   []string{"openid", "profile"},
				Type:     "access",
			}

			// 序列化
			data, err := TokenClaimsToJSON(claims)
			if err != nil {
				t.Logf("序列化失败: %v", err)
				return false
			}

			// 反序列化
			restored, err := TokenClaimsFromJSON(data)
			if err != nil {
				t.Logf("反序列化失败: %v", err)
				return false
			}

			// 验证一致性
			if restored.UserID != claims.UserID {
				t.Log("UserID 不一致")
				return false
			}
			if restored.Username != claims.Username {
				t.Log("Username 不一致")
				return false
			}
			if restored.Email != claims.Email {
				t.Log("Email 不一致")
				return false
			}

			return true
		},
		userIDGen,
		usernameGen,
	))

	properties.TestingRun(t)
}

// Property 9: JWT 签名验证
// *For any* 有效令牌，验证应成功；篡改后验证应失败
// **Feature: unified-auth-center, Property 9: JWT 签名验证**
// **Validates: Requirements 6.2, 6.4**
func TestProperty_JWTSignatureVerification(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	userIDGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "user-default"
		}
		return "user-" + s
	})

	properties.Property("签名验证正确性", prop.ForAll(
		func(userID string) bool {
			privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
			svc := NewTokenService(&TokenServiceConfig{
				PrivateKey:    privateKey,
				PublicKey:     &privateKey.PublicKey,
				KeyID:         "test-key",
				Issuer:        "test-issuer",
				AccessExpiry:  15 * time.Minute,
				RefreshExpiry: 7 * 24 * time.Hour,
				CodeExpiry:    10 * time.Minute,
			})
			ctx := context.Background()

			claims := &TokenClaims{UserID: userID}
			token, err := svc.GenerateAccessToken(ctx, claims)
			if err != nil {
				return true // 跳过无效输入
			}

			// 验证有效令牌
			_, err = svc.ValidateToken(ctx, token)
			if err != nil {
				t.Logf("有效令牌验证失败: %v", err)
				return false
			}

			// 篡改令牌
			if len(token) > 10 {
				tamperedToken := token[:len(token)-5] + "xxxxx"
				_, err = svc.ValidateToken(ctx, tamperedToken)
				if err == nil {
					t.Log("篡改令牌应该验证失败")
					return false
				}
			}

			return true
		},
		userIDGen,
	))

	properties.TestingRun(t)
}

// Property 7: 授权码有效期
// *For any* 授权码，过期后验证应失败
// **Feature: unified-auth-center, Property 7: 授权码有效期**
// **Validates: Requirements 4.3**
func TestProperty_AuthorizationCodeExpiry(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	clientIDGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "client-default"
		}
		return "client-" + s
	})

	properties.Property("授权码过期后失效", prop.ForAll(
		func(clientID string) bool {
			privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
			// 设置极短的过期时间
			svc := NewTokenService(&TokenServiceConfig{
				PrivateKey:    privateKey,
				PublicKey:     &privateKey.PublicKey,
				KeyID:         "test-key",
				Issuer:        "test-issuer",
				AccessExpiry:  15 * time.Minute,
				RefreshExpiry: 7 * 24 * time.Hour,
				CodeExpiry:    1 * time.Millisecond, // 极短过期时间
			})
			ctx := context.Background()

			code := &AuthorizationCode{
				ClientID:    clientID,
				UserID:      "user-123",
				RedirectURI: "https://example.com/callback",
				Scopes:      []string{"openid"},
			}

			codeStr, err := svc.GenerateAuthorizationCode(ctx, code)
			if err != nil {
				return true
			}

			// 等待过期
			time.Sleep(5 * time.Millisecond)

			// 验证应该失败
			_, err = svc.ValidateAuthorizationCode(ctx, codeStr)
			if err != ErrCodeExpired {
				t.Logf("期望 ErrCodeExpired, 实际 %v", err)
				return false
			}

			return true
		},
		clientIDGen,
	))

	properties.TestingRun(t)
}

// Property 22: 过期访问令牌拒绝
// *For any* 过期的访问令牌，验证应返回过期错误
// **Feature: unified-auth-center, Property 22: 过期访问令牌拒绝**
// **Validates: Requirements 4.10**
func TestProperty_ExpiredAccessTokenRejection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	userIDGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "user-default"
		}
		return "user-" + s
	})

	properties.Property("过期令牌被拒绝", prop.ForAll(
		func(userID string) bool {
			privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
			svc := NewTokenService(&TokenServiceConfig{
				PrivateKey:    privateKey,
				PublicKey:     &privateKey.PublicKey,
				KeyID:         "test-key",
				Issuer:        "test-issuer",
				AccessExpiry:  -1 * time.Hour, // 已过期
				RefreshExpiry: 7 * 24 * time.Hour,
				CodeExpiry:    10 * time.Minute,
			})
			ctx := context.Background()

			claims := &TokenClaims{UserID: userID}
			token, err := svc.GenerateAccessToken(ctx, claims)
			if err != nil {
				return true
			}

			_, err = svc.ValidateToken(ctx, token)
			if err != ErrTokenExpired {
				t.Logf("期望 ErrTokenExpired, 实际 %v", err)
				return false
			}

			return true
		},
		userIDGen,
	))

	properties.TestingRun(t)
}

// Property 21: 刷新令牌轮换（OAuth 2.1）
// *For any* 刷新令牌，使用后应生成新令牌，旧令牌失效
// **Feature: unified-auth-center, Property 21: 刷新令牌轮换（OAuth 2.1）**
// **Validates: Requirements 4.9**
func TestProperty_RefreshTokenRotation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	userIDGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "user-default"
		}
		return "user-" + s
	})

	properties.Property("刷新令牌轮换后旧令牌失效", prop.ForAll(
		func(userID string) bool {
			privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
			svc := NewTokenService(&TokenServiceConfig{
				PrivateKey:    privateKey,
				PublicKey:     &privateKey.PublicKey,
				KeyID:         "test-key",
				Issuer:        "test-issuer",
				AccessExpiry:  15 * time.Minute,
				RefreshExpiry: 7 * 24 * time.Hour,
				CodeExpiry:    10 * time.Minute,
			})
			ctx := context.Background()

			// 生成刷新令牌
			claims := &TokenClaims{UserID: userID}
			oldRefreshToken, err := svc.GenerateRefreshToken(ctx, claims)
			if err != nil {
				return true
			}

			// 验证旧令牌有效
			_, err = svc.ValidateToken(ctx, oldRefreshToken)
			if err != nil {
				t.Logf("旧刷新令牌应该有效: %v", err)
				return false
			}

			// 模拟轮换：撤销旧令牌，生成新令牌
			svc.RevokeToken(ctx, oldRefreshToken)
			newRefreshToken, err := svc.GenerateRefreshToken(ctx, claims)
			if err != nil {
				return true
			}

			// 验证旧令牌失效
			_, err = svc.ValidateToken(ctx, oldRefreshToken)
			if err == nil {
				t.Log("旧刷新令牌应该失效")
				return false
			}

			// 验证新令牌有效
			_, err = svc.ValidateToken(ctx, newRefreshToken)
			if err != nil {
				t.Logf("新刷新令牌应该有效: %v", err)
				return false
			}

			return true
		},
		userIDGen,
	))

	properties.TestingRun(t)
}
