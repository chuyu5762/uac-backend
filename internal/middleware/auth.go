package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

// JWTAuth JWT 认证中间件
func JWTAuth(tokenService service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization 头获取令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.ErrorWithMsg(c, response.CodeInvalidToken, "未提供认证令牌")
			c.Abort()
			return
		}

		// 检查 Bearer 前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.ErrorWithMsg(c, response.CodeInvalidToken, "认证令牌格式错误")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证令牌
		claims, err := tokenService.ValidateToken(c.Request.Context(), tokenString)
		if err != nil {
			switch err {
			case service.ErrTokenExpired:
				response.ErrorWithMsg(c, response.CodeInvalidToken, "令牌已过期")
			case service.ErrInvalidToken:
				response.Error(c, response.CodeInvalidToken)
			default:
				response.ErrorWithMsg(c, response.CodeInvalidToken, "认证失败")
			}
			c.Abort()
			return
		}

		// 检查令牌类型
		if claims.Type != "access" {
			response.ErrorWithMsg(c, response.CodeInvalidToken, "无效的令牌类型")
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("scopes", claims.Scopes)
		c.Set("claims", claims)

		c.Next()
	}
}

// OptionalJWTAuth 可选的 JWT 认证中间件（不强制要求登录）
func OptionalJWTAuth(tokenService service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		claims, err := tokenService.ValidateToken(c.Request.Context(), parts[1])
		if err == nil && claims.Type == "access" {
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("email", claims.Email)
			c.Set("scopes", claims.Scopes)
			c.Set("claims", claims)
		}

		c.Next()
	}
}
