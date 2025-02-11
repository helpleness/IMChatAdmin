package model

import (
	"gorm.io/gorm"
	"time"
)

// 结构体，登录注册，初始化创建表单调用
type User struct {
	gorm.Model
	Username  string `gorm:"unique"`
	Password  string `gorm:"size:512"`          //哈希加密
	AvatarURL string `gorm:"type:varchar(255)"` // 头像URL
}

// MessageType 描述系统中不同类型的消息
type MessageType int

const (
	TEXT           MessageType = iota // 文本消息
	IMAGE                             // 图片消息
	FILE                              // 文件消息
	FRIEND_REQUEST                    // 好友请求
	GROUP_INVITE                      // 群组邀请
	ONLINE_STATUS                     // 在线状态更新
)

// MyMessage 聊天消息结构
type MyMessage struct {
	gorm.Model
	MessageID  string      `gorm:"primaryKey;type:varchar(36)"` // 消息唯一标识
	UserFrom   string      `gorm:"type:varchar(36);not null"`   // 发送者用户ID
	SendTarget string      `gorm:"type:varchar(36);not null"`   // 接收者用户ID或群组ID
	Content    string      `gorm:"type:text"`                   // 消息内容
	Type       MessageType `gorm:"type:int"`                    // 消息类型
	SendTime   time.Time   `gorm:"type:bigint"`                 // 发送时间（Unix时间戳）
}

// 定义 Friends 结构体，好友关系表
type Friends struct {
	UserID    int       `gorm:"primaryKey;not null"`
	FriendID  int       `gorm:"not null;uniqueIndex:idx_user_friend"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	User   User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Friend User `gorm:"foreignKey:FriendID;constraint:OnDelete:CASCADE"`
}

// Group 表示群聊的基本信息
type Group struct {
	GroupID     string        `gorm:"primaryKey;type:varchar(36)"`                                       // 群聊唯一ID，使用字符串
	GroupName   string        `gorm:"type:varchar(100);not null"`                                        // 群聊名称，最长100字符
	OwnerID     int           `gorm:"not null"`                                                          // 群主的用户ID
	CreatedTime time.Time     `gorm:"autoCreateTime"`                                                    // 群聊创建时间
	Members     []GroupMember `gorm:"foreignKey:GroupID;references:GroupID;constraint:OnDelete:CASCADE"` // 群聊成员列表，外键关联
}

// GroupMember 表示群聊中的成员信息
type GroupMember struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`  // 主键ID
	GroupID  string    `gorm:"not null"`                  // 群聊ID，关联Group表
	UserID   int       `gorm:"not null"`                  // 成员的用户ID
	JoinTime time.Time `gorm:"autoCreateTime"`            // 加入群聊的时间
	Role     string    `gorm:"type:varchar(20);not null"` // 成员角色，例如 "owner", "admin", "member"
}

// 好友/群聊加入申请
// 群聊创建
// 持久化 发送给暂时不在线用户的消息
type HeartBeat struct {
	UserID string `json:"userID,omitempty"`
}
