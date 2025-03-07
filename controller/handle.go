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
	"log"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
	"net/http"
)

// HandleFriendAdd 处理好友申请
func HandleFriendAdd(ctx *gin.Context) {
	var req model.FriendAdd
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ID, _ := ctx.Get("userid")
	UserID := ID.(uint)
	req.UserID = int(UserID)
	db := database.GetDB()
	redisCli := database.GetRedisClient()
	var err error
	_, err = middleware.Isuserexist(ctx, req.FriendID)
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
	req.Status = model.Accepted // 或 request.Rejected
	// 从数据库中获取当前记录
	var currentReq model.FriendAdd
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
	fmt.Printf("originalRequests: %v\n", cacheKey)
	fmt.Printf("originalRequests: %v\n", originalRequests)
	// 找到并删除匹配的原始好友申请
	for _, originalRequest := range originalRequests {
		var originalReq model.FriendAdd
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

// HandleGroupApplication 处理群组申请
func HandleGroupApplication(ctx *gin.Context) {
	var req model.GroupApplication
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ID, _ := ctx.Get("userid")
	UserID := ID.(uint)
	req.UserID = int(UserID)

	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 使用WaitGroup来协调所有并发操作
	var wg sync.WaitGroup
	var mu sync.Mutex // 用于保护错误变量和共享数据

	// 错误通道，用于收集并发操作的错误
	errChan := make(chan error, 2)
	var group model.Group
	var isMember bool

	// 同时检查群组是否存在和用户是否已经是群组成员
	wg.Add(2)

	// 检查群组是否存在
	go func() {
		defer wg.Done()
		var err error
		_, g := isgroupexist(ctx, req.GroupID)
		if err != nil {
			errChan <- fmt.Errorf("groupid err: %v", err)
			return
		}
		mu.Lock()
		group = g
		mu.Unlock()
	}()

	// 检查用户是否已经是群组成员
	go func() {
		defer wg.Done()
		var err error
		ismember, err := IsGroupMember(ctx, req.UserID, req.GroupID)
		if err != nil {
			errChan <- fmt.Errorf("group member check err: %v", err)
			return
		}
		mu.Lock()
		isMember = ismember
		mu.Unlock()
	}()

	// 等待并发检查完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for err := range errChan {
		ctx.JSON(200, gin.H{"error": err.Error()})
		return
	}

	// 检查用户是否已经是成员
	if isMember {
		ctx.JSON(200, gin.H{"msg": "user already in group"})
		return
	}

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新请求状态
	req.Status = model.Accepted // 或 request.Rejected

	// 从数据库中获取当前记录
	var currentReq model.GroupApplication
	if result := tx.Where("user_id = ? AND group_id = ?", req.UserID, req.GroupID).First(&currentReq).Error; result != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 只更新 Status 字段
	currentReq.Status = req.Status
	if result := tx.Save(&currentReq).Error; result != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 将群组成员关系添加到数据库
	groupMember := model.GroupMember{
		GroupID:  req.GroupID,
		UserID:   req.UserID,
		Role:     "member",
		JoinTime: time.Now(),
	}
	if result := tx.Table("group_members").Create(&groupMember).Error; result != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 立即返回成功响应，缓存清理工作在后台进行
	ctx.JSON(http.StatusOK, gin.H{"message": "群组申请处理成功"})

	// 后台并发处理所有的缓存操作
	go func() {
		// 用于协调缓存操作的WaitGroup
		var cachewg sync.WaitGroup
		cachewg.Add(3)

		// 处理群组申请缓存
		go func() {
			defer cachewg.Done()
			applicationCacheKey := fmt.Sprintf("GroupApplicationList:%d", group.OwnerID)

			// 批量处理方式1: 使用管道清除整个列表并重新加载，如果列表很长更有效
			ctx := context.Background() // 使用新的上下文
			pipe := redisCli.Pipeline()
			pipe.Del(ctx, applicationCacheKey)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Printf("清除群组申请缓存错误: %v", err)
			}

			// 如果需要重新加载缓存，可以在这里查询并添加
			// 这里省略了重新加载代码，实际使用时可以根据需求添加
		}()

		// 缓存群组成员关系
		go func() {
			defer cachewg.Done()
			ctx := context.Background()
			memberCacheKey := fmt.Sprintf("group_member:%d", req.GroupID)
			memberCache, err := json.Marshal(groupMember)
			if err != nil {
				log.Printf("序列化群组成员错误: %v", err)
				return
			}

			// 使用管道操作优化Redis访问
			pipe := redisCli.Pipeline()
			pipe.LPush(ctx, memberCacheKey, memberCache)
			pipe.Expire(ctx, memberCacheKey, 7*24*time.Hour)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Printf("缓存群组成员关系错误: %v", err)
			}
		}()

		// 更新用户群组列表缓存
		go func() {
			defer cachewg.Done()
			ctx := context.Background()
			userGroupCacheKey := fmt.Sprintf("groupList:%d", req.UserID)
			if err := redisCli.Del(ctx, userGroupCacheKey).Err(); err != nil {
				log.Printf("删除用户群组缓存错误: %v", err)
			}
		}()

		// 等待所有缓存操作完成
		cachewg.Wait()
		log.Printf("所有缓存操作已完成")
	}()
}

// 删除过期的 FriendAdd 请求
func DeleteExpiredFriendAdds() {
	db := database.GetDB()
	var expiredFriendAdds []model.FriendAdd

	// 查询所有过期的请求
	result := db.Where("status = ? AND created_at <= ?", model.Pending, time.Now().Add(-7*24*time.Hour)).Find(&expiredFriendAdds)
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
	var expiredGroupApplications []model.GroupApplication

	// 查询所有过期的请求
	result := db.Where("status = ? AND created_at <= ?", model.Pending, time.Now().Add(-7*24*time.Hour)).Find(&expiredGroupApplications)
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

// deleteRequestHandler 是一个通用的请求删除处理函数
func deleteRequestHandler[T any](
	ctx *gin.Context,
	findRequest func(*gorm.DB, int) (T, error),
	deleteRequest func(*gorm.DB, T) error,
	requestType string,
) {
	// 解析请求ID
	id := ctx.Query("id")
	fmt.Printf("id:%s", id)
	reqID, _ := strconv.Atoi(id)
	fmt.Printf("reqID:%d", reqID)
	//if err != nil {
	//	ctx.JSON(http.StatusBadRequest, gin.H{"error": err})
	//	return
	//}

	// 获取数据库连接
	db := database.GetDB()

	// 查找请求
	req, err := findRequest(db, reqID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "请求未找到"})
			return
		}
		log.Printf("查询 %s 请求错误: %v", requestType, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查找失败"})
		return
	}

	// 检查请求是否过期
	var isExpired bool
	switch v := any(req).(type) {
	case model.FriendAdd:
		isExpired = v.CreatedAt.Add(7 * 24 * time.Hour).Before(time.Now())
	case model.GroupApplication:
		isExpired = v.CreatedAt.Add(7 * 24 * time.Hour).Before(time.Now())
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "未知的请求类型"})
		return
	}

	// 处理过期请求
	if isExpired {
		// 开启事务
		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 删除请求
		if err := deleteRequest(tx, req); err != nil {
			tx.Rollback()
			log.Printf("删除 %s 请求错误: %v", requestType, err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
			return
		}

		// 提交事务
		if err := tx.Commit().Error; err != nil {
			log.Printf("事务提交错误: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "请求删除成功"})
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求未过期"})
	}
}

// DeleteFriendAddByID 删除指定 ID 的加好友请求
func DeleteFriendAddByID(ctx *gin.Context) {
	deleteRequestHandler(
		ctx,
		// 查找请求的函数
		func(db *gorm.DB, id int) (model.FriendAdd, error) {
			var req model.FriendAdd
			err := db.Where("id=?", id).First(&req).Error
			return req, err
		},
		// 删除请求的函数
		func(db *gorm.DB, req model.FriendAdd) error {
			return db.Delete(&req).Error
		},
		"加好友",
	)
}

// DeleteGroupApplicationByID 删除指定 ID 的加入群聊请求
func DeleteGroupApplicationByID(ctx *gin.Context) {
	deleteRequestHandler(
		ctx,
		// 查找请求的函数
		func(db *gorm.DB, id int) (model.GroupApplication, error) {
			var req model.GroupApplication
			err := db.Where("id=?", id).First(&req).Error
			return req, err
		},
		// 删除请求的函数
		func(db *gorm.DB, req model.GroupApplication) error {
			return db.Delete(&req).Error
		},
		"加群",
	)
}
