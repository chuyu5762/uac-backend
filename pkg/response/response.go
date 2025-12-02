package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 标准响应结构
// 字段顺序：code -> msg -> data
type Response struct {
	Code int         `json:"code"` // 业务状态码，0 表示成功
	Msg  string      `json:"msg"`  // 响应消息（中文）
	Data interface{} `json:"data"` // 响应数据
}

// 业务错误码
const (
	CodeSuccess = 0 // 操作成功

	// 参数错误 10xxx
	CodeInvalidRequest = 10001 // 请求参数无效
	CodeInvalidFormat  = 10002 // 参数格式错误
	CodeMissingParam   = 10003 // 必填参数缺失

	// 认证错误 20xxx
	CodeInvalidCredentials = 20001 // 用户名或密码错误
	CodeInvalidToken       = 20002 // 令牌无效或已过期
	CodeInvalidClient      = 20003 // 客户端认证失败
	CodeAccountLocked      = 20004 // 账户已被锁定
	CodeMFARequired        = 20005 // 需要多因素认证
	CodeInvalidCode        = 20006 // 验证码错误
	CodeAccessDenied       = 20007 // 用户拒绝授权
	CodeForbidden          = 20008 // 无权访问该资源

	// OAuth 错误 30xxx
	CodeInvalidAuthCode      = 30001 // 授权码无效或已过期
	CodeInvalidRefreshToken  = 30002 // 刷新令牌无效或已过期
	CodeUnsupportedGrantType = 30003 // 不支持的授权类型
	CodeUnsupportedResponse  = 30004 // 不支持的响应类型
	CodeInvalidScope         = 30005 // 请求的权限范围无效
	CodePKCEFailed           = 30006 // PKCE 验证失败

	// 资源不存在 40xxx
	CodeUserNotFound = 40001 // 用户不存在
	CodeOrgNotFound  = 40002 // 组织不存在
	CodeAppNotFound  = 40003 // 应用不存在
	CodeRoleNotFound = 40004 // 角色不存在

	// 冲突错误 50xxx
	CodeUserExists  = 50001 // 该用户名已被注册
	CodeEmailExists = 50002 // 该邮箱已被注册
	CodePhoneExists = 50003 // 该手机号已被注册

	// 服务器错误 90xxx
	CodeServerError = 90001 // 服务器内部错误
	CodeUnavailable = 90002 // 服务暂时不可用
	CodeTooManyReq  = 90003 // 请求过于频繁
)

// 错误码对应的消息
var codeMessages = map[int]string{
	CodeSuccess:              "操作成功",
	CodeInvalidRequest:       "请求参数无效",
	CodeInvalidFormat:        "参数格式错误",
	CodeMissingParam:         "必填参数缺失",
	CodeInvalidCredentials:   "用户名或密码错误",
	CodeInvalidToken:         "令牌无效或已过期",
	CodeInvalidClient:        "客户端认证失败",
	CodeAccountLocked:        "账户已被锁定，请稍后重试",
	CodeMFARequired:          "需要进行多因素认证",
	CodeInvalidCode:          "验证码错误",
	CodeAccessDenied:         "用户拒绝授权",
	CodeForbidden:            "无权访问该资源",
	CodeInvalidAuthCode:      "授权码无效或已过期",
	CodeInvalidRefreshToken:  "刷新令牌无效或已过期",
	CodeUnsupportedGrantType: "不支持的授权类型",
	CodeUnsupportedResponse:  "不支持的响应类型",
	CodeInvalidScope:         "请求的权限范围无效",
	CodePKCEFailed:           "PKCE 验证失败",
	CodeUserNotFound:         "用户不存在",
	CodeOrgNotFound:          "组织不存在",
	CodeAppNotFound:          "应用不存在",
	CodeRoleNotFound:         "角色不存在",
	CodeUserExists:           "该用户名已被注册",
	CodeEmailExists:          "该邮箱已被注册",
	CodePhoneExists:          "该手机号已被注册",
	CodeServerError:          "服务器内部错误，请稍后重试",
	CodeUnavailable:          "服务暂时不可用",
	CodeTooManyReq:           "请求过于频繁，请稍后重试",
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  codeMessages[CodeSuccess],
		Data: data,
	})
}

// SuccessWithMsg 成功响应（自定义消息）
func SuccessWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  msg,
		Data: data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int) {
	msg, ok := codeMessages[code]
	if !ok {
		msg = "未知错误"
	}
	c.JSON(codeToHTTPStatus(code), Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}

// ErrorWithMsg 错误响应（自定义消息）
func ErrorWithMsg(c *gin.Context, code int, msg string) {
	c.JSON(codeToHTTPStatus(code), Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}

// codeToHTTPStatus 业务错误码转 HTTP 状态码
func codeToHTTPStatus(code int) int {
	switch {
	case code == CodeSuccess:
		return http.StatusOK
	case code >= 10000 && code < 20000:
		return http.StatusBadRequest
	case code >= 20000 && code < 30000:
		if code == CodeInvalidToken || code == CodeInvalidClient || code == CodeInvalidCredentials {
			return http.StatusUnauthorized
		}
		return http.StatusForbidden
	case code >= 30000 && code < 40000:
		return http.StatusBadRequest
	case code >= 40000 && code < 50000:
		return http.StatusNotFound
	case code >= 50000 && code < 60000:
		return http.StatusConflict
	case code == CodeTooManyReq:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
