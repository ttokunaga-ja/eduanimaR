package domain

import (
	"time"

	"github.com/google/uuid"
)

// FileStatus はアップロードファイルの処理状態
type FileStatus string

const (
	FileStatusUploading  FileStatus = "uploading"
	FileStatusUploaded   FileStatus = "uploaded"
	FileStatusProcessing FileStatus = "processing"
	FileStatusCompleted  FileStatus = "completed"
	FileStatusFailed     FileStatus = "failed"
	FileStatusArchived   FileStatus = "archived"
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
	ProcessedAt  *time.Time // status=completed になった時刻
}
