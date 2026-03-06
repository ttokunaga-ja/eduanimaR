package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/testhelper"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/usecases"
)

// ─── テストヘルパー ────────────────────────────────────────────────

// ptrStr は string ポインタを返す。
func ptrStr(s string) *string { return &s }

// ptrInt は int ポインタを返す。
func ptrInt(i int) *int { return &i }

// collectEvents は onEvent コールバックで発生した全イベントを収集する。
func collectEvents() (func(domain.SSEEventType, any) error, *[]domain.SSEEventType) {
	events := make([]domain.SSEEventType, 0)
	fn := func(et domain.SSEEventType, _ any) error {
		events = append(events, et)
		return nil
	}
	return fn, &events
}

// newChatUseCase はテスト用依存を注入した ChatUseCase を返す。
func newChatUseCase(
	subjectRepo *testhelper.MockSubjectRepository,
	qaRepo *testhelper.MockQASessionRepository,
	chunkRepo *testhelper.MockChunkRepository,
	llm *testhelper.MockLLMClient,
	librarian *testhelper.MockLibrarianClient,
) *usecases.ChatUseCase {
	return usecases.NewChatUseCase(subjectRepo, qaRepo, chunkRepo, llm, librarian)
}

// ─── Ask 正常系 ──────────────────────────────────────────────────

func TestChatUseCase_Ask_Success(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID
	question := "テスト質問"

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subject := testhelper.NewSubject()

	// subject 所有権確認
	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).Return(subject, nil)

	// QASession 作成（session.ID は uuid.New() で動的生成されるため mock.Anything）
	qaRepo.On("Create", ctx, mock.AnythingOfType("*domain.QASession")).Return(nil)

	// Librarian Think: エビデンスなし（フォールバック経路）
	thinkResult := &ports.LibrarianThinkResult{
		Evidences:     []ports.LibrarianEvidence{},
		CoverageNotes: "テスト推論",
	}
	librarianClient.On("Think",
		ctx,
		mock.AnythingOfType("string"), // session.ID.String()
		question,
		subjectID,
		userID,
		mock.Anything, // onSearchRequest func
	).Return(thinkResult, nil)

	// LLM ストリーミング: "テスト回答" を1チャンクで返す
	llmClient.On("GenerateAnswerStream",
		ctx,
		question,
		mock.Anything, // []string（空スライス）
		mock.Anything, // func(string) error
	).Return(nil).Run(func(args mock.Arguments) {
		onChunk := args.Get(3).(func(string) error)
		_ = onChunk("テスト回答")
	})

	// UpdateAnswer（session.ID は動的生成のため mock.Anything）
	updatedSession := testhelper.NewQASession(func(s *domain.QASession) {
		s.Answer = ptrStr("テスト回答")
	})
	qaRepo.On("UpdateAnswer",
		ctx,
		mock.Anything, // session.ID
		"テスト回答",
		mock.Anything, // []domain.Source
	).Return(updatedSession, nil)

	onEvent, events := collectEvents()
	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	session, err := uc.Ask(ctx, subjectID, userID, question, onEvent)

	require.NoError(t, err)
	require.NotNil(t, session)

	// 期待イベント: thinking → answer → done
	assert.Contains(t, *events, domain.SSEEventThinking)
	assert.Contains(t, *events, domain.SSEEventAnswer)
	assert.Contains(t, *events, domain.SSEEventDone)

	subjectRepo.AssertExpectations(t)
	qaRepo.AssertExpectations(t)
	llmClient.AssertExpectations(t)
	librarianClient.AssertExpectations(t)
}

// ─── Ask: subject が見つからない ──────────────────────────────────

func TestChatUseCase_Ask_SubjectNotFound(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).
		Return((*domain.Subject)(nil), domain.ErrNotFound)

	onEvent, events := collectEvents()
	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	session, err := uc.Ask(ctx, subjectID, userID, "質問", onEvent)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	assert.Nil(t, session)
	// イベントは一切発生しないこと
	assert.Empty(t, *events)

	subjectRepo.AssertExpectations(t)
	qaRepo.AssertNotCalled(t, "Create")
	librarianClient.AssertNotCalled(t, "Think")
}

// ─── Ask: QASession 作成失敗 ─────────────────────────────────────

func TestChatUseCase_Ask_CreateSessionFails(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subject := testhelper.NewSubject()
	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).Return(subject, nil)

	dbErr := errors.New("db connection error")
	qaRepo.On("Create", ctx, mock.AnythingOfType("*domain.QASession")).Return(dbErr)

	onEvent, _ := collectEvents()
	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	session, err := uc.Ask(ctx, subjectID, userID, "質問", onEvent)

	assert.Error(t, err)
	assert.Nil(t, session)
	librarianClient.AssertNotCalled(t, "Think")
}

// ─── Ask: Librarian エラー時 SSEEventError を送信 ─────────────────

func TestChatUseCase_Ask_LibrarianError_SendsSSEError(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subject := testhelper.NewSubject()
	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).Return(subject, nil)
	qaRepo.On("Create", ctx, mock.AnythingOfType("*domain.QASession")).Return(nil)

	librarianErr := errors.New("librarian unavailable")
	librarianClient.On("Think",
		ctx, mock.Anything, "質問", subjectID, userID, mock.Anything,
	).Return((*ports.LibrarianThinkResult)(nil), librarianErr)

	var gotErrorEvent bool
	onEvent := func(et domain.SSEEventType, _ any) error {
		if et == domain.SSEEventError {
			gotErrorEvent = true
		}
		return nil
	}

	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	session, err := uc.Ask(ctx, subjectID, userID, "質問", onEvent)

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.True(t, gotErrorEvent, "SSEEventError が送信されるべき")

	librarianClient.AssertExpectations(t)
	llmClient.AssertNotCalled(t, "GenerateAnswerStream")
}

// ─── Ask: LLM ストリームエラー時 SSEEventError を送信 ──────────────

func TestChatUseCase_Ask_LLMStreamError_SendsSSEError(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID
	question := "ストリームエラーテスト"

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subject := testhelper.NewSubject()
	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).Return(subject, nil)
	qaRepo.On("Create", ctx, mock.AnythingOfType("*domain.QASession")).Return(nil)

	thinkResult := &ports.LibrarianThinkResult{
		Evidences:     []ports.LibrarianEvidence{},
		CoverageNotes: "推論",
	}
	librarianClient.On("Think",
		ctx, mock.Anything, question, subjectID, userID, mock.Anything,
	).Return(thinkResult, nil)

	streamErr := errors.New("LLM stream broken")
	llmClient.On("GenerateAnswerStream",
		ctx, question, mock.Anything, mock.Anything,
	).Return(streamErr)

	var gotErrorEvent bool
	onEvent := func(et domain.SSEEventType, _ any) error {
		if et == domain.SSEEventError {
			gotErrorEvent = true
		}
		return nil
	}

	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	session, err := uc.Ask(ctx, subjectID, userID, question, onEvent)

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.True(t, gotErrorEvent, "SSEEventError が送信されるべき")

	llmClient.AssertExpectations(t)
	qaRepo.AssertNotCalled(t, "UpdateAnswer")
}

// ─── ListSessions ─────────────────────────────────────────────────

func TestChatUseCase_ListSessions_Success(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subject := testhelper.NewSubject()
	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).Return(subject, nil)

	sessions := []*domain.QASession{testhelper.NewQASession()}
	qaRepo.On("ListBySubjectID", ctx, subjectID, userID, 20, 0).Return(sessions, nil)

	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	result, err := uc.ListSessions(ctx, subjectID, userID, 20, 0)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	subjectRepo.AssertExpectations(t)
	qaRepo.AssertExpectations(t)
}

func TestChatUseCase_ListSessions_SubjectNotFound(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).
		Return((*domain.Subject)(nil), domain.ErrForbidden)

	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	result, err := uc.ListSessions(ctx, subjectID, userID, 20, 0)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden))
	assert.Nil(t, result)
	qaRepo.AssertNotCalled(t, "ListBySubjectID")
}

// ─── UpdateFeedback ───────────────────────────────────────────────

func TestChatUseCase_UpdateFeedback_Success(t *testing.T) {
	ctx := context.Background()
	sessionID := testhelper.FixtureSessionID
	userID := testhelper.FixtureUserID

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	updated := testhelper.NewQASession(func(s *domain.QASession) {
		s.Feedback = ptrInt(1)
	})
	qaRepo.On("UpdateFeedback", ctx, sessionID, userID, 1).Return(updated, nil)

	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	result, err := uc.UpdateFeedback(ctx, sessionID, userID, 1)

	require.NoError(t, err)
	require.NotNil(t, result.Feedback)
	assert.Equal(t, 1, *result.Feedback)
	qaRepo.AssertExpectations(t)
}

func TestChatUseCase_UpdateFeedback_NotFound(t *testing.T) {
	ctx := context.Background()
	sessionID := testhelper.FixtureSessionID
	userID := testhelper.FixtureUserID

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	qaRepo.On("UpdateFeedback", ctx, sessionID, userID, -1).
		Return((*domain.QASession)(nil), domain.ErrNotFound)

	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	result, err := uc.UpdateFeedback(ctx, sessionID, userID, -1)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	assert.Nil(t, result)
}
