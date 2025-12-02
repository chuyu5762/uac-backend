// Package model 数据模型定义
package model

import (
	"time"
)

// Session 用户会话
type Session struct {
	ID         string    `json:"id" gorm:"type:char(36);primaryKey"`
	UserID     string    `json:"user_id" gorm:"type:char(36);index;not null"`
	TGTID      string    `json:"tgt_id,omitempty" gorm:"type:char(36);index"` // CAS TGT ID
	DeviceInfo string    `json:"device_info" gorm:"type:varchar(500)"`
	IPAddress  string    `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent  string    `json:"user_agent" gorm:"type:varchar(500)"`
	ExpiresAt  time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 表名
func (Session) TableName() string {
	return "sessions"
}

// IsExpired 检查会话是否过期
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// TGT CAS Ticket Granting Ticket
type TGT struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired 检查 TGT 是否过期
func (t *TGT) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// ServiceTicket CAS Service Ticket
type ServiceTicket struct {
	Ticket    string    `json:"ticket"`
	TGTID     string    `json:"tgt_id"`
	UserID    string    `json:"user_id"`
	Service   string    `json:"service"`
	Used      bool      `json:"used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired 检查 ST 是否过期
func (st *ServiceTicket) IsExpired() bool {
	return time.Now().After(st.ExpiresAt)
}
