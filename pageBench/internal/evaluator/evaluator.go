// Package evaluator provides the main evaluation loop.
package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/ttokunaga-ja/pagebench/internal/backend"
	"github.com/ttokunaga-ja/pagebench/internal/checkpoint"
	"github.com/ttokunaga-ja/pagebench/internal/config"
	"github.com/ttokunaga-ja/pagebench/internal/domain"
	geminiutil "github.com/ttokunaga-ja/pagebench/internal/gemini"
	"github.com/ttokunaga-ja/pagebench/internal/judge"
	"github.com/ttokunaga-ja/pagebench/internal/prepare"
	"github.com/ttokunaga-ja/pagebench/internal/reporter"
)

// Options は評価実行オプション。
type Options struct {
	DomainDir  string
	Cfg        *config.Config
	Limit      int // 0 = 全件
	SkipJudge  bool
	Resume     bool
	NoCleanup  bool
	Force      bool // true の場合は既存 0c/0d の上書きを許可
	UploadOnly bool // アップロードのみ実行して評価ループをスキップ（後方互換）
}

// Run は指定ドメインディレクトリに対して評価を実行する。
// Cfg.ExecuteEvaluation が false の場合はスキップする。
func Run(ctx context.Context, opts Options) ([]domain.EvalRecord, error) {
	// フェーズ制御
	if opts.Cfg != nil && !opts.Cfg.ExecuteEvaluation {
		fmt.Printf("[SKIP] 評価をスキップ（PAGEBENCH_EXECUTE_EVALUATION=false）: %s\n", opts.DomainDir)
		return nil, nil
	}

	domainName := filepath.Base(opts.DomainDir)

	if !opts.UploadOnly && !opts.Force {
		hasRows, checkErr := domain.HasNonHeaderRows(opts.DomainDir, domain.FileEvaluate)
		if checkErr != nil {
			return nil, fmt.Errorf("%s の既存データ確認に失敗: %w", domain.FileEvaluate, checkErr)
		}
		if hasRows {
			return nil, fmt.Errorf("%s に既存データがあります。上書きするには --force を指定してください", domain.FileEvaluate)
		}

		reportPath := filepath.Join(opts.DomainDir, domain.FileReport)
		if st, err := os.Stat(reportPath); err == nil && st.Size() > 0 {
			return nil, fmt.Errorf("%s に既存データがあります。上書きするには --force を指定してください", domain.FileReport)
		}
	}

	// データ読み込み
	corpus, err := domain.LoadCorpus(opts.DomainDir)
	if err != nil {
		return nil, fmt.Errorf("corpus 読み込み失敗: %w", err)
	}

	var qaPairs []domain.QAPair
	if !opts.UploadOnly {
		qaPairs, err = domain.LoadQAPairs(opts.DomainDir)
		if err != nil {
			return nil, fmt.Errorf("%s 読み込み失敗: %w", domain.FileQAPairs, err)
		}
		if len(qaPairs) == 0 {
			fmt.Fprintf(os.Stderr, "[WARN] %s が空です: %s\n", domain.FileQAPairs, opts.DomainDir)
			return nil, nil
		}
		if opts.Limit > 0 && opts.Limit < len(qaPairs) {
			qaPairs = qaPairs[:opts.Limit]
		}
	}

	// 設定表示
	judgeLabel := "スキップ"
	if !opts.SkipJudge {
		judgeLabel = opts.Cfg.Gemini.JudgeModel
	}
	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("  pageBench — RAG システム性能評価\n")
	fmt.Printf("  ドメイン  : %s\n", domainName)
	fmt.Printf("  バックエンド: %s\n", opts.Cfg.BackendDisplay())
	fmt.Printf("  対象 QA   : %d 件\n", len(qaPairs))
	fmt.Printf("  LLM Judge : %s\n", judgeLabel)
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))

	// チェックポイント
	cp, err := checkpoint.New(opts.DomainDir)
	if err != nil {
		return nil, fmt.Errorf("checkpoint 初期化失敗: %w", err)
	}
	if opts.Resume && cp.Exists() {
		fmt.Printf("[resume] チェックポイントを発見: %d 件完了済み → 再開します\n", cp.DoneCount())
	} else if !opts.Resume && cp.Exists() {
		fmt.Println("[info] 未完了のチェックポイントが存在します。--resume で再開できます。新規評価を開始します。")
		if err := cp.Clear(); err != nil {
			return nil, err
		}
	}

	// バックエンド初期化
	b := backend.NewAgentBackend(
		opts.Cfg.Agent.APIBase,
		opts.Cfg.Agent.APIKey,
		opts.Cfg.Agent.Model,
	)

	// Evaluation Preparation state の確認
	// .pagebench_prep.json が存在する場合はアップロード/インデックス待機をスキップ
	prepState, _ := prepare.LoadState(opts.DomainDir)
	hasPrepState := prepState != nil && prepState.IsReady()

	// コレクション作成 or 再利用
	collectionID := ""
	if opts.Resume {
		collectionID = cp.GetCollectionID()
	}

	if hasPrepState && collectionID == "" {
		collectionID = fmt.Sprintf("prepared:%s", domainName)
		fmt.Printf("[prep] .pagebench_prep.json を発見 (%s, %d ファイル) → アップロードをスキップ\n",
			prepState.PreparedAt.Format("2006-01-02 15:04:05"), prepState.FileCount)
	} else if collectionID != "" {
		fmt.Printf("[resume] 既存コレクションを再利用: %s...\n", collectionID[:min(16, len(collectionID))])
	} else {
		fmt.Println("[1/4] コレクション（科目）を作成中...")
		benchName := fmt.Sprintf("[bench] %s %s", domainName, time.Now().Format("20060102_150405"))
		collectionID, err = b.CreateCollection(benchName)
		if err != nil {
			return nil, fmt.Errorf("コレクション作成失敗: %w", err)
		}
		if err := cp.SetCollectionID(collectionID); err != nil {
			return nil, err
		}
		fmt.Printf("      collection_id: %s\n\n", collectionID)

		// PDF アップロード
		fmt.Printf("[2/4] ドキュメントをアップロード中 (%d ファイル)...\n", len(corpus))
		uploaded := 0
		for _, entry := range corpus {
			pdfPath := filepath.Join(opts.DomainDir, "source", entry.FileName)
			f, err := os.Open(pdfPath)
			if err != nil {
				fmt.Printf("  [SKIP] ファイルが見つかりません: %s\n", entry.FileName)
				continue
			}
			_, err = b.UploadDocument(collectionID, entry.FileName, f)
			f.Close()
			if err != nil {
				fmt.Printf("  ✗ %s: アップロード失敗 (%v)\n", entry.FileName, err)
				continue
			}
			uploaded++
			fmt.Printf("  ✓ %s\n", entry.FileName)
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Printf("  → %d/%d ファイルをアップロード完了\n\n", uploaded, len(corpus))

		// インデックス完了待機
		fmt.Println("[3/4] インデックス処理の完了を待機中...")
		ready, err := b.WaitForReady(collectionID, 0, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] インデックス確認エラー: %v\n", err)
		}
		if !ready {
			fmt.Fprintln(os.Stderr, "[WARN] インデックス完了未確認。評価を続行しますが精度が低下する可能性があります。")
		}
		fmt.Println()
	}

	// UploadOnly モードはアップロードフェーズで終了
	if opts.UploadOnly {
		fmt.Printf("  [upload] コレクション ID: %s\n", collectionID)
		fmt.Println("  [upload] ドキュメントのアップロードが完了しました。")
		return nil, nil
	}

	// Judge 初期化
	var j *judge.Judge
	if !opts.SkipJudge {
		if opts.Cfg.Gemini.APIKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY が設定されていません。--skip-judge で Judge をスキップできます")
		}
		j, err = judge.New(ctx, opts.Cfg.Gemini.JudgeModel, opts.Cfg.Gemini.ThinkingLevel)
		if err != nil {
			return nil, fmt.Errorf("Judge 初期化失敗: %w", err)
		}
	}

	// Gemini クライアント初期化（SemanticSimilarity 計算用）
	// GEMINI_API_KEY が設定されていれば初期化する（SkipJudge に関係なく）
	var geminiClient *genai.Client
	if opts.Cfg.Gemini.APIKey != "" {
		gc, gcErr := geminiutil.NewClient(ctx)
		if gcErr != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Gemini クライアント初期化失敗（SemanticSimilarity をスキップ）: %v\n", gcErr)
		} else {
			geminiClient = gc
			fmt.Println("[info] Gemini embedding クライアント初期化完了（SemanticSimilarity 計算有効）")
		}
	} else {
		fmt.Println("[info] GEMINI_API_KEY 未設定。SemanticSimilarity 計算をスキップします。")
	}

	// 評価ループ
	fmt.Printf("[4/4] 評価開始 (%d 件)...\n", len(qaPairs))
	results := cp.GetResults()
	if len(results) > 0 {
		fmt.Printf("  → %d 件をスキップ（チェックポイント済み）\n", len(results))
	}

	rateSleep := time.Duration(opts.Cfg.Gemini.RateLimitSleepMs) * time.Millisecond

	for i, qa := range qaPairs {
		if opts.Resume && cp.IsDone(qa.QID) {
			continue
		}

		fmt.Printf("\n  [%03d/%03d] %s...\n", i+1, len(qaPairs), truncate(qa.Question, 70))

		// 質問送信
		var ragAnswer string
		var latencyMS int
		var ragSourcesStr string
		var fileHit, pageHit int
		var retrievedFilePages string
		var refFileFound, refPageFound, refFilePageFound int
		var answerability string
		var qResult *backend.QueryResult

		qResult, err = b.Query(collectionID, qa.Question)
		if err != nil {
			fmt.Printf("          [ERROR] 回答取得失敗: %v\n", err)
		} else {
			ragAnswer = qResult.Answer
			latencyMS = qResult.LatencyMS
			answerability = qResult.Answerability
			srcJSON, _ := json.Marshal(qResult.Sources)
			ragSourcesStr = string(srcJSON)
			fmt.Printf("          回答取得完了 (%d ms) | 文字数: %d | answerability: %s\n",
				latencyMS, len(ragAnswer), answerability)

			// ファイル・ページ一致検証（LLM なし）
			retrievedFilePages, refFileFound, refPageFound, refFilePageFound = evaluateRetrievedSources(qResult.Sources, qa.RefFile, qa.RefPage)
			fileHit = refFileFound
			pageHit = refFilePageFound
			fmt.Printf("          file_hit=%d  page_hit=%d  ref_file_found=%d  ref_page_found=%d  ref_file_page_found=%d\n",
				fileHit, pageHit, refFileFound, refPageFound, refFilePageFound)
			fmt.Printf("          retrieved=%s\n", truncate(retrievedFilePages, 220))
		}

		// answerability 表示（unanswerable 専用）
		if qa.QuestionType == "unanswerable" {
			fmt.Printf("          unanswerable | answerability=%s\n", answerability)
		}

		// SemanticSimilarity（answerable かつ ref_answer / rag_answer あり、Gemini 有効時のみ）
		var semSim float64
		if qa.QuestionType != "unanswerable" && ragAnswer != "" && qa.RefAnswer != "" && geminiClient != nil {
			var simErr error
			semSim, simErr = computeSemanticSimilarity(ctx, geminiClient, ragAnswer, qa.RefAnswer)
			if simErr != nil {
				fmt.Printf("          [WARN] SemanticSimilarity 計算失敗: %v\n", simErr)
				semSim = 0.0
			} else {
				fmt.Printf("          semantic_similarity=%.4f\n", semSim)
			}
		}

		// LLM-as-Judge
		var judgeAccuracy, judgeFaithful, judgeComplete, judgeOverall string
		var judgeHallucination, judgeReasoning string
		if j != nil && ragAnswer != "" {
			if qa.QuestionType == "unanswerable" {
				// unanswerable: ハルシネーション採点
				juResult, err := j.ScoreUnanswerable(ctx, qa.Question, ragAnswer)
				if err != nil {
					fmt.Printf("          [WARN] Judge (unanswerable) エラー: %v\n", err)
				} else {
					judgeHallucination = fmt.Sprintf("%d", juResult.Hallucination)
					judgeOverall = fmt.Sprintf("%d", juResult.Overall)
					judgeReasoning = juResult.Reasoning
					fmt.Printf("          Judge(unanswerable): hallucination=%s overall=%s\n",
						judgeHallucination, judgeOverall)
				}
			} else {
				// answerable: 通常採点
				jResult, err := j.Score(ctx, qa.Question, qa.RefAnswer, ragAnswer, qa.RefEvidence)
				if err != nil {
					fmt.Printf("          [WARN] Judge エラー: %v\n", err)
				} else {
					judgeAccuracy = fmt.Sprintf("%d", jResult.Accuracy)
					judgeFaithful = fmt.Sprintf("%d", jResult.Faithfulness)
					judgeComplete = fmt.Sprintf("%d", jResult.Completeness)
					judgeOverall = fmt.Sprintf("%d", jResult.Overall)
					judgeReasoning = jResult.Reasoning
					fmt.Printf("          Judge: acc=%s faith=%s comp=%s overall=%s\n",
						judgeAccuracy, judgeFaithful, judgeComplete, judgeOverall)
				}
			}
		}

		loopCount := 0
		librarianMS := 0
		answerGenMS := 0
		if qResult != nil {
			loopCount = qResult.LoopCount
			librarianMS = qResult.LibrarianMS
			answerGenMS = qResult.AnswerGenMS
		}

		rec := domain.EvalRecord{
			QID:                qa.QID,
			Domain:             domainName,
			Question:           qa.Question,
			QuestionType:       qa.QuestionType,
			Difficulty:         qa.Difficulty,
			RefAnswer:          qa.RefAnswer,
			RefEvidence:        qa.RefEvidence,
			RefFile:            qa.RefFile,
			RefPage:            qa.RefPage,
			RagAnswer:          ragAnswer,
			RagSources:         ragSourcesStr,
			FileHit:            fileHit,
			PageHit:            pageHit,
			RetrievedFilePages: retrievedFilePages,
			RefFileFound:       refFileFound,
			RefPageFound:       refPageFound,
			RefFilePageFound:   refFilePageFound,
			SemanticSimilarity: semSim,
			Answerability:      answerability,
			LatencyMS:          latencyMS,
			LoopCount:          loopCount,
			LibrarianMS:        librarianMS,
			AnswerGenMS:        answerGenMS,
			JudgeAccuracy:      judgeAccuracy,
			JudgeFaithful:      judgeFaithful,
			JudgeComplete:      judgeComplete,
			JudgeOverall:       judgeOverall,
			JudgeHallucination: judgeHallucination,
			JudgeReasoning:     judgeReasoning,
		}

		results = append(results, rec)
		if err := cp.MarkDone(qa.QID, rec); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] checkpoint 保存失敗: %v\n", err)
		}

		time.Sleep(rateSleep)
	}

	// 結果出力
	if err := domain.WriteEvaluate(opts.DomainDir, results); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] %s 出力失敗: %v\n", domain.FileEvaluate, err)
	}

	summary := reporter.ComputeSummary(results)
	if err := reporter.WriteMarkdownReport(results, opts.DomainDir, domainName, opts.Cfg.BackendDisplay(), summary); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] %s 出力失敗: %v\n", domain.FileReport, err)
	}

	reporter.PrintSummary(summary, domainName)
	fmt.Printf("  結果 CSV  : %s\n", filepath.Join(opts.DomainDir, domain.FileEvaluate))
	fmt.Printf("  レポート  : %s\n", filepath.Join(opts.DomainDir, domain.FileReport))

	// クリーンアップ
	if !opts.NoCleanup {
		if err := b.Cleanup(collectionID); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] cleanup 失敗: %v\n", err)
		}
	}
	_ = cp.Clear()

	return results, nil
}

// computeSemanticSimilarity は 2 つのテキスト間の SemanticSimilarity（コサイン類似度）を計算する。
// gemini-embedding-001 + task_type="SEMANTIC_SIMILARITY" を使用する。
func computeSemanticSimilarity(ctx context.Context, client *genai.Client, a, b string) (float64, error) {
	embA, err := geminiutil.EmbedText(ctx, client, a)
	if err != nil {
		return 0, fmt.Errorf("回答の埋め込み失敗: %w", err)
	}
	embB, err := geminiutil.EmbedText(ctx, client, b)
	if err != nil {
		return 0, fmt.Errorf("参照解答の埋め込み失敗: %w", err)
	}
	return geminiutil.CosineSimilarity(embA, embB), nil
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type retrievedFilePagesEntry struct {
	FileName string   `json:"file_name"`
	Pages    []string `json:"pages,omitempty"`
}

func evaluateRetrievedSources(sources []backend.Source, refFile, refPage string) (string, int, int, int) {
	pageSetByFile := make(map[string]map[string]struct{}, len(sources))
	refFileFound := 0
	refPageFound := 0
	refFilePageFound := 0

	refFileNorm := normalizeFileNameForMatch(refFile)
	refPageNorm := strings.TrimSpace(refPage)

	for _, src := range sources {
		name := strings.TrimSpace(src.Name)
		if name == "" {
			continue
		}
		page := strings.TrimSpace(src.Page)

		if _, ok := pageSetByFile[name]; !ok {
			pageSetByFile[name] = map[string]struct{}{}
		}
		if page != "" {
			pageSetByFile[name][page] = struct{}{}
		}

		if refFileNorm != "" && normalizeFileNameForMatch(name) == refFileNorm {
			refFileFound = 1
			if refPageNorm != "" && page == refPageNorm {
				refFilePageFound = 1
			}
		}
		if refPageNorm != "" && page == refPageNorm {
			refPageFound = 1
		}
	}

	entries := make([]retrievedFilePagesEntry, 0, len(pageSetByFile))
	fileNames := make([]string, 0, len(pageSetByFile))
	for fileName := range pageSetByFile {
		fileNames = append(fileNames, fileName)
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		pages := make([]string, 0, len(pageSetByFile[fileName]))
		for page := range pageSetByFile[fileName] {
			pages = append(pages, page)
		}
		sort.Strings(pages)
		entries = append(entries, retrievedFilePagesEntry{FileName: fileName, Pages: pages})
	}

	bJSON, err := json.Marshal(entries)
	if err != nil {
		return "[]", refFileFound, refPageFound, refFilePageFound
	}
	return string(bJSON), refFileFound, refPageFound, refFilePageFound
}

func normalizeFileNameForMatch(name string) string {
	replaced := strings.ReplaceAll(name, "\u3000", " ")
	trimmed := strings.TrimSpace(replaced)
	compact := strings.Join(strings.Fields(trimmed), " ")
	return strings.ToLower(compact)
}
