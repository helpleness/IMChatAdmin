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

// 处理好友申请
func HandleFriendAdd(ctx *gin.Context) {
	var req request.FriendAdd
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 更新请求状态
	req.Status = request.Accepted // 或 request.Rejected
	if result := db.Save(&req).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 删除缓存中的好友申请
	cacheKey := "friend_request:" + strconv.Itoa(req.UserID) + strconv.Itoa(req.FriendID)
	if _, err := redisCli.Del(ctx, cacheKey).Result(); err != nil {
		log.Printf("Error deleting friend request from cache: %v", err)
	}

	// 将好友关系添加到数据库
	friendship1 := model.Friends{
		UserID:    req.UserID,
		FriendID:  req.FriendID,
		CreatedAt: time.Now(),
	}
	if result := db.Create(&friendship1).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}
	// 将好友关系缓存到 Redis
	friendshipCacheKey := "friendship:" + strconv.Itoa(req.UserID)
	friendshipCache, _ := json.Marshal(friendship1)
	if err := redisCli.LPush(ctx, friendshipCacheKey, friendshipCache, 7*24*time.Hour).Err(); err != nil {
		log.Printf("Error caching friendship: %v", err)
	}
	// 将好友关系添加到数据库
	friendship2 := model.Friends{
		UserID:    req.FriendID,
		FriendID:  req.UserID,
		CreatedAt: time.Now(),
	}
	if result := db.Create(&friendship2).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}
	friendshipCacheKey = "friendship:" + strconv.Itoa(req.FriendID)
	friendshipCache, _ = json.Marshal(friendship2)
	if err := redisCli.LPush(ctx, friendshipCacheKey, friendshipCache, 7*24*time.Hour).Err(); err != nil {
		log.Printf("Error caching friendship: %v", err)
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Request handled successfully"})
}

// 处理群组申请
func HandleGroupApplication(ctx *gin.Context) {
	var req request.GroupApplication
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 更新请求状态
	req.Status = request.Accepted // 或 request.Rejected
	if result := db.Save(&req).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 删除缓存中的群组申请
	cacheKey := "group_application:" + strconv.Itoa(req.UserID) + ":" + strconv.Itoa(req.GroupID)
	if _, err := redisCli.Del(ctx, cacheKey).Result(); err != nil {
		log.Printf("Error deleting group application from cache: %v", err)
	}

	// 将群组成员关系添加到数据库
	groupMember := model.GroupMember{
		GroupID: strconv.Itoa(req.GroupID),
		UserID:  req.UserID,
		Role:    "member",
	}
	if result := db.Create(&groupMember).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 将群组成员关系缓存到 Redis
	memberCacheKey := "group_member:" + strconv.Itoa(req.GroupID)
	memberCache, _ := json.Marshal(groupMember)
	if err := redisCli.LPush(ctx, memberCacheKey, memberCache).Err(); err != nil {
		log.Printf("Error caching group member: %v", err)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Request handled successfully"})
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
