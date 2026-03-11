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
	Difficulty      string `json:"difficulty"`       // "simple" | "reasoning" | "multi_chunk" | "N/A"
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
			"difficulty":       {Type: genai.TypeString, Description: `answerable の場合は "simple" | "reasoning" | "multi_chunk"、unanswerable の場合は "N/A"`},
			"reference_answer": {Type: genai.TypeString, Description: "質問への模範解答（unanswerable の場合は空文字）"},
			"evidence_text":    {Type: genai.TypeString, Description: "解答の根拠となる文書内のテキスト（100字以内。unanswerable の場合は空文字）"},
			"target_page":      {Type: genai.TypeString, Description: "根拠ページ番号（不明・unanswerable の場合は空文字）"},
		},
		Required: []string{"question", "question_type", "difficulty", "reference_answer", "evidence_text"},
	},
}

const generatePromptTmpl = `You are an expert QA dataset creator for benchmarking RAG (Retrieval-Augmented Generation) and Agentic Search systems.

## Language Rule (ABSOLUTE)
Detect the primary language of document "%s" and generate ALL output — questions, reference answers, and evidence — in that EXACT same language.

## Task
Generate %d high-quality QA pairs from the provided document to evaluate retrieval accuracy, multi-chunk reasoning, and hallucination resistance.

## Distribution Requirements

**Answerable (~80%%)** — set question_type to "answerable":
- simple (~30%%): Single factual lookup from one sentence or chunk. Set difficulty "simple".
- reasoning (~30%%): Requires deduction, calculation, or combining facts within one section. Set difficulty "reasoning".
- multi_chunk (~20%%): Requires synthesizing information from two or more separate sections. Set difficulty "multi_chunk".

**Unanswerable (~20%%)** — set question_type to "unanswerable", difficulty to "N/A", reference_answer to "", evidence_text to "":
- Must be topically relevant to the document's named entities, but the answer must NOT appear anywhere in the text.
- Valid types: comparisons with external entities, future predictions beyond document scope, undisclosed technical or pricing details.

## Question Uniqueness and Entity Binding (CRITICAL)
Each question must be universally unique and specifically anchored to this document so that it cannot be confused with any other document in a multi-document corpus.

**Rule 1 — Entity Binding (applies to >70%% of questions)**
Include highly specific proper nouns, named entities, unique product or concept names, specific years, or unique terminology found in the document.

**Rule 2 — Numeric Anchoring (applies to >20%% of questions)**
Include specific numbers, metrics, percentages, dates, or quantities that appear in the document.

**Rule 3 — No Vague References (applies to ALL questions)**
NEVER use "this document", "this paper", "the author", "the report", or "the system". Always refer to entities by their specific names as they appear in the document.
- Bad: "What is the main finding of this research?"
- Good: "What accuracy did the AMAQA benchmark achieve on the 2025 cross-lingual evaluation task?"

Each question must be fully self-contained and independent (no question may assume knowledge of any other question's answer).`

// Options は generate サブコマンドのオプション。
type Options struct {
	DomainDir     string
	Model         string
	ThinkingLevel string
	QAPerDoc      int            // 1 ドキュメントあたりの QA 件数
	Cfg           *config.Config // フェーズ実行制御のために使用（nil = 全フェーズ実行）
	Force         bool           // true の場合は既存 0a/0b の上書きを許可
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
		if !opts.Force {
			hasRows, checkErr := domain.HasNonHeaderRows(opts.DomainDir, domain.FileRegistry)
			if checkErr != nil {
				return fmt.Errorf("%s の既存データ確認に失敗: %w", domain.FileRegistry, checkErr)
			}
			if hasRows {
				return fmt.Errorf("%s に既存データがあります。上書きするには --force を指定してください", domain.FileRegistry)
			}
		}

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
	if !opts.Force {
		hasRows, checkErr := domain.HasNonHeaderRows(opts.DomainDir, domain.FileQAPairs)
		if checkErr != nil {
			return fmt.Errorf("%s の既存データ確認に失敗: %w", domain.FileQAPairs, checkErr)
		}
		if hasRows {
			return fmt.Errorf("%s に既存データがあります。上書きするには --force を指定してください", domain.FileQAPairs)
		}
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
				Difficulty:   p.Difficulty,
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
