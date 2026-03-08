// Package checkpoint provides evaluation resume support.
// 評価ループが途中でクラッシュしても --resume フラグで再開できる。
package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ttokunaga-ja/pagebench/internal/domain"
)

const defaultFileName = ".checkpoint.json"

// data はチェックポイントファイルの構造。
type data struct {
	CollectionID string              `json:"collection_id"`
	DoneIDs      []string            `json:"done_ids"`
	Results      []domain.EvalRecord `json:"results"`
}

// Checkpoint は評価の進捗を JSON ファイルに永続化する。
type Checkpoint struct {
	path string
	d    data
}

// New はドメインディレクトリ内の .checkpoint.json を対象とした Checkpoint を返す。
func New(domainDir string) (*Checkpoint, error) {
	path := filepath.Join(domainDir, defaultFileName)
	c := &Checkpoint{path: path}
	if err := c.load(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Checkpoint) load() error {
	b, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return nil // 初回は空でOK
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &c.d)
}

func (c *Checkpoint) save() error {
	b, err := json.MarshalIndent(c.d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, b, 0o644)
}

// Exists はチェックポイントファイルが存在し、データがあるか確認する。
func (c *Checkpoint) Exists() bool {
	_, err := os.Stat(c.path)
	return err == nil && (c.d.CollectionID != "" || len(c.d.DoneIDs) > 0)
}

// GetCollectionID は保存済みの collection_id を返す（空文字 = 未設定）。
func (c *Checkpoint) GetCollectionID() string {
	return c.d.CollectionID
}

// SetCollectionID は collection_id を保存する。
func (c *Checkpoint) SetCollectionID(id string) error {
	c.d.CollectionID = id
	return c.save()
}

// IsDone は指定した q_id が完了済みか確認する。
func (c *Checkpoint) IsDone(qID string) bool {
	for _, id := range c.d.DoneIDs {
		if id == qID {
			return true
		}
	}
	return false
}

// MarkDone は q_id を完了済みとしてマークし、結果を保存する。
func (c *Checkpoint) MarkDone(qID string, rec domain.EvalRecord) error {
	if !c.IsDone(qID) {
		c.d.DoneIDs = append(c.d.DoneIDs, qID)
	}
	c.d.Results = append(c.d.Results, rec)
	return c.save()
}

// GetResults は完了済みの評価結果リストを返す。
func (c *Checkpoint) GetResults() []domain.EvalRecord {
	return c.d.Results
}

// DoneCount は完了済みの問題数を返す。
func (c *Checkpoint) DoneCount() int {
	return len(c.d.DoneIDs)
}

// Clear はチェックポイントをリセットしてファイルを削除する。
func (c *Checkpoint) Clear() error {
	c.d = data{}
	if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
