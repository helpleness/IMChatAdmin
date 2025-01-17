package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/model/request"
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
	if result := db.Where("id = ?", req.ID).First(&req).Error; result != nil {
		if errors.Is(result, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 更新请求状态
	req.Status = request.Accepted // 或 request.Rejected
	if result := db.Save(&req).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
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
	if result := db.Where("id = ?", req.ID).First(&req).Error; result != nil {
		if errors.Is(result, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	// 更新请求状态
	req.Status = request.Accepted // 或 request.Rejected
	if result := db.Save(&req).Error; result != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Request handled successfully"})
}

// 删除过期的 FriendAdd 请求
func DeleteExpiredFriendAdds(ctx *gin.Context) {
	db := database.GetDB()
	var expiredFriendAdds []request.FriendAdd

	// 查询所有过期的请求
	result := db.Where("status = ? AND created_at <= ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Find(&expiredFriendAdds)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 删除过期的请求
	result = db.Where("status = ? AND created_at <= ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Delete(&request.FriendAdd{})
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Expired requests deleted successfully", "expired_requests": expiredFriendAdds})
}

// 删除过期的 GroupApplication 请求
func DeleteExpiredGroupApplications(ctx *gin.Context) {
	db := database.GetDB()
	var expiredGroupApplications []request.GroupApplication

	// 查询所有过期的请求
	result := db.Where("status = ? AND created_at <= ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Find(&expiredGroupApplications)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 删除过期的请求
	result = db.Where("status = ? AND created_at <= ?", request.Pending, time.Now().Add(-7*24*time.Hour)).Delete(&request.GroupApplication{})
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Expired requests deleted successfully", "expired_requests": expiredGroupApplications})
}
