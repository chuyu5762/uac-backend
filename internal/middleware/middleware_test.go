package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestLogger 测试日志中间件
func TestLogger(t *testing.T) {
	router := gin.New()
	router.Use(Logger())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 发送请求
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	// 验证 X-Request-ID 头
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("期望 X-Request-ID 头存在")
	}
}

// TestLoggerWithRequestID 测试日志中间件使用已有的请求 ID
func TestLoggerWithRequestID(t *testing.T) {
	router := gin.New()
	router.Use(Logger())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 发送带有 X-Request-ID 的请求
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "custom-request-id")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应中的 X-Request-ID 与请求中的一致
	requestID := w.Header().Get("X-Request-ID")
	if requestID != "custom-request-id" {
		t.Errorf("期望 X-Request-ID 为 custom-request-id, 实际 %s", requestID)
	}
}

// TestRecovery 测试恢复中间件
func TestRecovery(t *testing.T) {
	router := gin.New()
	router.Use(Logger()) // Recovery 依赖 Logger 设置的 request_id
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("测试 panic")
	})

	// 发送请求
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证返回 500 状态码
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码 500, 实际 %d", w.Code)
	}
}

// TestCORS 测试 CORS 中间件
func TestCORS(t *testing.T) {
	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 发送带 Origin 的请求
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证 CORS 头
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Error("期望 Access-Control-Allow-Origin 头存在")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("期望 Access-Control-Allow-Methods 头存在")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("期望 Access-Control-Allow-Headers 头存在")
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("期望 Access-Control-Allow-Credentials 为 true")
	}
}

// TestCORSPreflight 测试 CORS 预检请求
func TestCORSPreflight(t *testing.T) {
	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 发送 OPTIONS 预检请求
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证返回 204 状态码
	if w.Code != http.StatusNoContent {
		t.Errorf("期望状态码 204, 实际 %d", w.Code)
	}
}

// TestSecurityHeaders 测试安全响应头
func TestSecurityHeaders(t *testing.T) {
	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证安全头
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("期望 X-Content-Type-Options 为 nosniff")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("期望 X-Frame-Options 为 DENY")
	}
	if w.Header().Get("X-XSS-Protection") == "" {
		t.Error("期望 X-XSS-Protection 头存在")
	}
}

// TestGetLogger 测试获取日志实例
func TestGetLogger(t *testing.T) {
	l := GetLogger()
	if l == nil {
		t.Error("GetLogger() 返回 nil")
	}
}
