package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/config"
	"github.com/pu-ac-cn/uac-backend/internal/database"
	"github.com/pu-ac-cn/uac-backend/internal/middleware"
	"github.com/pu-ac-cn/uac-backend/internal/redis"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库连接
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()
	log.Println("数据库连接成功")

	// 初始化 Redis 连接
	if err := redis.Init(&cfg.Redis); err != nil {
		log.Fatalf("初始化 Redis 失败: %v", err)
	}
	defer redis.Close()
	log.Println("Redis 连接成功")

	// 设置 Gin 模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router := gin.New()

	// 全局中间件
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		// 检查数据库连接
		dbStatus := "ok"
		if err := database.Ping(); err != nil {
			dbStatus = "error"
		}

		// 检查 Redis 连接
		redisStatus := "ok"
		redisClient := redis.GetClient()
		if redisClient == nil {
			redisStatus = "error"
		} else if err := redisClient.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = "error"
		}

		response.Success(c, gin.H{
			"status":   "ok",
			"time":     time.Now().Format(time.RFC3339),
			"database": dbStatus,
			"redis":    redisStatus,
		})
	})

	// API 路由组
	api := router.Group("/api/v1")
	{
		// TODO: 注册路由
		api.GET("/ping", func(c *gin.Context) {
			response.Success(c, "pong")
		})
	}

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器
	go func() {
		log.Printf("服务启动，监听地址: %s", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务...")

	// 优雅关闭，等待 5 秒
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务关闭失败: %v", err)
	}

	// 关闭数据库和 Redis 连接
	database.Close()
	redis.Close()

	log.Println("服务已关闭")
}
