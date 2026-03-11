package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
)

// LibrarianSearchRequest は Librarian から Professor への検索リクエスト
type LibrarianSearchRequest struct {
	QueriesText     []string // 全文検索クエリ
	QueriesVector   []string // ベクトル検索クエリ（空の場合は全文検索のみ）
	Rationale       string   // 検索理由（日本語自然文・UX表示用）
	ExcludeChunkIDs []string // 既読チャンクID（B-2: IDフィルタリング）
}

// LibrarianSearchResponse は Professor から Librarian への検索結果
type LibrarianSearchResponse struct {
	Results []domain.SearchResult
}

// LibrarianThinkResult は Librarian の推論完了結果
type LibrarianThinkResult struct {
	Evidences     []LibrarianEvidence
	CoverageNotes string // 充足している点・不確実な点の説明
	IsPartial     bool   // max_retries 未達でも回答に進んだ場合 true
	ErrorType     string // エラー発生時のエラー種別（空文字の場合は正常）
}

// LibrarianEvidence は Librarian が選定したエビデンスチャンクの参照情報
// chunk_id ベースに変更（temp_index廃止）
type LibrarianEvidence struct {
	ChunkID     string // チャンクID（UUID文字列）
	WhyRelevant string // 選定理由
}

// LibrarianClient は Professor から Librarian への gRPC 通信を抽象化する
type LibrarianClient interface {
	// Think は双方向ストリーミングで Librarian に推論を依頼する。
	// onSearchRequest: Librarian が検索を要求するたびに呼ばれるコールバック
	//   → Professor は subject_id/user_id による物理制約を強制してから検索を実行する
	// thinkingLevel: "flash-lite" | "flash" | "" (空文字はflashとして扱う)
	//   → Librarian が使用するモデルを決定する（C要件）
	Think(
		ctx context.Context,
		requestID string,
		userQuery string,
		subjectID uuid.UUID,
		userID uuid.UUID,
		maxLoops int32,
		thinkingLevel string,
		interpretedQuery string, // LLM が解釈した質問（Pre-search Step1 結果）
		completionCriteria []string, // 終了基準リスト（judge_sufficiency に渡す）
		onSearchRequest func(req LibrarianSearchRequest) (*LibrarianSearchResponse, error),
	) (*LibrarianThinkResult, error)
}
