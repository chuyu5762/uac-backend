// Package main 数据库迁移工具
package main

import (
	"flag"
	"log"

	"github.com/pu-ac-cn/uac-backend/internal/config"
	"github.com/pu-ac-cn/uac-backend/internal/database"
	"github.com/pu-ac-cn/uac-backend/internal/model"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	var cfg *config.Config
	var err error
	if *configPath != "" {
		cfg, err = config.LoadFromFile(*configPath)
	} else {
		cfg, err = config.Load()
	}
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库连接
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()
	log.Println("数据库连接成功")

	// 执行迁移
	log.Println("开始执行数据库迁移...")

	// 迁移所有模型
	models := []any{
		&model.User{},
		&model.Organization{},
		&model.Application{},
		&model.UserOrgBinding{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RolePermission{},
	}

	for _, m := range models {
		if err := database.AutoMigrate(m); err != nil {
			log.Fatalf("迁移失败: %v", err)
		}
	}

	// 额外迁移：允许 applications.org_id 为 NULL（系统级应用）
	if cfg.Database.Driver == "postgres" {
		if err := database.GetDB().Exec("ALTER TABLE applications ALTER COLUMN org_id DROP NOT NULL").Error; err != nil {
			log.Printf("忽略 org_id DROP NOT NULL 迁移错误（可能已是 NULL）: %v", err)
		}
	} else if cfg.Database.Driver == "mysql" {
		if err := database.GetDB().Exec("ALTER TABLE applications MODIFY COLUMN org_id char(36) NULL").Error; err != nil {
			log.Printf("忽略 org_id NULL 迁移错误（可能已是 NULL）: %v", err)
		}
	}

	log.Println("数据库迁移完成！")

	// 打印创建的表
	log.Println("已创建/更新的表:")
	log.Println("  - users (用户表)")
	log.Println("  - organizations (组织表)")
	log.Println("  - applications (应用表)")
	log.Println("  - user_org_bindings (用户-组织绑定表)")
	log.Println("  - roles (角色表)")
	log.Println("  - permissions (权限表)")
	log.Println("  - user_roles (用户角色关联表)")
	log.Println("  - role_permissions (角色权限关联表)")
}
