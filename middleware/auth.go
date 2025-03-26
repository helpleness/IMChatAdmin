package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func Isuserexist(ctx context.Context, UserID int, db *gorm.DB, redisCli *redis.Client) (model.User, error) {

	// 查找数据库中是否存在用户
	cacheKey := "user:" + strconv.Itoa(UserID)
	userCache, err := redisCli.Get(ctx, cacheKey).Result()
	var user model.User

	if err == nil {
		// 缓存命中，解析缓存数据
		if err := json.Unmarshal([]byte(userCache), &user); err != nil {
			log.Printf("Error unmarshalling user from cache: %v", err)
			//ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling user from cache"})
			return user, err
		}
	} else {
		// 缓存未命中，从数据库中查询
		if result := db.Table("users").Where("id =?", UserID).First(&user); result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				//ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return user, err
			}
			//ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return user, err
		}

		// 缓存用户信息
		userCacheMarshal, _ := json.Marshal(user)
		if err := redisCli.Set(ctx, cacheKey, userCacheMarshal, 0).Err(); err != nil {
			log.Printf("Error caching user: %v", err)
		}
	}

	return user, nil
}
func AuthMiddleWare() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.GetHeader("Authorization")
		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"data": "token验证失败",
				"mag":  "token 验证失败",
			})
			ctx.Abort()
			return
		}
		tokenString = tokenString[7:]
		token, claims, err := ParseToken(tokenString)
		if err != nil || !token.Valid {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"data": "token失效",
				"msg":  "token失效",
			})
			ctx.Abort()
			return
		}
		userID := claims.UserID
		//DB := database.GetDB()
		var user model.User
		//DB.Table("users").Where("id = ?", userID).First(&user)
		db := database.GetDB()
		redisCli := database.GetRedisClient()
		user, err = Isuserexist(ctx, int(userID), db, redisCli)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if user.ID == 0 {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{
				"code": 401,
				"data": "用户不存在",
				"msg":  "用户不存在",
			})
			ctx.Abort()
			return
		}
		ctx.Set("user", user)
		ctx.Set("userid", user.ID)
		ctx.Next()
	}
}
