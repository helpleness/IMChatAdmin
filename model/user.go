package model

import (
	"gorm.io/gorm"
	"time"
)

// 结构体，登录注册，初始化创建表单调用
type User struct {
	gorm.Model
	Username string `gorm:"unique"`
	Password string `gorm:"size:512"` //哈希加密
}

// 定义 Friends 结构体，好友关系表
type Friends struct {
	UserID    int       `gorm:"primaryKey;not null"`
	FriendID  int       `gorm:"primaryKey;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	// 设置外键关联
	User   User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	Friend User `gorm:"foreignKey:FriendID;constraint:OnDelete:CASCADE;"`
}

// Group 表示群聊的基本信息
type Group struct {
	GroupID     string        `gorm:"primaryKey;type:varchar(36)"` // 群聊唯一ID，使用字符串
	GroupName   string        `gorm:"type:varchar(100);not null"`  // 群聊名称，最长100字符
	OwnerID     int           `gorm:"not null"`                    // 群主的用户ID
	CreatedTime time.Time     `gorm:"autoCreateTime"`              // 群聊创建时间
	Members     []GroupMember `gorm:"foreignKey:GroupID"`          // 群聊成员列表，外键关联
}

// GroupMember 表示群聊中的成员信息
type GroupMember struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`  // 主键ID
	GroupID  string    `gorm:"not null"`                  // 群聊ID，关联Group表
	UserID   int       `gorm:"not null"`                  // 成员的用户ID
	JoinTime time.Time `gorm:"autoCreateTime"`            // 加入群聊的时间
	Role     string    `gorm:"type:varchar(20);not null"` // 成员角色，例如 "owner", "admin", "member"
}

type HeartBeat struct {
	UserID string `json:"userID,omitempty"`
}
