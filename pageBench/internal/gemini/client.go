// Package gemini provides a shared Gemini API client and thinking budget helpers.
package gemini

import (
	"context"
	"fmt"
	"math"
	"os"

	"google.golang.org/genai"
)

// NewClient は GEMINI_API_KEY 環境変数を使って genai.Client を生成する。
// api_key が空の場合は環境変数から自動解決する。
func NewClient(ctx context.Context) (*genai.Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY が設定されていません")
	}
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
}

// ThinkingLevel は thinking_level 文字列を genai.ThinkingLevel 定数に変換する。
//
//	high    → ThinkingLevelHigh    (3 Pro / 3 Flash, デフォルト推論深度)
//	medium  → ThinkingLevelMedium  (3.1 Pro / 3 Flash, バランス型)
//	low     → ThinkingLevelLow     (3 Pro / 3 Flash, 浅い推論)
//	minimal → ThinkingLevelMinimal (3 Flash / 3.1 Flash-Lite のみ, 最小コスト)
//
// 注意: minimal は Flash 系モデル専用。Pro 系では low が最低レベル。
func ThinkingLevel(level string) genai.ThinkingLevel {
	switch level {
	case "high":
		return genai.ThinkingLevelHigh
	case "medium":
		return genai.ThinkingLevelMedium
	case "low":
		return genai.ThinkingLevelLow
	default: // "minimal" or anything else
		return genai.ThinkingLevelMinimal
	}
}

// EmbedText は gemini-embedding-001 で SEMANTIC_SIMILARITY タスク用の埋め込みベクトルを取得する。
// unanswerable 問題や空文字の場合は呼び出し側でスキップすること。
func EmbedText(ctx context.Context, client *genai.Client, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("空のテキストは埋め込みできません")
	}
	contents := []*genai.Content{
		{
			Parts: []*genai.Part{{Text: text}},
			Role:  "user",
		},
	}
	resp, err := client.Models.EmbedContent(ctx, "gemini-embedding-001", contents, &genai.EmbedContentConfig{
		TaskType: "SEMANTIC_SIMILARITY",
	})
	if err != nil {
		return nil, fmt.Errorf("EmbedContent 失敗: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("EmbedContent のレスポンスに Embeddings が含まれていません")
	}
	return resp.Embeddings[0].Values, nil
}

// CosineSimilarity は 2 つの埋め込みベクトル間のコサイン類似度を計算する（範囲: -1.0〜1.0）。
// 長さが 0 またはゼロベクトルの場合は 0.0 を返す。
func CosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0.0
	}
	var dot, normA, normB float64
	for i := range a {
		fA := float64(a[i])
		fB := float64(b[i])
		dot += fA * fB
		normA += fA * fA
		normB += fB * fB
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	// 丸め誤差のクランプ
	if sim > 1.0 {
		sim = 1.0
	} else if sim < -1.0 {
		sim = -1.0
	}
	return sim
}
