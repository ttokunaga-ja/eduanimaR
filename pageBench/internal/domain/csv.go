package domain

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ファイル名定数
const (
	FileRegistry = "0a_registry.csv"         // PDFリスト（source/ スキャンで自動生成）
	FileQAPairs  = "0b_qa_pairs.csv"         // QAペア（generate コマンドが生成）
	FileEvaluate = "0c_evaluation.csv"       // 評価結果（eval コマンドが生成）
	FileReport   = "0d_evaluation_report.md" // 評価レポート（eval/report コマンドが生成）
)

// ScanAndWriteRegistry は domainDir/source/ 内の PDF ファイルをスキャンして
// 0a_registry.csv を生成する。既存エントリは保持し、新規 PDF のみ追記する。
func ScanAndWriteRegistry(domainDir string) ([]CorpusEntry, error) {
	sourceDir := filepath.Join(domainDir, "source")
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("source/ ディレクトリが見つかりません: %s", sourceDir)
	}

	// 既存レジストリを読み込んで既登録のファイル名を把握
	existing := map[string]CorpusEntry{}
	registryPath := filepath.Join(domainDir, FileRegistry)
	if prev, err := LoadCorpus(domainDir); err == nil {
		for _, e := range prev {
			existing[e.FileName] = e
		}
	}

	// source/ をスキャンして PDF ファイルを収集
	dirEntries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("source/ の読み込み失敗: %w", err)
	}

	var entries []CorpusEntry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".pdf") {
			continue
		}
		if prev, ok := existing[name]; ok {
			// 既存エントリを保持（タイトル等ユーザー編集を守る）
			// ただし file_pages / file_url が未設定の場合は補完する
			if prev.FilePages == "" {
				prev.FilePages = fmt.Sprintf("%d", countPDFPages(filepath.Join(sourceDir, name)))
			}
			if prev.FileURL == "" {
				prev.FileURL = "NULL"
			}
			entries = append(entries, prev)
		} else {
			// 新規エントリ: ファイル名（拡張子なし）をタイトルの初期値に
			title := strings.TrimSuffix(name, filepath.Ext(name))
			pages := countPDFPages(filepath.Join(sourceDir, name))
			entries = append(entries, CorpusEntry{
				FileName:  name,
				FileTitle: title,
				FileURL:   "NULL",
				FilePages: fmt.Sprintf("%d", pages),
			})
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("source/ に PDF ファイルが見つかりません: %s", sourceDir)
	}

	// 0a_registry.csv に書き出す
	f, err := os.Create(registryPath)
	if err != nil {
		return nil, fmt.Errorf("0a_registry.csv の作成失敗: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.Write([]string{"file_name", "file_title", "file_url", "file_pages"})
	for _, e := range entries {
		fileURL := e.FileURL
		if fileURL == "" {
			fileURL = "NULL"
		}
		_ = w.Write([]string{e.FileName, e.FileTitle, fileURL, e.FilePages})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("0a_registry.csv の書き込み失敗: %w", err)
	}

	return entries, nil
}

// LoadCorpus は 0a_registry.csv を読み込んで CorpusEntry スライスを返す。
func LoadCorpus(domainDir string) ([]CorpusEntry, error) {
	path := filepath.Join(domainDir, FileRegistry)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s を開けません: %w", FileRegistry, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s の読み込みに失敗: %w", FileRegistry, err)
	}
	if len(records) < 2 {
		return nil, nil
	}

	header := records[0]
	idx := headerIndex(header)

	var entries []CorpusEntry
	for _, row := range records[1:] {
		e := CorpusEntry{
			FileName:  getField(row, idx, "file_name"),
			FileTitle: getField(row, idx, "file_title"),
			FileURL:   getField(row, idx, "file_url"),
			FilePages: getField(row, idx, "file_pages"),
		}
		if e.FileName != "" {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

// LoadQAPairs は 0b_qa_pairs.csv を読み込んで QAPair スライスを返す。
func LoadQAPairs(domainDir string) ([]QAPair, error) {
	path := filepath.Join(domainDir, FileQAPairs)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s を開けません: %w", FileQAPairs, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s の読み込みに失敗: %w", FileQAPairs, err)
	}
	if len(records) < 2 {
		return nil, nil
	}

	header := records[0]
	idx := headerIndex(header)

	var pairs []QAPair
	for _, row := range records[1:] {
		qt := getField(row, idx, "question_type")
		if qt == "" {
			qt = "answerable" // 後方互換: 空の場合は answerable とみなす
		}
		pairs = append(pairs, QAPair{
			QID:          getField(row, idx, "q_id"),
			Question:     getField(row, idx, "question"),
			QuestionType: qt,
			RefAnswer:    getField(row, idx, "ref_answer"),
			RefEvidence:  getField(row, idx, "ref_evidence"),
			RefFile:      getField(row, idx, "ref_file"),
			RefPage:      getField(row, idx, "ref_page"),
		})
	}
	return pairs, nil
}

// WriteQAPairs は QAPair スライスを 0b_qa_pairs.csv に書き出す。
func WriteQAPairs(domainDir string, pairs []QAPair) error {
	path := filepath.Join(domainDir, FileQAPairs)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%s を作成できません: %w", FileQAPairs, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.Write([]string{"q_id", "question", "question_type", "ref_answer", "ref_evidence", "ref_file", "ref_page"})
	for _, p := range pairs {
		qt := p.QuestionType
		if qt == "" {
			qt = "answerable"
		}
		_ = w.Write([]string{p.QID, p.Question, qt, p.RefAnswer, p.RefEvidence, p.RefFile, p.RefPage})
	}
	w.Flush()
	return w.Error()
}

// LoadEvaluate は 0c_evaluation.csv を読み込んで EvalRecord スライスを返す。
func LoadEvaluate(domainDir string) ([]EvalRecord, error) {
	path := filepath.Join(domainDir, FileEvaluate)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s を開けません: %w", FileEvaluate, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s の読み込みに失敗: %w", FileEvaluate, err)
	}
	if len(records) < 2 {
		return nil, nil
	}

	header := records[0]
	idx := headerIndex(header)

	var out []EvalRecord
	for _, row := range records[1:] {
		var rougeL float64
		fmt.Sscanf(getField(row, idx, "rouge_l"), "%f", &rougeL)
		var em int
		fmt.Sscanf(getField(row, idx, "exact_match"), "%d", &em)
		var latencyMS int
		fmt.Sscanf(getField(row, idx, "latency_ms"), "%d", &latencyMS)
		var loopCount int
		fmt.Sscanf(getField(row, idx, "loop_count"), "%d", &loopCount)
		var librarianMS int
		fmt.Sscanf(getField(row, idx, "librarian_ms"), "%d", &librarianMS)
		var answerGenMS int
		fmt.Sscanf(getField(row, idx, "answer_gen_ms"), "%d", &answerGenMS)
		var fileHit int
		fmt.Sscanf(getField(row, idx, "file_hit"), "%d", &fileHit)
		var pageHit int
		fmt.Sscanf(getField(row, idx, "page_hit"), "%d", &pageHit)
		var refFileFound int
		fmt.Sscanf(getField(row, idx, "ref_file_found"), "%d", &refFileFound)
		var refPageFound int
		fmt.Sscanf(getField(row, idx, "ref_page_found"), "%d", &refPageFound)
		var refFilePageFound int
		fmt.Sscanf(getField(row, idx, "ref_file_page_found"), "%d", &refFilePageFound)
		var ragRefused int
		fmt.Sscanf(getField(row, idx, "rag_refused"), "%d", &ragRefused)

		qt := getField(row, idx, "question_type")
		if qt == "" {
			qt = "answerable" // 後方互換
		}

		out = append(out, EvalRecord{
			QID:                getField(row, idx, "q_id"),
			Domain:             getField(row, idx, "domain"),
			Question:           getField(row, idx, "question"),
			QuestionType:       qt,
			RefAnswer:          getField(row, idx, "ref_answer"),
			RefEvidence:        getField(row, idx, "ref_evidence"),
			RefFile:            getField(row, idx, "ref_file"),
			RefPage:            getField(row, idx, "ref_page"),
			RagAnswer:          getField(row, idx, "rag_answer"),
			RagSources:         getField(row, idx, "rag_sources"),
			FileHit:            fileHit,
			PageHit:            pageHit,
			RetrievedFilePages: getField(row, idx, "retrieved_file_pages"),
			RefFileFound:       refFileFound,
			RefPageFound:       refPageFound,
			RefFilePageFound:   refFilePageFound,
			RougeL:             rougeL,
			ExactMatch:         em,
			RagRefused:         ragRefused,
			LatencyMS:          latencyMS,
			LoopCount:          loopCount,
			LibrarianMS:        librarianMS,
			AnswerGenMS:        answerGenMS,
			JudgeAccuracy:      getField(row, idx, "judge_accuracy"),
			JudgeFaithful:      getField(row, idx, "judge_faithfulness"),
			JudgeComplete:      getField(row, idx, "judge_completeness"),
			JudgeOverall:       getField(row, idx, "judge_overall"),
			JudgeHallucination: getField(row, idx, "judge_hallucination"),
			JudgeReasoning:     getField(row, idx, "judge_reasoning"),
		})
	}
	return out, nil
}

// WriteEvaluate は EvalRecord スライスを 0c_evaluation.csv に書き出す。
func WriteEvaluate(domainDir string, records []EvalRecord) error {
	path := filepath.Join(domainDir, FileEvaluate)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%s を作成できません: %w", FileEvaluate, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.Write([]string{
		// 識別情報
		"q_id", "domain",
		// QA内容（0b 共通）
		"question", "question_type", "ref_answer", "ref_evidence", "ref_file", "ref_page",
		// RAG回答
		"rag_answer", "rag_sources",
		// LLMなし自動メトリクス
		"file_hit", "page_hit", "retrieved_file_pages", "ref_file_found", "ref_page_found", "ref_file_page_found", "rouge_l", "exact_match", "rag_refused", "latency_ms", "loop_count", "librarian_ms", "answer_gen_ms",
		// LLM-as-Judge
		"judge_accuracy", "judge_faithfulness", "judge_completeness", "judge_overall", "judge_hallucination", "judge_reasoning",
	})
	for _, r := range records {
		qt := r.QuestionType
		if qt == "" {
			qt = "answerable"
		}
		_ = w.Write([]string{
			r.QID, r.Domain,
			r.Question, qt, r.RefAnswer, r.RefEvidence, r.RefFile, r.RefPage,
			r.RagAnswer, r.RagSources,
			fmt.Sprintf("%d", r.FileHit), fmt.Sprintf("%d", r.PageHit),
			r.RetrievedFilePages,
			fmt.Sprintf("%d", r.RefFileFound), fmt.Sprintf("%d", r.RefPageFound), fmt.Sprintf("%d", r.RefFilePageFound),
			fmt.Sprintf("%.4f", r.RougeL), fmt.Sprintf("%d", r.ExactMatch),
			fmt.Sprintf("%d", r.RagRefused),
			fmt.Sprintf("%d", r.LatencyMS),
			fmt.Sprintf("%d", r.LoopCount),
			fmt.Sprintf("%d", r.LibrarianMS),
			fmt.Sprintf("%d", r.AnswerGenMS),
			r.JudgeAccuracy, r.JudgeFaithful, r.JudgeComplete, r.JudgeOverall, r.JudgeHallucination, r.JudgeReasoning,
		})
	}
	w.Flush()
	return w.Error()
}

// ─── 後方互換エイリアス（旧名称 → 新名称へのブリッジ） ────────────────

// LoadTestset は LoadQAPairs の後方互換エイリアス。
// Deprecated: LoadQAPairs を使用してください。
func LoadTestset(domainDir string) ([]QAPair, error) { return LoadQAPairs(domainDir) }

// LoadResultsCSV は LoadEvaluate の後方互換エイリアス。
// Deprecated: LoadEvaluate を使用してください。
func LoadResultsCSV(domainDir string) ([]EvalRecord, error) { return LoadEvaluate(domainDir) }

// WriteResults は WriteEvaluate の後方互換エイリアス。
// Deprecated: WriteEvaluate を使用してください。
func WriteResults(domainDir string, records []EvalRecord) error {
	return WriteEvaluate(domainDir, records)
}

// ─── PDF ユーティリティ ────────────────────────────────────

// countPDFPages は PDF ファイルのページ数を返す。
// 外部ライブラリを使わず、PDF バイナリ中の "/Type /Page" エントリ数をカウントする。
// 標準的な PDF（学術論文・財務資料・政策文書など）で正確に動作する。
func countPDFPages(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	// PDF ページオブジェクトは "<</Type /Page ..." の形式で記録される
	n := bytes.Count(data, []byte("/Type /Page"))
	if n == 0 {
		// スペースなしの形式も考慮
		n = bytes.Count(data, []byte("/Type/Page"))
	}
	return n
}

// ─── ヘルパー ──────────────────────────────────────────────

func headerIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[h] = i
	}
	return m
}

func getField(row []string, idx map[string]int, key string) string {
	i, ok := idx[key]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}
