// Package storage はオブジェクトストレージアダプターを提供する。
package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

type minioAdapter struct {
	client *minio.Client
	bucket string
}

// NewMinioAdapter は MinIO を使った ObjectStorage 実装を返す。
// Phase 1 用。Phase 2 では GCS に差し替える。
func NewMinioAdapter(endpoint, accessKey, secretKey, bucket string, useSSL bool) (ports.ObjectStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	// バケットが存在しない場合は作成
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &minioAdapter{client: client, bucket: bucket}, nil
}

// Upload はファイルを MinIO にアップロードし、ストレージキーを返す。
func (a *minioAdapter) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := a.client.PutObject(ctx, a.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

// Download は MinIO からファイルをダウンロードする。
func (a *minioAdapter) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := a.client.GetObject(ctx, a.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Delete は MinIO からファイルを削除する。
func (a *minioAdapter) Delete(ctx context.Context, key string) error {
	return a.client.RemoveObject(ctx, a.bucket, key, minio.RemoveObjectOptions{})
}

// GetPresignedURL は MinIO の署名付き一時 URL を生成する。
func (a *minioAdapter) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := a.client.PresignedGetObject(ctx, a.bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
