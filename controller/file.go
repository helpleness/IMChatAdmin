package controller

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/helpleness/IMChatAdmin/database"
	"github.com/helpleness/IMChatAdmin/model"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"io"
	"net/http"
)

func UploadFile(ctx *gin.Context) {
	bucketName := viper.GetString("minio.bucket")
	name := ctx.PostForm("name")
	//这里是从form表单里拿到key为file的文件
	file, err := ctx.FormFile("file")
	if err != nil {
		fmt.Println("upload file err:", err)
		return
	}
	//临时保存
	ctx.SaveUploadedFile(file, "./upload/"+name)
	MC := database.GetMinioClisnt()
	bucket, err := MC.BucketExists(context.Background(), bucketName)
	if err != nil {
		fmt.Println("minio bucket err:", err)
		return
	}
	if !bucket {
		fmt.Println("minio bucket doesn't exist")
		return
	}
	uploadInfo, err := MC.FPutObject(context.Background(), bucketName, name, "./upload/"+name, minio.PutObjectOptions{})
	if err != nil {
		fmt.Println("minio upload err:", err)
		return
	}
	//数据库
	DB := database.GetDB()
	newFile := &model.File{
		Name:   name,
		Bucket: bucketName,
	}
	//在数据库打上一条文件上传的命令
	DB.Table("files").Create(newFile)
	ctx.JSON(http.StatusOK, gin.H{
		"code": "200",
		"data": uploadInfo,
		"msg":  "上传成功",
	})
}
func DownloadFile(ctx *gin.Context) {
	bucketName := viper.GetString("minio.bucket")
	name := ctx.Query("name")
	MC := database.GetMinioClisnt()
	bucket, err := MC.BucketExists(context.Background(), bucketName)
	if err != nil {
		fmt.Println("minio bucket err:", err)
		return
	}
	if !bucket {
		fmt.Println("minio bucket doesn't exist")
		return
	}
	file, err := MC.GetObject(context.Background(), bucketName, name, minio.GetObjectOptions{})
	if err != nil {
		fmt.Println("minio download err:", err)
		return
	}
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, file)
	ctx.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", name))
	ctx.Writer.Header().Add("Content-Type", "application/octet-stream")
	ctx.Writer.Header().Add("Content-Transfer-Encoding", "binary")
	ctx.Data(http.StatusOK, "application/octet-stream", buf.Bytes())
}
