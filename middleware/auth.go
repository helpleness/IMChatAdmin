package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/model"
	"net/http"
	"strings"
)

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
		DB := database.GetDB()
		var user model.User
		DB.Table("users").Where("id = ?", userID).First(&user)
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
