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
}
