package database

import (
	"testing"

	"github.com/pu-ac-cn/uac-backend/internal/config"
)

// 测试用的数据库配置
func getTestPostgresConfig() *config.DatabaseConfig {
	return &config.DatabaseConfig{
		Driver: "postgres",
		Postgres: config.PostgresConfig{
			Host:     "1.95.88.239",
			Port:     5432,
			User:     "uac",
			Password: "j4x3mdzttNnanCsB",
			DBName:   "uac",
			SSLMode:  "disable",
		},
	}
}

func getTestMySQLConfig() *config.DatabaseConfig {
	return &config.DatabaseConfig{
		Driver: "mysql",
		MySQL: config.MySQLConfig{
			Host:      "1.95.88.239",
			Port:      3306,
			User:      "uac",
			Password:  "j4x3mdzttNnanCsB",
			DBName:    "uac",
			Charset:   "utf8mb4",
			ParseTime: true,
			Loc:       "Local",
		},
	}
}

// TestInitPostgres 测试 PostgreSQL 初始化
func TestInitPostgres(t *testing.T) {
	cfg := getTestPostgresConfig()
	err := Init(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 PostgreSQL: %v", err)
	}
	defer Close()

	// 验证数据库实例已初始化
	d := GetDB()
	if d == nil {
		t.Error("GetDB() 返回 nil")
	}
}

// TestInitMySQL 测试 MySQL 初始化
func TestInitMySQL(t *testing.T) {
	cfg := getTestMySQLConfig()
	err := Init(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 MySQL: %v", err)
	}
	defer Close()

	// 验证数据库实例已初始化
	d := GetDB()
	if d == nil {
		t.Error("GetDB() 返回 nil")
	}
}

// TestInitUnsupportedDriver 测试不支持的数据库驱动
func TestInitUnsupportedDriver(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "unsupported",
	}
	err := Init(cfg)
	if err == nil {
		t.Error("期望返回错误，但没有")
	}
}

// TestPing 测试数据库连接检查
func TestPing(t *testing.T) {
	cfg := getTestPostgresConfig()
	err := Init(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接数据库: %v", err)
	}
	defer Close()

	// 测试 Ping
	if err := Ping(); err != nil {
		t.Errorf("Ping 失败: %v", err)
	}
}

// TestPingNotInitialized 测试未初始化时的 Ping
func TestPingNotInitialized(t *testing.T) {
	// 重置数据库实例
	db = nil

	err := Ping()
	if err == nil {
		t.Error("期望返回错误，但没有")
	}
}

// TestClose 测试关闭数据库连接
func TestClose(t *testing.T) {
	cfg := getTestPostgresConfig()
	err := Init(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接数据库: %v", err)
	}

	// 关闭连接
	if err := Close(); err != nil {
		t.Errorf("Close 失败: %v", err)
	}
}

// TestCloseNil 测试关闭未初始化的连接
func TestCloseNil(t *testing.T) {
	// 重置数据库实例
	db = nil

	// 关闭应该不报错
	if err := Close(); err != nil {
		t.Errorf("Close nil 数据库应该不报错: %v", err)
	}
}

// TestAutoMigrate 测试自动迁移
func TestAutoMigrate(t *testing.T) {
	cfg := getTestPostgresConfig()
	err := Init(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接数据库: %v", err)
	}
	defer Close()

	// 定义测试模型
	type TestModel struct {
		ID   string `gorm:"primaryKey"`
		Name string
	}

	// 执行迁移
	if err := AutoMigrate(&TestModel{}); err != nil {
		t.Errorf("AutoMigrate 失败: %v", err)
	}

	// 清理测试表
	GetDB().Exec("DROP TABLE IF EXISTS test_models")
}

// TestAutoMigrateNotInitialized 测试未初始化时的自动迁移
func TestAutoMigrateNotInitialized(t *testing.T) {
	// 重置数据库实例
	db = nil

	type TestModel struct {
		ID string `gorm:"primaryKey"`
	}

	err := AutoMigrate(&TestModel{})
	if err == nil {
		t.Error("期望返回错误，但没有")
	}
}
