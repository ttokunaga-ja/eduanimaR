package ports

import (
	"context"
	"io"
	"time"
)

// ObjectStorage はオブジェクトストレージ操作を抽象化する。
// Phase 1: MinIO（S3互換）実装 / Phase 2: GCS 実装に差し替え
type ObjectStorage interface {
	// Upload はファイルをアップロードし、ストレージパスを返す
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (storagePath string, err error)
	// Download はファイルをダウンロードする
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete はファイルを削除する
	Delete(ctx context.Context, key string) error
	// GetPresignedURL は署名付き一時 URL を生成する（クライアント向けダウンロード用）
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
