package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/controller"
	"github.com/helpleness/IMChatAdmin/middleware"
	"net/http"
)

func Collectrouters(r *gin.Engine) *gin.Engine {
	r.GET("/ip", func(ctx *gin.Context) { //函数返回ip
		ctx.String(http.StatusOK, ctx.ClientIP())
	})
	r.POST("/register", controller.Register)
	r.POST("/login", controller.Login)
	r.GET("/userinfo", middleware.AuthMiddleWare(), controller.Userinfo)
	r.POST("/upload", middleware.AuthMiddleWare(), controller.UploadFile)
	r.GET("/download", middleware.AuthMiddleWare(), controller.DownloadFile)
	r.POST("/friendAdd", middleware.AuthMiddleWare(), controller.FriendAdd)
	r.POST("/groupCreated", middleware.AuthMiddleWare(), controller.GroupCreated)
	r.POST("/groupAdd", middleware.AuthMiddleWare(), controller.GroupAdd)
	r.POST("/groupAddRedis", middleware.AuthMiddleWare(), controller.GroupAddRedis)
	r.POST("/groupApplication", middleware.AuthMiddleWare(), controller.GroupApplication)
	r.POST("/groupApplicationRedis", middleware.AuthMiddleWare(), controller.GroupApplicationRedis)
	r.GET("/queryAllActiveFriendAdds", middleware.AuthMiddleWare(), controller.QueryAllActiveFriendAdds)
	r.GET("/queryFriendAddByID", middleware.AuthMiddleWare(), controller.QueryFriendAddByID)
	r.GET("/getFriendRequestByID", middleware.AuthMiddleWare(), controller.GetFriendRequestByID)
	r.GET("/queryAllActiveGroupApplications", middleware.AuthMiddleWare(), controller.QueryAllActiveGroupApplications)
	r.GET("/queryGroupApplicationByID", middleware.AuthMiddleWare(), controller.QueryGroupApplicationByID)
	r.POST("/handleFriendAdd", middleware.AuthMiddleWare(), controller.HandleFriendAdd)
	r.POST("/handleGroupApplication", middleware.AuthMiddleWare(), controller.HandleGroupApplication)
	r.POST("/deleteFriendAddByID", middleware.AuthMiddleWare(), controller.DeleteFriendAddByID)
	r.POST("/deleteGroupApplicationByID", middleware.AuthMiddleWare(), controller.DeleteGroupApplicationByID)
	return r
}
