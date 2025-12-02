package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/pu-ac-cn/uac-backend/internal/config"
	"github.com/pu-ac-cn/uac-backend/internal/database"
	"github.com/pu-ac-cn/uac-backend/internal/model"
)

// 一个只清理 UAC 相关表的重置工具：
// - 默认按依赖顺序 Drop 表，然后可选地 AutoMigrate 重建。
// - 仅影响本项目的业务表，不会删除数据库、用户或其它非 UAC 表。
// 用法：
//   go run ./cmd/resetdb -force
// 可选参数：
//   -recreate  重建表（默认 true）
//   -force     必须为 true 才会执行（安全开关）
func main() {
	recreate := flag.Bool("recreate", true, "是否在清空后重建表")
	force := flag.Bool("force", false, "确认执行清空操作")
	flag.Parse()

	if !*force {
		log.Fatal("为避免误操作，请加上 -force 参数：go run ./cmd/resetdb -force")
	}

	// 加载配置并连接数据库
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	db := database.GetDB()
	m := db.Migrator()

	// 注意依赖顺序：先删子表再删父表
	dropOrder := []any{
		&model.RolePermission{},
		&model.UserRole{},
		&model.UserOrgBinding{},
		&model.Application{},
		&model.Permission{},
		&model.Role{},
		&model.User{},
		&model.Organization{},
	}

	fmt.Println("开始清空数据库中的 UAC 相关表...")
	for _, t := range dropOrder {
		if m.HasTable(t) {
			if err := m.DropTable(t); err != nil {
				log.Fatalf("删除表失败: %v", err)
			}
			// 打印表名
			fmt.Printf("已删除表: %T\n", t)
		}
	}

	if *recreate {
		// 重新创建（按依赖自底向上）
		createOrder := []any{
			&model.Organization{},
			&model.User{},
			&model.Role{},
			&model.Permission{},
			&model.Application{},
			&model.UserOrgBinding{},
			&model.UserRole{},
			&model.RolePermission{},
		}
		for _, t := range createOrder {
			if err := m.AutoMigrate(t); err != nil {
				log.Fatalf("创建表失败: %v", err)
			}
			fmt.Printf("已创建/更新表: %T\n", t)
		}
	}

	fmt.Println("完成。")
}