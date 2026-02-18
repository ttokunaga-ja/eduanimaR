package domain

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus は Kafka 非同期 Ingestion ジョブの処理状態
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// IngestJob は OCR/Embedding 処理の非同期ジョブエンティティ
type IngestJob struct {
	ID           uuid.UUID
	FileID       uuid.UUID
	Status       JobStatus
	RetryCount   int
	MaxRetries   int
	ErrorMessage *string
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

// CanRetry は再試行可能かどうかを返す
func (j *IngestJob) CanRetry() bool {
	return j.RetryCount < j.MaxRetries
}
