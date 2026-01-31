package models

import (
	"time"

	"gorm.io/gorm"
)

// Device 客户端 Agent 上报的设备信息
type Device struct {
	gorm.Model
	ID               string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"` // 使用 UUID 作为主键
	UniqueHardwareID string `gorm:"uniqueIndex;not null" json:"unique_hardware_id"`          // 设备的唯一硬件ID (BIOS UUID, Serial Number等)
	OS               string `json:"os"`                                                        // 操作系统
	Hostname         string `json:"hostname"`                                                  // 主机名
	LastSeenAt       time.Time `json:"last_seen_at"`                                             // 最后一次 Agent 上报时间
	// 其他可以采集的设备信息...
}

// UserDeviceBinding 用户与设备的绑定关系
type UserDeviceBinding struct {
	gorm.Model
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	KeycloakUserID string `gorm:"uniqueIndex:idx_user_device_binding;not null" json:"keycloak_user_id"` // Keycloak 中用户的 ID (sub)
	DeviceID     string `gorm:"uniqueIndex:idx_user_device_binding;not null" json:"device_id"`          // 关联的设备 ID
	Status       string `gorm:"default:'active';not null" json:"status"`                            // 绑定状态: active, inactive, pending_approval
	BoundAt      time.Time `json:"bound_at"`
	UnboundAt    *time.Time `json:"unbound_at"` // 解绑时间，可为空
	// Device         Device `gorm:"foreignKey:DeviceID"` // 可选，如果需要GORM自动加载关联
}

// Rule 代理规则
type Rule struct {
	gorm.Model
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name        string `gorm:"uniqueIndex;not null" json:"name"` // 规则名称
	Type        string `gorm:"not null" json:"type"`             // 规则类型: http-proxy, tcp-proxy
	Match       string `gorm:"not null" json:"match"`            // 匹配条件: 域名, IP:Port
	Action      string `gorm:"not null" json:"action"`           // 动作: proxy, block, direct
	Description string `json:"description"`
}

// KeycloakUser 用于前端显示 Keycloak 用户信息 (简化 DTO)
type KeycloakUser struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	FirstName          string `json:"firstName"`
	LastName           string `json:"lastName"`
	Enabled            bool   `json:"enabled"`
	EmailVerified      bool   `json:"emailVerified"`
	FederatedIdentities []struct { // 联合身份，例如 Google
		IdentityProvider string `json:"identityProvider"`
		UserID           string `json:"userId"`
		UserName         string `json:"userName"`
	} `json:"federatedIdentities"`
	// ... 其他您可能需要的 Keycloak 用户字段
}
