// Package web 提供前端静态文件嵌入和服务功能
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var embeddedFS embed.FS

// StaticMode 静态文件服务模式
type StaticMode string

const (
	// ModeEmbed 嵌入模式，使用 go:embed 嵌入的文件
	ModeEmbed StaticMode = "embed"
	// ModeDisk 磁盘模式，直接读取磁盘文件（支持热更新）
	ModeDisk StaticMode = "disk"
)

// StaticConfig 静态文件服务配置
type StaticConfig struct {
	// Mode 服务模式：embed 或 disk
	Mode StaticMode
	// DiskPath 磁盘模式下的文件路径（相对于工作目录）
	DiskPath string
	// IndexFile 默认首页文件
	IndexFile string
	// APIPrefix API 路径前缀（这些路径不会被静态文件服务处理）
	APIPrefix []string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *StaticConfig {
	return &StaticConfig{
		Mode:      ModeEmbed,
		DiskPath:  "./web/dist",
		IndexFile: "index.html",
		APIPrefix: []string{"/api/", "/oauth/", "/.well-known/", "/health"},
	}
}

// StaticHandler 静态文件处理器
type StaticHandler struct {
	config *StaticConfig
	fs     http.FileSystem
}

// NewStaticHandler 创建静态文件处理器
func NewStaticHandler(config *StaticConfig) *StaticHandler {
	if config == nil {
		config = DefaultConfig()
	}

	handler := &StaticHandler{config: config}

	if config.Mode == ModeDisk {
		// 磁盘模式：直接使用文件系统
		handler.fs = http.Dir(config.DiskPath)
	} else {
		// 嵌入模式：使用 embed.FS
		subFS, err := fs.Sub(embeddedFS, "dist")
		if err != nil {
			// 如果嵌入文件不存在，回退到磁盘模式
			handler.fs = http.Dir(config.DiskPath)
		} else {
			handler.fs = http.FS(subFS)
		}
	}

	return handler
}

// GetFileSystem 返回文件系统
func (h *StaticHandler) GetFileSystem() http.FileSystem {
	return h.fs
}

// IsAPIPath 检查路径是否为 API 路径
func (h *StaticHandler) IsAPIPath(path string) bool {
	for _, prefix := range h.config.APIPrefix {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// IsStaticFile 检查路径是否为静态资源文件
func (h *StaticHandler) IsStaticFile(path string) bool {
	extensions := []string{
		".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg",
		".ico", ".woff", ".woff2", ".ttf", ".eot", ".map",
		".json", ".xml", ".txt", ".html", ".htm",
	}
	for _, ext := range extensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}
	return false
}

// FileExists 检查文件是否存在
func (h *StaticHandler) FileExists(path string) bool {
	if h.config.Mode == ModeDisk {
		fullPath := filepath.Join(h.config.DiskPath, path)
		_, err := os.Stat(fullPath)
		return err == nil
	}

	// embed 模式
	file, err := h.fs.Open(path)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// ServeFile 服务单个文件
func (h *StaticHandler) ServeFile(c *gin.Context, path string) {
	// 尝试打开文件
	file, err := h.fs.Open(path)
	if err != nil {
		// 文件不存在，返回 index.html（SPA 路由）
		h.serveIndex(c)
		return
	}
	defer file.Close()

	// 获取文件信息
	stat, err := file.Stat()
	if err != nil {
		h.serveIndex(c)
		return
	}

	// 如果是目录，尝试返回 index.html
	if stat.IsDir() {
		indexPath := strings.TrimSuffix(path, "/") + "/" + h.config.IndexFile
		if h.FileExists(indexPath) {
			path = indexPath
		} else {
			h.serveIndex(c)
			return
		}
	}

	// 使用 Gin 的文件服务
	c.FileFromFS(path, h.fs)
}

// serveIndex 返回首页（用于 SPA 路由）
func (h *StaticHandler) serveIndex(c *gin.Context) {
	c.FileFromFS(h.config.IndexFile, h.fs)
}

// Middleware 返回 Gin 中间件
func (h *StaticHandler) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 路径直接跳过
		if h.IsAPIPath(path) {
			c.Next()
			return
		}

		// 只处理 GET 和 HEAD 请求
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			c.Next()
			return
		}

		// 静态资源文件直接服务
		if h.IsStaticFile(path) {
			h.ServeFile(c, path)
			c.Abort()
			return
		}

		// 其他路径继续处理（可能是 API 或 SPA 路由）
		c.Next()
	}
}

// SPAHandler 返回 SPA 路由处理器（用于 NoRoute）
func (h *StaticHandler) SPAHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 路径返回 404
		if h.IsAPIPath(path) {
			c.JSON(http.StatusNotFound, gin.H{
				"code": 404,
				"msg":  "接口不存在",
			})
			return
		}

		// 只处理 GET 请求
		if c.Request.Method != "GET" {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"code": 405,
				"msg":  "方法不允许",
			})
			return
		}

		// 返回 index.html（SPA 路由）
		h.serveIndex(c)
	}
}

// SetupRoutes 设置静态文件路由
func (h *StaticHandler) SetupRoutes(router *gin.Engine) {
	// 静态资源中间件
	router.Use(h.Middleware())

	// SPA 路由处理（NoRoute）
	router.NoRoute(h.SPAHandler())
}
