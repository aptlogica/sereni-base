package minio_storage

import (
	"fmt"
	"serenibase/internal/config"
	"serenibase/internal/providers/storage/interfaces"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewStorageProvider(cfg *config.StorageMinioConfig) (interfaces.StorageProvider, error) {
	endpoint := cfg.Endpoint
	accessKeyID := cfg.AccessKey
	secretAccessKey := cfg.SecretKey
	useSSL := cfg.UseSSL

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		fmt.Println(err)
	}

	utility := NewLocalStorageService(minioClient, cfg.Bucket)
	return utility, nil
}
