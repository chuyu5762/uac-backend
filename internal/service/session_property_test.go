package service

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/redis/go-redis/v9"
)

// 生成随机用户 ID
func genUserID() gopter.Gen {
	return gen.Const(nil).Map(func(_ interface{}) string {
		return "user-" + uuid.New().String()[:8]
	})
}

// 生成随机设备信息
func genDevice() gopter.Gen {
	return gen.OneConstOf(
		"Chrome on Windows",
		"Safari on macOS",
		"Firefox on Linux",
		"Mobile Safari on iOS",
		"Chrome on Android",
	)
}

// 生成随机服务 URL
func genService() gopter.Gen {
	return gen.OneConstOf(
		"https://app1.example.com",
		"https://app2.example.com/callback",
		"https://service.internal.net",
		"http://localhost:8080",
	)
}

// Property 19: SLO 会话销毁
// *For any* 用户会话，执行登出后查询该会话应返回不存在
// Validates: Requirements 14.1
func TestProperty_SLO_SessionDestroy(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("SLO 会话销毁：登出后会话不存在", prop.ForAll(
		func(userID string, device string) bool {
			// 创建会话
			session := &model.Session{
				UserID:     userID,
				DeviceInfo: device,
				IPAddress:  "192.168.1.1",
			}
			if err := svc.Create(ctx, session); err != nil {
				return false
			}

			// 验证会话存在
			retrieved, err := svc.Get(ctx, session.ID)
			if err != nil || retrieved == nil {
				return false
			}

			// 执行登出（删除会话）
			if err := svc.Delete(ctx, session.ID); err != nil {
				return false
			}

			// 验证会话不存在
			_, err = svc.Get(ctx, session.ID)
			return err == ErrSessionNotFound
		},
		genUserID(),
		genDevice(),
	))

	properties.TestingRun(t)
}

// Property 19 扩展: 用户所有会话销毁
// *For any* 用户的多个会话，执行全部登出后所有会话应返回不存在
// Validates: Requirements 14.1
func TestProperty_SLO_AllSessionsDestroy(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("SLO 全部会话销毁：登出后所有会话不存在", prop.ForAll(
		func(userID string, count int) bool {
			var sessionIDs []string

			// 创建多个会话
			for i := 0; i < count; i++ {
				session := &model.Session{
					UserID:     userID,
					DeviceInfo: "Device " + string(rune('A'+i)),
				}
				if err := svc.Create(ctx, session); err != nil {
					return false
				}
				sessionIDs = append(sessionIDs, session.ID)
			}

			// 验证会话存在
			sessions, err := svc.ListByUserID(ctx, userID)
			if err != nil || len(sessions) != count {
				return false
			}

			// 执行全部登出
			if err := svc.DeleteByUserID(ctx, userID); err != nil {
				return false
			}

			// 验证所有会话不存在
			for _, sessionID := range sessionIDs {
				_, err := svc.Get(ctx, sessionID)
				if err != ErrSessionNotFound {
					return false
				}
			}

			// 验证用户会话列表为空
			sessions, err = svc.ListByUserID(ctx, userID)
			return err == nil && len(sessions) == 0
		},
		genUserID(),
		gen.IntRange(1, 5),
	))

	properties.TestingRun(t)
}

// Property: CAS ST 单次使用
// *For any* Service Ticket，首次验证应成功，第二次验证应失败
// Validates: Requirements 13.6
func TestProperty_CAS_ST_SingleUse(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	svc := NewSessionService(client, nil)
	ctx := context.Background()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("CAS ST 单次使用：首次验证成功，第二次失败", prop.ForAll(
		func(service string) bool {
			// 创建 TGT
			tgt, err := svc.CreateTGT(ctx, "user-123", "session-456")
			if err != nil {
				return false
			}

			// 创建 ST
			st, err := svc.CreateST(ctx, tgt.ID, service)
			if err != nil {
				return false
			}

			// 首次验证应成功
			validated, err := svc.ValidateST(ctx, st.Ticket, service)
			if err != nil || validated == nil || !validated.Used {
				return false
			}

			// 第二次验证应失败
			_, err = svc.ValidateST(ctx, st.Ticket, service)
			return err == ErrSTUsed
		},
		genService(),
	))

	properties.TestingRun(t)
}
