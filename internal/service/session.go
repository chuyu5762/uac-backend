// Package service 业务逻辑层
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/redis/go-redis/v9"
)

var (
	ErrSessionNotFound = errors.New("会话不存在")
	ErrSessionExpired  = errors.New("会话已过期")
	ErrTGTNotFound     = errors.New("TGT 不存在")
	ErrTGTExpired      = errors.New("TGT 已过期")
	ErrSTNotFound      = errors.New("Service Ticket 不存在")
	ErrSTExpired       = errors.New("Service Ticket 已过期")
	ErrSTUsed          = errors.New("Service Ticket 已被使用")
)

// SessionService 会话服务接口
type SessionService interface {
	// 会话管理
	Create(ctx context.Context, session *model.Session) error
	Get(ctx context.Context, sessionID string) (*model.Session, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID string) error
	ListByUserID(ctx context.Context, userID string) ([]*model.Session, error)

	// TGT 管理（CAS 协议）
	CreateTGT(ctx context.Context, userID, sessionID string) (*model.TGT, error)
	GetTGT(ctx context.Context, tgtID string) (*model.TGT, error)
	DeleteTGT(ctx context.Context, tgtID string) error

	// ST 管理（CAS 协议）
	CreateST(ctx context.Context, tgtID, service string) (*model.ServiceTicket, error)
	ValidateST(ctx context.Context, ticket, service string) (*model.ServiceTicket, error)
}

// SessionServiceConfig 会话服务配置
type SessionServiceConfig struct {
	SessionExpiry time.Duration // 会话有效期，默认 7 天
	TGTExpiry     time.Duration // TGT 有效期，默认 8 小时
	STExpiry      time.Duration // ST 有效期，默认 5 分钟
}

type sessionService struct {
	redis  *redis.Client
	config *SessionServiceConfig
}

// NewSessionService 创建会话服务
func NewSessionService(redisClient *redis.Client, config *SessionServiceConfig) SessionService {
	if config == nil {
		config = &SessionServiceConfig{}
	}
	if config.SessionExpiry == 0 {
		config.SessionExpiry = 7 * 24 * time.Hour // 默认 7 天
	}
	if config.TGTExpiry == 0 {
		config.TGTExpiry = 8 * time.Hour // 默认 8 小时
	}
	if config.STExpiry == 0 {
		config.STExpiry = 5 * time.Minute // 默认 5 分钟
	}
	return &sessionService{
		redis:  redisClient,
		config: config,
	}
}

// Redis key 前缀
const (
	sessionKeyPrefix   = "session:"
	userSessionsPrefix = "user_sessions:"
	tgtKeyPrefix       = "tgt:"
	stKeyPrefix        = "st:"
)

// Create 创建会话
func (s *sessionService) Create(ctx context.Context, session *model.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(s.config.SessionExpiry)
	}
	session.CreatedAt = time.Now()

	// 序列化会话数据
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("序列化会话失败: %w", err)
	}

	// 计算过期时间
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return errors.New("会话过期时间无效")
	}

	// 存储会话
	key := sessionKeyPrefix + session.ID
	if err := s.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("存储会话失败: %w", err)
	}

	// 添加到用户会话列表
	userKey := userSessionsPrefix + session.UserID
	if err := s.redis.SAdd(ctx, userKey, session.ID).Err(); err != nil {
		return fmt.Errorf("添加用户会话索引失败: %w", err)
	}
	// 设置用户会话列表过期时间（比最长会话稍长）
	s.redis.Expire(ctx, userKey, s.config.SessionExpiry+time.Hour)

	return nil
}

// Get 获取会话
func (s *sessionService) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	key := sessionKeyPrefix + sessionID
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	var session model.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("反序列化会话失败: %w", err)
	}

	if session.IsExpired() {
		// 直接删除 key，避免递归调用
		s.redis.Del(ctx, key)
		// 从用户会话列表中移除
		userKey := userSessionsPrefix + session.UserID
		s.redis.SRem(ctx, userKey, sessionID)
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// Delete 删除会话
func (s *sessionService) Delete(ctx context.Context, sessionID string) error {
	// 先获取会话以获取用户 ID
	session, err := s.Get(ctx, sessionID)
	if err != nil && !errors.Is(err, ErrSessionNotFound) && !errors.Is(err, ErrSessionExpired) {
		return err
	}

	// 删除会话
	key := sessionKeyPrefix + sessionID
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除会话失败: %w", err)
	}

	// 从用户会话列表中移除
	if session != nil {
		userKey := userSessionsPrefix + session.UserID
		s.redis.SRem(ctx, userKey, sessionID)
	}

	return nil
}

// DeleteByUserID 删除用户的所有会话
func (s *sessionService) DeleteByUserID(ctx context.Context, userID string) error {
	userKey := userSessionsPrefix + userID
	sessionIDs, err := s.redis.SMembers(ctx, userKey).Result()
	if err != nil {
		return fmt.Errorf("获取用户会话列表失败: %w", err)
	}

	// 删除所有会话
	for _, sessionID := range sessionIDs {
		key := sessionKeyPrefix + sessionID
		s.redis.Del(ctx, key)
	}

	// 删除用户会话列表
	s.redis.Del(ctx, userKey)

	return nil
}

// ListByUserID 列出用户的所有会话
func (s *sessionService) ListByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	userKey := userSessionsPrefix + userID
	sessionIDs, err := s.redis.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("获取用户会话列表失败: %w", err)
	}

	var sessions []*model.Session
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, sessionID)
		if err != nil {
			// 跳过已过期或不存在的会话
			if errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrSessionExpired) {
				s.redis.SRem(ctx, userKey, sessionID)
				continue
			}
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// CreateTGT 创建 TGT（CAS 协议）
func (s *sessionService) CreateTGT(ctx context.Context, userID, sessionID string) (*model.TGT, error) {
	tgt := &model.TGT{
		ID:        "TGT-" + uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(s.config.TGTExpiry),
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(tgt)
	if err != nil {
		return nil, fmt.Errorf("序列化 TGT 失败: %w", err)
	}

	key := tgtKeyPrefix + tgt.ID
	if err := s.redis.Set(ctx, key, data, s.config.TGTExpiry).Err(); err != nil {
		return nil, fmt.Errorf("存储 TGT 失败: %w", err)
	}

	return tgt, nil
}

// GetTGT 获取 TGT
func (s *sessionService) GetTGT(ctx context.Context, tgtID string) (*model.TGT, error) {
	key := tgtKeyPrefix + tgtID
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrTGTNotFound
		}
		return nil, fmt.Errorf("获取 TGT 失败: %w", err)
	}

	var tgt model.TGT
	if err := json.Unmarshal(data, &tgt); err != nil {
		return nil, fmt.Errorf("反序列化 TGT 失败: %w", err)
	}

	if tgt.IsExpired() {
		s.DeleteTGT(ctx, tgtID)
		return nil, ErrTGTExpired
	}

	return &tgt, nil
}

// DeleteTGT 删除 TGT
func (s *sessionService) DeleteTGT(ctx context.Context, tgtID string) error {
	key := tgtKeyPrefix + tgtID
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除 TGT 失败: %w", err)
	}
	return nil
}

// CreateST 创建 Service Ticket（CAS 协议）
func (s *sessionService) CreateST(ctx context.Context, tgtID, service string) (*model.ServiceTicket, error) {
	// 验证 TGT
	tgt, err := s.GetTGT(ctx, tgtID)
	if err != nil {
		return nil, err
	}

	st := &model.ServiceTicket{
		Ticket:    "ST-" + uuid.New().String(),
		TGTID:     tgtID,
		UserID:    tgt.UserID,
		Service:   service,
		Used:      false,
		ExpiresAt: time.Now().Add(s.config.STExpiry),
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(st)
	if err != nil {
		return nil, fmt.Errorf("序列化 ST 失败: %w", err)
	}

	key := stKeyPrefix + st.Ticket
	if err := s.redis.Set(ctx, key, data, s.config.STExpiry).Err(); err != nil {
		return nil, fmt.Errorf("存储 ST 失败: %w", err)
	}

	return st, nil
}

// ValidateST 验证 Service Ticket（CAS 协议）
func (s *sessionService) ValidateST(ctx context.Context, ticket, service string) (*model.ServiceTicket, error) {
	key := stKeyPrefix + ticket
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSTNotFound
		}
		return nil, fmt.Errorf("获取 ST 失败: %w", err)
	}

	var st model.ServiceTicket
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("反序列化 ST 失败: %w", err)
	}

	// 检查是否过期
	if st.IsExpired() {
		s.redis.Del(ctx, key)
		return nil, ErrSTExpired
	}

	// 检查是否已使用（ST 只能使用一次）
	if st.Used {
		return nil, ErrSTUsed
	}

	// 检查服务是否匹配
	if st.Service != service {
		return nil, errors.New("服务不匹配")
	}

	// 标记为已使用
	st.Used = true
	updatedData, _ := json.Marshal(st)
	s.redis.Set(ctx, key, updatedData, time.Until(st.ExpiresAt))

	return &st, nil
}
