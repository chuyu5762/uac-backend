// 为现有用户分配超级管理员角色的工具
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pu-ac-cn/uac-backend/internal/config"
	"github.com/pu-ac-cn/uac-backend/internal/database"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
	"github.com/pu-ac-cn/uac-backend/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: assign-admin <用户名或邮箱>")
		fmt.Println("示例: assign-admin admin@example.com")
		os.Exit(1)
	}

	username := os.Args[1]

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// 初始化 Repository
	userRepo := repository.NewUserRepository(database.GetDB())
	roleRepo := repository.NewRoleRepository(database.GetDB())
	permRepo := repository.NewPermissionRepository(database.GetDB())
	userRoleRepo := repository.NewUserRoleRepository(database.GetDB())

	// 初始化 Service
	rbacService := service.NewRBACService(roleRepo, permRepo, userRoleRepo)

	// 确保默认角色和权限已初始化
	if err := rbacService.InitDefaultRolesAndPermissions(ctx); err != nil {
		log.Printf("初始化默认角色和权限失败: %v", err)
	}

	// 查找用户
	user, err := userRepo.GetByUsername(ctx, username)
	if err != nil {
		user, err = userRepo.GetByEmail(ctx, username)
		if err != nil {
			log.Fatalf("用户不存在: %s", username)
		}
	}

	// 分配超级管理员角色
	if err := rbacService.AssignRoleByCode(ctx, user.ID, model.RoleSuperAdmin); err != nil {
		log.Fatalf("分配角色失败: %v", err)
	}

	fmt.Printf("成功为用户 %s (%s) 分配超级管理员角色\n", user.Username, user.Email)
}
