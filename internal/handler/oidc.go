// Package handler HTTP 处理器
package handler

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

// OIDCHandler OIDC 处理器
type OIDCHandler struct {
	userService  service.UserService
	tokenService service.TokenService
	issuer       string
}

// NewOIDCHandler 创建 OIDC 处理器
func NewOIDCHandler(userSvc service.UserService, tokenSvc service.TokenService, issuer string) *OIDCHandler {
	return &OIDCHandler{
		userService:  userSvc,
		tokenService: tokenSvc,
		issuer:       issuer,
	}
}

// UserInfo 用户信息端点
// GET /oauth/userinfo
func (h *OIDCHandler) UserInfo(c *gin.Context) {
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

	// 获取请求的 scopes
	scopes, _ := c.Get("scopes")
	scopeList, _ := scopes.([]string)

	// 构建响应
	resp := gin.H{
		"sub": user.ID,
	}

	// 根据 scope 返回不同的信息
	if containsScope(scopeList, "profile") {
		resp["name"] = user.DisplayName
		resp["preferred_username"] = user.Username
		if user.AvatarURL != "" {
			resp["picture"] = user.AvatarURL
		}
	}

	if containsScope(scopeList, "email") {
		resp["email"] = user.Email
		resp["email_verified"] = user.EmailVerified
	}

	if containsScope(scopeList, "phone") && user.Phone != "" {
		resp["phone_number"] = user.Phone
		resp["phone_number_verified"] = user.PhoneVerified
	}

	c.JSON(http.StatusOK, resp)
}

// Discovery OIDC 发现文档端点
// GET /.well-known/openid-configuration
func (h *OIDCHandler) Discovery(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                h.issuer,
		"authorization_endpoint":                h.issuer + "/oauth/authorize",
		"token_endpoint":                        h.issuer + "/oauth/token",
		"userinfo_endpoint":                     h.issuer + "/oauth/userinfo",
		"jwks_uri":                              h.issuer + "/.well-known/jwks.json",
		"revocation_endpoint":                   h.issuer + "/oauth/revoke",
		"introspection_endpoint":                h.issuer + "/oauth/introspect",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "phone", "offline_access"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"claims_supported": []string{
			"sub", "iss", "aud", "exp", "iat", "auth_time",
			"name", "preferred_username", "email", "email_verified",
			"phone_number", "phone_number_verified", "picture",
		},
		"code_challenge_methods_supported": []string{"plain", "S256"},
	})
}

// JWKS JSON Web Key Set 端点
// GET /.well-known/jwks.json
func (h *OIDCHandler) JWKS(c *gin.Context) {
	publicKey := h.tokenService.GetPublicKey()
	keyID := h.tokenService.GetKeyID()

	jwk := rsaPublicKeyToJWK(publicKey, keyID)

	c.JSON(http.StatusOK, gin.H{
		"keys": []gin.H{jwk},
	})
}

// rsaPublicKeyToJWK 将 RSA 公钥转换为 JWK 格式
func rsaPublicKeyToJWK(key *rsa.PublicKey, keyID string) gin.H {
	return gin.H{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": keyID,
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}
