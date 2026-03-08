// Package backend provides the Agent RAG backend implementation.
// モデル指定はエージェント側で固定されているため、エンドポイントへの
// シンプルな HTTP POST のみを行う。
package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AgentBackend は Agent システム（RAG バックエンド）の実装。
// OpenAI chat completions 互換形式でリクエストを送信するが、
// モデル指定は行わない（エージェント側で固定）。
type AgentBackend struct {
	apiBase    string
	apiKey     string
	httpClient *http.Client
}

// NewAgentBackend は AgentBackend を構築する。
func NewAgentBackend(apiBase, apiKey string) *AgentBackend {
	return &AgentBackend{
		apiBase:    strings.TrimRight(apiBase, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 180 * time.Second},
	}
}

// CreateCollection は chat_completions モードでは仮想 ID を返す（no-op）。
func (b *AgentBackend) CreateCollection(name string) (string, error) {
	return fmt.Sprintf("virtual:%d", time.Now().UnixMilli()), nil
}

// UploadDocument は chat_completions モードでは no-op。
func (b *AgentBackend) UploadDocument(collectionID string, name string, r io.Reader) (string, error) {
	return "", nil
}

// WaitForReady は chat_completions モードでは即時 true を返す。
func (b *AgentBackend) WaitForReady(collectionID string, timeoutSecs int, pollIntervalSecs int) (bool, error) {
	return true, nil
}

// Query はエージェントエンドポイントに質問を送信して QueryResult を返す。
// リクエストは OpenAI chat completions 互換形式（model フィールドなし）。
func (b *AgentBackend) Query(collectionID string, question string) (*QueryResult, error) {
	body, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": question},
		},
		"stream": false,
	})

	start := time.Now()

	req, err := http.NewRequest("POST", b.apiBase+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("リクエスト作成失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.apiKey)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Agent API 呼び出し失敗: %w", err)
	}
	defer resp.Body.Close()

	latencyMS := int(time.Since(start).Milliseconds())

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("レスポンスのデコード失敗: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %v", resp.StatusCode, result)
	}

	choices, _ := result["choices"].([]any)
	if len(choices) == 0 {
		return nil, fmt.Errorf("choices が空です: %v", result)
	}
	msg, _ := choices[0].(map[string]any)["message"].(map[string]any)
	answer, _ := msg["content"].(string)

	// sources: レスポンスに含まれる場合は抽出（エージェント実装依存）
	var sources []Source
	if rawSources, ok := result["sources"].([]any); ok {
		for _, s := range rawSources {
			if sm, ok := s.(map[string]any); ok {
				src := Source{}
				if n, ok := sm["name"].(string); ok {
					src.Name = n
				}
				if p, ok := sm["page"].(string); ok {
					src.Page = p
				}
				sources = append(sources, src)
			}
		}
	}

	return &QueryResult{Answer: answer, LatencyMS: latencyMS, Sources: sources}, nil
}

// Cleanup は chat_completions モードでは no-op。
func (b *AgentBackend) Cleanup(collectionID string) error {
	return nil
}
