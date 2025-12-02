package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
	"go.uber.org/zap"
)

// Recovery 恢复中间件
// 捕获 panic，记录日志，返回友好错误
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 获取请求 ID
				requestID, _ := c.Get("request_id")

				// 记录错误日志
				logger.Error("服务器内部错误",
					zap.Any("request_id", requestID),
					zap.Any("error", r),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("ip", c.ClientIP()),
				)

				// 返回错误响应
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
					Code: response.CodeServerError,
					Msg:  "服务器内部错误，请稍后重试",
					Data: nil,
				})
			}
		}()
		c.Next()
	}
}
