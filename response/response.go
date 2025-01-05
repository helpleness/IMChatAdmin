package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Success(ctx *gin.Context, code int, data, msg string) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": code,
		"data": data,
		"msg":  msg,
	})
}
func Fail(ctx *gin.Context, code int, data, msg string) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": code,
		"data": msg,
		"msg":  msg,
	})
}
func Responce(ctx *gin.Context, httpStatus, code int, data, msg string) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": code,
		"data": data,
		"msg":  msg,
	})
}
