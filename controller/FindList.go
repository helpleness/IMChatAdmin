package controller

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/model/request"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"time"
)

// 查询当前用户的所有未过期的 FriendAdd 请求
func QueryAllActiveFriendAdds(ctx *gin.Context) {
	// 获取当前用户ID
	db := database.GetDB()
	userID := ctx.Query("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// 将字符串转换为整数
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
		return
	}

	// 查询所有未过期的请求
	var friendAdds []request.FriendAdd
	result := db.Where("user_id = ? AND status = ? AND created_at > ?", userIDInt, request.Pending, time.Now().Add(-7*24*time.Hour)).Find(&friendAdds)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	ctx.JSON(http.StatusOK, friendAdds)
}

// 根据 ID 查询好友申请
func QueryFriendAddByID(ctx *gin.Context) {
	var req request.FriendAdd
	id := ctx.Query("id")

	// 将字符串 ID 转换为 int
	reqID, err := strconv.Atoi(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	db := database.GetDB()
	if result := db.First(&req, reqID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	ctx.JSON(http.StatusOK, req)
}

// 旁路缓存查询指定 ID 的好友请求
func GetFriendRequestByID(ctx *gin.Context) {
	userid := ctx.Query("user_id")
	friendid := ctx.Query("friend_id")
	cacheKey := "friend_request:" + userid + friendid

	db := database.GetDB()
	redisCli := database.GetRedisClient()
	// 从 Redis 缓存中获取
	var req request.FriendAdd
	cacheValue, err := redisCli.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中
		if err := json.Unmarshal([]byte(cacheValue), &req); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling cached data"})
			return
		}
		ctx.JSON(http.StatusOK, req)
		return
	} else if err != redis.Nil {
		// 缓存获取失败
		log.Printf("Error getting friend request from cache: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting data from cache"})
		return
	}

	// 缓存未命中，从数据库中获取
	if result := db.Where(" user_id = ?", userid).First(&req); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 缓存到 Redis
	if err := redisCli.Set(ctx, cacheKey, req, 7*24*time.Hour).Err(); err != nil {
		log.Printf("Error caching friend request: %v", err)
	}

	ctx.JSON(http.StatusOK, req)
}

// 查询所有未过期的 GroupApplication 请求
func QueryAllActiveGroupApplications(ctx *gin.Context) {
	userid := ctx.Query("user_id")
	cacheKey := "active_group_applications" + userid
	var groupApplications []request.GroupApplication
	db := database.GetDB()
	redisCli := database.GetRedisClient()
	// 从缓存中获取未过期的请求
	cacheValue, err := redisCli.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，解析缓存数据
		if err := json.Unmarshal([]byte(cacheValue), &groupApplications); err != nil {
			log.Printf("Error unmarshalling group applications from cache: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling group applications from cache"})
			return
		}
	} else if err != redis.Nil {
		// 缓存获取失败
		log.Printf("Error getting group applications from cache: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting group applications from cache"})
		return
	} else {
		// 缓存未命中，从数据库中查询
		result := db.Where("status = ? AND created_at > ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Find(&groupApplications)
		if result.Error != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		// 缓存查询结果
		cacheMarshal, _ := json.Marshal(groupApplications)
		if err := redisCli.Set(ctx, cacheKey, cacheMarshal, 7*24*time.Hour).Err(); err != nil {
			log.Printf("Error caching group applications: %v", err)
		}
	}

	ctx.JSON(http.StatusOK, groupApplications)
}

// 根据 ID 查询群聊加入申请
func QueryGroupApplicationByID(ctx *gin.Context) {
	userid := ctx.Param("user_id")
	groupid := ctx.Param("group_id")

	// 将字符串 ID 转换为 int
	reqID, err := strconv.Atoi(userid)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	redisCli := database.GetRedisClient()
	cacheKey := "group_application:" + userid + groupid
	var groupApplication request.GroupApplication

	// 从缓存中获取请求
	cacheValue, err := redisCli.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，解析缓存数据
		if err := json.Unmarshal([]byte(cacheValue), &groupApplication); err != nil {
			log.Printf("Error unmarshalling group application from cache: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling group application from cache"})
			return
		}
	} else if err != redis.Nil {
		// 缓存获取失败
		log.Printf("Error getting group application from cache: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting group application from cache"})
		return
	} else {
		// 缓存未命中，从数据库中查询
		db := database.GetDB()
		if result := db.First(&groupApplication, reqID); result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		// 缓存查询结果
		cacheMarshal, _ := json.Marshal(groupApplication)
		if err := redisCli.Set(ctx, cacheKey, cacheMarshal, 7*24*time.Hour).Err(); err != nil {
			log.Printf("Error caching group application: %v", err)
		}
	}

	ctx.JSON(http.StatusOK, groupApplication)
}
