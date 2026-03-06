package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/postgres/sqlcgen"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

type qaSessionRepo struct {
	q *sqlcgen.Queries
}

// NewQASessionRepo は ports.QASessionRepository の postgres 実装を返す。
func NewQASessionRepo(db *sql.DB) ports.QASessionRepository {
	return &qaSessionRepo{q: sqlcgen.New(db)}
}

func (r *qaSessionRepo) Create(ctx context.Context, session *domain.QASession) error {
	_, err := r.q.CreateQASession(ctx, sqlcgen.CreateQASessionParams{
		SessionID: session.ID,
		UserID:    session.UserID,
		SubjectID: session.SubjectID,
		Question:  session.Question,
	})
	return err
}

func (r *qaSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.QASession, error) {
	row, err := r.q.GetQASessionByID(ctx, id)
	if err != nil {
		return nil, mapDBError(err)
	}
	return sqlcQASessionToDomain(row)
}

func (r *qaSessionRepo) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.QASession, error) {
	row, err := r.q.GetQASessionByIDAndUserID(ctx, sqlcgen.GetQASessionByIDAndUserIDParams{
		SessionID: id,
		UserID:    userID,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	return sqlcQASessionToDomain(row)
}

func (r *qaSessionRepo) ListBySubjectID(ctx context.Context, subjectID, userID uuid.UUID, limit, offset int) ([]*domain.QASession, error) {
	rows, err := r.q.ListQASessionsBySubjectID(ctx, sqlcgen.ListQASessionsBySubjectIDParams{
		SubjectID: subjectID,
		UserID:    userID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.QASession, 0, len(rows))
	for _, row := range rows {
		s, err := sqlcQASessionToDomain(row)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *qaSessionRepo) CountBySubjectID(ctx context.Context, subjectID, userID uuid.UUID) (int64, error) {
	return r.q.CountQASessionsBySubjectID(ctx, sqlcgen.CountQASessionsBySubjectIDParams{
		SubjectID: subjectID,
		UserID:    userID,
	})
}

func (r *qaSessionRepo) UpdateAnswer(ctx context.Context, id uuid.UUID, answer string, sources []domain.Source) (*domain.QASession, error) {
	sourcesJSON, err := sourcesToNullRawMessage(sources)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateQASessionAnswer(ctx, sqlcgen.UpdateQASessionAnswerParams{
		SessionID: id,
		Answer:    sql.NullString{String: answer, Valid: true},
		Sources:   sourcesJSON,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	return sqlcQASessionToDomain(row)
}

func (r *qaSessionRepo) UpdateFeedback(ctx context.Context, id, userID uuid.UUID, feedback int) (*domain.QASession, error) {
	row, err := r.q.UpdateQASessionFeedback(ctx, sqlcgen.UpdateQASessionFeedbackParams{
		SessionID: id,
		UserID:    userID,
		Feedback:  sql.NullInt16{Int16: int16(feedback), Valid: true},
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	return sqlcQASessionToDomain(row)
}

// ─── 変換ヘルパー ─────────────────────────────────────────────────

func sqlcQASessionToDomain(row sqlcgen.QaSession) (*domain.QASession, error) {
	s := &domain.QASession{
		ID:        row.SessionID,
		UserID:    row.UserID,
		SubjectID: row.SubjectID,
		Question:  row.Question,
		CreatedAt: row.CreatedAt,
	}
	if row.Answer.Valid {
		s.Answer = &row.Answer.String
	}
	if row.Feedback.Valid {
		v := int(row.Feedback.Int16)
		s.Feedback = &v
	}
	if row.AnsweredAt.Valid {
		s.AnsweredAt = &row.AnsweredAt.Time
	}
	if row.Sources.Valid {
		var srcs []domain.Source
		if err := json.Unmarshal(row.Sources.RawMessage, &srcs); err != nil {
			return nil, err
		}
		s.Sources = srcs
	}
	return s, nil
}

func sourcesToNullRawMessage(sources []domain.Source) (pqtype.NullRawMessage, error) {
	if len(sources) == 0 {
		return pqtype.NullRawMessage{Valid: false}, nil
	}
	b, err := json.Marshal(sources)
	if err != nil {
		return pqtype.NullRawMessage{}, err
	}
	return pqtype.NullRawMessage{RawMessage: b, Valid: true}, nil
}
