package request

type RequestStatus int

const (
	Pending  RequestStatus = iota // 0: pending
	Accepted                      // 1: accepted
	Rejected                      // 2: rejected
)

// FriendAdd 表示添加好友的请求
type FriendAdd struct {
	UserID   int           `gorm:"type:int;not null" json:"user_id"`   // 发起添加请求的用户ID
	FriendID int           `gorm:"type:int;not null" json:"friend_id"` // 要添加的好友的ID
	Message  string        `gorm:"type:text" json:"message"`           // 添加好友时的附加消息
	Status   RequestStatus `gorm:"type:int;default:0" json:"status"`   // 请求的处理状态
}

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

// GroupApplication 表示申请加入群组的请求
// GroupApplication 表示申请加入群组的请求
type GroupApplication struct {
	UserID  int           `gorm:"type:int;not null" json:"user_id"`  // 申请加入群组的用户的ID
	GroupID int           `gorm:"type:int;not null" json:"group_id"` // 群组的ID
	Message string        `gorm:"type:text" json:"message"`          // 申请加入群组时的附加消息
	Status  RequestStatus `gorm:"type:int;default:0" json:"status"`  // 请求的处理状态
}
