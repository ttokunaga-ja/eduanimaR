// Package postgres は PostgreSQL を使った ports インターフェースの実装を提供する。
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

type subjectRepo struct {
	q *sqlcgen.Queries
}

// NewSubjectRepo は SubjectRepository 実装を返す。
func NewSubjectRepo(db *sql.DB) ports.SubjectRepository {
	return &subjectRepo{q: sqlcgen.New(db)}
}

func (r *subjectRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Subject, error) {
	rows, err := r.q.ListSubjectsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Subject, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSubjectDomainFromList(row))
	}
	return out, nil
}

func (r *subjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subject, error) {
	row, err := r.q.GetSubjectByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toSubjectDomainFromGet(row), nil
}

func (r *subjectRepo) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Subject, error) {
	row, err := r.q.GetSubjectByIDAndUserID(ctx, sqlcgen.GetSubjectByIDAndUserIDParams{
		ID:          id,
		OwnerUserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toSubjectDomainFromGetByUser(row), nil
}

func (r *subjectRepo) Create(ctx context.Context, s *domain.Subject) error {
	ns := sql.NullString{}
	if s.LMSCourseID != nil {
		ns = sql.NullString{String: *s.LMSCourseID, Valid: true}
	}
	created, err := r.q.CreateSubject(ctx, sqlcgen.CreateSubjectParams{
		SubjectID:   s.ID,
		UserID:      s.UserID,
		Name:        s.Name,
		LmsCourseID: ns,
	})
	if err != nil {
		return err
	}
	s.CreatedAt = created.CreatedAt
	s.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *subjectRepo) UpdateName(ctx context.Context, id, userID uuid.UUID, name string) (*domain.Subject, error) {
	row, err := r.q.UpdateSubjectName(ctx, sqlcgen.UpdateSubjectNameParams{
		Name:      name,
		SubjectID: id,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toSubjectDomainFromUpdate(row), nil
}

func (r *subjectRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteSubject(ctx, sqlcgen.DeleteSubjectParams{
		SubjectID: id,
		UserID:    userID,
	})
}

func toSubjectDomainCore(subjectID, userID uuid.UUID, name string, lmsCourseID sql.NullString, createdAt, updatedAt time.Time) *domain.Subject {
	s := &domain.Subject{
		ID:        subjectID,
		UserID:    userID,
		Name:      name,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if lmsCourseID.Valid {
		s.LMSCourseID = &lmsCourseID.String
	}
	return s
}

func toSubjectDomainFromList(row sqlcgen.ListSubjectsByUserIDRow) *domain.Subject {
	return toSubjectDomainCore(row.SubjectID, row.UserID, row.Name, row.LmsCourseID, row.CreatedAt, row.UpdatedAt)
}

func toSubjectDomainFromGet(row sqlcgen.GetSubjectByIDRow) *domain.Subject {
	return toSubjectDomainCore(row.SubjectID, row.UserID, row.Name, row.LmsCourseID, row.CreatedAt, row.UpdatedAt)
}

func toSubjectDomainFromGetByUser(row sqlcgen.GetSubjectByIDAndUserIDRow) *domain.Subject {
	return toSubjectDomainCore(row.SubjectID, row.UserID, row.Name, row.LmsCourseID, row.CreatedAt, row.UpdatedAt)
}

func toSubjectDomainFromUpdate(row sqlcgen.UpdateSubjectNameRow) *domain.Subject {
	return toSubjectDomainCore(row.SubjectID, row.UserID, row.Name, row.LmsCourseID, row.CreatedAt, row.UpdatedAt)
}
