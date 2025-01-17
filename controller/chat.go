package controller

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
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

// 好友添加和处理
func FriendAdd(ctx *gin.Context) {
	var req request.FriendAdd
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 这里添加处理好友添加请求的逻辑
	// 例如，验证用户ID，发送好友请求等
	if req.UserID <= 0 || req.FriendID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID不正确"})
	}
	// 查找数据库中是否存在这两个ID
	var user, friend model.User
	db := database.GetDB()
	if result := db.First(&user, req.UserID); result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if result := db.First(&friend, req.FriendID); result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Friend not found"})
		return
	}
	// 发送好友请求
	req.Status = request.Pending // 设置请求状态为待处理
	db.Create(&req)
	// 假设我们已经处理了请求，并且添加成功
	ctx.JSON(http.StatusOK, gin.H{"message": "Friend request sent successfully"})
}

// 群聊创建
// 没写完
func GroupCreated(ctx *gin.Context) {
	var req request.GroupCreated
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证创建者ID
	if req.CreatorID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Creator ID不正确"})
		return
	}

	// 查找数据库中是否存在创建者
	var creator model.User
	db := database.GetDB()
	if result := db.First(&creator, req.CreatorID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Creator not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 创建群聊记录
	group := model.Group{
		GroupName: req.GroupName,
		OwnerID:   req.CreatorID,
	}
	if result := db.Create(&group).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
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
	// 添加初始成员
	for _, memberID := range req.InitialMembers {
		var member model.User
		if result := db.First(&member, memberID); result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "Member not found"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
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
	}

	// 假设我们已经创建了群聊
	ctx.JSON(http.StatusOK, gin.H{"message": "Group created successfully", "group_id": group.GroupID})
}

// 添加用户到群组的请求
func GroupAdd(ctx *gin.Context) {
	var req request.GroupAdd
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证用户ID和群组ID
	if req.UserID <= 0 || req.GroupID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID不正确"})
		return
	}

	// 查找数据库中是否存在用户
	var user model.User
	db := database.GetDB()
	if result := db.First(&user, req.UserID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 查找数据库中是否存在群组
	var group model.Group
	if result := db.First(&group, req.GroupID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 检查用户是否已经是群组成员
	var existingMember model.GroupMember
	if result := db.Where("group_id = ? AND user_id = ?", req.GroupID, req.UserID).First(&existingMember).Error; result == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "User is already a member of the group"})
		return
	}

	// 发送加入申请
	groupMember := model.GroupMember{
		GroupID: group.GroupID,
		UserID:  int(user.ID),
		Role:    "member",
	}
	if result := db.Create(&groupMember).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 假设申请已发送
	ctx.JSON(http.StatusOK, gin.H{"message": "Group join request sent successfully"})
}

// 申请加入群组的请求
func GroupApplication(ctx *gin.Context) {
	var req request.GroupApplication
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证用户ID和群组ID
	if req.UserID <= 0 || req.GroupID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID不正确"})
		return
	}

	// 查找数据库中是否存在用户
	var user model.User
	db := database.GetDB()
	if result := db.First(&user, req.UserID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 查找数据库中是否存在群组
	var group model.Group
	if result := db.First(&group, req.GroupID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 检查用户是否已经发送过申请
	var existingApplication request.GroupApplication
	if result := db.Where("user_id = ? AND group_id = ?", req.UserID, req.GroupID).First(&existingApplication).Error; result == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Application already exists"})
		return
	}

	// 发送申请
	req.Status = request.Pending // 设置请求状态为待处理
	if result := db.Create(&req).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 假设申请已发送
	ctx.JSON(http.StatusOK, gin.H{"message": "Group application sent successfully"})
}
