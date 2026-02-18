package domain

import (
	"time"

	"github.com/google/uuid"
)

// FileStatus はアップロードファイルの処理状態
type FileStatus string

const (
	FileStatusPending    FileStatus = "pending"    // アップロード受付済み・OCR待ち
	FileStatusProcessing FileStatus = "processing" // OCR/Embedding 処理中
	FileStatusReady      FileStatus = "ready"      // 検索可能な状態
	FileStatusFailed     FileStatus = "failed"     // 処理失敗
)

// File はアップロードファイルエンティティ
// StoragePath: Phase 1 は MinIO パス / Phase 2 は GCS パス
type File struct {
	ID           uuid.UUID
	SubjectID    uuid.UUID
	UserID       uuid.UUID
	Name         string
	StoragePath  string // "minio://bucket/key" (Phase 1) / "gs://bucket/key" (Phase 2)
	MimeType     string
	SizeBytes    int64
	Status       FileStatus
	ErrorMessage *string // status=failed 時のエラー詳細
	UploadedAt   time.Time
	ProcessedAt  *time.Time // status=ready になった時刻
}
