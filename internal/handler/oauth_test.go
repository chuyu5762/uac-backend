package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAppService 模拟应用服务
type mockAppService struct {
	apps map[string]*model.Application
}

func newMockAppService() *mockAppService {
	return &mockAppService{
		apps: make(map[string]*model.Application),
	}
}

func (m *mockAppService) GetByClientID(clientID string) (*model.Application, error) {
	if app, ok := m.apps[clientID]; ok {
		return app, nil
	}
	return nil, service.ErrAppIDEmpty
}

func (m *mockAppService) AddApp(app *model.Application) {
	m.apps[app.ClientID] = app
}

func setupOAuthTestRouter(t *testing.T) (*gin.Engine, *OAuthHandler, service.TokenService) {
	gin.SetMode(gin.TestMode)

	// 生成 RSA 密钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// 创建令牌服务
	tokenService := service.NewTokenService(&service.TokenServiceConfig{
		PrivateKey:    privateKey,
		PublicKey:     &privateKey.PublicKey,
		KeyID:         "test-key",
		Issuer:        "http://localhost:8080",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
		CodeExpiry:    10 * time.Minute,
	})

	// 创建 OAuth handler（使用 nil 作为 appService 和 sessionService，因为我们只测试部分功能）
	oauthHandler := &OAuthHandler{
		tokenService: tokenService,
	}

	router := gin.New()
	return router, oauthHandler, tokenService
}

func TestOAuthHandler_Token_RefreshToken(t *testing.T) {
	router, oauthHandler, tokenService := setupOAuthTestRouter(t)

	router.POST("/oauth/token", oauthHandler.Token)

	// 先生成一个刷新令牌
	claims := &service.TokenClaims{
		UserID:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Scopes:   []string{"openid", "profile"},
	}
	refreshToken, err := tokenService.GenerateRefreshToken(nil, claims)
	require.NoError(t, err)

	// 使用刷新令牌获取新令牌
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
	assert.Equal(t, "Bearer", resp["token_type"])
}

func TestOAuthHandler_Token_UnsupportedGrantType(t *testing.T) {
	router, oauthHandler, _ := setupOAuthTestRouter(t)

	router.POST("/oauth/token", oauthHandler.Token)

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", "test")
	form.Set("password", "test123")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "unsupported_grant_type", resp["error"])
}

func TestOAuthHandler_Token_InvalidRefreshToken(t *testing.T) {
	router, oauthHandler, _ := setupOAuthTestRouter(t)

	router.POST("/oauth/token", oauthHandler.Token)

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "invalid_grant", resp["error"])
}

func TestOAuthHandler_Revoke(t *testing.T) {
	router, oauthHandler, tokenService := setupOAuthTestRouter(t)

	router.POST("/oauth/revoke", oauthHandler.Revoke)

	// 生成一个令牌
	claims := &service.TokenClaims{
		UserID: "user-123",
	}
	accessToken, _ := tokenService.GenerateAccessToken(nil, claims)

	// 撤销令牌
	form := url.Values{}
	form.Set("token", accessToken)

	req := httptest.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// RFC 7009: 无论令牌是否有效，都返回 200
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOAuthHandler_Introspect(t *testing.T) {
	router, oauthHandler, tokenService := setupOAuthTestRouter(t)

	router.POST("/oauth/introspect", oauthHandler.Introspect)

	// 生成一个有效令牌
	claims := &service.TokenClaims{
		UserID:   "user-123",
		Username: "testuser",
		Scopes:   []string{"openid", "profile"},
	}
	accessToken, _ := tokenService.GenerateAccessToken(nil, claims)

	// 内省令牌
	form := url.Values{}
	form.Set("token", accessToken)

	req := httptest.NewRequest(http.MethodPost, "/oauth/introspect", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, true, resp["active"])
	assert.Equal(t, "testuser", resp["username"])
}

func TestOAuthHandler_Introspect_InvalidToken(t *testing.T) {
	router, oauthHandler, _ := setupOAuthTestRouter(t)

	router.POST("/oauth/introspect", oauthHandler.Introspect)

	form := url.Values{}
	form.Set("token", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/oauth/introspect", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, false, resp["active"])
}

func TestVerifyPKCE_Plain(t *testing.T) {
	h := &OAuthHandler{}

	verifier := "test-verifier-12345"
	challenge := verifier

	assert.True(t, h.verifyPKCE(challenge, "plain", verifier))
	assert.True(t, h.verifyPKCE(challenge, "", verifier))
	assert.False(t, h.verifyPKCE(challenge, "plain", "wrong-verifier"))
}

func TestVerifyPKCE_S256(t *testing.T) {
	h := &OAuthHandler{}

	// 使用已知的测试向量
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	assert.True(t, h.verifyPKCE(challenge, "S256", verifier))
	assert.False(t, h.verifyPKCE(challenge, "S256", "wrong-verifier"))
}
