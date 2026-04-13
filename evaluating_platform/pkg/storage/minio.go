package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"evaluating_platform/pkg/config"
)

// MinIOClient MinIO 存储客户端
type MinIOClient struct {
	client *minio.Client
	bucket string
}

// NewMinIOClient 创建 MinIO 客户端
func NewMinIOClient(cfg config.StorageConfig) (*MinIOClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	// 确保 bucket 存在
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &MinIOClient{client: client, bucket: cfg.Bucket}, nil
}

// Upload 上传数据到指定路径
func (m *MinIOClient) Upload(ctx context.Context, path string, data []byte, contentType string) error {
	reader := bytes.NewReader(data)
	_, err := m.client.PutObject(ctx, m.bucket, path, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("upload to %s: %w", path, err)
	}
	return nil
}

// Download 从指定路径下载数据
func (m *MinIOClient) Download(ctx context.Context, path string) ([]byte, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", path, err)
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", path, err)
	}
	return data, nil
}

// Delete 删除指定路径的文件
func (m *MinIOClient) Delete(ctx context.Context, path string) error {
	return m.client.RemoveObject(ctx, m.bucket, path, minio.RemoveObjectOptions{})
}
