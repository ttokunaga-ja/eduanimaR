package testhelper

import (
	"time"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
)

// FixtureUserID と FixtureSubjectID は全テストで共通利用する固定 UUID
var (
	FixtureUserID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	FixtureSubjectID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	FixtureFileID    = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	FixtureJobID     = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	FixtureChunkID   = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	FixtureSessionID = uuid.MustParse("00000000-0000-0000-0000-000000000006")
)

// NewSubject はテスト用 Subject を生成する。
func NewSubject(opts ...func(*domain.Subject)) *domain.Subject {
	s := &domain.Subject{
		ID:        FixtureSubjectID,
		UserID:    FixtureUserID,
		Name:      "テスト科目",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewQASession はテスト用 QASession を生成する。
func NewQASession(opts ...func(*domain.QASession)) *domain.QASession {
	s := &domain.QASession{
		ID:        FixtureSessionID,
		UserID:    FixtureUserID,
		SubjectID: FixtureSubjectID,
		Question:  "テスト質問",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewIngestJob はテスト用 IngestJob を生成する。
func NewIngestJob(status domain.JobStatus) *domain.IngestJob {
	return &domain.IngestJob{
		ID:         FixtureJobID,
		FileID:     FixtureFileID,
		Status:     status,
		MaxRetries: 3,
		CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// NewFile はテスト用 File を生成する。
func NewFile(status domain.FileStatus) *domain.File {
	return &domain.File{
		ID:          FixtureFileID,
		UserID:      FixtureUserID,
		SubjectID:   FixtureSubjectID,
		Name:        "test.pdf",
		MimeType:    "application/pdf",
		StoragePath: "subjects/test/test.pdf",
		SizeBytes:   1024,
		Status:      status,
		UploadedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}
