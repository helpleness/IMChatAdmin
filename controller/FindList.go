package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"time"
)

// QueryAllActiveFriendAdds 查询当前用户的所有未过期的 FriendAdd 请求
func QueryAllActiveFriendAdds(ctx *gin.Context) {
	// 获取当前用户ID
	ID, _ := ctx.Get("userid")
	UserID := ID.(uint)
	userIDInt := int(UserID)

	// 缓存键
	cacheKey := "friend_request:" + strconv.Itoa(userIDInt)
	// 尝试从缓存中获取数据
	redisCli := database.GetRedisClient()
	var friendAddCaches []string
	var err error
	friendAddCaches, err = redisCli.LRange(ctx, cacheKey, 0, -1).Result()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取cache错误"})
		return
	}
	var friendAdds []model.FriendAdd
	fmt.Println("Redis Cache Data:", friendAddCaches) // 打印缓存数据，看看是否为空
	if err == nil {
		// 缓存命中，解析缓存数据
		for _, friendAddCache := range friendAddCaches {
			fmt.Println("Redis Cache Data:", friendAddCache) // 打印缓存数据，看看是否为空
			var friendAdd model.FriendAdd
			if err := json.Unmarshal([]byte(friendAddCache), &friendAdd); err != nil {
				log.Printf("Error unmarshalling friend add request from cache: %v", err)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling friend add request from cache"})
				return
			}
			friendAdds = append(friendAdds, friendAdd)
		}
	} else {
		// 缓存未命中，从数据库中查询
		db := database.GetDB()
		// 查询所有未过期的请求
		result := db.Table("friend_adds").Where("user_id = ? AND status = ? AND created_at > ?", userIDInt, model.Pending, time.Now().Add(-7*24*time.Hour)).Find(&friendAdds)
		if result.Error != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		// 缓存查询结果
		for _, friendAdd := range friendAdds {
			friendAddCache, _ := json.Marshal(friendAdd)
			if err := redisCli.LPush(ctx, cacheKey, friendAddCache).Err(); err != nil {
				log.Printf("Error caching friend add request: %v", err)
			}
		}
		// 设置缓存过期时间
		if err := redisCli.Expire(ctx, cacheKey, 7*24*time.Hour).Err(); err != nil {
			log.Printf("Error setting cache expiration: %v", err)
		}
	}

	// 返回查询结果
	ctx.JSON(http.StatusOK, friendAdds)
}

//// 根据 ID 查询好友申请
//func QueryFriendAddByID(ctx *gin.Context) {
//	var req request.FriendAdd
//	id := ctx.Query("id")
//
//	// 将字符串 ID 转换为 int
//	reqID, err := strconv.Atoi(id)
//	if err != nil {
//		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
//		return
//	}
//
//	db := database.GetDB()
//	if result := db.First(&req, reqID); result.Error != nil {
//		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
//			ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
//			return
//		}
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
//		return
//	}
//
//	ctx.JSON(http.StatusOK, req)
//}
//
//// 旁路缓存查询指定 ID 的好友请求
//func GetFriendRequestByID(ctx *gin.Context) {
//	userid := ctx.Query("user_id")
//	cacheKey := "friend_request:" + userid
//
//	db := database.GetDB()
//	redisCli := database.GetRedisClient()
//	// 从 Redis 缓存中获取
//	var req request.FriendAdd
//	cacheValue, err := redisCli.Get(ctx, cacheKey).Result()
//	if err == nil {
//		// 缓存命中
//		if err := json.Unmarshal([]byte(cacheValue), &req); err != nil {
//			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling cached data"})
//			return
//		}
//		ctx.JSON(http.StatusOK, req)
//		return
//	} else if err != redis.Nil {
//		// 缓存获取失败
//		log.Printf("Error getting friend request from cache: %v", err)
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting data from cache"})
//		return
//	}
//
//	// 缓存未命中，从数据库中获取
//	if result := db.Where(" user_id = ?", userid).First(&req); result.Error != nil {
//		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
//			ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
//			return
//		}
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
//		return
//	}
//
//	// 缓存到 Redis
//	if err := redisCli.LPush(ctx, cacheKey, req, 7*24*time.Hour).Err(); err != nil {
//		log.Printf("Error caching friend request: %v", err)
//	}
//
//	ctx.JSON(http.StatusOK, req)
//}

// 查询所有未过期的 GroupApplication 请求
func QueryAllActiveGroupApplications(ctx *gin.Context) {
	ID, _ := ctx.Get("userid")
	UserID := ID.(uint)
	userIdToInt := int(UserID)

	var groupApplications []model.GroupApplication
	id := userIdToInt
	var err error
	err, groupApplications = GetPendingGroupApplications(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, groupApplications)
}

//// 根据 ID 查询群聊加入申请
//func QueryGroupApplicationByID(ctx *gin.Context) {
//	userid := ctx.Param("user_id")
//	groupid := ctx.Param("group_id")
//
//	// 将字符串 ID 转换为 int
//	reqID, err := strconv.Atoi(userid)
//	if err != nil {
//		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
//		return
//	}
//	// 检查群组是否存在
//	atoi, _ := strconv.Atoi(groupid)
//	group, err := checkGroupExistence(ctx, atoi)
//	if err != nil {
//		return
//	}
//	redisCli := database.GetRedisClient()
//	cacheKey := "group_application:" + strconv.Itoa(group.OwnerID)
//	var groupApplication request.GroupApplication
//
//	// 从缓存中获取请求
//	cacheValue, err := redisCli.Get(ctx, cacheKey).Result()
//	if err == nil {
//		// 缓存命中，解析缓存数据
//		if err := json.Unmarshal([]byte(cacheValue), &groupApplication); err != nil {
//			log.Printf("Error unmarshalling group application from cache: %v", err)
//			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling group application from cache"})
//			return
//		}
//	} else if err != redis.Nil {
//		// 缓存获取失败
//		log.Printf("Error getting group application from cache: %v", err)
//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting group application from cache"})
//		return
//	} else {
//		// 缓存未命中，从数据库中查询
//		db := database.GetDB()
//		if result := db.First(&groupApplication, reqID); result.Error != nil {
//			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
//				ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
//				return
//			}
//			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
//			return
//		}
//
//		// 缓存查询结果
//		cacheMarshal, _ := json.Marshal(groupApplication)
//		if err := redisCli.Set(ctx, cacheKey, cacheMarshal, 7*24*time.Hour).Err(); err != nil {
//			log.Printf("Error caching group application: %v", err)
//		}
//	}
//
//	ctx.JSON(http.StatusOK, groupApplication)
//}

// getListHandler 是一个通用的获取列表的处理函数
func getListHandler[T any](
	ctx *gin.Context,
	userID int,
	cacheKeyPrefix string,
	findUserRelations func(*gorm.DB, int) ([]int, error),
	findItems func(*gorm.DB, []int) ([]T, error),
	cacheItem func(T) ([]byte, error),
) (error, []T) {
	db := database.GetDB()
	redisCli := database.GetRedisClient()

	// 缓存键
	cacheKey := fmt.Sprintf("%s:%d", cacheKeyPrefix, userID)

	// 尝试从缓存中获取数据
	var items []T
	cachedItems, err := redisCli.LRange(ctx, cacheKey, 0, -1).Result()

	if err == nil && len(cachedItems) > 0 {
		// 缓存命中，解析缓存数据
		for _, itemCache := range cachedItems {
			var item T
			if unmarshalErr := json.Unmarshal([]byte(itemCache), &item); unmarshalErr != nil {
				log.Printf("Error unmarshalling item from cache: %v", unmarshalErr)
				break
			}
			items = append(items, item)
		}

		// 如果成功解析，直接返回
		if len(items) > 0 {
			return nil, items
		}
	}

	// 缓存未命中或解析失败，从数据库中查询
	// 先查询用户关系（群组成员或好友关系）
	relatedIDs, err := findUserRelations(db, userID)
	if err != nil {
		log.Printf("Error fetching user relations: %v", err)
		return err, nil
	}

	// 如果没有关联关系，直接返回空列表
	if len(relatedIDs) == 0 {
		return nil, []T{}
	}

	// 根据关联ID查询具体项目
	items, err = findItems(db, relatedIDs)
	if err != nil {
		log.Printf("Error fetching items: %v", err)
		return err, nil
	}

	// 缓存查询结果
	go func() {
		// 清除旧缓存
		redisCli.Del(ctx, cacheKey)

		// 缓存新数据
		for _, item := range items {
			itemCache, err := cacheItem(item)
			if err != nil {
				log.Printf("Error marshalling item: %v", err)
				continue
			}

			if err := redisCli.LPush(ctx, cacheKey, itemCache).Err(); err != nil {
				log.Printf("Error caching item: %v", err)
			}
		}

		// 设置缓存过期时间
		if err := redisCli.Expire(ctx, cacheKey, 7*24*time.Hour).Err(); err != nil {
			log.Printf("Error setting cache expiration: %v", err)
		}
	}()

	return nil, items
}

// GetGroup 获取用户已经加入的群组列表
func GetGroup(ctx *gin.Context, UserID int) (error, []model.Group) {
	return getListHandler(
		ctx,
		UserID,
		"groupList",
		// 查找用户群组关系
		func(db *gorm.DB, userID int) ([]int, error) {
			var userGroups []model.GroupMember
			if result := db.Where("user_id = ?", userID).Find(&userGroups); result.Error != nil {
				return nil, result.Error
			}

			groupIDs := make([]int, len(userGroups))
			for i, ug := range userGroups {
				groupIDs[i] = ug.GroupID
			}
			return groupIDs, nil
		},
		// 根据群组ID查找群组
		func(db *gorm.DB, groupIDs []int) ([]model.Group, error) {
			var groups []model.Group
			result := db.Where("group_id IN ?", groupIDs).Find(&groups)
			return groups, result.Error
		},
		// 缓存项目的序列化方法
		func(group model.Group) ([]byte, error) {
			return json.Marshal(group)
		},
	)
}

// GetFriends 获取用户已经添加的好友列表
func GetFriends(ctx *gin.Context, UserID int) (error, []model.User) {
	return getListHandler(
		ctx,
		UserID,
		"friendList",
		// 查找用户好友关系
		func(db *gorm.DB, userID int) ([]int, error) {
			var friendships []model.Friends
			if result := db.Where("user_id = ? OR friend_id = ?", userID, userID).Find(&friendships); result.Error != nil {
				return nil, result.Error
			}

			friendIDs := make([]int, 0, len(friendships))
			for _, friendship := range friendships {
				friendID := friendship.UserID
				if friendID == userID {
					friendID = friendship.FriendID
				}
				friendIDs = append(friendIDs, friendID)
			}
			return friendIDs, nil
		},
		// 根据好友ID查找用户信息
		func(db *gorm.DB, friendIDs []int) ([]model.User, error) {
			var friends []model.User
			result := db.Where("id IN ?", friendIDs).Find(&friends)
			return friends, result.Error
		},
		// 缓存项目的序列化方法
		func(user model.User) ([]byte, error) {
			return json.Marshal(user)
		},
	)
}

// HTTP 处理器示例
func GetGroupHandler(ctx *gin.Context) {
	// 从 JWT 或会话中获取用户ID
	userID, exists := ctx.Get("userid")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证的用户"})
		return
	}

	// 类型转换
	userIDInt, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "用户ID类型错误"})
		return
	}

	// 获取群组列表
	err, groups := GetGroup(ctx, int(userIDInt))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取群组失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"groups": groups,
	})
}

// HTTP 处理器示例
func GetFriendsHandler(ctx *gin.Context) {
	// 从 JWT 或会话中获取用户ID
	userID, exists := ctx.Get("userid")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证的用户"})
		return
	}

	// 类型转换
	userIDInt, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "用户ID类型错误"})
		return
	}

	// 获取好友列表
	err, friends := GetFriends(ctx, int(userIDInt))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取好友失败"})
		return
	}

	type FriendResp struct {
		ID        uint   `json:"id"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	}

	jsonData, err := json.Marshal(friends)
	//// 转换好友列表为响应格式
	var friendsResponse []FriendResp
	if err := json.Unmarshal(jsonData, &friendsResponse); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "数据转换失败"})
		return
	}
	//for _, friend := range friends {
	//	friendsResponse = append(friendsResponse, FriendResp{
	//		ID:        friend.ID,
	//		Username:  friend.Username,
	//		AvatarURL: friend.AvatarURL,
	//	})
	//}

	ctx.JSON(http.StatusOK, gin.H{
		"friends": friendsResponse,
	})
}

// GetGroupMembers 获取群成员列表的接口
func GetGroupMembers(ctx *gin.Context) {
	// 获取URL参数中的群组ID
	groupIDStr := ctx.Query("group_id")
	if groupIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "群组ID不能为空"})
		return
	}

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	// 获取Redis和数据库连接
	redisCli := database.GetRedisClient()
	db := database.GetDB()

	// 首先尝试从缓存获取群成员
	members, err := getGroupMembersFromCache(ctx, redisCli, groupID)

	// 如果缓存中没有数据或发生错误，则从数据库获取
	if err != nil || len(members) == 0 {
		members, err = getGroupMembersFromDB(ctx, db, groupID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取群成员失败: %v", err)})
			return
		}

		// 将数据库中获取的成员信息缓存到Redis
		go cacheGroupMembers(context.Background(), redisCli, groupID, members)
	}

	// 返回结果
	ctx.JSON(http.StatusOK, gin.H{
		"group_id": groupID,
		"count":    len(members),
		"members":  members,
	})
}

// getGroupMembersFromCache 从Redis缓存中获取群成员
func getGroupMembersFromCache(ctx context.Context, redisCli *redis.Client, groupID int) ([]model.GroupMember, error) {
	cacheKey := fmt.Sprintf("group_member:%d", groupID)

	// 从Redis列表中获取所有成员数据
	cachedData, err := redisCli.LRange(ctx, cacheKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	if len(cachedData) == 0 {
		return nil, nil
	}

	var members []model.GroupMember

	// 反序列化每个成员数据
	for _, data := range cachedData {
		var member model.GroupMember
		if err := json.Unmarshal([]byte(data), &member); err != nil {
			log.Printf("解析群成员缓存错误: %v", err)
			continue
		}
		members = append(members, member)
	}

	return members, nil
}

// getGroupMembersFromDB 从数据库中获取群成员
func getGroupMembersFromDB(ctx context.Context, db *gorm.DB, groupID int) ([]model.GroupMember, error) {
	startTime := time.Now()

	var members []model.GroupMember

	result := db.Where("group_id = ?", groupID).Find(&members)

	if result.Error != nil {
		return nil, result.Error
	}

	log.Printf("从数据库获取群成员耗时: %v", time.Since(startTime))

	return members, nil
}

// cacheGroupMembers 将群成员信息缓存到Redis
func cacheGroupMembers(ctx context.Context, redisCli *redis.Client, groupID int, members []model.GroupMember) {
	startTime := time.Now()
	cacheKey := fmt.Sprintf("group_member:%d", groupID)

	// 使用管道操作批量处理
	pipe := redisCli.Pipeline()

	// 首先删除旧缓存
	pipe.Del(ctx, cacheKey)

	// 添加所有成员到缓存
	for _, member := range members {
		memberJSON, err := json.Marshal(member)
		if err != nil {
			log.Printf("序列化群成员错误: %v", err)
			continue
		}
		pipe.LPush(ctx, cacheKey, memberJSON)
	}

	// 设置过期时间为7天
	pipe.Expire(ctx, cacheKey, 7*24*time.Hour)

	// 执行所有命令
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("缓存群成员错误: %v", err)
	} else {
		log.Printf("缓存群成员成功, 耗时: %v, 成员数: %d", time.Since(startTime), len(members))
	}
}

// 优化版：GetGroupMembersV2使用聚合结果集的缓存策略
func GetGroupMembersV2(ctx *gin.Context) {
	// 获取URL参数中的群组ID
	groupIDStr := ctx.Query("group_id")
	if groupIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "群组ID不能为空"})
		return
	}

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	// 获取Redis和数据库连接
	redisCli := database.GetRedisClient()
	db := database.GetDB()

	// 缓存键
	cacheKey := fmt.Sprintf("group_members_v2:%d", groupID)

	// 尝试从缓存获取
	cachedData, err := redisCli.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中
		var response gin.H
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			ctx.JSON(http.StatusOK, response)
			return
		}
	}

	// 缓存未命中，从数据库获取
	var members []model.GroupMember
	if err := db.Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取群成员失败: %v", err)})
		return
	}

	// 构建响应
	response := gin.H{
		"group_id": groupID,
		"count":    len(members),
		"members":  members,
	}

	// 将响应缓存到Redis (10分钟过期)
	go func() {
		if responseData, err := json.Marshal(response); err == nil {
			redisCli.Set(context.Background(), cacheKey, responseData, 10*time.Minute)
		}
	}()

	// 返回结果
	ctx.JSON(http.StatusOK, response)
}
