// Package generate creates QA test sets from PDF documents using Gemini File API.
package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/ttokunaga-ja/pagebench/internal/config"
	"github.com/ttokunaga-ja/pagebench/internal/domain"
	geminiutil "github.com/ttokunaga-ja/pagebench/internal/gemini"
)

// geminiQAPair は Gemini からのレスポンス用内部型（target_file 除く）。
type geminiQAPair struct {
	Question        string `json:"question"`
	QuestionType    string `json:"question_type"`    // "answerable" | "unanswerable"
	ReferenceAnswer string `json:"reference_answer"` // unanswerable の場合は空文字
	EvidenceText    string `json:"evidence_text"`    // unanswerable の場合は空文字
	TargetPage      string `json:"target_page"`
}

// qaListSchema は Gemini Structured Output 用のスキーマ（QA リスト）。
var qaListSchema = &genai.Schema{
	Type: genai.TypeArray,
	Items: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"question":         {Type: genai.TypeString, Description: "質問文"},
			"question_type":    {Type: genai.TypeString, Description: `"answerable"（文書から回答可能）または "unanswerable"（文書に答えがない）`},
			"reference_answer": {Type: genai.TypeString, Description: "質問への模範解答（unanswerable の場合は空文字）"},
			"evidence_text":    {Type: genai.TypeString, Description: "解答の根拠となる文書内のテキスト（100字以内。unanswerable の場合は空文字）"},
			"target_page":      {Type: genai.TypeString, Description: "根拠ページ番号（不明・unanswerable の場合は空文字）"},
		},
		Required: []string{"question", "question_type", "reference_answer", "evidence_text"},
	},
}

const generatePromptTmpl = `あなたは RAG システム評価用の QA データセット作成の専門家です。
提供された文書（%s）から、RAG システムの検索・回答能力を評価するための質問と解答ペアを %d 件作成してください。

要件:
1. 質問は文書の具体的な内容（数値、固有名詞、因果関係など）に基づくこと
2. 参照解答は文書から直接導出できる正確な内容であること
3. 質問の難易度は多様にすること（ファクトual質問と推論を要する質問を混在）
4. 各質問は独立していること（前の質問への回答を前提としない）
5. 日本語と英語が混在する文書の場合は、質問・解答ともに日本語で作成すること
6. 全体の約20%%を「回答不能質問（unanswerable）」とすること:
   - 文書の主題と関連するが、文書内に根拠がない質問（他社比較・将来予測・文書外の情報など）
   - question_type を "unanswerable" に設定し、reference_answer と evidence_text は空文字にすること
   - 残りの約80%%は question_type を "answerable" に設定すること

JSON 配列形式で出力してください。`

// Options は generate サブコマンドのオプション。
type Options struct {
	DomainDir     string
	Model         string
	ThinkingLevel string
	QAPerDoc      int            // 1 ドキュメントあたりの QA 件数
	Cfg           *config.Config // フェーズ実行制御のために使用（nil = 全フェーズ実行）
}

// Run は source/ のPDFをスキャンして 0a_registry.csv を生成し、
// 各PDFから QA ペアを生成して 0b_qa_pairs.csv に書き出す。
// Cfg.ExecuteRegistry / Cfg.ExecuteQA により各フェーズをスキップ可能。
func Run(ctx context.Context, opts Options) error {
	// ── Step 1: source/ をスキャンして 0a_registry.csv を生成 ──────────
	var corpus []domain.CorpusEntry
	var err error

	skipRegistry := opts.Cfg != nil && !opts.Cfg.ExecuteRegistry
	if skipRegistry {
		fmt.Printf("[SKIP] %s 生成をスキップ（PAGEBENCH_EXECUTE_REGISTRY=false）\n", domain.FileRegistry)
		corpus, err = domain.LoadCorpus(opts.DomainDir)
		if err != nil {
			return fmt.Errorf("既存の %s 読み込み失敗: %w", domain.FileRegistry, err)
		}
		if len(corpus) == 0 {
			return fmt.Errorf("既存の %s にエントリがありません。PAGEBENCH_EXECUTE_REGISTRY=true で先に生成してください", domain.FileRegistry)
		}
		fmt.Printf("      既存 %s から %d 件を読み込みました\n\n", domain.FileRegistry, len(corpus))
	} else {
		fmt.Printf("[1/3] source/ をスキャンして %s を生成中...\n", domain.FileRegistry)
		corpus, err = domain.ScanAndWriteRegistry(opts.DomainDir)
		if err != nil {
			return err
		}
		fmt.Printf("      %d 件の PDF を検出しました\n\n", len(corpus))
	}

	// ── Step 2-3: QA ペア生成 ────────────────────────────────────────
	skipQA := opts.Cfg != nil && !opts.Cfg.ExecuteQA
	if skipQA {
		fmt.Printf("[SKIP] %s 生成をスキップ（PAGEBENCH_EXECUTE_QA=false）\n", domain.FileQAPairs)
		return nil
	}

	// Gemini クライアント初期化
	client, err := geminiutil.NewClient(ctx)
	if err != nil {
		return err
	}

	perDoc := opts.QAPerDoc
	if perDoc <= 0 {
		perDoc = 10
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("  pageBench — QA テストセット生成\n")
	fmt.Printf("  ドメイン    : %s\n", filepath.Base(opts.DomainDir))
	fmt.Printf("  モデル      : %s\n", opts.Model)
	fmt.Printf("  thinking    : %s\n", opts.ThinkingLevel)
	fmt.Printf("  ドキュメント: %d 件 × %d QA/doc\n", len(corpus), perDoc)
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))

	// 各 PDF から QA を生成
	var allPairs []domain.QAPair
	qidCounter := 1

	for i, entry := range corpus {
		pdfPath := filepath.Join(opts.DomainDir, "source", entry.FileName)

		// ページ数に応じた動的 QA 件数計算
		qaCount := calcQAPerDoc(entry.FilePages, perDoc, opts.Cfg)
		fmt.Printf("[%02d/%02d] %s を処理中... (%d QA)\n", i+1, len(corpus), entry.FileName, qaCount)

		pairs, err := generateForFile(ctx, client, opts.Model, opts.ThinkingLevel, pdfPath, entry.FileName, qaCount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [ERROR] %s の QA 生成失敗: %v\n", entry.FileName, err)
			time.Sleep(5 * time.Second)
			continue
		}

		// q_id を連番で付与してドメイン型に変換
		answerableCount, unanswerableCount := 0, 0
		for _, p := range pairs {
			qt := p.QuestionType
			if qt != "unanswerable" {
				qt = "answerable" // デフォルト
			}
			allPairs = append(allPairs, domain.QAPair{
				QID:          fmt.Sprintf("Q%04d", qidCounter),
				Question:     p.Question,
				QuestionType: qt,
				RefAnswer:    p.ReferenceAnswer,
				RefEvidence:  p.EvidenceText,
				RefFile:      entry.FileName,
				RefPage:      p.TargetPage,
			})
			if qt == "unanswerable" {
				unanswerableCount++
			} else {
				answerableCount++
			}
			qidCounter++
		}
		fmt.Printf("  → %d 件の QA を生成 (answerable=%d, unanswerable=%d)\n",
			len(pairs), answerableCount, unanswerableCount)
		time.Sleep(3 * time.Second) // レート制限対策
	}

	if len(allPairs) == 0 {
		return fmt.Errorf("QA ペアが 1 件も生成されませんでした")
	}

	// 0b_qa_pairs.csv に書き出す
	if err := domain.WriteQAPairs(opts.DomainDir, allPairs); err != nil {
		return fmt.Errorf("%s 書き出し失敗: %w", domain.FileQAPairs, err)
	}

	fmt.Printf("\n✓ %d 件の QA ペアを生成しました: %s/%s\n",
		len(allPairs), opts.DomainDir, domain.FileQAPairs)
	return nil
}

// generateForFile は 1 つの PDF ファイルに対して QA を生成する。
func generateForFile(ctx context.Context, client *genai.Client, model, thinkingLevel, pdfPath, fileName string, perDoc int) ([]geminiQAPair, error) {
	// Gemini File API でファイルをアップロード
	f, err := os.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("PDF を開けません: %w", err)
	}
	defer f.Close()

	uploadedFile, err := client.Files.Upload(ctx, f, &genai.UploadFileConfig{
		MIMEType:    "application/pdf",
		DisplayName: fileName,
		HTTPOptions: &genai.HTTPOptions{Headers: http.Header{}}, // SDK パニック回避
	})
	if err != nil {
		return nil, fmt.Errorf("File API アップロード失敗: %w", err)
	}
	defer func() {
		_, _ = client.Files.Delete(ctx, uploadedFile.Name, nil)
	}()

	// ファイルの処理完了を待機
	for uploadedFile.State == genai.FileStateProcessing {
		time.Sleep(2 * time.Second)
		uploadedFile, err = client.Files.Get(ctx, uploadedFile.Name, nil)
		if err != nil {
			return nil, fmt.Errorf("File API ステータス確認失敗: %w", err)
		}
	}
	if uploadedFile.State != genai.FileStateActive {
		return nil, fmt.Errorf("File API の処理失敗: state=%s", uploadedFile.State)
	}

	// プロンプト構築
	prompt := fmt.Sprintf(generatePromptTmpl, fileName, perDoc)

	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   qaListSchema,
		Temperature:      ptr(float32(0.5)), // QA 多様性のため適度なランダム性を維持
		Seed:             ptr(int32(42)),    // 乱数シード固定（再現性確保）
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: geminiutil.ThinkingLevel(thinkingLevel),
		},
	}

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{FileData: &genai.FileData{FileURI: uploadedFile.URI, MIMEType: "application/pdf"}},
				{Text: prompt},
			},
			Role: "user",
		},
	}

	resp, err := client.Models.GenerateContent(ctx, model, contents, cfg)
	if err != nil {
		return nil, fmt.Errorf("Gemini 生成失敗: %w", err)
	}

	var pairs []geminiQAPair
	if err := json.Unmarshal([]byte(resp.Text()), &pairs); err != nil {
		return nil, fmt.Errorf("レスポンスデコード失敗: %w (raw: %s)", err, resp.Text())
	}

	return pairs, nil
}

// calcQAPerDoc はページ数に応じた QA 件数を計算する。
// Cfg.QA.Density > 0 の場合は動的計算:
//
//	QA = clamp(round(pages × density), qa_min, qa_max)
//
// Density が未設定（0）の場合は defaultPerDoc をそのまま返す。
func calcQAPerDoc(filePages string, defaultPerDoc int, cfg *config.Config) int {
	if cfg == nil || cfg.QA.Density <= 0 {
		return defaultPerDoc
	}
	pages, err := strconv.Atoi(filePages)
	if err != nil || pages <= 0 {
		return cfg.QA.Min
	}
	qa := int(math.Round(float64(pages) * cfg.QA.Density))
	if qa < cfg.QA.Min {
		qa = cfg.QA.Min
	}
	if qa > cfg.QA.Max {
		qa = cfg.QA.Max
	}
	return qa
}

func ptr[T any](v T) *T { return &v }
