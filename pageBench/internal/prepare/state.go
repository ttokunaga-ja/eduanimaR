// Package prepare は evaluation preparation フェーズ（RAG インデックス作成）を提供する。
package prepare

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// stateFileName は domainDir 直下に保存される preparation 状態ファイル名。
const stateFileName = ".pagebench_prep.json"

// State は evaluation preparation の永続化状態。
// pagebench prepare コマンドが完了すると domainDir/.pagebench_prep.json に保存される。
// pagebench eval コマンドはこのファイルが存在する場合、アップロード/インデックス待機をスキップする。
type State struct {
	PreparedAt  time.Time `json:"prepared_at"`
	DomainDir   string    `json:"domain_dir"`
	FileIDs     []string  `json:"file_ids"`
	FileCount   int       `json:"file_count"`
	IndexStatus string    `json:"index_status"` // "ready" | "partial"
}

// IsReady は preparation が正常完了済みかどうかを返す。
func (s *State) IsReady() bool {
	return s.IndexStatus == "ready" && len(s.FileIDs) > 0
}

// StatePath は domainDir の state ファイルのフルパスを返す。
func StatePath(domainDir string) string {
	return filepath.Join(domainDir, stateFileName)
}

// LoadState は domainDir の state ファイルを読み込んで返す。
// ファイルが存在しない場合は os.ErrNotExist を含むエラーを返す。
func LoadState(domainDir string) (*State, error) {
	data, err := os.ReadFile(StatePath(domainDir))
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("state ファイルのパース失敗: %w", err)
	}
	return &s, nil
}

// SaveState は state を domainDir/.pagebench_prep.json に書き込む。
func SaveState(domainDir string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("state のシリアライズ失敗: %w", err)
	}
	return os.WriteFile(StatePath(domainDir), data, 0o644)
}

// ClearState は state ファイルを削除する。ファイルが存在しない場合は何もしない。
func ClearState(domainDir string) error {
	path := StatePath(domainDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}
