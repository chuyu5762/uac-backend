package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"
)

// 创建测试用的令牌服务
func newTestTokenService() TokenService {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return NewTokenService(&TokenServiceConfig{
		PrivateKey:    privateKey,
		PublicKey:     &privateKey.PublicKey,
		KeyID:         "test-key-1",
		Issuer:        "test-issuer",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
		CodeExpiry:    10 * time.Minute,
	})
}

// TestTokenService_GenerateAccessToken 测试生成访问令牌
func TestTokenService_GenerateAccessToken(t *testing.T) {
	svc := newTestTokenService()
	ctx := context.Background()

	claims := &TokenClaims{
		UserID:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Scopes:   []string{"openid", "profile"},
	}

	token, err := svc.GenerateAccessToken(ctx, claims)
	if err != nil {
		t.Fatalf("生成访问令牌失败: %v", err)
	}

	if token == "" {
		t.Error("令牌不应为空")
	}

	// 验证令牌
	validatedClaims, err := svc.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("验证令牌失败: %v", err)
	}

	if validatedClaims.UserID != claims.UserID {
		t.Errorf("UserID 不匹配: 期望 %s, 实际 %s", claims.UserID, validatedClaims.UserID)
	}

	if validatedClaims.Type != "access" {
		t.Errorf("Type 不匹配: 期望 access, 实际 %s", validatedClaims.Type)
	}
}

// TestTokenService_GenerateRefreshToken 测试生成刷新令牌
func TestTokenService_GenerateRefreshToken(t *testing.T) {
	svc := newTestTokenService()
	ctx := context.Background()

	claims := &TokenClaims{
		UserID: "user-123",
	}

	token, err := svc.GenerateRefreshToken(ctx, claims)
	if err != nil {
		t.Fatalf("生成刷新令牌失败: %v", err)
	}

	validatedClaims, err := svc.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("验证令牌失败: %v", err)
	}

	if validatedClaims.Type != "refresh" {
		t.Errorf("Type 不匹配: 期望 refresh, 实际 %s", validatedClaims.Type)
	}
}

// TestTokenService_ValidateExpiredToken 测试验证过期令牌
func TestTokenService_ValidateExpiredToken(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	svc := NewTokenService(&TokenServiceConfig{
		PrivateKey:    privateKey,
		PublicKey:     &privateKey.PublicKey,
		KeyID:         "test-key-1",
		Issuer:        "test-issuer",
		AccessExpiry:  -1 * time.Hour, // 已过期
		RefreshExpiry: 7 * 24 * time.Hour,
		CodeExpiry:    10 * time.Minute,
	})
	ctx := context.Background()

	claims := &TokenClaims{UserID: "user-123"}
	token, _ := svc.GenerateAccessToken(ctx, claims)

	_, err := svc.ValidateToken(ctx, token)
	if err != ErrTokenExpired {
		t.Errorf("期望 ErrTokenExpired, 实际 %v", err)
	}
}

// TestTokenService_RevokeToken 测试撤销令牌
func TestTokenService_RevokeToken(t *testing.T) {
	svc := newTestTokenService()
	ctx := context.Background()

	claims := &TokenClaims{UserID: "user-123"}
	token, _ := svc.GenerateAccessToken(ctx, claims)

	// 验证令牌有效
	_, err := svc.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("令牌应该有效: %v", err)
	}

	// 撤销令牌
	svc.RevokeToken(ctx, token)

	// 验证令牌已失效
	_, err = svc.ValidateToken(ctx, token)
	if err != ErrInvalidToken {
		t.Errorf("期望 ErrInvalidToken, 实际 %v", err)
	}
}

// TestTokenService_AuthorizationCode 测试授权码
func TestTokenService_AuthorizationCode(t *testing.T) {
	svc := newTestTokenService()
	ctx := context.Background()

	code := &AuthorizationCode{
		ClientID:    "client-123",
		UserID:      "user-123",
		RedirectURI: "https://example.com/callback",
		Scopes:      []string{"openid", "profile"},
	}

	codeStr, err := svc.GenerateAuthorizationCode(ctx, code)
	if err != nil {
		t.Fatalf("生成授权码失败: %v", err)
	}

	// 验证授权码
	validatedCode, err := svc.ValidateAuthorizationCode(ctx, codeStr)
	if err != nil {
		t.Fatalf("验证授权码失败: %v", err)
	}

	if validatedCode.ClientID != code.ClientID {
		t.Errorf("ClientID 不匹配")
	}

	// 再次使用应该失败
	_, err = svc.ValidateAuthorizationCode(ctx, codeStr)
	if err != ErrCodeUsed {
		t.Errorf("期望 ErrCodeUsed, 实际 %v", err)
	}
}

// TestTokenClaimsSerialization 测试令牌声明序列化
func TestTokenClaimsSerialization(t *testing.T) {
	claims := &TokenClaims{
		UserID:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Scopes:   []string{"openid", "profile"},
		Type:     "access",
	}

	// 序列化
	data, err := TokenClaimsToJSON(claims)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 反序列化
	restored, err := TokenClaimsFromJSON(data)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if restored.UserID != claims.UserID {
		t.Errorf("UserID 不匹配")
	}
	if restored.Username != claims.Username {
		t.Errorf("Username 不匹配")
	}
	if len(restored.Scopes) != len(claims.Scopes) {
		t.Errorf("Scopes 长度不匹配")
	}
}
