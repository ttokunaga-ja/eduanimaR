// Package postgres は PostgreSQL を使った ports インターフェースの実装を提供する。
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
		out = append(out, toSubjectDomain(row))
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
	return toSubjectDomain(row), nil
}

func (r *subjectRepo) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Subject, error) {
	row, err := r.q.GetSubjectByIDAndUserID(ctx, sqlcgen.GetSubjectByIDAndUserIDParams{
		SubjectID: id,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toSubjectDomain(row), nil
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
		SubjectID: id,
		UserID:    userID,
		Name:      name,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toSubjectDomain(row), nil
}

func (r *subjectRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteSubject(ctx, sqlcgen.DeleteSubjectParams{
		SubjectID: id,
		UserID:    userID,
	})
}

func toSubjectDomain(row sqlcgen.Subject) *domain.Subject {
	s := &domain.Subject{
		ID:        row.SubjectID,
		UserID:    row.UserID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.LmsCourseID.Valid {
		s.LMSCourseID = &row.LmsCourseID.String
	}
	return s
}
