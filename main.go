package main

import (
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/config"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/routers"
	"github.com/helpleness/IMChatAdmin/utils"
	"github.com/spf13/viper"
)

func main() {
	//初始化yml配置
	config.ConfigInit()
	//mysql数据库初始化
	database.InitMysql()
	database.InitMinioClient()
	// 初始化连接 Single Redis 服务端
	database.InitClusterClient()

	//使用gin创建一个路由
	r := gin.Default()
	//路由绑定
	r = routers.Collectrouters(r)

	//获取运行端口
	port := viper.GetString("server.port")

	//初始化协程池，进行过期数据删除任务
	utils.InitPoll()

	//如果端口不为空就加上端口运行
	if port != "" {
		panic(r.Run(":" + port))
	}
	//端口为空就直接运行
	panic(r.Run())

}
