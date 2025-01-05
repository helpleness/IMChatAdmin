package database

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
	"log"
)

var MC *minio.Client

func InitMinioClient() *minio.Client {
	minioEndpoint := viper.GetString("minio.endpoint")
	accessID := viper.GetString("minio.accessID")
	secretKey := viper.GetString("minio.accessKey")
	minioOpt := minio.Options{
		Creds: credentials.NewStaticV4(accessID, secretKey, ""),
	}
	mc, err := minio.New(minioEndpoint, &minioOpt)
	if err != nil {
		log.Fatalln(err)
	}
	MC = mc
	return mc
}

func GetMinioClisnt() *minio.Client {
	return MC
}
