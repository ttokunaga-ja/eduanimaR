// Package gemini provides a shared Gemini API client and thinking budget helpers.
package gemini

import (
	"context"
	"fmt"
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
