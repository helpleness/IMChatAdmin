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
	return r
}
