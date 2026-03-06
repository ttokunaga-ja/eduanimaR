package domain

import (
	"time"

	"github.com/google/uuid"
)

// Subject は科目エンティティ
type Subject struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	LMSCourseID *string // nullable（将来の LMS 連携用 / Phase 1 未使用）
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
