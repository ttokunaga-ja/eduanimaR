package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/postgres/sqlcgen"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

type ingestJobRepo struct {
	q *sqlcgen.Queries
}

// NewIngestJobRepo は IngestJobRepository 実装を返す。
func NewIngestJobRepo(db *sql.DB) ports.IngestJobRepository {
	return &ingestJobRepo{q: sqlcgen.New(db)}
}

func (r *ingestJobRepo) Create(ctx context.Context, job *domain.IngestJob) error {
	created, err := r.q.CreateIngestJob(ctx, sqlcgen.CreateIngestJobParams{
		JobID:      job.ID,
		FileID:     job.FileID,
		MaxRetries: int32(job.MaxRetries),
	})
	if err != nil {
		return err
	}
	job.CreatedAt = created.CreatedAt
	return nil
}

func (r *ingestJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.IngestJob, error) {
	row, err := r.q.GetIngestJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toIngestJobDomain(row), nil
}

func (r *ingestJobRepo) GetByFileID(ctx context.Context, fileID uuid.UUID) (*domain.IngestJob, error) {
	row, err := r.q.GetIngestJobByFileID(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toIngestJobDomain(row), nil
}

func (r *ingestJobRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errMsg *string) (*domain.IngestJob, error) {
	ns := sql.NullString{}
	if errMsg != nil {
		ns = sql.NullString{String: *errMsg, Valid: true}
	}
	row, err := r.q.UpdateIngestJobStatus(ctx, sqlcgen.UpdateIngestJobStatusParams{
		JobID:        id,
		Status:       sqlcgen.JobStatus(status),
		ErrorMessage: ns,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toIngestJobDomain(row), nil
}

func toIngestJobDomain(row sqlcgen.IngestJob) *domain.IngestJob {
	job := &domain.IngestJob{
		ID:         row.JobID,
		FileID:     row.FileID,
		Status:     domain.JobStatus(row.Status),
		RetryCount: int(row.RetryCount),
		MaxRetries: int(row.MaxRetries),
		CreatedAt:  row.CreatedAt,
	}
	if row.ErrorMessage.Valid {
		job.ErrorMessage = &row.ErrorMessage.String
	}
	if row.StartedAt.Valid {
		t := row.StartedAt.Time
		job.StartedAt = &t
	}
	if row.CompletedAt.Valid {
		t := row.CompletedAt.Time
		job.CompletedAt = &t
	}
	return job
}
