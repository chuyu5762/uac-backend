package model

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// Application 应用模型
// 应用属于组织，继承组织的品牌配置
type Application struct {
	BaseModel
	OrgID            string      `gorm:"type:char(36);index;not null" json:"org_id"`        // 所属组织 ID
	Name             string      `gorm:"type:varchar(255);not null" json:"name"`            // 应用名称
	ClientID         string      `gorm:"type:varchar(64);uniqueIndex" json:"client_id"`     // OAuth Client ID
	ClientSecretHash string      `gorm:"type:varchar(255)" json:"-"`                        // Client Secret 哈希
	OAuthVersion     string      `gorm:"type:varchar(10);default:2.1" json:"oauth_version"` // OAuth 版本：2.0 或 2.1
	RedirectURIs     StringSlice `gorm:"type:json" json:"redirect_uris"`                    // 回调地址列表
	AllowedScopes    StringSlice `gorm:"type:json" json:"allowed_scopes"`                   // 允许的权限范围
	Protocol         string      `gorm:"type:varchar(20);default:oauth" json:"protocol"`    // 协议：oauth, saml, cas
	Status           string      `gorm:"type:varchar(20);default:active" json:"status"`     // 状态
	Description      string      `gorm:"type:text" json:"description"`                      // 应用描述

	// 关联
	Organization *Organization `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
}

// TableName 指定表名
func (Application) TableName() string {
	return "applications"
}

// StringSlice 字符串切片类型，用于 JSON 存储
type StringSlice []string

// Value 实现 driver.Valuer 接口
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("无法将值转换为 []byte")
	}
	return json.Unmarshal(bytes, s)
}

// IsActive 检查应用是否启用
func (a *Application) IsActive() bool {
	return a.Status == StatusActive
}

// IsOAuth21 检查是否为 OAuth 2.1 模式
func (a *Application) IsOAuth21() bool {
	return a.OAuthVersion == "2.1"
}

// SetClientSecret 设置 Client Secret（哈希存储）
func (a *Application) SetClientSecret(secret string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.ClientSecretHash = string(hash)
	return nil
}

// VerifyClientSecret 验证 Client Secret
func (a *Application) VerifyClientSecret(secret string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.ClientSecretHash), []byte(secret))
	return err == nil
}

// GenerateClientID 生成 Client ID
func GenerateClientID() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateClientSecret 生成 Client Secret
func GenerateClientSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HasScope 检查应用是否允许指定的权限范围
func (a *Application) HasScope(scope string) bool {
	for _, s := range a.AllowedScopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasRedirectURI 检查回调地址是否在允许列表中
func (a *Application) HasRedirectURI(uri string) bool {
	for _, u := range a.RedirectURIs {
		if u == uri {
			return true
		}
	}
	return false
}

// OAuth 版本常量
const (
	OAuthVersion20 = "2.0"
	OAuthVersion21 = "2.1"
)

// 协议常量
const (
	ProtocolOAuth = "oauth"
	ProtocolSAML  = "saml"
	ProtocolCAS   = "cas"
)
