package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/adapters/postgres/sqlcgen"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

func nullUUID(id uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: id, Valid: true}
}

type ingestJobRepo struct {
	q *sqlcgen.Queries
}

// NewIngestJobRepo は IngestJobRepository 実装を返す。
func NewIngestJobRepo(db *sql.DB) ports.IngestJobRepository {
	return &ingestJobRepo{q: sqlcgen.New(db)}
}

func (r *ingestJobRepo) Create(ctx context.Context, job *domain.IngestJob) error {
	created, err := r.q.CreateIngestJob(ctx, sqlcgen.CreateIngestJobParams{
		ID:              job.ID,
		TargetRawFileID: nullUUID(job.FileID),
		Status:          sqlcgen.JobStatus(job.Status),
		MaxRetries:      int32(job.MaxRetries),
		IdempotencyKey:  fmt.Sprintf("ingest:%s:%s", job.FileID, job.ID),
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
	return toIngestJobDomainFromGetByID(row), nil
}

func (r *ingestJobRepo) GetByFileID(ctx context.Context, fileID uuid.UUID) (*domain.IngestJob, error) {
	row, err := r.q.GetIngestJobByFileID(ctx, nullUUID(fileID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toIngestJobDomainFromGetByFile(row), nil
}

func (r *ingestJobRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errMsg *string) (*domain.IngestJob, error) {
	ns := sql.NullString{}
	if errMsg != nil {
		ns = sql.NullString{String: *errMsg, Valid: true}
	}
	row, err := r.q.UpdateIngestJobStatus(ctx, sqlcgen.UpdateIngestJobStatusParams{
		ID:           id,
		Status:       sqlcgen.JobStatus(status),
		ErrorMessage: ns,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toIngestJobDomainFromUpdate(row), nil
}

func toIngestJobDomainCore(jobID uuid.UUID, fileID uuid.NullUUID, status sqlcgen.JobStatus, retryCount, maxRetries int32, errorMessage sql.NullString, createdAt time.Time, startedAt, completedAt sql.NullTime) *domain.IngestJob {
	job := &domain.IngestJob{
		ID:         jobID,
		Status:     domain.JobStatus(status),
		RetryCount: int(retryCount),
		MaxRetries: int(maxRetries),
		CreatedAt:  createdAt,
	}
	if fileID.Valid {
		job.FileID = fileID.UUID
	}
	if errorMessage.Valid {
		job.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		t := startedAt.Time
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		job.CompletedAt = &t
	}
	return job
}

func toIngestJobDomainFromGetByID(row sqlcgen.GetIngestJobByIDRow) *domain.IngestJob {
	return toIngestJobDomainCore(row.JobID, row.FileID, row.Status, row.RetryCount, row.MaxRetries, row.ErrorMessage, row.CreatedAt, row.StartedAt, row.CompletedAt)
}

func toIngestJobDomainFromGetByFile(row sqlcgen.GetIngestJobByFileIDRow) *domain.IngestJob {
	return toIngestJobDomainCore(row.JobID, row.FileID, row.Status, row.RetryCount, row.MaxRetries, row.ErrorMessage, row.CreatedAt, row.StartedAt, row.CompletedAt)
}

func toIngestJobDomainFromUpdate(row sqlcgen.UpdateIngestJobStatusRow) *domain.IngestJob {
	return toIngestJobDomainCore(row.JobID, row.FileID, row.Status, row.RetryCount, row.MaxRetries, row.ErrorMessage, row.CreatedAt, row.StartedAt, row.CompletedAt)
}
