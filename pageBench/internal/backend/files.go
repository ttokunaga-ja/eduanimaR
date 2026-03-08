// Package backend provides the FileUploadBackend implementation.
// OpenAI Files API 互換の POST /files + GET /files/{id} を用いて
// ドキュメントのアップロードとインデックス完了待機を行う。
package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"
)

// FileUploadBackend は OpenAI Files API 互換の RAG システム向けバックエンド。
//
// アップロード: POST {apiBase}/files  (multipart/form-data)
// 状態確認:     GET  {apiBase}/files/{id}
// クエリ:       AgentBackend に委譲 (POST {apiBase}/chat/completions)
type FileUploadBackend struct {
	apiBase          string
	apiKey           string
	purpose          string // POST /files の purpose フィールド
	fileStatusReady  string // インデックス完了とみなす status 値 (例: "processed")
	timeoutSecs      int    // WaitForReady タイムアウト（秒）。0=無制限
	pollIntervalSecs int    // WaitForReady ポーリング間隔（秒）

	httpClient   *http.Client
	queryBackend *AgentBackend // Query 委譲先

	mu      sync.Mutex
	fileIDs map[string][]string // collectionID → []fileID
}

// NewFileUploadBackend は FileUploadBackend を構築する。
// purpose が空の場合は POST /files に purpose フィールドを送らない。
// fileStatusReady が空の場合は "processed" をデフォルト値として使用する。
func NewFileUploadBackend(apiBase, apiKey, purpose, fileStatusReady string, timeoutSecs, pollIntervalSecs int) *FileUploadBackend {
	if fileStatusReady == "" {
		fileStatusReady = "processed"
	}
	if pollIntervalSecs <= 0 {
		pollIntervalSecs = 5
	}
	return &FileUploadBackend{
		apiBase:          strings.TrimRight(apiBase, "/"),
		apiKey:           apiKey,
		purpose:          purpose,
		fileStatusReady:  fileStatusReady,
		timeoutSecs:      timeoutSecs,
		pollIntervalSecs: pollIntervalSecs,
		httpClient:       &http.Client{Timeout: 180 * time.Second},
		queryBackend:     NewAgentBackend(apiBase, apiKey),
		fileIDs:          make(map[string][]string),
	}
}

// CreateCollection は仮想コレクション ID を返す（Files API に collection 概念はない）。
func (b *FileUploadBackend) CreateCollection(name string) (string, error) {
	collID := fmt.Sprintf("files:%s:%d", name, time.Now().UnixMilli())
	b.mu.Lock()
	b.fileIDs[collID] = []string{}
	b.mu.Unlock()
	return collID, nil
}

// UploadDocument は POST {apiBase}/files にファイルをアップロードして file_id を返す。
// multipart/form-data 形式: field "file"=<binary>、field "purpose"=<purpose>（purpose 未設定時は省略）
func (b *FileUploadBackend) UploadDocument(collectionID string, name string, r io.Reader) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile("file", name)
	if err != nil {
		return "", fmt.Errorf("multipart フォーム作成失敗: %w", err)
	}
	if _, err := io.Copy(fw, r); err != nil {
		return "", fmt.Errorf("ファイルコピー失敗: %w", err)
	}
	if b.purpose != "" {
		if err := w.WriteField("purpose", b.purpose); err != nil {
			return "", fmt.Errorf("purpose フィールド設定失敗: %w", err)
		}
	}
	w.Close()

	req, err := http.NewRequest("POST", b.apiBase+"/files", &buf)
	if err != nil {
		return "", fmt.Errorf("リクエスト作成失敗: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if b.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.apiKey)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST /files 失敗: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("レスポンスデコード失敗: %w", err)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %v", resp.StatusCode, result)
	}

	fileID, _ := result["id"].(string)
	if fileID == "" {
		return "", fmt.Errorf("file_id が取得できません: %v", result)
	}

	b.mu.Lock()
	b.fileIDs[collectionID] = append(b.fileIDs[collectionID], fileID)
	b.mu.Unlock()

	return fileID, nil
}

// WaitForReady は collectionID に紐付けられたすべての file_id について
// GET {apiBase}/files/{id} をポーリングし、全ファイルが fileStatusReady になるまで待機する。
// timeoutSecs / pollIntervalSecs が 0 の場合はコンストラクタの値を使用する。
func (b *FileUploadBackend) WaitForReady(collectionID string, timeoutSecs int, pollIntervalSecs int) (bool, error) {
	if timeoutSecs <= 0 {
		timeoutSecs = b.timeoutSecs
	}
	if pollIntervalSecs <= 0 {
		pollIntervalSecs = b.pollIntervalSecs
	}

	b.mu.Lock()
	ids := make([]string, len(b.fileIDs[collectionID]))
	copy(ids, b.fileIDs[collectionID])
	b.mu.Unlock()

	// アップロードされたファイルがない場合は即時 true
	if len(ids) == 0 {
		return true, nil
	}

	// fileStatusReady が未設定の RAG（ポーリング不要）は即時 true
	if b.fileStatusReady == "-" || b.fileStatusReady == "" {
		return true, nil
	}

	var deadline time.Time
	if timeoutSecs > 0 {
		deadline = time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	}

	// file_id → 完了済みフラグ
	pending := make(map[string]bool, len(ids))
	for _, id := range ids {
		pending[id] = true
	}

	for len(pending) > 0 {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return false, fmt.Errorf("タイムアウト (%ds): %d 件のファイルが未完了", timeoutSecs, len(pending))
		}

		for fileID := range pending {
			status, err := b.getFileStatus(fileID)
			if err != nil {
				fmt.Printf("        [WARN] GET /files/%s: %v\n", truncateID(fileID, 12), err)
				continue
			}
			if status == b.fileStatusReady {
				delete(pending, fileID)
				fmt.Printf("        %s → %s ✓\n", truncateID(fileID, 16), status)
			}
		}

		if len(pending) > 0 {
			fmt.Printf("        polling... (%d 件待機中)\n", len(pending))
			time.Sleep(time.Duration(pollIntervalSecs) * time.Second)
		}
	}

	return true, nil
}

// getFileStatus は GET {apiBase}/files/{fileID} で status を取得する。
func (b *FileUploadBackend) getFileStatus(fileID string) (string, error) {
	req, err := http.NewRequest("GET", b.apiBase+"/files/"+fileID, nil)
	if err != nil {
		return "", err
	}
	if b.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.apiKey)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	status, _ := result["status"].(string)
	return status, nil
}

// Query は AgentBackend に委譲して POST {apiBase}/chat/completions を呼ぶ。
func (b *FileUploadBackend) Query(collectionID string, question string) (*QueryResult, error) {
	return b.queryBackend.Query(collectionID, question)
}

// Cleanup はコレクションの内部状態をクリアする（Files API では no-op）。
func (b *FileUploadBackend) Cleanup(collectionID string) error {
	b.mu.Lock()
	delete(b.fileIDs, collectionID)
	b.mu.Unlock()
	return nil
}

// GetFileIDs はコレクションに紐付けられたファイル ID 一覧を返す。
// RAGBackend インターフェース外のメソッド（prepare パッケージから使用）。
func (b *FileUploadBackend) GetFileIDs(collectionID string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	ids := make([]string, len(b.fileIDs[collectionID]))
	copy(ids, b.fileIDs[collectionID])
	return ids
}

func truncateID(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
