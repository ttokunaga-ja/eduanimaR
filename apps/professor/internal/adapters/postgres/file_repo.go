package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/postgres/sqlcgen"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
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
	return toFileDomain(row), nil
}

func (r *fileRepo) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.File, error) {
	row, err := r.q.GetFileByIDAndUserID(ctx, sqlcgen.GetFileByIDAndUserIDParams{
		FileID: id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toFileDomain(row), nil
}

func (r *fileRepo) ListBySubjectID(ctx context.Context, subjectID uuid.UUID) ([]*domain.File, error) {
	rows, err := r.q.ListFilesBySubjectID(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.File, 0, len(rows))
	for _, row := range rows {
		out = append(out, toFileDomain(row))
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
		MimeType:    f.MimeType,
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
	ns := sql.NullString{}
	if errMsg != nil {
		ns = sql.NullString{String: *errMsg, Valid: true}
	}
	row, err := r.q.UpdateFileStatus(ctx, sqlcgen.UpdateFileStatusParams{
		FileID:       id,
		Status:       sqlcgen.FileStatus(status),
		ErrorMessage: ns,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toFileDomain(row), nil
}

func (r *fileRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteFile(ctx, sqlcgen.DeleteFileParams{
		FileID: id,
		UserID: userID,
	})
}

func toFileDomain(row sqlcgen.File) *domain.File {
	f := &domain.File{
		ID:          row.FileID,
		SubjectID:   row.SubjectID,
		UserID:      row.UserID,
		Name:        row.Name,
		StoragePath: row.StoragePath,
		MimeType:    row.MimeType,
		SizeBytes:   row.SizeBytes,
		Status:      domain.FileStatus(row.Status),
		UploadedAt:  row.UploadedAt,
	}
	if row.ErrorMessage.Valid {
		f.ErrorMessage = &row.ErrorMessage.String
	}
	if row.ProcessedAt.Valid {
		t := row.ProcessedAt.Time
		f.ProcessedAt = &t
	}
	_ = time.Time{} // suppress unused import if needed
	return f
}
