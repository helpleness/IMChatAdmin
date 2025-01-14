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
	"log"
	"net/http"
	"strconv"
	"time"
)

func PushMessage(ctx context.Context, user model.User) {

	time.Sleep(50 * time.Millisecond)
	reidsClient := database.GetRedisClient()
	key := fmt.Sprintf("messages:%s", user.ID)
	data, err := reidsClient.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Printf("Error 获取 %s 消息: %v", user.ID, err)
		return
	}

	for {
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
			err = reidsClient.LPush(ctx, queueName, data).Err()
			if err != nil {
				log.Printf("Error pushing message to Redis queue: %v", err)
				return
			}
			return
		} else {
			time.Sleep(50 * time.Millisecond)
		}
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
	// 假设我们已经处理了请求，并且添加成功
	ctx.JSON(http.StatusOK, gin.H{"message": "Friend request sent successfully"})
}

// 群聊创建
func GroupCreated(ctx *gin.Context) {
	var req request.GroupCreated
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 这里添加创建群聊的逻辑
	// 例如，验证创建者ID，创建群聊记录，添加初始成员等
	// 假设我们已经创建了群聊
	ctx.JSON(http.StatusOK, gin.H{"message": "Group created successfully", "group_id": 123}) // 假设返回新创建的群组ID
}

// 添加用户到群组的请求
func GroupAdd(ctx *gin.Context) {
	var req request.GroupAdd
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 这里添加处理群聊加入申请的逻辑
	// 例如，验证用户ID和群组ID，发送加入申请等
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

	// 这里添加处理群聊邀请加入申请的逻辑
	// 例如，验证用户ID和群组ID，发送邀请等
	// 假设邀请已发送
	ctx.JSON(http.StatusOK, gin.H{"message": "Group invitation sent successfully"})
}
