package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/adapters/postgres/sqlcgen"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
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
		ID:        session.ID,
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
	return sqlcQASessionToDomainFromGet(row)
}

func (r *qaSessionRepo) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.QASession, error) {
	row, err := r.q.GetQASessionByIDAndUserID(ctx, sqlcgen.GetQASessionByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	return sqlcQASessionToDomainFromGetByUser(row)
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
		s, err := sqlcQASessionToDomainFromList(row)
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
		ID:                  id,
		FinalAnswerMarkdown: sql.NullString{String: answer, Valid: true},
		EvidenceSnippets:    sourcesJSON,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	return sqlcQASessionToDomainFromUpdateAnswer(row)
}

func (r *qaSessionRepo) UpdateFeedback(ctx context.Context, id, userID uuid.UUID, feedback int) (*domain.QASession, error) {
	row, err := r.q.UpdateQASessionFeedback(ctx, sqlcgen.UpdateQASessionFeedbackParams{
		ID:      id,
		UserID:  userID,
		Column2: feedback,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	return sqlcQASessionToDomainFromUpdateFeedback(row)
}

// ─── 変換ヘルパー ─────────────────────────────────────────────────

func sqlcQASessionToDomainCore(
	sessionID, userID, subjectID uuid.UUID,
	question string,
	answer sql.NullString,
	sources pqtype.NullRawMessage,
	feedback int16,
	createdAt time.Time,
	answeredAt sql.NullTime,
) (*domain.QASession, error) {
	s := &domain.QASession{
		ID:        sessionID,
		UserID:    userID,
		SubjectID: subjectID,
		Question:  question,
		CreatedAt: createdAt,
	}
	if answer.Valid {
		s.Answer = &answer.String
	}
	if feedback != 0 {
		v := int(feedback)
		s.Feedback = &v
	}
	if answeredAt.Valid {
		s.AnsweredAt = &answeredAt.Time
	}
	if sources.Valid {
		var srcs []domain.Source
		if err := json.Unmarshal(sources.RawMessage, &srcs); err != nil {
			return nil, err
		}
		s.Sources = srcs
	}
	return s, nil
}

func sqlcQASessionToDomainFromGet(row sqlcgen.GetQASessionByIDRow) (*domain.QASession, error) {
	return sqlcQASessionToDomainCore(row.SessionID, row.UserID, row.SubjectID, row.Question, row.Answer, row.Sources, row.Feedback, row.CreatedAt, row.AnsweredAt)
}

func sqlcQASessionToDomainFromGetByUser(row sqlcgen.GetQASessionByIDAndUserIDRow) (*domain.QASession, error) {
	return sqlcQASessionToDomainCore(row.SessionID, row.UserID, row.SubjectID, row.Question, row.Answer, row.Sources, row.Feedback, row.CreatedAt, row.AnsweredAt)
}

func sqlcQASessionToDomainFromList(row sqlcgen.ListQASessionsBySubjectIDRow) (*domain.QASession, error) {
	return sqlcQASessionToDomainCore(row.SessionID, row.UserID, row.SubjectID, row.Question, row.Answer, row.Sources, row.Feedback, row.CreatedAt, row.AnsweredAt)
}

func sqlcQASessionToDomainFromUpdateAnswer(row sqlcgen.UpdateQASessionAnswerRow) (*domain.QASession, error) {
	return sqlcQASessionToDomainCore(row.SessionID, row.UserID, row.SubjectID, row.Question, row.Answer, row.Sources, row.Feedback, row.CreatedAt, row.AnsweredAt)
}

func sqlcQASessionToDomainFromUpdateFeedback(row sqlcgen.UpdateQASessionFeedbackRow) (*domain.QASession, error) {
	return sqlcQASessionToDomainCore(row.SessionID, row.UserID, row.SubjectID, row.Question, row.Answer, row.Sources, row.Feedback, row.CreatedAt, row.AnsweredAt)
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
