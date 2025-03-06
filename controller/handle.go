package controller

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/model"
	"github.com/helpleness/IMChatAdmin/model/request"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"
	"net/http"
)

// HandleFriendAdd 处理好友申请
func HandleFriendAdd(ctx *gin.Context) {
	var req request.FriendAdd
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db := database.GetDB()
	redisCli := database.GetRedisClient()
	err := isuserexist(ctx, req.UserID)
	if err != nil {
		ctx.JSON(200, gin.H{"error": "userid err"})
		return
	}
	err = isuserexist(ctx, req.FriendID)
	if err != nil {
		ctx.JSON(200, gin.H{"error": "friendid err"})
		return
	}
	isfriendsexist, err := IsFriends(ctx, req.UserID, req.FriendID)
	if err != nil {
		ctx.JSON(200, gin.H{"error": "friend err"})
		return
	}
	if isfriendsexist == true {
		ctx.JSON(200, gin.H{"msg": "friend ship exist"})
		return
	}
	// 更新请求状态
	req.Status = request.Accepted // 或 request.Rejected
	// 从数据库中获取当前记录
	var currentReq request.FriendAdd
	if result := db.Where("user_id = ? AND friend_id = ?", req.UserID, req.FriendID).First(&currentReq).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 只更新 Status 字段
	currentReq.Status = req.Status
	if result := db.Save(&currentReq).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}
	// 缓存键
	cacheKey := "friend_request:" + strconv.Itoa(req.FriendID)

	// 获取缓存中的所有好友申请
	originalRequests, err := redisCli.LRange(ctx, cacheKey, 0, -1).Result()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		log.Printf("Error cache original friend request: %v", err)
	}
	// 找到并删除匹配的原始好友申请
	for _, originalRequest := range originalRequests {
		var originalReq request.FriendAdd
		if err := json.Unmarshal([]byte(originalRequest), &originalReq); err != nil {
			log.Printf("Error unmarshalling original friend request: %v", err)
			continue
		}

		// 检查是否是需要删除的请求
		if originalReq.UserID == req.UserID && originalReq.FriendID == req.FriendID {
			// 删除匹配的原始请求
			if _, err := redisCli.LRem(ctx, cacheKey, 0, originalRequest).Result(); err != nil {
				log.Printf("Error deleting friend request from cache: %v", err)
			}
			break
		}
	}
	// 将好友关系添加到数据库
	friendship1 := model.Friends{
		UserID:    req.UserID,
		FriendID:  req.FriendID,
		CreatedAt: time.Now(),
	}
	if result := db.Table("friends").Create(&friendship1).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 将好友关系缓存到 Redis
	friendshipCacheKey := "friendship:" + strconv.Itoa(req.UserID)
	friendshipCache, _ := json.Marshal(friendship1)
	pipe := redisCli.Pipeline()
	pipe.LPush(ctx, friendshipCacheKey, friendshipCache)
	pipe.Expire(ctx, friendshipCacheKey, 7*24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		log.Printf("Error caching friend request with expiration: %v", err)
	}

	// 为对方也添加好友关系
	friendship2 := model.Friends{
		UserID:    req.FriendID,
		FriendID:  req.UserID,
		CreatedAt: time.Now(),
	}
	if result := db.Table("friends").Create(&friendship2).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	friendshipCacheKey = "friendship:" + strconv.Itoa(req.FriendID)
	friendshipCache, _ = json.Marshal(friendship2)

	pipe = redisCli.Pipeline()
	pipe.LPush(ctx, friendshipCacheKey, friendshipCache)
	pipe.Expire(ctx, friendshipCacheKey, 7*24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		log.Printf("Error caching friend request with expiration: %v", err)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Request handled successfully"})
}

// 处理群组申请
// HandleGroupApplication 处理群组申请
func HandleGroupApplication(ctx *gin.Context) {
	var req request.GroupApplication
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 检查用户是否存在
	err := isuserexist(ctx, req.UserID)
	if err != nil {
		ctx.JSON(200, gin.H{"error": "userid err"})
		return
	}

	// 检查群组是否存在
	err, group := isgroupexist(ctx, req.GroupID)
	if err != nil {
		ctx.JSON(200, gin.H{"error": "groupid err"})
		return
	}

	// 检查用户是否已经是群组成员
	isMember, err := IsGroupMember(ctx, req.UserID, req.GroupID)
	if err != nil {
		ctx.JSON(200, gin.H{"error": "group member check err"})
		return
	}
	if isMember {
		ctx.JSON(200, gin.H{"msg": "user already in group"})
		return
	}

	// 更新请求状态
	req.Status = request.Accepted // 或 request.Rejected

	// 从数据库中获取当前记录
	var currentReq request.GroupApplication
	if result := db.Where("user_id = ? AND group_id = ?", req.UserID, req.GroupID).First(&currentReq).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 只更新 Status 字段
	currentReq.Status = req.Status
	if result := db.Save(&currentReq).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 缓存键
	cacheKey := "group_application:" + strconv.Itoa(group.OwnerID)

	// 获取缓存中的所有群组申请
	originalRequests, err := redisCli.LRange(ctx, cacheKey, 0, -1).Result()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		log.Printf("获取原始群组申请缓存错误: %v", err)
	}

	// 找到并删除匹配的原始群组申请
	for _, originalRequest := range originalRequests {
		var originalReq request.GroupApplication
		if err := json.Unmarshal([]byte(originalRequest), &originalReq); err != nil {
			log.Printf("解析原始群组申请错误: %v", err)
			continue
		}

		// 检查是否是需要删除的请求
		if originalReq.UserID == req.UserID && originalReq.GroupID == req.GroupID {
			// 删除匹配的原始请求
			if _, err := redisCli.LRem(ctx, cacheKey, 0, originalRequest).Result(); err != nil {
				log.Printf("从缓存中删除群组申请错误: %v", err)
			}
			break
		}
	}

	// 将群组成员关系添加到数据库
	groupMember := model.GroupMember{
		GroupID:  strconv.Itoa(req.GroupID),
		UserID:   req.UserID,
		Role:     "member",
		JoinTime: time.Now(),
	}
	if result := db.Table("group_members").Create(&groupMember).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 将群组成员关系缓存到 Redis
	memberCacheKey := "group_member:" + strconv.Itoa(req.GroupID)
	memberCache, _ := json.Marshal(groupMember)

	pipe := redisCli.Pipeline()
	pipe.LPush(ctx, memberCacheKey, memberCache)
	pipe.Expire(ctx, memberCacheKey, 7*24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		log.Printf("缓存群组成员关系错误: %v", err)
	}

	// 更新用户的群组列表缓存
	userGroupCacheKey := "groupList:" + strconv.Itoa(req.UserID)
	if _, err := redisCli.Del(ctx, userGroupCacheKey).Result(); err != nil {
		log.Printf("删除用户群组缓存错误: %v", err)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "群组申请处理成功"})
}

// 删除过期的 FriendAdd 请求
func DeleteExpiredFriendAdds() {
	db := database.GetDB()
	var expiredFriendAdds []request.FriendAdd

	// 查询所有过期的请求
	result := db.Where("status = ? AND created_at <= ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Find(&expiredFriendAdds)
	if result.Error != nil {
		log.Printf("error: %v", result.Error.Error())
		return
	}

	// 删除过期的请求
	result = db.Delete(&expiredFriendAdds)
	if result.Error != nil {
		log.Printf("error: %v", result.Error.Error())
		return
	}
	log.Println("Expired requests deleted successfully")
}

// 删除过期的 GroupApplication 请求
func DeleteExpiredGroupApplications() {
	db := database.GetDB()
	var expiredGroupApplications []request.GroupApplication

	// 查询所有过期的请求
	result := db.Where("status = ? AND created_at <= ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Find(&expiredGroupApplications)
	if result.Error != nil {
		log.Printf("error: %v", result.Error.Error())
		return
	}

	// 删除过期的请求
	result = db.Delete(&expiredGroupApplications)
	if result.Error != nil {
		log.Printf("error: %v", result.Error.Error())
		return
	}
	log.Println("Expired requests deleted successfully")
}

// 删除指定 ID 的加好友请求
// 每次获取加好友列表时调用这个接口查看其是否过期删除
func DeleteFriendAddByID(ctx *gin.Context) {
	id := ctx.Param("id")
	reqID, err := strconv.Atoi(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	db := database.GetDB()
	var reqs []request.FriendAdd
	if result := db.First(&reqs, reqID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		log.Printf("Error querying friend add request: %v", result.Error)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	for _, req := range reqs {
		if req.CreatedAt.Add(7 * 24 * time.Hour).Before(time.Now()) {
			if result := db.Delete(&req); result.Error != nil {
				log.Printf("Error deleting friend add request: %v", result.Error)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"message": "Request deleted successfully"})
		} else {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Request is not expired"})
		}
	}

}

// 删除指定 ID 的加入群聊请求
// 每次获取加群聊列表时调用这个接口查看其是否过期删除
func DeleteGroupApplicationByID(ctx *gin.Context) {
	id := ctx.Param("id")
	reqID, err := strconv.Atoi(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	db := database.GetDB()
	var reqs []request.GroupApplication
	if result := db.First(&reqs, reqID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		log.Printf("Error querying group application request: %v", result.Error)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查找失败"})
		return
	}

	for _, req := range reqs {
		if req.CreatedAt.Add(7 * 24 * time.Hour).Before(time.Now()) {
			if result := db.Delete(&req); result.Error != nil {
				log.Printf("Error deleting group application request: %v", result.Error)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"message": "Request deleted successfully"})
		} else {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Request is not expired"})
		}
	}

}
