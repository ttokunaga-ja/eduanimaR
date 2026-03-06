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

// LLMClient は Gemini API 呼び出しを抽象化する。
// Phase 1: 高速推論モデル（OCR/Embedding） + 高精度推論モデル（最終回答）を使い分ける。
type LLMClient interface {
	// OCRAndChunk は PDF/画像ファイルのバイト列を受け取り、
	// Markdown化・意味単位チャンク分割を行う（高速推論モデル使用）
	OCRAndChunk(ctx context.Context, fileContent []byte, mimeType string) (*OCRResult, error)

	// GenerateEmbedding はテキストの埋め込みベクトル（768次元）を生成する
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateAnswer は選定済みエビデンスチャンクと質問から最終回答を生成する
	// （高精度推論モデル使用）
	GenerateAnswer(ctx context.Context, question string, evidences []string) (string, error)

	// GenerateAnswerStream は GenerateAnswer のストリーミング版
	// onChunk コールバックに回答テキストを逐次的に渡す
	GenerateAnswerStream(ctx context.Context, question string, evidences []string, onChunk func(text string) error) error
}
