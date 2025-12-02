package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 创建测试用的 Redis 客户端
func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, func() {
		client.Close()
		mr.Close()
	}
}

func TestSessionService_Create(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	session := &model.Session{
		UserID:     "user-123",
		DeviceInfo: "Chrome on Windows",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	err := svc.Create(ctx, session)
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.False(t, session.ExpiresAt.IsZero())
}

func TestSessionService_Get(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	// 创建会话
	session := &model.Session{
		UserID:     "user-123",
		DeviceInfo: "Chrome on Windows",
		IPAddress:  "192.168.1.1",
	}
	err := svc.Create(ctx, session)
	require.NoError(t, err)

	// 获取会话
	retrieved, err := svc.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.DeviceInfo, retrieved.DeviceInfo)
}

func TestSessionService_Get_NotFound(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	_, err := svc.Get(ctx, "non-existent-id")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionService_Delete(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	// 创建会话
	session := &model.Session{
		UserID:     "user-123",
		DeviceInfo: "Chrome on Windows",
	}
	err := svc.Create(ctx, session)
	require.NoError(t, err)

	// 删除会话
	err = svc.Delete(ctx, session.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = svc.Get(ctx, session.ID)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionService_DeleteByUserID(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	userID := "user-123"

	// 创建多个会话
	for i := 0; i < 3; i++ {
		session := &model.Session{
			UserID:     userID,
			DeviceInfo: "Device " + string(rune('A'+i)),
		}
		err := svc.Create(ctx, session)
		require.NoError(t, err)
	}

	// 验证会话存在
	sessions, err := svc.ListByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// 删除所有会话
	err = svc.DeleteByUserID(ctx, userID)
	require.NoError(t, err)

	// 验证已删除
	sessions, err = svc.ListByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestSessionService_ListByUserID(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	userID := "user-123"

	// 创建多个会话
	for i := 0; i < 3; i++ {
		session := &model.Session{
			UserID:     userID,
			DeviceInfo: "Device " + string(rune('A'+i)),
		}
		err := svc.Create(ctx, session)
		require.NoError(t, err)
	}

	// 列出会话
	sessions, err := svc.ListByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestSessionService_TGT_Create_Get_Delete(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	// 创建 TGT
	tgt, err := svc.CreateTGT(ctx, "user-123", "session-456")
	require.NoError(t, err)
	assert.NotEmpty(t, tgt.ID)
	assert.True(t, len(tgt.ID) > 4 && tgt.ID[:4] == "TGT-")

	// 获取 TGT
	retrieved, err := svc.GetTGT(ctx, tgt.ID)
	require.NoError(t, err)
	assert.Equal(t, tgt.ID, retrieved.ID)
	assert.Equal(t, tgt.UserID, retrieved.UserID)

	// 删除 TGT
	err = svc.DeleteTGT(ctx, tgt.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = svc.GetTGT(ctx, tgt.ID)
	assert.ErrorIs(t, err, ErrTGTNotFound)
}

func TestSessionService_ST_Create_Validate(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	// 创建 TGT
	tgt, err := svc.CreateTGT(ctx, "user-123", "session-456")
	require.NoError(t, err)

	// 创建 ST
	service := "https://app.example.com"
	st, err := svc.CreateST(ctx, tgt.ID, service)
	require.NoError(t, err)
	assert.NotEmpty(t, st.Ticket)
	assert.True(t, len(st.Ticket) > 3 && st.Ticket[:3] == "ST-")
	assert.False(t, st.Used)

	// 验证 ST
	validated, err := svc.ValidateST(ctx, st.Ticket, service)
	require.NoError(t, err)
	assert.Equal(t, st.Ticket, validated.Ticket)
	assert.True(t, validated.Used)

	// 再次验证应失败（ST 只能使用一次）
	_, err = svc.ValidateST(ctx, st.Ticket, service)
	assert.ErrorIs(t, err, ErrSTUsed)
}

func TestSessionService_ST_ServiceMismatch(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	// 创建 TGT
	tgt, err := svc.CreateTGT(ctx, "user-123", "session-456")
	require.NoError(t, err)

	// 创建 ST
	st, err := svc.CreateST(ctx, tgt.ID, "https://app1.example.com")
	require.NoError(t, err)

	// 使用不同的服务验证应失败
	_, err = svc.ValidateST(ctx, st.Ticket, "https://app2.example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "服务不匹配")
}

func TestSessionService_ExpiredSession(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	// 使用很短的过期时间
	svc := NewSessionService(client, &SessionServiceConfig{
		SessionExpiry: 100 * time.Millisecond,
	})
	ctx := context.Background()

	session := &model.Session{
		UserID:     "user-123",
		DeviceInfo: "Chrome on Windows",
	}
	err := svc.Create(ctx, session)
	require.NoError(t, err)

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 获取应返回过期错误
	_, err = svc.Get(ctx, session.ID)
	assert.ErrorIs(t, err, ErrSessionExpired)
}
