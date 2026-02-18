package ports

import "context"

// IngestMessage は Kafka トピック "eduanima.ingest.jobs" に送信するメッセージ
type IngestMessage struct {
	JobID       string `json:"job_id"`
	FileID      string `json:"file_id"`
	SubjectID   string `json:"subject_id"`
	UserID      string `json:"user_id"`
	StoragePath string `json:"storage_path"` // MinIO/GCS パス
	MimeType    string `json:"mime_type"`
}

// MessagePublisher は Kafka プロデューサーを抽象化する
type MessagePublisher interface {
	PublishIngestJob(ctx context.Context, msg IngestMessage) error
	Close() error
}

// MessageConsumer は Kafka コンシューマーを抽象化する
type MessageConsumer interface {
	// ConsumeIngestJobs はメッセージを継続的に受信し、handler を呼び出す
	// handler がエラーを返した場合はリトライ（max_retries まで）
	ConsumeIngestJobs(ctx context.Context, handler func(ctx context.Context, msg IngestMessage) error) error
	Close() error
}
