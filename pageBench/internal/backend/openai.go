// Package backend provides the Agent RAG backend implementation.
// OpenAI互換の model フィールドで品質レベルを指定し、
// エンドポイントへシンプルな HTTP POST を行う。
package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AgentBackend は Agent システム（RAG バックエンド）の実装。
// OpenAI chat completions 互換形式でリクエストを送信する。
// model フィールドには eduanima-flash/eduanima/eduanima-pro などを指定できる。
type AgentBackend struct {
	apiBase    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAgentBackend は AgentBackend を構築する。
func NewAgentBackend(apiBase, apiKey string, model ...string) *AgentBackend {
	backendModel := "eduanima"
	if len(model) > 0 && strings.TrimSpace(model[0]) != "" {
		backendModel = strings.TrimSpace(model[0])
	}
	return &AgentBackend{
		apiBase:    strings.TrimRight(apiBase, "/"),
		apiKey:     apiKey,
		model:      backendModel,
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
// リクエストは OpenAI chat completions 互換形式（model フィールドあり）。
func (b *AgentBackend) Query(collectionID string, question string) (*QueryResult, error) {
	body, _ := json.Marshal(map[string]any{
		"model": b.model,
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

	// sources: 互換のため legacy("sources") と eduanima 拡張("eduanima_sources") の両方を受ける。
	var sources []Source
	rawSources, ok := result["sources"].([]any)
	if !ok {
		rawSources, _ = result["eduanima_sources"].([]any)
	}
	if len(rawSources) > 0 {
		for _, s := range rawSources {
			if sm, ok := s.(map[string]any); ok {
				src := Source{}
				if n, ok := sm["name"].(string); ok {
					src.Name = n
				} else if n, ok := sm["file_name"].(string); ok {
					src.Name = n
				}
				if p, ok := sm["page"].(string); ok {
					src.Page = p
				} else if p, ok := sm["page_number"].(string); ok {
					src.Page = p
				} else if p, ok := sm["page_number"].(float64); ok {
					src.Page = strconv.Itoa(int(p))
				} else if p, ok := sm["page_number"].(int); ok {
					src.Page = strconv.Itoa(p)
				}
				sources = append(sources, src)
			}
		}
	}

	qr := &QueryResult{Answer: answer, LatencyMS: latencyMS, Sources: sources}
	if meta, ok := result["eduanima_meta"].(map[string]any); ok {
		if v, ok := meta["loop_count"].(float64); ok {
			qr.LoopCount = int(v)
		}
		if v, ok := meta["loop_count"].(int); ok {
			qr.LoopCount = v
		}
		if v, ok := meta["librarian_ms"].(float64); ok {
			qr.LibrarianMS = int(v)
		}
		if v, ok := meta["librarian_ms"].(int); ok {
			qr.LibrarianMS = v
		}
		if v, ok := meta["answer_gen_ms"].(float64); ok {
			qr.AnswerGenMS = int(v)
		}
		if v, ok := meta["answer_gen_ms"].(int); ok {
			qr.AnswerGenMS = v
		}
		// answerability: professor GenerateAnswerMeta から取得（"answerable" | "unanswerable"）
		if v, ok := meta["answerability"].(string); ok {
			qr.Answerability = v
		}
	}

	return qr, nil
}

// Cleanup は chat_completions モードでは no-op。
func (b *AgentBackend) Cleanup(collectionID string) error {
	return nil
}
