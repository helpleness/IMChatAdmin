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
	r.POST("/friendAdd", middleware.AuthMiddleWare(), controller.FriendAdd)       //好友添加
	r.POST("/groupCreated", middleware.AuthMiddleWare(), controller.GroupCreated) //群聊创建
	//	r.POST("/groupAdd", middleware.AuthMiddleWare(), controller.GroupAdd)                                              //// 添加用户到群组的请求
	r.POST("/groupAddRedis", middleware.AuthMiddleWare(), controller.GroupAddRedis) //旁路缓存添加用户到群组的请求
	//	r.POST("/groupApplication", middleware.AuthMiddleWare(), controller.GroupApplication)                              //申请加入群组的请求
	r.POST("/groupApplicationRedis", middleware.AuthMiddleWare(), controller.GroupApplicationRedis)      // 旁路缓存申请加入群组的请求
	r.GET("/queryAllActiveFriendAdds", middleware.AuthMiddleWare(), controller.QueryAllActiveFriendAdds) // QueryAllActiveFriendAdds 查询当前用户的所有未过期的 FriendAdd 请求
	//	r.GET("/queryFriendAddByID", middleware.AuthMiddleWare(), controller.QueryFriendAddByID)                           // 根据 ID 查询好友申请
	//	r.GET("/getFriendRequestByID", middleware.AuthMiddleWare(), controller.GetFriendRequestByID)                       // 旁路缓存查询指定 ID 的好友请求
	r.GET("/queryAllActiveGroupApplications", middleware.AuthMiddleWare(), controller.QueryAllActiveGroupApplications) //查询所有未过期的 GroupApplication 请求
	//r.GET("/queryGroupApplicationByID", middleware.AuthMiddleWare(), controller.QueryGroupApplicationByID)             // 根据 ID 查询群聊加入申请
	r.POST("/handleFriendAdd", middleware.AuthMiddleWare(), controller.HandleFriendAdd)                      // HandleFriendAdd 处理好友申请
	r.POST("/handleGroupApplication", middleware.AuthMiddleWare(), controller.HandleGroupApplication)        // 处理群组申请
	r.GET("/deleteFriendAddByID", middleware.AuthMiddleWare(), controller.DeleteFriendAddByID)               // 删除指定 ID 的加好友请求 // 每次获取加好友列表时调用这个接口查看其是否过期删除
	r.GET("/deleteGroupApplicationByID", middleware.AuthMiddleWare(), controller.DeleteGroupApplicationByID) // 删除指定 ID 的加入群聊请求 // 每次获取加群聊列表时调用这个接口查看其是否过期删除
	return r
}
