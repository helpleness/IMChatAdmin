package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/controller"
	"github.com/panjf2000/ants/v2"
	"github.com/robfig/cron/v3"
	"log"
	"time"
)

func InitPoll(ctx *gin.Context) {

	// 创建一个 ants 池
	p, _ := ants.NewPool(10, ants.WithExpiryDuration(5*time.Second))
	defer p.Release()

	// 创建一个定时任务调度器
	c := cron.New(cron.WithSeconds())

	// 添加定时任务，每小时执行一次
	_, err := c.AddFunc("@hourly", func() {
		_ = p.Submit(func() {
			controller.DeleteExpiredGroupApplications(ctx)
		})
	})
	if err != nil {
		log.Fatalf("Error adding cron job: %v", err)
	}

	// 启动定时任务
	c.Start()
}
