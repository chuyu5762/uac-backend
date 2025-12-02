package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Organization 组织模型
// 组织是顶层实体，应用继承组织配置，用户通过绑定组织获得访问权限
type Organization struct {
	BaseModel
	TenantID    string   `gorm:"type:char(36);index" json:"tenant_id"`          // 租户 ID
	Name        string   `gorm:"type:varchar(255);not null" json:"name"`        // 组织名称
	Slug        string   `gorm:"type:varchar(100);uniqueIndex" json:"slug"`     // 组织标识（URL 友好）
	Description string   `gorm:"type:text" json:"description"`                  // 组织描述
	Branding    Branding `gorm:"type:json" json:"branding"`                     // 品牌配置
	Status      string   `gorm:"type:varchar(20);default:active" json:"status"` // 状态：active, disabled
}

// TableName 指定表名
func (Organization) TableName() string {
	return "organizations"
}

// Branding 品牌配置
type Branding struct {
	LogoURL      string `json:"logo_url"`      // Logo URL
	FaviconURL   string `json:"favicon_url"`   // Favicon URL
	PrimaryColor string `json:"primary_color"` // 主题色
	CustomCSS    string `json:"custom_css"`    // 自定义 CSS
}

// Value 实现 driver.Valuer 接口，用于数据库存储
func (b Branding) Value() (driver.Value, error) {
	return json.Marshal(b)
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
func (b *Branding) Scan(value interface{}) error {
	if value == nil {
		*b = Branding{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("无法将值转换为 []byte")
	}
	return json.Unmarshal(bytes, b)
}

// IsActive 检查组织是否启用
func (o *Organization) IsActive() bool {
	return o.Status == StatusActive
}
