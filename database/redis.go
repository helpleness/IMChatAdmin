package database

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"time"
)

// 定义一个全局变量
var (
	RedisClient *redis.Client
)

// 结构体方法
func InitClusterClient() *redis.Client {

	masteraddr := viper.GetString("redis.masteraddr")
	password := viper.GetString("redis.password")

	RedisClient = redis.NewClient(&redis.Options{
		Addr:         masteraddr,             // redis服务ip:port
		Password:     password,               // redis的认证密码
		DB:           0,                      // 连接的database库
		PoolSize:     100,                    // 连接池
		MinIdleConns: 10,                     // 最小空闲连接数
		DialTimeout:  500 * time.Millisecond, // 连接超时
		ReadTimeout:  500 * time.Millisecond, // 读取超时
		WriteTimeout: 500 * time.Millisecond, // 写入超时
		PoolTimeout:  1 * time.Second,        // 连接池超时
		MaxRetries:   3,                      // 命令执行失败时的最大重试次数
	})
	//defer func(RedisClient *redis.Client) {
	//	err := RedisClient.Close()
	//	if err != nil {
	//		log.Printf("err:%v", err)
	//		panic(err)
	//	}
	//}(RedisClient)
	fmt.Printf("Redis Config - Addr: %s, Password: %s\n", masteraddr, password)

	// go-redis库v8版本相关命令都需要传递context.Context参数,Background 返回一个非空的Context,它永远不会被取消，没有值，也没有期限。
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 验证是否连接到redis服务端
	res, err := RedisClient.Ping(ctx).Result()

	if err != nil {
		fmt.Printf("Connect Failed! Err: %v\n", err)
		panic(err)
	}
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Connect Failed! Err: %v\n", err)
		}
	}()
	// 输出连接成功标识
	fmt.Printf("Connect Successful! \nPing => %v\n", res)
	return RedisClient

}

func GetRedisClient() *redis.Client {
	return RedisClient
}

/*

// 程序执行完毕释放资源
	defer func(redisClient *redis.Client) {
		err := redisClient.Close()
		if err != nil {

		}
	}(redisClient)

*/
