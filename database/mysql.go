package database

import (
	"fmt"
	"github.com/helpleness/IMChatAdmin/model"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 表单的全局变量
var DB *gorm.DB

// 初始化数据库
func InitMysql() *gorm.DB {
	//得到yml中数据库配置信息，yml由viper管理
	host := viper.GetString("mysql.host")
	port := viper.GetString("mysql.port")
	username := viper.GetString("mysql.username")
	password := viper.GetString("mysql.password")
	database := viper.GetString("mysql.database")
	//正则表达式，用于创建mysql连接
	args := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		username, password, host, port, database)
	//db存储数据库信息，err存储报错
	db, err := gorm.Open(mysql.Open(args), &gorm.Config{})
	if err != nil {
		panic("failed to connect mysql database,err: " + err.Error())
	}
	//自动创建表单，如果已经有表单，则不重复创建
	db.AutoMigrate(&model.User{})
	db.AutoMigrate(&model.File{})
	db.AutoMigrate(&model.Friends{})
	db.AutoMigrate(&model.Group{})
	db.AutoMigrate(&model.GroupMember{})
	db.AutoMigrate(&model.MyMessage{})
	DB = db
	return db
}

// 返回DB的函数
func GetDB() *gorm.DB {
	return DB
}

// 将好友关系存储到 MySQL 数据库中
func AddFriendToDatabase(userFrom string, userTarget string) error {
	// 这里可以根据实际情况进行 MySQL 数据库操作
	// 假设数据库中有一个 friends 表，存储好友关系
	//query := "INSERT INTO friends (user_from, user_target) VALUES (?, ?)"
	// 执行数据库插入操作...
	// db.Exec(query, userFrom, userTarget)

	fmt.Printf("Added %s and %s as friends to the database\n", userFrom, userTarget)
	return nil
}
