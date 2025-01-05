package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/middleware"
	"github.com/helpleness/IMChatAdmin/model"
	"github.com/helpleness/IMChatAdmin/response"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
)

// 注册函数
func Register(ctx *gin.Context) {
	DB := database.GetDB()
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")
	//过滤错误信息
	if len(username) == 0 || len(password) == 0 {
		response.Fail(ctx, 400, "username or password is empty", "username or password is empty")
		return
	}
	if len(password) < 6 {
		response.Fail(ctx, 400, "password is too short", "password is too short")
		return
	}
	if isUserExits(DB, username) {
		response.Success(ctx, 400, "user exits", "user exits")
		return
	}
	//hash加密密码
	hashdPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	//返回加密时的错误
	if err != nil {
		response.Fail(ctx, 500, "fail", "fail")
		return
	}
	//用户信息
	newUser := model.User{
		Username: username,
		Password: string(hashdPassword),
	}
	//把用户信息写入数据库中
	DB.Create(&newUser)
	//写入成功。注册成功。
	var user model.User
	DB.Table("users").Where("username = ?", username).First(&user)

	token, err := middleware.ReleaseToken(user)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": "500",
			"msg":  "加密错误",
			"data": "加密错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": "200",
		"data": gin.H{"token": token},
		"msg":  "注册成功",
	})
}

// 登录函数
func Login(ctx *gin.Context) {
	//得到数据库信息
	DB := database.GetDB()
	//得到从前端获取的账号密码
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")
	//过滤错误信息
	if len(password) < 6 {
		response.Fail(ctx, 400, "password is too short", "password is too short")
		return
	}
	//创建用户名变量，在数据库中查找获取到的用户名
	var user model.User
	DB.Table("users").Where("username = ?", username).First(&user)
	//找不到用户时：
	if user.ID == 0 {
		response.Fail(ctx, 400, "user exits", "user exits")
		return
	}
	//匹配用户密码，不匹配返回错误信息
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		response.Fail(ctx, 422, "password is wrong", "password is wrong")
		return
	}
	token, err := middleware.ReleaseToken(user)
	if err != nil {
		response.Fail(ctx, 500, "token加密错误", "token加密错误")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{

		"code": 200,
		"data": gin.H{
			"token": token,
		},
		"msg": "login success",
	})
	return
}

// 查找用户名是否存在的函数。
func isUserExits(db *gorm.DB, username string) bool {
	var user model.User
	db.Table("users").Where("username = ?", username).First(&user)
	return user.ID != 0
}

func Userinfo(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	u := user.(model.User)
	ctx.JSON(http.StatusOK, gin.H{
		"code": "200",
		"data": gin.H{
			"ID":       u.ID,
			"username": u.Username,
		},
		"msg": "注册成功",
	})

}