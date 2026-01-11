package storage

import (
	"context"
	"corechain-communication/internal/config"
	"log"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	Client *minio.Client
	Bucket string
}

var Instance *MinioClient

func InitMinio() {
	cfg := config.Get()
	endpoint := strings.Replace(cfg.MinioPublicURL, "http://", "", 1)
	endpoint = strings.Replace(endpoint, "https://", "", 1)
	accessKey := cfg.MinIOAccessKey
	secretKey := cfg.MinIOSecretKey
	bucketName := cfg.MinIOBucketName
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln("Failed to initialize MinIO:", err)
	}

	Instance = &MinioClient{
		Client: minioClient,
		Bucket: bucketName,
	}

	log.Println("MinIO initialized successfully")
}

func GetPresignedURL(objectName string) (string, error) {
	if objectName == "" {
		return "", nil
	}

	expiry := time.Hour * 1

	presignedURL, err := Instance.Client.PresignedGetObject(
		context.Background(),
		Instance.Bucket,
		objectName,
		expiry,
		nil,
	)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}
