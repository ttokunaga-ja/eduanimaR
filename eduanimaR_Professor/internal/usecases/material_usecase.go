package usecases

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

// MaterialUseCase は教材（ファイル）に関するビジネスロジックを提供する。
type MaterialUseCase struct {
	files     ports.FileRepository
	jobs      ports.IngestJobRepository
	storage   ports.ObjectStorage
	publisher ports.MessagePublisher
	subjects  ports.SubjectRepository
}

// NewMaterialUseCase は MaterialUseCase を生成する。
func NewMaterialUseCase(
	files ports.FileRepository,
	jobs ports.IngestJobRepository,
	storage ports.ObjectStorage,
	publisher ports.MessagePublisher,
	subjects ports.SubjectRepository,
) *MaterialUseCase {
	return &MaterialUseCase{
		files:     files,
		jobs:      jobs,
		storage:   storage,
		publisher: publisher,
		subjects:  subjects,
	}
}

// ListBySubject は科目に属する教材一覧を返す。
func (uc *MaterialUseCase) ListBySubject(ctx context.Context, subjectID, userID uuid.UUID) ([]*domain.File, error) {
	// subject の所有権確認
	if _, err := uc.subjects.GetByIDAndUserID(ctx, subjectID, userID); err != nil {
		return nil, err
	}
	return uc.files.ListBySubjectID(ctx, subjectID)
}

// UploadMaterialInput はファイルアップロードの入力値
type UploadMaterialInput struct {
	SubjectID uuid.UUID
	UserID    uuid.UUID
	FileName  string
	MimeType  string
	Size      int64
	Reader    io.Reader
}

// Upload は教材ファイルをアップロードし、非同期 OCR/Embedding ジョブを登録する。
func (uc *MaterialUseCase) Upload(ctx context.Context, in UploadMaterialInput) (*domain.File, error) {
	// subject の所有権確認
	if _, err := uc.subjects.GetByIDAndUserID(ctx, in.SubjectID, in.UserID); err != nil {
		return nil, err
	}

	fileID := uuid.New()
	// MinIO キー: {userID}/{subjectID}/{fileID}/{fileName}
	key := fmt.Sprintf("%s/%s/%s/%s", in.UserID, in.SubjectID, fileID, in.FileName)

	storagePath, err := uc.storage.Upload(ctx, key, in.Reader, in.Size, in.MimeType)
	if err != nil {
		return nil, err
	}

	file := &domain.File{
		ID:          fileID,
		SubjectID:   in.SubjectID,
		UserID:      in.UserID,
		Name:        in.FileName,
		StoragePath: storagePath,
		MimeType:    in.MimeType,
		SizeBytes:   in.Size,
		Status:      domain.FileStatusPending,
		UploadedAt:  time.Now().UTC(),
	}
	if err := uc.files.Create(ctx, file); err != nil {
		return nil, err
	}

	// 非同期 OCR/Embedding ジョブを作成
	job := &domain.IngestJob{
		ID:         uuid.New(),
		FileID:     fileID,
		Status:     domain.JobStatusPending,
		RetryCount: 0,
		MaxRetries: 3,
		CreatedAt:  time.Now().UTC(),
	}
	if err := uc.jobs.Create(ctx, job); err != nil {
		return nil, err
	}

	// Kafka メッセージ送信（失敗してもユーザーにはエラーを返さない）
	msg := ports.IngestMessage{
		JobID:       job.ID.String(),
		FileID:      fileID.String(),
		SubjectID:   in.SubjectID.String(),
		UserID:      in.UserID.String(),
		StoragePath: storagePath,
		MimeType:    in.MimeType,
	}
	if err := uc.publisher.PublishIngestJob(ctx, msg); err != nil {
		// Kafka 失敗はログだけ（ワーカーが DB をスキャンしてリカバリ可能）
		slog.Warn("kafka publish failed, worker will retry from DB",
			"job_id", job.ID.String(),
			"error", err,
		)
	}

	return file, nil
}

// Delete は教材ファイルをストレージと DB から削除する。
func (uc *MaterialUseCase) Delete(ctx context.Context, fileID, userID uuid.UUID) error {
	file, err := uc.files.GetByIDAndUserID(ctx, fileID, userID)
	if err != nil {
		return err
	}
	// ストレージから削除
	if err := uc.storage.Delete(ctx, file.StoragePath); err != nil {
		slog.Warn("storage delete failed", "key", file.StoragePath, "error", err)
	}
	return uc.files.Delete(ctx, fileID, userID)
}
