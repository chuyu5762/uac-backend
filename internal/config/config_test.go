package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoad 测试配置加载
func TestLoad(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  addr: ":9090"
  mode: "release"
  read_timeout: "15s"
  write_timeout: "15s"

database:
  driver: "postgres"
  postgres:
    host: "testhost"
    port: 5433
    user: "testuser"
    password: "testpass"
    dbname: "testdb"
    sslmode: "require"

redis:
  addr: "testredis:6380"
  password: "redispass"
  db: 1

jwt:
  issuer: "test-issuer"
  access_expiry: "1h"
  refresh_expiry: "24h"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	// 测试从文件加载配置
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证服务器配置
	if cfg.Server.Addr != ":9090" {
		t.Errorf("Server.Addr 期望 :9090, 实际 %s", cfg.Server.Addr)
	}
	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode 期望 release, 实际 %s", cfg.Server.Mode)
	}

	// 验证数据库配置
	if cfg.Database.Driver != "postgres" {
		t.Errorf("Database.Driver 期望 postgres, 实际 %s", cfg.Database.Driver)
	}
	if cfg.Database.Postgres.Host != "testhost" {
		t.Errorf("Database.Postgres.Host 期望 testhost, 实际 %s", cfg.Database.Postgres.Host)
	}
	if cfg.Database.Postgres.Port != 5433 {
		t.Errorf("Database.Postgres.Port 期望 5433, 实际 %d", cfg.Database.Postgres.Port)
	}

	// 验证 Redis 配置
	if cfg.Redis.Addr != "testredis:6380" {
		t.Errorf("Redis.Addr 期望 testredis:6380, 实际 %s", cfg.Redis.Addr)
	}
	if cfg.Redis.DB != 1 {
		t.Errorf("Redis.DB 期望 1, 实际 %d", cfg.Redis.DB)
	}

	// 验证 JWT 配置
	if cfg.JWT.Issuer != "test-issuer" {
		t.Errorf("JWT.Issuer 期望 test-issuer, 实际 %s", cfg.JWT.Issuer)
	}
}

// TestLoadDefaults 测试默认配置
func TestLoadDefaults(t *testing.T) {
	// 创建空配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证默认值
	if cfg.Server.Addr != ":8080" {
		t.Errorf("默认 Server.Addr 期望 :8080, 实际 %s", cfg.Server.Addr)
	}
	if cfg.Database.Driver != "postgres" {
		t.Errorf("默认 Database.Driver 期望 postgres, 实际 %s", cfg.Database.Driver)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("默认 Redis.Addr 期望 localhost:6379, 实际 %s", cfg.Redis.Addr)
	}
}

// TestGet 测试获取全局配置
func TestGet(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  addr: ":8888"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	// 加载配置
	_, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 获取全局配置
	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() 返回 nil")
	}
	if cfg.Server.Addr != ":8888" {
		t.Errorf("Get().Server.Addr 期望 :8888, 实际 %s", cfg.Server.Addr)
	}
}

// TestLoadFromFileNotFound 测试加载不存在的配置文件
func TestLoadFromFileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("期望返回错误，但没有")
	}
}
