package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOIDCTestRouter(t *testing.T) (*gin.Engine, *OIDCHandler, service.TokenService) {
	gin.SetMode(gin.TestMode)

	// 生成 RSA 密钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// 创建令牌服务
	tokenService := service.NewTokenService(&service.TokenServiceConfig{
		PrivateKey:    privateKey,
		PublicKey:     &privateKey.PublicKey,
		KeyID:         "test-key-1",
		Issuer:        "http://localhost:8080",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
		CodeExpiry:    10 * time.Minute,
	})

	// 创建 OIDC handler
	oidcHandler := NewOIDCHandler(nil, tokenService, "http://localhost:8080")

	router := gin.New()
	return router, oidcHandler, tokenService
}

func TestOIDCHandler_Discovery(t *testing.T) {
	router, oidcHandler, _ := setupOIDCTestRouter(t)

	router.GET("/.well-known/openid-configuration", oidcHandler.Discovery)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// 验证必需字段
	assert.Equal(t, "http://localhost:8080", resp["issuer"])
	assert.Equal(t, "http://localhost:8080/oauth/authorize", resp["authorization_endpoint"])
	assert.Equal(t, "http://localhost:8080/oauth/token", resp["token_endpoint"])
	assert.Equal(t, "http://localhost:8080/oauth/userinfo", resp["userinfo_endpoint"])
	assert.Equal(t, "http://localhost:8080/.well-known/jwks.json", resp["jwks_uri"])

	// 验证支持的响应类型
	responseTypes, ok := resp["response_types_supported"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, responseTypes, "code")

	// 验证支持的签名算法
	signingAlgs, ok := resp["id_token_signing_alg_values_supported"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, signingAlgs, "RS256")

	// 验证支持的 scopes
	scopes, ok := resp["scopes_supported"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, scopes, "openid")
	assert.Contains(t, scopes, "profile")
	assert.Contains(t, scopes, "email")
}

func TestOIDCHandler_JWKS(t *testing.T) {
	router, oidcHandler, _ := setupOIDCTestRouter(t)

	router.GET("/.well-known/jwks.json", oidcHandler.JWKS)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// 验证 keys 数组存在
	keys, ok := resp["keys"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, keys, 1)

	// 验证第一个 key 的字段
	key := keys[0].(map[string]interface{})
	assert.Equal(t, "RSA", key["kty"])
	assert.Equal(t, "sig", key["use"])
	assert.Equal(t, "RS256", key["alg"])
	assert.Equal(t, "test-key-1", key["kid"])
	assert.NotEmpty(t, key["n"])
	assert.NotEmpty(t, key["e"])
}

func TestOIDCHandler_UserInfo_Unauthorized(t *testing.T) {
	router, oidcHandler, _ := setupOIDCTestRouter(t)

	router.GET("/oauth/userinfo", oidcHandler.UserInfo)

	req := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 没有认证信息应返回错误
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
