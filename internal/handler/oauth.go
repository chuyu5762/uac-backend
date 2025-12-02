// Package handler HTTP 处理器
package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

// OAuthHandler OAuth 2.0/2.1 处理器
type OAuthHandler struct {
	appService     service.ApplicationService
	tokenService   service.TokenService
	sessionService service.SessionService
}

// NewOAuthHandler 创建 OAuth 处理器
func NewOAuthHandler(appSvc service.ApplicationService, tokenSvc service.TokenService, sessionSvc service.SessionService) *OAuthHandler {
	return &OAuthHandler{
		appService:     appSvc,
		tokenService:   tokenSvc,
		sessionService: sessionSvc,
	}
}

// AuthorizeRequest 授权请求参数
type AuthorizeRequest struct {
	ResponseType        string `form:"response_type" binding:"required"`
	ClientID            string `form:"client_id" binding:"required"`
	RedirectURI         string `form:"redirect_uri" binding:"required"`
	Scope               string `form:"scope"`
	State               string `form:"state"`
	CodeChallenge       string `form:"code_challenge"`
	CodeChallengeMethod string `form:"code_challenge_method"`
	Nonce               string `form:"nonce"` // OIDC
}

// TokenRequest 令牌请求参数
type TokenRequest struct {
	GrantType    string `form:"grant_type" binding:"required"`
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	CodeVerifier string `form:"code_verifier"`
	RefreshToken string `form:"refresh_token"`
	Scope        string `form:"scope"`
}

// Authorize 授权端点
// GET /oauth/authorize
func (h *OAuthHandler) Authorize(c *gin.Context) {
	var req AuthorizeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.redirectError(c, req.RedirectURI, "invalid_request", "参数错误", req.State)
		return
	}

	// 验证客户端
	app, err := h.appService.GetByClientID(c.Request.Context(), req.ClientID)
	if err != nil {
		h.redirectError(c, req.RedirectURI, "invalid_client", "客户端不存在", req.State)
		return
	}

	// 验证重定向 URI
	if !h.isValidRedirectURI(app.RedirectURIs, req.RedirectURI) {
		h.redirectError(c, req.RedirectURI, "invalid_request", "重定向 URI 无效", req.State)
		return
	}

	// 验证响应类型
	if req.ResponseType != "code" {
		// OAuth 2.1 不支持隐式模式
		if req.ResponseType == "token" && app.OAuthVersion == "2.1" {
			h.redirectError(c, req.RedirectURI, "unsupported_response_type", "OAuth 2.1 不支持隐式模式", req.State)
			return
		}
		if req.ResponseType != "token" {
			h.redirectError(c, req.RedirectURI, "unsupported_response_type", "不支持的响应类型", req.State)
			return
		}
	}

	// OAuth 2.1 强制要求 PKCE
	if app.OAuthVersion == "2.1" && req.CodeChallenge == "" {
		h.redirectError(c, req.RedirectURI, "invalid_request", "OAuth 2.1 要求使用 PKCE", req.State)
		return
	}

	// 验证 code_challenge_method
	if req.CodeChallenge != "" {
		if req.CodeChallengeMethod == "" {
			req.CodeChallengeMethod = "plain"
		}
		if req.CodeChallengeMethod != "plain" && req.CodeChallengeMethod != "S256" {
			h.redirectError(c, req.RedirectURI, "invalid_request", "不支持的 code_challenge_method", req.State)
			return
		}
	}

	// 验证权限范围
	requestedScopes := strings.Split(req.Scope, " ")
	if !h.isValidScopes(app.AllowedScopes, requestedScopes) {
		h.redirectError(c, req.RedirectURI, "invalid_scope", "请求的权限范围无效", req.State)
		return
	}

	// 检查用户是否已登录
	userID, exists := c.Get("user_id")
	if !exists {
		// 重定向到登录页面，登录后返回
		loginURL := "/login?redirect=" + url.QueryEscape(c.Request.URL.String())
		c.Redirect(http.StatusFound, loginURL)
		return
	}

	// 生成授权码
	authCode := &service.AuthorizationCode{
		ClientID:            req.ClientID,
		UserID:              userID.(string),
		RedirectURI:         req.RedirectURI,
		Scopes:              strings.Split(req.Scope, " "),
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	}

	code, err := h.tokenService.GenerateAuthorizationCode(c.Request.Context(), authCode)
	if err != nil {
		h.redirectError(c, req.RedirectURI, "server_error", "生成授权码失败", req.State)
		return
	}

	// 重定向回客户端
	redirectURL, _ := url.Parse(req.RedirectURI)
	query := redirectURL.Query()
	query.Set("code", code)
	if req.State != "" {
		query.Set("state", req.State)
	}
	redirectURL.RawQuery = query.Encode()

	c.Redirect(http.StatusFound, redirectURL.String())
}

// Token 令牌端点
// POST /oauth/token
func (h *OAuthHandler) Token(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBind(&req); err != nil {
		h.tokenError(c, "invalid_request", "参数错误")
		return
	}

	switch req.GrantType {
	case "authorization_code":
		h.handleAuthorizationCode(c, &req)
	case "refresh_token":
		h.handleRefreshToken(c, &req)
	case "password":
		// OAuth 2.1 不支持密码模式
		h.tokenError(c, "unsupported_grant_type", "不支持密码模式")
	case "client_credentials":
		h.handleClientCredentials(c, &req)
	default:
		h.tokenError(c, "unsupported_grant_type", "不支持的授权类型")
	}
}

// handleAuthorizationCode 处理授权码模式
func (h *OAuthHandler) handleAuthorizationCode(c *gin.Context, req *TokenRequest) {
	if req.Code == "" {
		h.tokenError(c, "invalid_request", "缺少授权码")
		return
	}

	// 验证授权码
	authCode, err := h.tokenService.ValidateAuthorizationCode(c.Request.Context(), req.Code)
	if err != nil {
		if err == service.ErrCodeExpired {
			h.tokenError(c, "invalid_grant", "授权码已过期")
			return
		}
		if err == service.ErrCodeUsed {
			h.tokenError(c, "invalid_grant", "授权码已被使用")
			return
		}
		h.tokenError(c, "invalid_grant", "授权码无效")
		return
	}

	// 验证客户端
	app, err := h.appService.GetByClientID(c.Request.Context(), authCode.ClientID)
	if err != nil {
		h.tokenError(c, "invalid_client", "客户端不存在")
		return
	}

	// 验证客户端凭证
	if req.ClientID != "" && req.ClientID != authCode.ClientID {
		h.tokenError(c, "invalid_client", "客户端 ID 不匹配")
		return
	}

	// 验证 Client Secret（如果提供）
	if req.ClientSecret != "" {
		if !app.VerifyClientSecret(req.ClientSecret) {
			h.tokenError(c, "invalid_client", "客户端密钥错误")
			return
		}
	}

	// 验证重定向 URI
	if req.RedirectURI != "" && req.RedirectURI != authCode.RedirectURI {
		h.tokenError(c, "invalid_grant", "重定向 URI 不匹配")
		return
	}

	// 验证 PKCE
	if authCode.CodeChallenge != "" {
		if req.CodeVerifier == "" {
			h.tokenError(c, "invalid_request", "缺少 code_verifier")
			return
		}
		if !h.verifyPKCE(authCode.CodeChallenge, authCode.CodeChallengeMethod, req.CodeVerifier) {
			h.tokenError(c, "invalid_grant", "PKCE 验证失败")
			return
		}
	}

	// 生成令牌
	claims := &service.TokenClaims{
		UserID:   authCode.UserID,
		ClientID: authCode.ClientID,
		Scopes:   authCode.Scopes,
	}

	accessToken, err := h.tokenService.GenerateAccessToken(c.Request.Context(), claims)
	if err != nil {
		h.tokenError(c, "server_error", "生成访问令牌失败")
		return
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken(c.Request.Context(), claims)
	if err != nil {
		h.tokenError(c, "server_error", "生成刷新令牌失败")
		return
	}

	// 构建响应
	scope := strings.Join(authCode.Scopes, " ")
	resp := gin.H{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    900, // 15 分钟
		"refresh_token": refreshToken,
		"scope":         scope,
	}

	// 如果请求了 openid scope，生成 ID Token
	if containsScope(authCode.Scopes, "openid") {
		idToken, err := h.tokenService.GenerateIDToken(c.Request.Context(), claims)
		if err == nil {
			resp["id_token"] = idToken
		}
	}

	c.JSON(http.StatusOK, resp)
}

// handleRefreshToken 处理刷新令牌
func (h *OAuthHandler) handleRefreshToken(c *gin.Context, req *TokenRequest) {
	if req.RefreshToken == "" {
		h.tokenError(c, "invalid_request", "缺少刷新令牌")
		return
	}

	// 验证刷新令牌
	claims, err := h.tokenService.ValidateToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.tokenError(c, "invalid_grant", "刷新令牌无效或已过期")
		return
	}

	if claims.Type != "refresh" {
		h.tokenError(c, "invalid_grant", "无效的令牌类型")
		return
	}

	// 撤销旧的刷新令牌（轮换）
	h.tokenService.RevokeToken(c.Request.Context(), req.RefreshToken)

	// 生成新令牌
	newClaims := &service.TokenClaims{
		UserID:   claims.UserID,
		ClientID: claims.ClientID,
		Username: claims.Username,
		Email:    claims.Email,
		Scopes:   claims.Scopes,
	}

	accessToken, _ := h.tokenService.GenerateAccessToken(c.Request.Context(), newClaims)
	refreshToken, _ := h.tokenService.GenerateRefreshToken(c.Request.Context(), newClaims)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    900,
		"refresh_token": refreshToken,
		"scope":         strings.Join(claims.Scopes, " "),
	})
}

// handleClientCredentials 处理客户端凭证模式
func (h *OAuthHandler) handleClientCredentials(c *gin.Context, req *TokenRequest) {
	if req.ClientID == "" || req.ClientSecret == "" {
		h.tokenError(c, "invalid_client", "缺少客户端凭证")
		return
	}

	// 验证客户端
	app, err := h.appService.GetByClientID(c.Request.Context(), req.ClientID)
	if err != nil {
		h.tokenError(c, "invalid_client", "客户端不存在")
		return
	}

	if !app.VerifyClientSecret(req.ClientSecret) {
		h.tokenError(c, "invalid_client", "客户端密钥错误")
		return
	}

	// 生成访问令牌（无用户上下文）
	claims := &service.TokenClaims{
		ClientID: req.ClientID,
		Scopes:   strings.Split(req.Scope, " "),
	}

	accessToken, err := h.tokenService.GenerateAccessToken(c.Request.Context(), claims)
	if err != nil {
		h.tokenError(c, "server_error", "生成访问令牌失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   900,
		"scope":        req.Scope,
	})
}

// Revoke 令牌撤销端点
// POST /oauth/revoke
func (h *OAuthHandler) Revoke(c *gin.Context) {
	token := c.PostForm("token")
	if token == "" {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "缺少令牌")
		return
	}

	// 撤销令牌
	h.tokenService.RevokeToken(c.Request.Context(), token)

	// RFC 7009: 无论令牌是否有效，都返回 200
	c.JSON(http.StatusOK, gin.H{})
}

// Introspect 令牌内省端点
// POST /oauth/introspect
func (h *OAuthHandler) Introspect(c *gin.Context) {
	token := c.PostForm("token")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	// 验证令牌
	claims, err := h.tokenService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active":     true,
		"scope":      strings.Join(claims.Scopes, " "),
		"client_id":  claims.ClientID,
		"username":   claims.Username,
		"token_type": "Bearer",
		"exp":        claims.ExpiresAt,
		"iat":        claims.IssuedAt,
		"sub":        claims.UserID,
		"iss":        claims.Issuer,
	})
}

// 辅助方法

// redirectError 重定向错误响应
func (h *OAuthHandler) redirectError(c *gin.Context, redirectURI, errorCode, errorDesc, state string) {
	if redirectURI == "" {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, errorDesc)
		return
	}

	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, errorDesc)
		return
	}

	query := redirectURL.Query()
	query.Set("error", errorCode)
	query.Set("error_description", errorDesc)
	if state != "" {
		query.Set("state", state)
	}
	redirectURL.RawQuery = query.Encode()

	c.Redirect(http.StatusFound, redirectURL.String())
}

// tokenError 令牌端点错误响应
func (h *OAuthHandler) tokenError(c *gin.Context, errorCode, errorDesc string) {
	status := http.StatusBadRequest
	if errorCode == "invalid_client" {
		status = http.StatusUnauthorized
	}

	c.JSON(status, gin.H{
		"error":             errorCode,
		"error_description": errorDesc,
	})
}

// isValidRedirectURI 验证重定向 URI
func (h *OAuthHandler) isValidRedirectURI(allowedURIs []string, uri string) bool {
	for _, allowed := range allowedURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

// isValidScopes 验证权限范围
func (h *OAuthHandler) isValidScopes(allowedScopes, requestedScopes []string) bool {
	allowedSet := make(map[string]bool)
	for _, s := range allowedScopes {
		allowedSet[s] = true
	}

	for _, s := range requestedScopes {
		if s == "" {
			continue
		}
		if !allowedSet[s] {
			return false
		}
	}
	return true
}

// verifyPKCE 验证 PKCE
func (h *OAuthHandler) verifyPKCE(challenge, method, verifier string) bool {
	if method == "plain" || method == "" {
		return challenge == verifier
	}

	if method == "S256" {
		hash := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(hash[:])
		return challenge == computed
	}

	return false
}

// containsScope 检查是否包含指定的 scope
func containsScope(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}
