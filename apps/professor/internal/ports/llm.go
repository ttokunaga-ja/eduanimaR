package ports

import "context"

// ChunkData は OCR/構造化 後の 1 チャンクのデータ
type ChunkData struct {
	Index      int    // ファイル内連番（0 始まり）
	Content    string // 抽出テキスト
	PageNumber *int   // PDF ページ番号（nil の場合は不明）
}

// OCRResult は PDF/画像ファイルの OCR・構造化結果
type OCRResult struct {
	Chunks []ChunkData
}

// AnswerMeta は最終回答の構造化メタデータ（answerability + document_summary）。
// GenerateAnswerStreamWithPDF の完了後に GenerateAnswerMeta で取得する。
type AnswerMeta struct {
	// Answerability は質問への回答可否を表す ENUM 値。
	// "answerable" | "unanswerable" | "partial"
	Answerability string `json:"answerability"`
	// DocumentSummary はコースマテリアルの簡潔な概要（1-2文）。
	// unanswerable の場合でも「何の資料か」の説明として使用する。
	DocumentSummary string `json:"document_summary"`
}

// QuestionClarity は質問の明確さを表す ENUM。
// Professor の Step1（質問分析）で判定し、Librarian を呼ぶか選択肢提示かを決定する。
type QuestionClarity string

const (
	// QuestionClarityClear は質問が十分に明確で、Librarian 検索に進めることを示す。
	QuestionClarityClear QuestionClarity = "clear"
	// QuestionClarityAmbiguous は質問が曖昧で、ユーザーへの選択肢提示が必要なことを示す。
	QuestionClarityAmbiguous QuestionClarity = "ambiguous"
)

// QuestionAnalysis は Professor が Librarian 呼び出し前に生成する質問分析結果。
// LLM の Structured Output（1回の呼び出し）で取得する。
type QuestionAnalysis struct {
	// InterpretedQuery は LLM が解釈した質問文（元の質問より精確）。
	// Librarian の初回 build_search_queries に使用する。
	InterpretedQuery string `json:"interpreted_query"`
	// CompletionCriteria は「何が揃えば回答できるか」の終了基準リスト。
	// Librarian の judge_sufficiency に渡し、Early Exit の判断基準にする。
	CompletionCriteria []string `json:"completion_criteria"`
	// Clarity は質問の明確さ ENUM。
	// "clear" → Librarian 検索へ進む。
	// "ambiguous" → GenerateClarificationOptions で選択肢提示へ分岐。
	Clarity QuestionClarity `json:"clarity"`
}

// ClarificationOptions は曖昧な質問に対してユーザーに提示する具体的な質問候補。
// GenerateClarificationOptions が返す Structured Output スキーマ。
type ClarificationOptions struct {
	// Options は 3〜5 個の具体的な質問候補テキスト。
	Options []string `json:"options"`
}

// LLMClient は Gemini API 呼び出しを抽象化する。
// Phase 1: 高速推論モデル（OCR/Embedding） + 高精度推論モデル（最終回答）を使い分ける。
type LLMClient interface {
	// OCRAndChunk は PDF/画像ファイルのバイト列を受け取り、
	// Markdown化・意味単位チャンク分割を行う（高速推論モデル使用）
	OCRAndChunk(ctx context.Context, fileContent []byte, mimeType string) (*OCRResult, error)

	// GenerateDocumentEmbedding はインジェスト用の埋め込みベクトル（1536次元）を生成する。
	// TaskType=RETRIEVAL_DOCUMENT を指定して品質を最適化する。
	GenerateDocumentEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateQueryEmbedding は検索クエリ用の埋め込みベクトル（1536次元）を生成する。
	// TaskType=RETRIEVAL_QUERY を指定して品質を最適化する。
	GenerateQueryEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateAnswer は選定済みエビデンスチャンクと質問から最終回答を生成する
	// （高精度推論モデル使用）
	GenerateAnswer(ctx context.Context, question string, evidences []string) (string, error)

	// GenerateAnswerStream は GenerateAnswer のストリーミング版
	// onChunk コールバックに回答テキストを逐次的に渡す
	GenerateAnswerStream(ctx context.Context, question string, evidences []string, onChunk func(text string) error) error

	// GenerateAnswerStreamWithPDF は PDF 原本バイト列とエビデンスチャンクを組み合わせて
	// 回答をストリーミング生成する。Gemini に PDF を直接渡すことで原本を参照した
	// 高精度な回答を実現する（テキスト抽出のみでは失われる表・図・数式も参照可能）。
	// pdfContent が nil または空の場合は GenerateAnswerStream と同じ動作をする。
	GenerateAnswerStreamWithPDF(ctx context.Context, question string, evidences []string, pdfContent []byte, mimeType string, modelOverride string, thinkingLevel string, onChunk func(text string) error) error

	// GenerateAnswerMeta は回答後の構造化メタデータを生成する。
	// GenerateAnswerStreamWithPDF 完了後に呼び出すことで、
	// answerability（回答可否）と document_summary（文書概要）を取得する。
	// question: ユーザーの質問
	// answer: ストリーミング完了後の完全な回答テキスト
	// evidenceCount: 使用したエビデンスチャンク数（0=回答不能の可能性が高い）
	GenerateAnswerMeta(ctx context.Context, question, answer string, evidenceCount int) (*AnswerMeta, error)

	// GenerateQuestionAnalysis は質問を分析し、解釈済み質問・終了基準・明確さを1回の
	// LLM 呼び出しで返す（Professor の Step1）。
	// 結果の Clarity により Librarian を呼ぶか選択肢提示かを決定する。
	GenerateQuestionAnalysis(ctx context.Context, question string) (*QuestionAnalysis, error)

	// GenerateClarificationOptions は曖昧な質問に対してユーザーに提示する
	// 3〜5 個の具体的な質問候補を生成する（最終回答モデル使用）。
	// Clarity == "ambiguous" の場合のみ呼び出す。
	GenerateClarificationOptions(ctx context.Context, question string) (*ClarificationOptions, error)
}
