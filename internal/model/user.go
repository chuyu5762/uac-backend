package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User 用户模型
type User struct {
	BaseModel
	Username         string     `gorm:"type:varchar(100);uniqueIndex" json:"username"`
	Email            string     `gorm:"type:varchar(255);uniqueIndex" json:"email"`
	Phone            string     `gorm:"type:varchar(20);index" json:"phone,omitempty"`
	PasswordHash     string     `gorm:"type:varchar(255)" json:"-"`
	DisplayName      string     `gorm:"type:varchar(100)" json:"display_name"`
	AvatarURL        string     `gorm:"type:varchar(500)" json:"avatar_url,omitempty"`
	Status           string     `gorm:"type:varchar(20);default:active" json:"status"`
	EmailVerified    bool       `gorm:"default:false" json:"email_verified"`
	PhoneVerified    bool       `gorm:"default:false" json:"phone_verified"`
	FailedLoginCount int        `gorm:"default:0" json:"-"`
	LockedUntil      *time.Time `json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// SetPassword 设置密码（哈希存储）
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// VerifyPassword 验证密码
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsActive 检查用户是否启用
func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

// IsLocked 检查用户是否被锁定
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// IncrementFailedLogin 增加登录失败次数
func (u *User) IncrementFailedLogin() {
	u.FailedLoginCount++
	if u.FailedLoginCount >= 5 {
		lockTime := time.Now().Add(15 * time.Minute)
		u.LockedUntil = &lockTime
	}
}

// ResetFailedLogin 重置登录失败次数
func (u *User) ResetFailedLogin() {
	u.FailedLoginCount = 0
	u.LockedUntil = nil
}

// UserOrgBinding 用户-组织绑定关系
type UserOrgBinding struct {
	BaseModel
	UserID string `gorm:"type:char(36);index;not null" json:"user_id"`
	OrgID  string `gorm:"type:char(36);index;not null" json:"org_id"`

	// 关联
	User         *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Organization *Organization `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
}

// TableName 指定表名
func (UserOrgBinding) TableName() string {
	return "user_org_bindings"
}
