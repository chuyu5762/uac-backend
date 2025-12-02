package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	"github.com/pu-ac-cn/uac-backend/web"
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
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Println("数据库迁移完成")

	// 初始化 Repository
	userRepo := repository.NewUserRepository(database.GetDB())
	orgRepo := repository.NewOrganizationRepository(database.GetDB())
	bindingRepo := repository.NewUserOrgBindingRepository(database.GetDB())

	// 加载或生成 RSA 密钥对
	privateKey, err := loadOrGenerateRSAKey(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatalf("加载 RSA 密钥失败: %v", err)
	}
	log.Println("RSA 密钥加载成功")

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

	// 初始化 RBAC 服务
	roleRepo := repository.NewRoleRepository(database.GetDB())
	permRepo := repository.NewPermissionRepository(database.GetDB())
	userRoleRepo := repository.NewUserRoleRepository(database.GetDB())
	rbacService := service.NewRBACService(roleRepo, permRepo, userRoleRepo)

	// 初始化默认角色和权限
	if err := rbacService.InitDefaultRolesAndPermissions(context.Background()); err != nil {
		log.Printf("初始化默认角色和权限失败: %v", err)
	} else {
		log.Println("默认角色和权限初始化完成")
	}

	// 初始化组织服务
	orgService := service.NewOrganizationService(orgRepo)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(userService, authService, tokenService, rbacService)
	oauthHandler := handler.NewOAuthHandler(appService, tokenService, sessionService)
	oidcHandler := handler.NewOIDCHandler(userService, tokenService, cfg.JWT.Issuer)
	rbacHandler := handler.NewRBACHandler(rbacService)
	userHandler := handler.NewUserHandler(userService)
	appHandler := handler.NewAppHandler(appService)
	orgHandler := handler.NewOrgHandler(orgService)

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
			authRequired.PUT("/auth/me", userHandler.UpdateCurrentUser)
			authRequired.POST("/auth/change-password", userHandler.ChangePassword)
			authRequired.GET("/auth/permissions", rbacHandler.GetCurrentUserPermissions)
		}

		// 用户管理路由（需要管理员权限）
		users := api.Group("/users")
		users.Use(middleware.JWTAuth(tokenService))
		users.Use(middleware.RequireAnyRole(rbacService, model.RoleSuperAdmin, model.RoleOrgAdmin))
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.POST("", userHandler.CreateUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}

		// 应用管理路由（需要管理员权限）
		apps := api.Group("/apps")
		apps.Use(middleware.JWTAuth(tokenService))
		apps.Use(middleware.RequireAnyRole(rbacService, model.RoleSuperAdmin, model.RoleOrgAdmin))
		{
			apps.GET("", appHandler.ListApps)
			apps.GET("/:id", appHandler.GetApp)
			apps.POST("", appHandler.CreateApp)
			apps.PUT("/:id", appHandler.UpdateApp)
			apps.DELETE("/:id", appHandler.DeleteApp)
			apps.POST("/:id/reset-secret", appHandler.ResetSecret)
		}

		// 组织管理路由（需要管理员权限）
		orgs := api.Group("/orgs")
		orgs.Use(middleware.JWTAuth(tokenService))
		orgs.Use(middleware.RequireAnyRole(rbacService, model.RoleSuperAdmin, model.RoleOrgAdmin))
		{
			orgs.GET("", orgHandler.ListOrgs)
			orgs.GET("/:id", orgHandler.GetOrg)
			orgs.POST("", orgHandler.CreateOrg)
			orgs.PUT("/:id", orgHandler.UpdateOrg)
			orgs.DELETE("/:id", orgHandler.DeleteOrg)
			orgs.PUT("/:id/branding", orgHandler.UpdateBranding)
		}

		// RBAC 管理路由（需要管理员权限）
		rbac := api.Group("")
		rbac.Use(middleware.JWTAuth(tokenService))
		rbac.Use(middleware.RequireAnyRole(rbacService, model.RoleSuperAdmin, model.RoleOrgAdmin))
		{
			// 角色管理
			rbac.POST("/roles", rbacHandler.CreateRole)
			rbac.GET("/roles", rbacHandler.ListRoles)
			rbac.GET("/roles/:id", rbacHandler.GetRole)
			rbac.PUT("/roles/:id", rbacHandler.UpdateRole)
			rbac.DELETE("/roles/:id", rbacHandler.DeleteRole)
			rbac.POST("/roles/:id/permissions", rbacHandler.AddPermissionsToRole)
			rbac.DELETE("/roles/:id/permissions", rbacHandler.RemovePermissionsFromRole)

			// 权限管理
			rbac.GET("/permissions", rbacHandler.ListPermissions)
			rbac.GET("/permissions/:id", rbacHandler.GetPermission)
			rbac.POST("/permissions", rbacHandler.CreatePermission)
			rbac.DELETE("/permissions/:id", rbacHandler.DeletePermission)

			// 获取角色权限
			rbac.GET("/roles/:id/permissions", rbacHandler.GetRolePermissions)

			// 用户角色管理（使用不同的路径避免冲突）
			rbac.GET("/user-roles/:user_id", rbacHandler.GetUserRoles)
			rbac.POST("/user-roles/:user_id", rbacHandler.AssignRole)
			rbac.DELETE("/user-roles/:user_id/:role_id", rbacHandler.RevokeRole)
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

	// 静态文件服务（前端嵌入）
	if cfg.Static.Enabled {
		staticMode := web.ModeEmbed
		if cfg.Static.Mode == "disk" {
			staticMode = web.ModeDisk
		}

		staticHandler := web.NewStaticHandler(&web.StaticConfig{
			Mode:      staticMode,
			DiskPath:  cfg.Static.Path,
			IndexFile: "index.html",
			APIPrefix: []string{"/api/", "/oauth/", "/.well-known/", "/health"},
		})

		// 设置静态文件路由和 SPA 处理
		staticHandler.SetupRoutes(router)
		log.Printf("静态文件服务已启用，模式: %s", cfg.Static.Mode)
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

// loadOrGenerateRSAKey 加载或生成 RSA 密钥对
// 如果密钥文件存在则加载，否则生成新密钥并保存到文件
func loadOrGenerateRSAKey(privateKeyPath, publicKeyPath string) (*rsa.PrivateKey, error) {
	// 尝试加载已有的私钥
	if privateKeyPath != "" {
		if privateKeyData, err := os.ReadFile(privateKeyPath); err == nil {
			block, _ := pem.Decode(privateKeyData)
			if block != nil && block.Type == "RSA PRIVATE KEY" {
				privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
				if err == nil {
					log.Printf("从文件加载 RSA 私钥: %s", privateKeyPath)
					return privateKey, nil
				}
			}
		}
	}

	// 生成新的密钥对
	log.Println("生成新的 RSA 密钥对...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// 保存私钥到文件
	if privateKeyPath != "" {
		if err := savePrivateKey(privateKeyPath, privateKey); err != nil {
			log.Printf("警告: 保存私钥失败: %v", err)
		} else {
			log.Printf("RSA 私钥已保存到: %s", privateKeyPath)
		}
	}

	// 保存公钥到文件
	if publicKeyPath != "" {
		if err := savePublicKey(publicKeyPath, &privateKey.PublicKey); err != nil {
			log.Printf("警告: 保存公钥失败: %v", err)
		} else {
			log.Printf("RSA 公钥已保存到: %s", publicKeyPath)
		}
	}

	return privateKey, nil
}

// savePrivateKey 保存私钥到 PEM 文件
func savePrivateKey(path string, key *rsa.PrivateKey) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, block)
}

// savePublicKey 保存公钥到 PEM 文件
func savePublicKey(path string, key *rsa.PublicKey) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	keyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: keyBytes,
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, block)
}
