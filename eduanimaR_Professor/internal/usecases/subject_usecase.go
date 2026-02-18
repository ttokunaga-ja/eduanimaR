// Package usecases はビジネスロジック（ユースケース層）を提供する。
// domain と ports にのみ依存し、HTTP/DB/外部サービスの詳細を知らない。
package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

// SubjectUseCase は科目に関するビジネスロジックを提供する。
type SubjectUseCase struct {
	subjects ports.SubjectRepository
}

// NewSubjectUseCase は SubjectUseCase を生成する。
func NewSubjectUseCase(subjects ports.SubjectRepository) *SubjectUseCase {
	return &SubjectUseCase{subjects: subjects}
}

// ListByUser は指定ユーザーの科目一覧を返す。
func (uc *SubjectUseCase) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Subject, error) {
	return uc.subjects.ListByUserID(ctx, userID)
}

// GetByIDAndUser は指定 ID かつ指定ユーザーの科目を返す。
func (uc *SubjectUseCase) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*domain.Subject, error) {
	return uc.subjects.GetByIDAndUserID(ctx, id, userID)
}

// CreateSubjectInput は科目作成の入力値
type CreateSubjectInput struct {
	Name        string
	LMSCourseID *string
}

// Create は新規科目を作成する。
func (uc *SubjectUseCase) Create(ctx context.Context, userID uuid.UUID, in CreateSubjectInput) (*domain.Subject, error) {
	if in.Name == "" {
		return nil, domain.ErrInvalidInput
	}
	now := time.Now().UTC()
	s := &domain.Subject{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        in.Name,
		LMSCourseID: in.LMSCourseID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.subjects.Create(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// UpdateName は科目名を更新する。
func (uc *SubjectUseCase) UpdateName(ctx context.Context, id, userID uuid.UUID, name string) (*domain.Subject, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	return uc.subjects.UpdateName(ctx, id, userID, name)
}

// Delete は科目を削除する。
func (uc *SubjectUseCase) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return uc.subjects.Delete(ctx, id, userID)
}
