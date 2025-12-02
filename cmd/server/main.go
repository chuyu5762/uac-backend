package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/config"
	"github.com/pu-ac-cn/uac-backend/internal/database"
	"github.com/pu-ac-cn/uac-backend/internal/handler"
	"github.com/pu-ac-cn/uac-backend/internal/middleware"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/redis"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
	"github.com/pu-ac-cn/uac-backend/internal/service"
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

	// 自动迁移数据库表
	if err := database.AutoMigrate(
		&model.User{},
		&model.Organization{},
		&model.Application{},
		&model.UserOrgBinding{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Println("数据库迁移完成")

	// 初始化 Repository
	userRepo := repository.NewUserRepository(database.GetDB())
	orgRepo := repository.NewOrganizationRepository(database.GetDB())
	bindingRepo := repository.NewUserOrgBindingRepository(database.GetDB())

	// 生成 RSA 密钥对（生产环境应从配置文件加载）
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("生成 RSA 密钥失败: %v", err)
	}

	// 初始化 Service
	userService := service.NewUserService(userRepo, bindingRepo, orgRepo)
	authService := service.NewAuthService(userRepo)
	tokenService := service.NewTokenService(&service.TokenServiceConfig{
		PrivateKey:    privateKey,
		PublicKey:     &privateKey.PublicKey,
		KeyID:         "key-1",
		Issuer:        cfg.JWT.Issuer,
		AccessExpiry:  cfg.JWT.AccessExpiry,
		RefreshExpiry: cfg.JWT.RefreshExpiry,
		CodeExpiry:    10 * time.Minute,
	})

	// 初始化应用服务
	appRepo := repository.NewApplicationRepository(database.GetDB())
	appService := service.NewApplicationService(appRepo, orgRepo)

	// 初始化会话服务
	sessionService := service.NewSessionService(redis.GetClient(), nil)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(userService, authService, tokenService)
	oauthHandler := handler.NewOAuthHandler(appService, tokenService, sessionService)
	oidcHandler := handler.NewOIDCHandler(userService, tokenService, cfg.JWT.Issuer)

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
		api.GET("/ping", func(c *gin.Context) {
			response.Success(c, "pong")
		})

		// 认证路由（公开）
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// 需要认证的路由
		authRequired := api.Group("")
		authRequired.Use(middleware.JWTAuth(tokenService))
		{
			authRequired.POST("/auth/logout", authHandler.Logout)
			authRequired.GET("/auth/me", authHandler.GetCurrentUser)
		}
	}

	// OAuth 2.0/2.1 路由
	oauth := router.Group("/oauth")
	{
		oauth.GET("/authorize", middleware.OptionalJWTAuth(tokenService), oauthHandler.Authorize)
		oauth.POST("/token", oauthHandler.Token)
		oauth.POST("/revoke", oauthHandler.Revoke)
		oauth.POST("/introspect", oauthHandler.Introspect)
		oauth.GET("/userinfo", middleware.JWTAuth(tokenService), oidcHandler.UserInfo)
	}

	// OIDC 发现端点
	router.GET("/.well-known/openid-configuration", oidcHandler.Discovery)
	router.GET("/.well-known/jwks.json", oidcHandler.JWKS)

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
