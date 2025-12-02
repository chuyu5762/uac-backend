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
	models := []interface{}{
		&model.User{},
		&model.Organization{},
		&model.Application{},
		&model.UserOrgBinding{},
	}

	for _, m := range models {
		if err := database.AutoMigrate(m); err != nil {
			log.Fatalf("迁移失败: %v", err)
		}
	}

	log.Println("数据库迁移完成！")

	// 打印创建的表
	log.Println("已创建/更新的表:")
	log.Println("  - users (用户表)")
	log.Println("  - organizations (组织表)")
	log.Println("  - applications (应用表)")
	log.Println("  - user_org_bindings (用户-组织绑定表)")
}
