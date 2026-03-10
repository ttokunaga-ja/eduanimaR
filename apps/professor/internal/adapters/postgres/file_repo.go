package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/adapters/postgres/sqlcgen"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

type fileRepo struct {
	q *sqlcgen.Queries
}

// NewFileRepo は FileRepository 実装を返す。
func NewFileRepo(db *sql.DB) ports.FileRepository {
	return &fileRepo{q: sqlcgen.New(db)}
}

func (r *fileRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	row, err := r.q.GetFileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toFileDomainFromGet(row), nil
}

func (r *fileRepo) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.File, error) {
	row, err := r.q.GetFileByIDAndUserID(ctx, sqlcgen.GetFileByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toFileDomainFromGetByUser(row), nil
}

func (r *fileRepo) ListBySubjectID(ctx context.Context, subjectID uuid.UUID) ([]*domain.File, error) {
	rows, err := r.q.ListFilesBySubjectID(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.File, 0, len(rows))
	for _, row := range rows {
		out = append(out, toFileDomainFromList(row))
	}
	return out, nil
}

func (r *fileRepo) Create(ctx context.Context, f *domain.File) error {
	created, err := r.q.CreateFile(ctx, sqlcgen.CreateFileParams{
		FileID:      f.ID,
		SubjectID:   f.SubjectID,
		UserID:      f.UserID,
		Name:        f.Name,
		StoragePath: f.StoragePath,
		MimeType:    sql.NullString{String: f.MimeType, Valid: f.MimeType != ""},
		SizeBytes:   f.SizeBytes,
		Status:      sqlcgen.FileStatus(f.Status),
	})
	if err != nil {
		return err
	}
	f.UploadedAt = created.UploadedAt
	return nil
}

func (r *fileRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FileStatus, errMsg *string) (*domain.File, error) {
	row, err := r.q.UpdateFileStatus(ctx, sqlcgen.UpdateFileStatusParams{
		FileID: id,
		Status: sqlcgen.FileStatus(status),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toFileDomainFromUpdate(row), nil
}

func (r *fileRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteFile(ctx, sqlcgen.DeleteFileParams{
		FileID: id,
		UserID: userID,
	})
}

func toFileDomainCore(
	fileID, subjectID, userID uuid.UUID,
	name, storagePath string,
	mimeType sql.NullString,
	sizeBytes int64,
	status sqlcgen.FileStatus,
	errorMessage sql.NullString,
	uploadedAt time.Time,
	processedAt sql.NullTime,
) *domain.File {
	f := &domain.File{
		ID:          fileID,
		SubjectID:   subjectID,
		UserID:      userID,
		Name:        name,
		StoragePath: storagePath,
		SizeBytes:   sizeBytes,
		Status:      domain.FileStatus(status),
		UploadedAt:  uploadedAt,
	}
	if mimeType.Valid {
		f.MimeType = mimeType.String
	}
	if errorMessage.Valid {
		f.ErrorMessage = &errorMessage.String
	}
	if processedAt.Valid {
		t := processedAt.Time
		f.ProcessedAt = &t
	}
	_ = time.Time{} // suppress unused import if needed
	return f
}

func toFileDomainFromGet(row sqlcgen.GetFileByIDRow) *domain.File {
	return toFileDomainCore(row.FileID, row.SubjectID, row.UserID, row.Name, row.StoragePath, row.MimeType, row.SizeBytes, row.Status, row.ErrorMessage, row.UploadedAt, row.ProcessedAt)
}

func toFileDomainFromGetByUser(row sqlcgen.GetFileByIDAndUserIDRow) *domain.File {
	return toFileDomainCore(row.FileID, row.SubjectID, row.UserID, row.Name, row.StoragePath, row.MimeType, row.SizeBytes, row.Status, row.ErrorMessage, row.UploadedAt, row.ProcessedAt)
}

func toFileDomainFromList(row sqlcgen.ListFilesBySubjectIDRow) *domain.File {
	return toFileDomainCore(row.FileID, row.SubjectID, row.UserID, row.Name, row.StoragePath, row.MimeType, row.SizeBytes, row.Status, row.ErrorMessage, row.UploadedAt, row.ProcessedAt)
}

func toFileDomainFromUpdate(row sqlcgen.UpdateFileStatusRow) *domain.File {
	return toFileDomainCore(row.FileID, row.SubjectID, row.UserID, row.Name, row.StoragePath, row.MimeType, row.SizeBytes, row.Status, row.ErrorMessage, row.UploadedAt, row.ProcessedAt)
}
