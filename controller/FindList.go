package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/model/request"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"
)

// 查询所有未过期的 FriendAdd 请求
func QueryAllActiveFriendAdds(ctx *gin.Context) {
	db := database.GetDB()
	var friendAdds []request.FriendAdd

	// 查询所有未过期的请求
	result := db.Where("status = ? AND created_at > ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Find(&friendAdds)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	ctx.JSON(http.StatusOK, friendAdds)
}

// 根据 ID 查询好友申请
func QueryFriendAddByID(ctx *gin.Context) {
	var req request.FriendAdd
	id := ctx.Param("id")

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

// 查询所有未过期的 GroupApplication 请求
func QueryAllActiveGroupApplications(ctx *gin.Context) {
	db := database.GetDB()
	var groupApplications []request.GroupApplication

	// 查询所有未过期的请求
	result := db.Where("status = ? AND created_at > ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Find(&groupApplications)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	ctx.JSON(http.StatusOK, groupApplications)
}

// 根据 ID 查询群聊加入申请
func QueryGroupApplicationByID(ctx *gin.Context) {
	var req request.GroupApplication
	id := ctx.Param("id")

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
