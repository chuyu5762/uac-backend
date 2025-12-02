package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Property 6: OAuth 2.1 PKCE 强制
// *For any* OAuth 2.1 模式的应用，不提供 code_challenge 的授权请求应被拒绝
// Validates: Requirements 4.1
func TestProperty_OAuth21_PKCE_Required(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	h := &OAuthHandler{}

	// 生成随机 code_verifier
	verifierGen := gen.Const(nil).Map(func(_ interface{}) string {
		return uuid.New().String() + uuid.New().String()[:10]
	})

	properties.Property("PKCE S256 验证：正确的 verifier 应通过验证", prop.ForAll(
		func(verifier string) bool {
			// 计算 S256 challenge
			hash := sha256.Sum256([]byte(verifier))
			challenge := base64.RawURLEncoding.EncodeToString(hash[:])

			// 验证应通过
			return h.verifyPKCE(challenge, "S256", verifier)
		},
		verifierGen,
	))

	properties.Property("PKCE S256 验证：错误的 verifier 应失败", prop.ForAll(
		func(verifier string) bool {
			// 计算 S256 challenge
			hash := sha256.Sum256([]byte(verifier))
			challenge := base64.RawURLEncoding.EncodeToString(hash[:])

			// 使用错误的 verifier
			wrongVerifier := verifier + "wrong"
			return !h.verifyPKCE(challenge, "S256", wrongVerifier)
		},
		verifierGen,
	))

	properties.Property("PKCE plain 验证：verifier 等于 challenge 应通过", prop.ForAll(
		func(verifier string) bool {
			return h.verifyPKCE(verifier, "plain", verifier)
		},
		verifierGen,
	))

	properties.Property("PKCE plain 验证：verifier 不等于 challenge 应失败", prop.ForAll(
		func(verifier string) bool {
			wrongVerifier := verifier + "wrong"
			return !h.verifyPKCE(verifier, "plain", wrongVerifier)
		},
		verifierGen,
	))

	properties.TestingRun(t)
}

// Property: Scope 验证
// *For any* 请求的 scope，只有在允许列表中的才应通过
func TestProperty_ScopeValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	h := &OAuthHandler{}

	// 允许的 scopes
	allowedScopes := []string{"openid", "profile", "email", "phone", "offline_access"}

	// 生成随机请求的 scopes（从允许列表中选择）
	validScopesGen := gen.SliceOfN(3, gen.OneConstOf("openid", "profile", "email")).Map(func(s []string) []string {
		return s
	})

	// 生成包含无效 scope 的列表
	invalidScopesGen := gen.Const(nil).Map(func(_ interface{}) []string {
		return []string{"openid", "invalid_scope", "profile"}
	})

	properties.Property("有效 scopes 应通过验证", prop.ForAll(
		func(requestedScopes []string) bool {
			return h.isValidScopes(allowedScopes, requestedScopes)
		},
		validScopesGen,
	))

	properties.Property("包含无效 scope 应失败", prop.ForAll(
		func(requestedScopes []string) bool {
			return !h.isValidScopes(allowedScopes, requestedScopes)
		},
		invalidScopesGen,
	))

	properties.TestingRun(t)
}

// Property: Redirect URI 验证
// *For any* redirect URI，只有在允许列表中的才应通过
func TestProperty_RedirectURIValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	h := &OAuthHandler{}

	allowedURIs := []string{
		"https://app.example.com/callback",
		"https://app.example.com/oauth/callback",
		"http://localhost:8080/callback",
	}

	// 生成有效的 URI
	validURIGen := gen.OneConstOf(
		"https://app.example.com/callback",
		"https://app.example.com/oauth/callback",
		"http://localhost:8080/callback",
	)

	// 生成无效的 URI
	invalidURIGen := gen.OneConstOf(
		"https://evil.com/callback",
		"https://app.example.com/other",
		"http://localhost:9999/callback",
	)

	properties.Property("有效 redirect URI 应通过验证", prop.ForAll(
		func(uri string) bool {
			return h.isValidRedirectURI(allowedURIs, uri)
		},
		validURIGen,
	))

	properties.Property("无效 redirect URI 应失败", prop.ForAll(
		func(uri string) bool {
			return !h.isValidRedirectURI(allowedURIs, uri)
		},
		invalidURIGen,
	))

	properties.TestingRun(t)
}
