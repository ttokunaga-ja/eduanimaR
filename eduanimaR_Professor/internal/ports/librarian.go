package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
)

// LibrarianSearchRequest は Librarian から Professor への検索リクエスト
type LibrarianSearchRequest struct {
	QueriesText   []string // 全文検索クエリ
	QueriesVector []string // ベクトル検索クエリ（空の場合は全文検索のみ）
	Rationale     string   // 検索理由（ログ・監査用）
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
type LibrarianEvidence struct {
	TempIndex   int    // Professor の検索結果配列インデックス
	WhyRelevant string // 選定理由
}

// LibrarianClient は Professor から Librarian への gRPC 通信を抽象化する
type LibrarianClient interface {
	// Think は双方向ストリーミングで Librarian に推論を依頼する。
	// onSearchRequest: Librarian が検索を要求するたびに呼ばれるコールバック
	//   → Professor は subject_id/user_id による物理制約を強制してから検索を実行する
	Think(
		ctx context.Context,
		requestID string,
		userQuery string,
		subjectID uuid.UUID,
		userID uuid.UUID,
		onSearchRequest func(req LibrarianSearchRequest) (*LibrarianSearchResponse, error),
	) (*LibrarianThinkResult, error)
}
