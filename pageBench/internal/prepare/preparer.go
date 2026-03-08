package prepare

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ttokunaga-ja/pagebench/internal/backend"
	"github.com/ttokunaga-ja/pagebench/internal/config"
	"github.com/ttokunaga-ja/pagebench/internal/domain"
)

// Options は preparation 実行オプション。
type Options struct {
	DomainDir string
	Cfg       *config.Config
	Force     bool // true = 既存の state を強制上書き
}

// Run は指定ドメインディレクトリに対して evaluation preparation を実行する。
//
// 動作:
//  1. source/ 内の PDF を POST {api_base}/files でアップロード
//  2. GET {api_base}/files/{id} をポーリングしてインデックス完了を確認
//  3. {domainDir}/.pagebench_prep.json に file_ids を保存
//
// Cfg.ExecuteEvaluationPreparation が false の場合はスキップする。
// 既存の state が IsReady() を返す場合は --force フラグがなければスキップする。
func Run(_ context.Context, opts Options) error {
	if opts.Cfg != nil && !opts.Cfg.ExecuteEvaluationPreparation {
		fmt.Printf("[SKIP] Evaluation Preparation をスキップ（PAGEBENCH_EXECUTE_EVALUATION_PREPARATION=false）: %s\n", opts.DomainDir)
		return nil
	}

	// 既存 state の確認（--force なし）
	if !opts.Force {
		existing, err := LoadState(opts.DomainDir)
		if err == nil && existing.IsReady() {
			fmt.Printf("[skip] .pagebench_prep.json が存在します（%s, %d ファイル, status=%s）\n",
				existing.PreparedAt.Format("2006-01-02 15:04:05"),
				existing.FileCount,
				existing.IndexStatus)
			fmt.Println("       上書きする場合は --force フラグを指定してください。")
			return nil
		}
	}

	domainName := filepath.Base(opts.DomainDir)

	// ソースファイル読み込み
	corpus, err := domain.LoadCorpus(opts.DomainDir)
	if err != nil {
		return fmt.Errorf("corpus 読み込み失敗: %w", err)
	}
	if len(corpus) == 0 {
		return fmt.Errorf("source/ ディレクトリに PDF ファイルがありません: %s", opts.DomainDir)
	}

	uc := opts.Cfg.Upload

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("  pageBench — Evaluation Preparation\n")
	fmt.Printf("  ドメイン    : %s\n", domainName)
	fmt.Printf("  API ベース  : %s\n", opts.Cfg.Agent.APIBase)
	fmt.Printf("  purpose     : %s\n", uc.Purpose)
	fmt.Printf("  ready 判定  : status=%q\n", uc.FileStatusReady)
	fmt.Printf("  対象ファイル: %d 件\n", len(corpus))
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))

	// FileUploadBackend 初期化
	b := backend.NewFileUploadBackend(
		opts.Cfg.Agent.APIBase,
		opts.Cfg.Agent.APIKey,
		uc.Purpose,
		uc.FileStatusReady,
		uc.TimeoutSecs,
		uc.PollIntervalSecs,
	)

	// 仮想コレクション作成
	collectionID, err := b.CreateCollection(domainName)
	if err != nil {
		return fmt.Errorf("コレクション作成失敗: %w", err)
	}

	// [1/2] ファイルアップロード
	fmt.Printf("[1/2] ドキュメントをアップロード中 (%d ファイル)...\n", len(corpus))
	fmt.Printf("      POST %s/files  [purpose=%s]\n", opts.Cfg.Agent.APIBase, uc.Purpose)

	uploaded := 0
	for _, entry := range corpus {
		pdfPath := filepath.Join(opts.DomainDir, "source", entry.FileName)
		f, err := os.Open(pdfPath)
		if err != nil {
			fmt.Printf("  [SKIP] ファイルが見つかりません: %s\n", entry.FileName)
			continue
		}
		fileID, err := b.UploadDocument(collectionID, entry.FileName, f)
		f.Close()
		if err != nil {
			fmt.Printf("  ✗ %s: アップロード失敗 (%v)\n", entry.FileName, err)
			continue
		}
		uploaded++
		fmt.Printf("  ✓ %-30s → %s\n", entry.FileName, truncate(fileID, 24))
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Printf("  → %d/%d ファイルをアップロード完了\n\n", uploaded, len(corpus))

	if uploaded == 0 {
		return fmt.Errorf("ファイルのアップロードに失敗しました（全 %d 件）", len(corpus))
	}

	// [2/2] インデックス完了待機
	fmt.Println("[2/2] インデックス完了を待機中...")
	fmt.Printf("      GET %s/files/{id}  [ready=%q]\n", opts.Cfg.Agent.APIBase, uc.FileStatusReady)

	ready, err := b.WaitForReady(collectionID, uc.TimeoutSecs, uc.PollIntervalSecs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [WARN] インデックス確認エラー: %v\n", err)
	}
	if ready {
		fmt.Println("  ✅ インデックス完了")
	} else {
		fmt.Fprintln(os.Stderr, "  [WARN] インデックス完了未確認。state を保存しますが精度が低下する可能性があります。")
	}

	// state 保存
	fileIDs := b.GetFileIDs(collectionID)
	indexStatus := "ready"
	if !ready {
		indexStatus = "partial"
	}
	state := &State{
		PreparedAt:  time.Now(),
		DomainDir:   opts.DomainDir,
		FileIDs:     fileIDs,
		FileCount:   len(fileIDs),
		IndexStatus: indexStatus,
	}
	if err := SaveState(opts.DomainDir, state); err != nil {
		return fmt.Errorf("state 保存失敗: %w", err)
	}

	fmt.Printf("\n  ✅ 準備完了 (%d/%d ファイル)\n", len(fileIDs), len(corpus))
	fmt.Printf("  状態ファイル: %s\n", StatePath(opts.DomainDir))
	fmt.Printf("  → pagebench eval --domain %s を実行してください\n", opts.DomainDir)
	fmt.Printf("%s\n", strings.Repeat("=", 60))

	return nil
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
