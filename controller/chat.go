package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/middleware"
	"github.com/helpleness/IMChatAdmin/model"
	"github.com/helpleness/IMChatAdmin/model/request"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"time"
)

func PushMessage(ctx context.Context, user model.User) {

	time.Sleep(50 * time.Millisecond)
	reidsClient := database.GetRedisClient()
	key := fmt.Sprintf("messages:%s", user.ID)
	datas, err := reidsClient.LRange(ctx, key, 0, -1).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Printf("Error 获取 %s 消息: %v", user.ID, err)
		return
	}

	status, err := reidsClient.HGet(context.Background(), strconv.Itoa(int(user.ID)), "status").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Printf("Error checking user %s status: %v", strconv.Itoa(int(user.ID)), err)
		return
	}
	if status == "online" {
		targetIP, err := reidsClient.Get(ctx, strconv.Itoa(int(user.ID))).Result()
		if err == redis.Nil {
			log.Printf("Key does not exist")
			return
		} else if err != nil {
			log.Printf("Error getting value: %v", err)
			return
		}
		queueName := fmt.Sprintf("message_queue" + targetIP) // 队列名，可以按需设置
		for data := range datas {
			err = reidsClient.LPush(ctx, queueName, data).Err()
			if err != nil {
				log.Printf("Error pushing message to Redis queue: %v", err)
				return
			}
		}
		return
	} else {
		time.Sleep(50 * time.Millisecond)
	}

}

// 旁路缓存好友添加和处理
func FriendAdd(ctx *gin.Context) {
	var req request.FriendAdd
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ID, _ := ctx.Get("userid")
	UserID := ID.(uint)
	req.UserID = int(UserID)
	// 这里添加处理好友添加请求的逻辑
	// 例如，验证用户ID，发送好友请求等
	if req.UserID <= 0 || req.FriendID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID不正确"})
	}
	// 查找数据库中是否存在这两个ID
	db := database.GetDB()
	_, err := middleware.Isuserexist(ctx, req.FriendID)
	if err != nil {
		return
	}
	// 发送好友请求
	req.Status = request.Pending // 设置请求状态为待处理
	db.Create(&req)
	// 缓存到 Redis
	redisCli := database.GetRedisClient()
	cacheKey := "friend_request:" + strconv.Itoa(req.FriendID)
	reqMarshal, _ := json.Marshal(req)
	pipe := redisCli.Pipeline()
	pipe.LPush(ctx, cacheKey, reqMarshal)
	pipe.Expire(ctx, cacheKey, 7*24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("Error caching friend request with expiration: %v", err)
	}

	// 假设我们已经处理了请求，并且添加成功
	ctx.JSON(http.StatusOK, gin.H{"message": "Friend request sent successfully"})
}

// 旁路缓存群聊创建
func GroupCreated(ctx *gin.Context) {
	var req request.GroupCreated
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ID, _ := ctx.Get("userid")
	UserID := ID.(uint)
	req.CreatorID = int(UserID)

	// 验证创建者ID
	if req.CreatorID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Creator ID不正确"})
		return
	}

	db := database.GetDB()
	var err error

	// 创建群聊记录
	group := model.Group{
		GroupName: req.GroupName,
		OwnerID:   req.CreatorID,
	}
	if result := db.Create(&group).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 缓存群聊信息
	redisCli := database.GetRedisClient()
	cacheKey := "group:" + strconv.Itoa(group.GroupID)
	groupMarshal, _ := json.Marshal(group)
	if err := redisCli.Set(ctx, cacheKey, groupMarshal, 7*24*time.Hour).Err(); err != nil {
		log.Printf("Error caching group: %v", err)
	}

	groupMember := model.GroupMember{
		GroupID: group.GroupID,
		UserID:  req.CreatorID,
		Role:    "owner",
	}
	if result := db.Create(&groupMember).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 缓存群主信息
	cacheKey = "group_member:" + strconv.Itoa(groupMember.GroupID)
	groupMemberMarshal, _ := json.Marshal(groupMember)
	if err := redisCli.LPush(ctx, cacheKey, groupMemberMarshal).Err(); err != nil {
		log.Printf("Error caching group member: %v", err)
	}
	err = redisCli.Expire(ctx, cacheKey, 7*24/time.Hour).Err()
	if err != nil {
		log.Printf("Error caching group member: %v", err)
	}

	// 添加初始成员
	for _, memberID := range req.InitialMembers {
		_, err := middleware.Isuserexist(ctx, memberID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		groupMember := model.GroupMember{
			GroupID: group.GroupID,
			UserID:  memberID,
			Role:    "member",
		}
		if result := db.Create(&groupMember).Error; result != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
			return
		}
		// 缓存成员信息
		cacheKey = "group_member:" + strconv.Itoa(groupMember.GroupID)
		groupMemberMarshal, _ := json.Marshal(groupMember)
		if err := redisCli.LPush(ctx, cacheKey, groupMemberMarshal).Err(); err != nil {
			log.Printf("Error caching group member: %v", err)
		}
		err = redisCli.Expire(ctx, cacheKey, 7*24/time.Hour).Err()
		if err != nil {
			log.Printf("Error caching group member: %v", err)
		}
	}

	// 假设我们已经创建了群聊
	ctx.JSON(http.StatusOK, gin.H{"message": "Group created successfully", "group_id": group.GroupID})
}

//// 添加用户到群组的请求
//func GroupAdd(ctx *gin.Context) {
//	var req request.GroupAdd
//	if err := ctx.ShouldBindJSON(&req); err != nil {
//		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//
//	// 验证用户ID和群组ID
//	if req.UserID <= 0 || req.GroupID <= 0 {
//		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID不正确"})
//		return
//	}
//
//	// 查找数据库中是否存在用户
//	var user model.User
//	db := database.GetDB()
//	if result := db.First(&user, req.UserID); result.Error != nil {
//		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
//			ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
//			return
//		}
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
//		return
//	}
//
//	// 查找数据库中是否存在群组
//	var group model.Group
//	if result := db.First(&group, req.GroupID); result.Error != nil {
//		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
//			ctx.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
//			return
//		}
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
//		return
//	}
//
//	// 检查用户是否已经是群组成员
//	var existingMember model.GroupMember
//	if result := db.Where("group_id = ? AND user_id = ?", req.GroupID, req.UserID).First(&existingMember).Error; result == nil {
//		ctx.JSON(http.StatusBadRequest, gin.H{"error": "User is already a member of the group"})
//		return
//	}
//
//	// 发送加入申请
//	groupMember := model.GroupMember{
//		GroupID: group.GroupID,
//		UserID:  int(user.ID),
//		Role:    "member",
//	}
//	if result := db.Create(&groupMember).Error; result != nil {
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
//		return
//	}
//
//	// 假设申请已发送
//	ctx.JSON(http.StatusOK, gin.H{"message": "Group join request sent successfully"})
//}

// 旁路缓存添加用户到群组的请求
func GroupAddRedis(ctx *gin.Context) {
	var req request.GroupAdd
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ID, _ := ctx.Get("userid")
	UserID := ID.(uint)
	req.UserID = int(UserID)
	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 验证用户ID和群组ID
	if req.UserID <= 0 || req.GroupID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID不正确"})
		return
	}

	var err error

	// 查找数据库中是否存在群组
	var group model.Group
	err, group = isgroupexist(ctx, req.GroupID)
	if err != nil {
		return
	}
	// 检查用户是否已经是群组成员
	cacheKey := "group_member:" + strconv.Itoa(req.GroupID)
	members, err := redisCli.LRange(ctx, cacheKey, 0, -1).Result()
	if err != nil && err != redis.Nil {
		log.Printf("Error getting group members from cache: %v", err)
	}

	var existingMember model.GroupMember
	for _, member := range members {
		if err := json.Unmarshal([]byte(member), &existingMember); err != nil {
			log.Printf("Error unmarshalling group member: %v", err)
			continue
		}
		if existingMember.UserID == req.UserID {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "User is already a member of the group"})
			return
		}
	}

	// 如果缓存中没有数据，从数据库中查询
	if len(members) == 0 {
		if result := db.Where("group_id = ? AND user_id = ?", req.GroupID, req.UserID).First(&existingMember).Error; result == nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "User is already a member of the group"})
			return
		}
	}

	// 发送加入申请
	groupMember := model.GroupMember{
		GroupID: group.GroupID,
		UserID:  req.UserID,
		Role:    "member",
	}
	if result := db.Create(&groupMember).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 缓存成员信息
	cacheKey = "group_member:" + strconv.Itoa(req.GroupID)
	groupMemberMarshal, _ := json.Marshal(groupMember)
	if err := redisCli.LPush(ctx, cacheKey, groupMemberMarshal).Err(); err != nil {
		log.Printf("Error caching group member: %v", err)
	}
	err = redisCli.Expire(ctx, cacheKey, 7*24/time.Hour).Err()
	if err != nil {
		log.Printf("Error caching group member: %v", err)
	}
	// 假设申请已发送
	ctx.JSON(http.StatusOK, gin.H{"message": "Group join request sent successfully"})
}

//// 申请加入群组的请求
//func GroupApplication(ctx *gin.Context) {
//	var req request.GroupApplication
//	if err := ctx.ShouldBindJSON(&req); err != nil {
//		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//
//	// 验证用户ID和群组ID
//	if req.UserID <= 0 || req.GroupID <= 0 {
//		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID不正确"})
//		return
//	}
//
//	// 查找数据库中是否存在用户
//	var user model.User
//	db := database.GetDB()
//	if result := db.First(&user, req.UserID); result.Error != nil {
//		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
//			ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
//			return
//		}
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
//		return
//	}
//
//	// 查找数据库中是否存在群组
//	var group model.Group
//	if result := db.First(&group, req.GroupID); result.Error != nil {
//		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
//			ctx.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
//			return
//		}
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
//		return
//	}
//
//	// 检查用户是否已经发送过申请
//	var existingApplication request.GroupApplication
//	if result := db.Where("user_id = ? AND group_id = ?", req.UserID, req.GroupID).First(&existingApplication).Error; result == nil {
//		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Application already exists"})
//		return
//	}
//
//	// 发送申请
//	req.Status = request.Pending // 设置请求状态为待处理
//	if result := db.Create(&req).Error; result != nil {
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
//		return
//	}
//
//	// 假设申请已发送
//	ctx.JSON(http.StatusOK, gin.H{"message": "Group application sent successfully"})
//}

// GroupApplicationRedis 处理用户申请加入群组的逻辑
func GroupApplicationRedis(ctx *gin.Context) {
	var req request.GroupApplication
	// 解析请求参数
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效", "details": err.Error()})
		return
	}
	ID, _ := ctx.Get("userid")
	UserID := ID.(uint)
	req.UserID = int(UserID)
	// 参数校验
	if err := validateApplicationRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取数据库和Redis客户端
	db := database.GetDB()
	redisCli := database.GetRedisClient()

	var err error

	// 检查群组是否存在
	group, err := checkGroupExistence(ctx, req.GroupID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查是否已存在申请
	if exists, err := checkExistingApplication(ctx, db, redisCli, req, group); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else if exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "您已提交过申请"})
		return
	}

	// 创建申请记录
	if err := createGroupApplication(ctx, db, redisCli, &req, group); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "申请提交失败", "details": err.Error()})
		return
	}

	// 成功响应
	ctx.JSON(http.StatusOK, gin.H{"message": "群组申请发送成功"})
}

// validateApplicationRequest 验证申请参数
func validateApplicationRequest(req *request.GroupApplication) error {
	if req.GroupID <= 0 {
		return errors.New("群组ID无效")
	}
	return nil
}

// checkGroupExistence 检查群组是否存在
func checkGroupExistence(ctx *gin.Context, groupID int) (*model.Group, error) {
	db := database.GetDB()
	var group model.Group
	if err := db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "群组不存在"})
			return nil, err
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "群组查询失败"})
		return nil, err
	}
	return &group, nil
}

// checkExistingApplication 检查是否已存在申请
func checkExistingApplication(
	ctx *gin.Context,
	db *gorm.DB,
	redisCli *redis.Client,
	req request.GroupApplication,
	group *model.Group,
) (bool, error) {
	cacheKey := "group_application:" + strconv.Itoa(group.OwnerID)

	// 先检查Redis缓存
	_, err := redisCli.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中
		return true, nil
	}

	if !errors.Is(err, redis.Nil) {
		// Redis错误
		return false, err
	}

	// 缓存未命中，检查数据库
	var existingApplication request.GroupApplication
	result := db.Where("user_id = ? AND group_id = ?", req.UserID, req.GroupID).First(&existingApplication)

	return result.Error == nil, nil
}

// createGroupApplication 创建群组申请
func createGroupApplication(
	ctx *gin.Context,
	db *gorm.DB,
	redisCli *redis.Client,
	req *request.GroupApplication,
	group *model.Group,
) error {
	// 设置申请状态
	req.Status = request.Pending

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建申请记录
	if err := tx.Create(req).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 缓存申请信息
	cacheKey := "group_application:" + strconv.Itoa(group.OwnerID)
	applicationCacheMarshal, err := json.Marshal(req)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 使用 Set 替代 LPush，避免重复存储
	if err := redisCli.LPush(ctx, cacheKey, applicationCacheMarshal).Err(); err != nil {
		tx.Rollback()
		return err
	}
	err = redisCli.Expire(ctx, cacheKey, 7*24/time.Hour).Err()
	if err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// 可以添加其他辅助函数，如通知群主、限制申请频率等

func isgroupexist(ctx *gin.Context, GroupID int) (error, model.Group) {
	db := database.GetDB()
	redisCli := database.GetRedisClient()
	// 查找数据库中是否存在群组
	cacheKey := "group:" + strconv.Itoa(GroupID)
	groupCache, err := redisCli.Get(ctx, cacheKey).Result()
	var group model.Group

	if err == nil {
		// 缓存命中，解析缓存数据
		if err := json.Unmarshal([]byte(groupCache), &group); err != nil {
			log.Printf("Error unmarshalling group from cache: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling group from cache"})
			return err, group
		}
	} else {
		// 缓存未命中，从数据库中查询
		if result := db.Table("groups").Where("group_id =?", GroupID).First(&group); result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
				return err, group
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return err, group
		}

		// 缓存群组信息
		groupCache, _ := json.Marshal(group)
		if err := redisCli.Set(ctx, cacheKey, groupCache, 7*24*time.Hour).Err(); err != nil {
			log.Printf("Error caching group: %v", err)
		}
	}
	return nil, group
}

// IsGroupMember 检查用户是否是群组成员
func IsGroupMember(ctx *gin.Context, userID int, groupID int) (bool, error) {
	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 先从 Redis 缓存中查询
	memberCacheKey := "group_member:" + strconv.Itoa(groupID)
	cachedMembers, err := redisCli.LRange(ctx, memberCacheKey, 0, -1).Result()

	if err == nil && len(cachedMembers) > 0 {
		// 缓存中有数据，遍历查找
		for _, memberJSON := range cachedMembers {
			var member model.GroupMember
			if err := json.Unmarshal([]byte(memberJSON), &member); err != nil {
				log.Printf("解析群组成员缓存错误: %v", err)
				continue
			}

			if member.UserID == userID {
				return true, nil
			}
		}
	}

	// 缓存中未找到或发生错误，从数据库查询
	var count int64
	result := db.Table("group_members").
		Where("group_id = ? AND user_id = ?", strconv.Itoa(groupID), userID).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	// 如果找到记录，将结果缓存到 Redis
	if count > 0 && (err != nil || len(cachedMembers) == 0) {
		// 从数据库查询所有成员并缓存
		var members []model.GroupMember
		if err := db.Table("group_members").Where("group_id = ?", strconv.Itoa(groupID)).Find(&members).Error; err != nil {
			log.Printf("查询群组成员错误: %v", err)
		} else {
			// 清除旧缓存
			redisCli.Del(ctx, memberCacheKey)

			// 缓存所有成员
			pipe := redisCli.Pipeline()
			for _, member := range members {
				memberJSON, _ := json.Marshal(member)
				pipe.LPush(ctx, memberCacheKey, memberJSON)
			}
			pipe.Expire(ctx, memberCacheKey, 7*24*time.Hour)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Printf("缓存群组成员错误: %v", err)
			}
		}
	}

	return count > 0, nil
}

// IsFriends 检查两个用户是否已经是好友关系
// IsFriends 检查两个用户是否已经是好友关系
func IsFriends(ctx *gin.Context, userID, friendID int) (bool, error) {
	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 缓存键
	cacheKey := "friendship:" + strconv.Itoa(userID)

	// 尝试从缓存中获取好友列表
	friendshipCaches, err := redisCli.LRange(ctx, cacheKey, 0, -1).Result()
	if err == nil && len(friendshipCaches) > 0 {
		// 缓存命中，解析缓存数据
		for _, friendshipCache := range friendshipCaches {
			var friendship model.Friends
			if err := json.Unmarshal([]byte(friendshipCache), &friendship); err != nil {
				log.Printf("解析好友关系缓存错误: %v", err)
				continue
			}
			if friendship.FriendID == friendID {
				return true, nil
			}
		}
		// 如果缓存中存在好友列表但未找到特定好友，可以认为不是好友
		return false, nil
	}

	// 缓存未命中或发生错误，从数据库中查询
	var count int64
	if result := db.Table("friends").Where("user_id = ? AND friend_id = ?", userID, friendID).Count(&count); result.Error != nil {
		log.Printf("从数据库查询好友关系错误: %v", result.Error)
		return false, result.Error
	}

	// 如果数据库中没有找到记录，则不是好友
	if count == 0 {
		return false, nil
	}

	// 如果找到记录但缓存不存在或为空，则刷新缓存
	if err != nil || len(friendshipCaches) == 0 {
		var friendships []model.Friends
		if err := db.Table("friends").Where("user_id = ?", userID).Find(&friendships).Error; err != nil {
			log.Printf("查询用户好友列表错误: %v", err)
			// 虽然查询好友列表失败，但我们已经确认了这两个用户是好友
			return true, nil
		}

		// 清除旧缓存
		if err := redisCli.Del(ctx, cacheKey).Err(); err != nil {
			log.Printf("清除好友关系缓存错误: %v", err)
		}

		// 缓存所有好友关系
		pipe := redisCli.Pipeline()
		for _, fs := range friendships {
			friendshipCache, _ := json.Marshal(fs)
			pipe.LPush(ctx, cacheKey, friendshipCache)
		}
		pipe.Expire(ctx, cacheKey, 7*24*time.Hour) // 设置7天过期
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("缓存好友关系错误: %v", err)
		}
	}

	return true, nil
}

//// GetGroup 获取用户已经加入的群组列表
//func GetGroup(ctx *gin.Context, UserID int) (error, []model.Group) {
//	db := database.GetDB()
//	redisCli := database.GetRedisClient()
//
//	// 缓存键
//	cacheKey := "groupList:" + strconv.Itoa(UserID)
//
//	// 尝试从缓存中获取数据
//	groupCaches, err := redisCli.LRange(ctx, cacheKey, 0, -1).Result()
//	var groups []model.Group
//
//	if err == nil {
//		// 缓存命中，解析缓存数据
//		for _, groupCache := range groupCaches {
//			var group model.Group
//			if err := json.Unmarshal([]byte(groupCache), &group); err != nil {
//				log.Printf("Error unmarshalling group from cache: %v", err)
//				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling group from cache"})
//				return err, groups
//			}
//			groups = append(groups, group)
//		}
//	} else {
//		// 缓存未命中，从数据库中查询
//		var userGroups []model.GroupMember
//		if result := db.Where("user_id = ?", UserID).Find(&userGroups); result.Error != nil {
//			log.Printf("Error fetching user groups from database: %v", result.Error)
//			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user groups from database"})
//			return result.Error, groups
//		}
//
//		// 获取群组信息
//		groupIDs := make([]string, len(userGroups))
//		for i, ug := range userGroups {
//			groupIDs[i] = strconv.Itoa(ug.GroupID)
//		}
//
//		if result := db.Where("group_id IN ?", groupIDs).Find(&groups); result.Error != nil {
//			log.Printf("Error fetching groups from database: %v", result.Error)
//			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching groups from database"})
//			return result.Error, groups
//		}
//
//		// 缓存群组信息
//		for _, group := range groups {
//			groupCache, _ := json.Marshal(group)
//			if err := redisCli.LPush(ctx, cacheKey, groupCache).Err(); err != nil {
//				log.Printf("Error caching group: %v", err)
//			}
//		}
//		// 设置缓存过期时间
//		if err := redisCli.Expire(ctx, cacheKey, 7*24*time.Hour).Err(); err != nil {
//			log.Printf("Error setting cache expiration: %v", err)
//		}
//	}
//
//	return nil, groups
//}

// GetPendingGroupApplications 获取群主和管理员收到的待处理群组申请信息列表
func GetPendingGroupApplications(ctx *gin.Context, UserID int) (error, []request.GroupApplication) {
	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 缓存键
	cacheKey := fmt.Sprintf("GroupApplicationList:%d", UserID)

	// 尝试从缓存中获取数据
	applications, err := fetchApplicationsFromCache(ctx, redisCli, cacheKey)
	if err == nil && len(applications) > 0 {
		return nil, applications
	}

	// 缓存未命中，从数据库中查询
	applications, err = fetchApplicationsFromDatabase(ctx, db, UserID)
	if err != nil {
		return err, nil
	}

	// 缓存查询结果
	if err := cacheApplications(ctx, redisCli, cacheKey, applications); err != nil {
		log.Printf("Error caching group applications: %v", err)
	}

	return nil, applications
}

// fetchApplicationsFromCache 从Redis缓存获取申请列表
func fetchApplicationsFromCache(
	ctx *gin.Context,
	redisCli *redis.Client,
	cacheKey string,
) ([]request.GroupApplication, error) {
	var applications []request.GroupApplication

	// 使用 JSON 序列化的字符串获取缓存
	applicationCaches, err := redisCli.LRange(ctx, cacheKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	for _, applicationCache := range applicationCaches {
		var application request.GroupApplication
		if unmarshalErr := json.Unmarshal([]byte(applicationCache), &application); unmarshalErr != nil {
			log.Printf("Error unmarshalling group application from cache: %v", unmarshalErr)
			// 如果单个反序列化失败，继续处理其他缓存项
			continue
		}
		applications = append(applications, application)
	}

	return applications, nil
}

// fetchApplicationsFromDatabase 从数据库获取申请列表
func fetchApplicationsFromDatabase(
	ctx *gin.Context,
	db *gorm.DB,
	UserID int,
) ([]request.GroupApplication, error) {
	// 查询用户作为群主的群组
	ownerGroups, err := fetchOwnerGroups(db, UserID)
	if err != nil {
		log.Printf("Error fetching owner groups: %v", err)
		return nil, err
	}

	// 查询用户作为管理员的群组
	adminGroups, err := fetchAdminGroups(db, UserID)
	if err != nil {
		log.Printf("Error fetching admin groups: %v", err)
		return nil, err
	}

	// 合并并去重群组ID
	groupIDs := mergeGroupIDs(ownerGroups, adminGroups)

	// 查询待处理的群组申请
	return fetchPendingApplications(db, groupIDs)
}

// fetchOwnerGroups 获取用户作为群主的群组
func fetchOwnerGroups(db *gorm.DB, userID int) ([]model.Group, error) {
	var ownerGroups []model.Group
	result := db.Where("owner_id = ?", userID).Find(&ownerGroups)
	return ownerGroups, result.Error
}

// fetchAdminGroups 获取用户作为管理员的群组
func fetchAdminGroups(db *gorm.DB, userID int) ([]model.GroupMember, error) {
	var adminGroups []model.GroupMember
	result := db.Where("user_id = ? AND role = ?", userID, "admin").Find(&adminGroups)
	return adminGroups, result.Error
}

// mergeGroupIDs 合并并去重群组ID
func mergeGroupIDs(
	ownerGroups []model.Group,
	adminGroups []model.GroupMember,
) []int {
	groupIDs := make(map[int]bool)

	for _, group := range ownerGroups {
		groupIDs[group.GroupID] = true
	}

	for _, member := range adminGroups {
		groupIDs[member.GroupID] = true
	}

	// 转换为切片
	result := make([]int, 0, len(groupIDs))
	for groupID := range groupIDs {
		result = append(result, groupID)
	}

	return result
}

// fetchPendingApplications 获取指定群组的待处理申请
func fetchPendingApplications(
	db *gorm.DB,
	groupIDs []int,
) ([]request.GroupApplication, error) {
	var applications []request.GroupApplication

	// 查询7天内的待处理申请
	result := db.Where(
		"group_id IN ? AND status = ? AND created_at > ?",
		groupIDs,
		request.Pending,
		time.Now().Add(-7*24*time.Hour),
	).Find(&applications)

	return applications, result.Error
}

// cacheApplications 缓存申请列表
func cacheApplications(
	ctx *gin.Context,
	redisCli *redis.Client,
	cacheKey string,
	applications []request.GroupApplication,
) error {
	// 清除旧缓存
	if err := redisCli.Del(ctx, cacheKey).Err(); err != nil {
		return err
	}

	// 缓存新数据
	for _, application := range applications {
		applicationCache, err := json.Marshal(application)
		if err != nil {
			log.Printf("Error marshalling application: %v", err)
			continue
		}

		if err := redisCli.LPush(ctx, cacheKey, applicationCache).Err(); err != nil {
			log.Printf("Error caching group application: %v", err)
		}
	}

	// 设置缓存过期时间
	return redisCli.Expire(ctx, cacheKey, 7*24*time.Hour).Err()
}
