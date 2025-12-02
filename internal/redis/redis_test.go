package redis

import (
	"context"
	"testing"
	"time"

	"github.com/pu-ac-cn/uac-backend/internal/config"
)

// 测试用的 Redis 配置
// 注意：这些测试需要运行中的 Redis 实例
func getTestConfig() *config.RedisConfig {
	return &config.RedisConfig{
		Addr:     "1.95.88.239:6379",
		Password: "123456",
		DB:       15, // 使用 DB 15 作为测试数据库，避免影响其他数据
	}
}

// TestInit 测试 Redis 初始化
func TestInit(t *testing.T) {
	cfg := getTestConfig()
	err := Init(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer Close()

	// 验证客户端已初始化
	c := GetClient()
	if c == nil {
		t.Error("GetClient() 返回 nil")
	}
}

// TestSetGet 测试 Set 和 Get 操作
func TestSetGet(t *testing.T) {
	cfg := getTestConfig()
	if err := Init(cfg); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer Close()

	ctx := context.Background()
	key := "test:key:setget"
	value := "test_value"

	// 设置值
	if err := Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set 失败: %v", err)
	}

	// 获取值
	got, err := Get(ctx, key)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got != value {
		t.Errorf("Get 期望 %s, 实际 %s", value, got)
	}

	// 清理
	Del(ctx, key)
}

// TestDel 测试删除操作
func TestDel(t *testing.T) {
	cfg := getTestConfig()
	if err := Init(cfg); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer Close()

	ctx := context.Background()
	key := "test:key:del"

	// 设置值
	Set(ctx, key, "value", time.Minute)

	// 删除
	if err := Del(ctx, key); err != nil {
		t.Fatalf("Del 失败: %v", err)
	}

	// 验证已删除
	exists, _ := Exists(ctx, key)
	if exists != 0 {
		t.Error("删除后键仍然存在")
	}
}

// TestExists 测试键存在检查
func TestExists(t *testing.T) {
	cfg := getTestConfig()
	if err := Init(cfg); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer Close()

	ctx := context.Background()
	key := "test:key:exists"

	// 键不存在
	exists, err := Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists 失败: %v", err)
	}
	if exists != 0 {
		t.Error("期望键不存在")
	}

	// 设置键
	Set(ctx, key, "value", time.Minute)

	// 键存在
	exists, err = Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists 失败: %v", err)
	}
	if exists != 1 {
		t.Error("期望键存在")
	}

	// 清理
	Del(ctx, key)
}

// TestExpire 测试过期时间设置
func TestExpire(t *testing.T) {
	cfg := getTestConfig()
	if err := Init(cfg); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer Close()

	ctx := context.Background()
	key := "test:key:expire"

	// 设置无过期时间的键
	Set(ctx, key, "value", 0)

	// 设置过期时间
	if err := Expire(ctx, key, time.Minute); err != nil {
		t.Fatalf("Expire 失败: %v", err)
	}

	// 获取 TTL
	ttl, err := TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL 失败: %v", err)
	}
	if ttl <= 0 || ttl > time.Minute {
		t.Errorf("TTL 期望在 0-60s 之间, 实际 %v", ttl)
	}

	// 清理
	Del(ctx, key)
}

// TestIncr 测试自增操作
func TestIncr(t *testing.T) {
	cfg := getTestConfig()
	if err := Init(cfg); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer Close()

	ctx := context.Background()
	key := "test:key:incr"

	// 清理可能存在的键
	Del(ctx, key)

	// 自增
	val, err := Incr(ctx, key)
	if err != nil {
		t.Fatalf("Incr 失败: %v", err)
	}
	if val != 1 {
		t.Errorf("Incr 期望 1, 实际 %d", val)
	}

	// 再次自增
	val, err = Incr(ctx, key)
	if err != nil {
		t.Fatalf("Incr 失败: %v", err)
	}
	if val != 2 {
		t.Errorf("Incr 期望 2, 实际 %d", val)
	}

	// 清理
	Del(ctx, key)
}

// TestIncrBy 测试自增指定值
func TestIncrBy(t *testing.T) {
	cfg := getTestConfig()
	if err := Init(cfg); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer Close()

	ctx := context.Background()
	key := "test:key:incrby"

	// 清理可能存在的键
	Del(ctx, key)

	// 自增 5
	val, err := IncrBy(ctx, key, 5)
	if err != nil {
		t.Fatalf("IncrBy 失败: %v", err)
	}
	if val != 5 {
		t.Errorf("IncrBy 期望 5, 实际 %d", val)
	}

	// 再自增 10
	val, err = IncrBy(ctx, key, 10)
	if err != nil {
		t.Fatalf("IncrBy 失败: %v", err)
	}
	if val != 15 {
		t.Errorf("IncrBy 期望 15, 实际 %d", val)
	}

	// 清理
	Del(ctx, key)
}

// TestHash 测试哈希操作
func TestHash(t *testing.T) {
	cfg := getTestConfig()
	if err := Init(cfg); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer Close()

	ctx := context.Background()
	key := "test:hash"

	// 清理可能存在的键
	Del(ctx, key)

	// 设置哈希字段
	if err := HSet(ctx, key, "field1", "value1", "field2", "value2"); err != nil {
		t.Fatalf("HSet 失败: %v", err)
	}

	// 获取单个字段
	val, err := HGet(ctx, key, "field1")
	if err != nil {
		t.Fatalf("HGet 失败: %v", err)
	}
	if val != "value1" {
		t.Errorf("HGet 期望 value1, 实际 %s", val)
	}

	// 获取所有字段
	all, err := HGetAll(ctx, key)
	if err != nil {
		t.Fatalf("HGetAll 失败: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("HGetAll 期望 2 个字段, 实际 %d", len(all))
	}

	// 删除字段
	if err := HDel(ctx, key, "field1"); err != nil {
		t.Fatalf("HDel 失败: %v", err)
	}

	// 验证字段已删除
	all, _ = HGetAll(ctx, key)
	if len(all) != 1 {
		t.Errorf("HDel 后期望 1 个字段, 实际 %d", len(all))
	}

	// 清理
	Del(ctx, key)
}

// TestClose 测试关闭连接
func TestClose(t *testing.T) {
	cfg := getTestConfig()
	if err := Init(cfg); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}

	// 关闭连接
	if err := Close(); err != nil {
		t.Errorf("Close 失败: %v", err)
	}
}

// TestCloseNil 测试关闭未初始化的连接
func TestCloseNil(t *testing.T) {
	// 重置客户端
	client = nil

	// 关闭应该不报错
	if err := Close(); err != nil {
		t.Errorf("Close nil 客户端应该不报错: %v", err)
	}
}
