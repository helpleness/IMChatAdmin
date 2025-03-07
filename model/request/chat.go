package request

// GroupCreated 表示创建群组的请求
type GroupCreated struct {
	CreatorID      int    `json:"creator_id"`      // 创建群组的用户的ID
	GroupName      string `json:"group_name"`      // 群组的名称
	InitialMembers []int  `json:"initial_members"` // 初始群友的ID列表
}

// GroupAdd 表示添加用户到群组的请求
type GroupAdd struct {
	GroupID  int    `json:"group_id"`                  // 群组的ID
	UserID   int    `json:"user_id"`                   // 要添加到群组的用户的ID
	UserFrom string `gorm:"type:varchar(36);not null"` // 发送者用户ID
}
