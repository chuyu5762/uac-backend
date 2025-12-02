// Package service 令牌服务
package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 令牌相关错误
var (
	ErrInvalidToken     = errors.New("无效的令牌")
	ErrTokenExpired     = errors.New("令牌已过期")
	ErrInvalidSignature = errors.New("签名验证失败")
	ErrInvalidIssuer    = errors.New("无效的签发者")
	ErrCodeExpired      = errors.New("授权码已过期")
	ErrCodeUsed         = errors.New("授权码已使用")
	ErrRefreshTokenUsed = errors.New("刷新令牌已使用")
)

// TokenClaims JWT 声明
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID   string   `json:"uid,omitempty"`
	Username string   `json:"username,omitempty"`
	Email    string   `json:"email,omitempty"`
	OrgID    string   `json:"org_id,omitempty"`
	AppID    string   `json:"app_id,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	Type     string   `json:"type,omitempty"` // access, refresh, id
}

// AuthorizationCode 授权码
type AuthorizationCode struct {
	Code                string    `json:"code"`
	ClientID            string    `json:"client_id"`
	UserID              string    `json:"user_id"`
	RedirectURI         string    `json:"redirect_uri"`
	Scopes              []string  `json:"scopes"`
	CodeChallenge       string    `json:"code_challenge,omitempty"`
	CodeChallengeMethod string    `json:"code_challenge_method,omitempty"`
	ExpiresAt           time.Time `json:"expires_at"`
	Used                bool      `json:"used"`
}

// TokenService 令牌服务接口
type TokenService interface {
	// GenerateAccessToken 生成访问令牌
	GenerateAccessToken(ctx context.Context, claims *TokenClaims) (string, error)
	// GenerateRefreshToken 生成刷新令牌
	GenerateRefreshToken(ctx context.Context, claims *TokenClaims) (string, error)
	// GenerateIDToken 生成 ID 令牌
	GenerateIDToken(ctx context.Context, claims *TokenClaims) (string, error)
	// ValidateToken 验证令牌
	ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error)
	// GenerateAuthorizationCode 生成授权码
	GenerateAuthorizationCode(ctx context.Context, code *AuthorizationCode) (string, error)
	// ValidateAuthorizationCode 验证授权码
	ValidateAuthorizationCode(ctx context.Context, code string) (*AuthorizationCode, error)
	// RevokeToken 撤销令牌
	RevokeToken(ctx context.Context, tokenString string) error
	// GetPublicKey 获取公钥（用于 JWKS）
	GetPublicKey() *rsa.PublicKey
	// GetKeyID 获取密钥 ID
	GetKeyID() string
}

// tokenService 令牌服务实现
type tokenService struct {
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	keyID         string
	issuer        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	codeExpiry    time.Duration
	// 存储授权码和已撤销令牌（生产环境应使用 Redis）
	codes         map[string]*AuthorizationCode
	revokedTokens map[string]time.Time
}

// TokenServiceConfig 令牌服务配置
type TokenServiceConfig struct {
	PrivateKey    *rsa.PrivateKey
	PublicKey     *rsa.PublicKey
	KeyID         string
	Issuer        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
	CodeExpiry    time.Duration
}

// NewTokenService 创建令牌服务
func NewTokenService(cfg *TokenServiceConfig) TokenService {
	return &tokenService{
		privateKey:    cfg.PrivateKey,
		publicKey:     cfg.PublicKey,
		keyID:         cfg.KeyID,
		issuer:        cfg.Issuer,
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
		codeExpiry:    cfg.CodeExpiry,
		codes:         make(map[string]*AuthorizationCode),
		revokedTokens: make(map[string]time.Time),
	}
}

// GenerateAccessToken 生成访问令牌
func (s *tokenService) GenerateAccessToken(ctx context.Context, claims *TokenClaims) (string, error) {
	now := time.Now()
	claims.Type = "access"
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    s.issuer,
		Subject:   claims.UserID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
		ID:        generateTokenID(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyID

	return token.SignedString(s.privateKey)
}

// GenerateRefreshToken 生成刷新令牌
func (s *tokenService) GenerateRefreshToken(ctx context.Context, claims *TokenClaims) (string, error) {
	now := time.Now()
	claims.Type = "refresh"
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    s.issuer,
		Subject:   claims.UserID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiry)),
		ID:        generateTokenID(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyID

	return token.SignedString(s.privateKey)
}

// GenerateIDToken 生成 ID 令牌
func (s *tokenService) GenerateIDToken(ctx context.Context, claims *TokenClaims) (string, error) {
	now := time.Now()
	claims.Type = "id"
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    s.issuer,
		Subject:   claims.UserID,
		Audience:  jwt.ClaimStrings{claims.AppID},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
		ID:        generateTokenID(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyID

	return token.SignedString(s.privateKey)
}

// ValidateToken 验证令牌
func (s *tokenService) ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	// 检查是否已撤销
	if _, revoked := s.revokedTokens[tokenString]; revoked {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidSignature
		}
		return s.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// 验证签发者
	if claims.Issuer != s.issuer {
		return nil, ErrInvalidIssuer
	}

	return claims, nil
}

// GenerateAuthorizationCode 生成授权码
func (s *tokenService) GenerateAuthorizationCode(ctx context.Context, code *AuthorizationCode) (string, error) {
	codeStr := generateSecureCode(32)
	code.Code = codeStr
	code.ExpiresAt = time.Now().Add(s.codeExpiry)
	code.Used = false
	s.codes[codeStr] = code
	return codeStr, nil
}

// ValidateAuthorizationCode 验证授权码
func (s *tokenService) ValidateAuthorizationCode(ctx context.Context, codeStr string) (*AuthorizationCode, error) {
	code, exists := s.codes[codeStr]
	if !exists {
		return nil, ErrInvalidToken
	}

	if code.Used {
		return nil, ErrCodeUsed
	}

	if time.Now().After(code.ExpiresAt) {
		delete(s.codes, codeStr)
		return nil, ErrCodeExpired
	}

	// 标记为已使用
	code.Used = true

	return code, nil
}

// RevokeToken 撤销令牌
func (s *tokenService) RevokeToken(ctx context.Context, tokenString string) error {
	s.revokedTokens[tokenString] = time.Now()
	return nil
}

// GetPublicKey 获取公钥
func (s *tokenService) GetPublicKey() *rsa.PublicKey {
	return s.publicKey
}

// GetKeyID 获取密钥 ID
func (s *tokenService) GetKeyID() string {
	return s.keyID
}

// generateTokenID 生成令牌 ID
func generateTokenID() string {
	return generateSecureCode(16)
}

// generateSecureCode 生成安全随机码
func generateSecureCode(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

// TokenClaimsToJSON 将令牌声明序列化为 JSON
func TokenClaimsToJSON(claims *TokenClaims) ([]byte, error) {
	return json.Marshal(claims)
}

// TokenClaimsFromJSON 从 JSON 反序列化令牌声明
func TokenClaimsFromJSON(data []byte) (*TokenClaims, error) {
	var claims TokenClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

// 令牌有效期常量
const (
	DefaultAccessExpiry  = 15 * time.Minute
	DefaultRefreshExpiry = 7 * 24 * time.Hour
	DefaultCodeExpiry    = 10 * time.Minute
)
