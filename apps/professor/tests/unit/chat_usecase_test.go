package unit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/testhelper"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/usecases"
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
// fileRepo と storage は既存テストでは使用しないため、内部で空モックを生成する。
func newChatUseCase(
	subjectRepo *testhelper.MockSubjectRepository,
	qaRepo *testhelper.MockQASessionRepository,
	chunkRepo *testhelper.MockChunkRepository,
	llm *testhelper.MockLLMClient,
	librarian *testhelper.MockLibrarianClient,
) *usecases.ChatUseCase {
	return usecases.NewChatUseCase(
		subjectRepo, qaRepo, chunkRepo,
		&testhelper.MockFileRepository{},
		&testhelper.MockObjectStorage{},
		llm, librarian,
	)
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

	// 質問分析（Step1）: clear として返す
	llmClient.On("GenerateQuestionAnalysis", ctx, question).
		Return(&ports.QuestionAnalysis{
			InterpretedQuery:   question,
			CompletionCriteria: []string{},
			Clarity:            ports.QuestionClarityClear,
		}, nil)

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
		mock.Anything, // maxLoops (int32)
		mock.Anything, // thinkingLevel (string)
		mock.Anything, // interpretedQuery (string)
		mock.Anything, // completionCriteria ([]string)
		mock.Anything, // onSearchRequest func
	).Return(thinkResult, nil)

	// LLM ストリーミング: "テスト回答" を1チャンクで返す
	llmClient.On("GenerateAnswerStreamWithPDF",
		ctx,
		question,
		mock.Anything, // []string（空スライス）
		mock.Anything, // []byte
		mock.Anything, // mime type
		mock.Anything, // model override
		mock.Anything, // thinking level
		mock.Anything, // func(string) error
	).Return(nil).Run(func(args mock.Arguments) {
		onChunk := args.Get(7).(func(string) error)
		_ = onChunk("テスト回答")
	})

	// GenerateAnswerMeta: 回答メタデータ取得
	llmClient.On("GenerateAnswerMeta", ctx, question, "テスト回答", 0).
		Return(&ports.AnswerMeta{Answerability: "answerable"}, nil)

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

	// 質問分析（Step1）: clear として返す
	llmClient.On("GenerateQuestionAnalysis", ctx, "質問").
		Return(&ports.QuestionAnalysis{
			InterpretedQuery:   "質問",
			CompletionCriteria: []string{},
			Clarity:            ports.QuestionClarityClear,
		}, nil)

	librarianErr := errors.New("librarian unavailable")
	librarianClient.On("Think",
		ctx,
		mock.AnythingOfType("string"), // session.ID.String()
		"質問",
		subjectID,
		userID,
		mock.Anything, // maxLoops (int32)
		mock.Anything, // thinkingLevel (string)
		mock.Anything, // interpretedQuery (string)
		mock.Anything, // completionCriteria ([]string)
		mock.Anything, // onSearchRequest func
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
	llmClient.AssertNotCalled(t, "GenerateAnswerStreamWithPDF")
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

	// 質問分析（Step1）: clear として返す
	llmClient.On("GenerateQuestionAnalysis", ctx, question).
		Return(&ports.QuestionAnalysis{
			InterpretedQuery:   question,
			CompletionCriteria: []string{},
			Clarity:            ports.QuestionClarityClear,
		}, nil)

	thinkResult := &ports.LibrarianThinkResult{
		Evidences:     []ports.LibrarianEvidence{},
		CoverageNotes: "推論",
	}
	librarianClient.On("Think",
		ctx,
		mock.AnythingOfType("string"), // session.ID.String()
		question,
		subjectID,
		userID,
		mock.Anything, // maxLoops (int32)
		mock.Anything, // thinkingLevel (string)
		mock.Anything, // interpretedQuery (string)
		mock.Anything, // completionCriteria ([]string)
		mock.Anything, // onSearchRequest func
	).Return(thinkResult, nil)

	streamErr := errors.New("LLM stream broken")
	llmClient.On("GenerateAnswerStreamWithPDF",
		ctx, question, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
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

// ─── Ask: 曖昧な質問 → SSEEventClarification を送信 ───────────────

// TestChatUseCase_Ask_AmbiguousQuestion_SendsClarification は、
// GenerateQuestionAnalysis が "ambiguous" を返した場合に
// SSEEventClarification + SSEEventDone が送信されることを検証する。
// Think は呼ばれないことも確認する。
func TestChatUseCase_Ask_AmbiguousQuestion_SendsClarification(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID
	question := "機械学習について教えて"

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subject := testhelper.NewSubject()
	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).Return(subject, nil)
	qaRepo.On("Create", ctx, mock.AnythingOfType("*domain.QASession")).Return(nil)

	// Step1: 曖昧と判定
	llmClient.On("GenerateQuestionAnalysis", ctx, question).
		Return(&ports.QuestionAnalysis{
			InterpretedQuery:   question,
			CompletionCriteria: []string{},
			Clarity:            ports.QuestionClarityAmbiguous,
		}, nil)

	// 選択肢生成
	clarificationOpts := []string{
		"機械学習の基本的なアルゴリズムについて知りたい",
		"機械学習を使った具体的な応用例が知りたい",
		"機械学習の学習方法・おすすめリソースが知りたい",
	}
	llmClient.On("GenerateClarificationOptions", ctx, question).
		Return(&ports.ClarificationOptions{Options: clarificationOpts}, nil)

	onEvent, events := collectEvents()
	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	session, err := uc.Ask(ctx, subjectID, userID, question, onEvent)

	// 正常終了（エラーなし）
	require.NoError(t, err)
	require.NotNil(t, session)

	// clarification + done イベントが送信されること
	assert.Contains(t, *events, domain.SSEEventClarification)
	assert.Contains(t, *events, domain.SSEEventDone)

	// thinking + answer イベントは発生しないこと
	assert.NotContains(t, *events, domain.SSEEventThinking)
	assert.NotContains(t, *events, domain.SSEEventAnswer)

	// Think は呼ばれないこと
	librarianClient.AssertNotCalled(t, "Think")

	subjectRepo.AssertExpectations(t)
	qaRepo.AssertExpectations(t)
	llmClient.AssertExpectations(t)
}

// ─── Ask: 質問分析エラー時にフォールバックして Think を呼ぶ ─────────

// TestChatUseCase_Ask_QuestionAnalysisError_FallsBackToClear は、
// GenerateQuestionAnalysis がエラーを返した場合に early return せず
// フォールバック（clear・元の質問）で Think が呼ばれることを検証する。
func TestChatUseCase_Ask_QuestionAnalysisError_FallsBackToClear(t *testing.T) {
	ctx := context.Background()
	subjectID := testhelper.FixtureSubjectID
	userID := testhelper.FixtureUserID
	question := "分析エラー時のフォールバックテスト"

	subjectRepo := &testhelper.MockSubjectRepository{}
	qaRepo := &testhelper.MockQASessionRepository{}
	chunkRepo := &testhelper.MockChunkRepository{}
	llmClient := &testhelper.MockLLMClient{}
	librarianClient := &testhelper.MockLibrarianClient{}

	subject := testhelper.NewSubject()
	subjectRepo.On("GetByIDAndUserID", ctx, subjectID, userID).Return(subject, nil)
	qaRepo.On("Create", ctx, mock.AnythingOfType("*domain.QASession")).Return(nil)

	// GenerateQuestionAnalysis はエラーを返す
	analysisErr := errors.New("LLM analysis timeout")
	llmClient.On("GenerateQuestionAnalysis", ctx, question).
		Return((*ports.QuestionAnalysis)(nil), analysisErr)

	// フォールバック後も Think が呼ばれることを確認（Think はエラーを返す）
	librarianErr := errors.New("librarian error after fallback")
	librarianClient.On("Think",
		ctx,
		mock.AnythingOfType("string"), // session.ID.String()
		question,
		subjectID,
		userID,
		mock.Anything, // maxLoops (int32)
		mock.Anything, // thinkingLevel (string)
		mock.Anything, // interpretedQuery (string) ← fallback: original question
		mock.Anything, // completionCriteria ([]string) ← fallback: []
		mock.Anything, // onSearchRequest func
	).Return((*ports.LibrarianThinkResult)(nil), librarianErr)

	var gotErrorEvent bool
	onEvent := func(et domain.SSEEventType, _ any) error {
		if et == domain.SSEEventError {
			gotErrorEvent = true
		}
		return nil
	}

	uc := newChatUseCase(subjectRepo, qaRepo, chunkRepo, llmClient, librarianClient)
	session, err := uc.Ask(ctx, subjectID, userID, question, onEvent)

	// エラーで終了（Librarian エラーが伝播）
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.True(t, gotErrorEvent, "SSEEventError が送信されるべき")

	// GenerateQuestionAnalysis がエラーを返しても Think が呼ばれること（early return しないこと）
	librarianClient.AssertExpectations(t)
	llmClient.AssertExpectations(t)
}
