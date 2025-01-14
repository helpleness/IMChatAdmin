package request

// FriendAdd 表示添加好友的请求
type FriendAdd struct {
	UserID   int    `json:"user_id"`   // 发起添加请求的用户ID
	FriendID int    `json:"friend_id"` // 要添加的好友的ID
	Message  string `json:"message"`   // 添加好友时的附加消息
}

// GroupCreated 表示创建群组的请求
type GroupCreated struct {
	CreatorID      int    `json:"creator_id"`      // 创建群组的用户的ID
	GroupName      string `json:"group_name"`      // 群组的名称
	Description    string `json:"description"`     // 群组的描述
	InitialMembers []int  `json:"initial_members"` // 初始群友的ID列表
}

// GroupAdd 表示添加用户到群组的请求
type GroupAdd struct {
	GroupID int `json:"group_id"` // 群组的ID
	UserID  int `json:"user_id"`  // 要添加到群组的用户的ID
}

// GroupApplication 表示申请加入群组的请求
type GroupApplication struct {
	UserID  int    `json:"user_id"`  // 申请加入群组的用户的ID
	GroupID int    `json:"group_id"` // 群组的ID
	Message string `json:"message"`  // 申请加入群组时的附加消息
}
