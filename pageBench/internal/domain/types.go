// Package domain defines core data types for pageBench.
package domain

// CorpusEntry はドキュメントコーパスの 1 エントリ（0a_registry.csv の 1 行）。
type CorpusEntry struct {
	FileName  string // file_name : PDFファイル名（0b.RefFile と結合可能）
	FileTitle string // file_title: ドキュメントタイトル
	FileURL   string // file_url  : 取得元URL
	FilePages string // file_pages: 総ページ数
}

// QAPair はテストセットの 1 件（0b_qa_pairs.csv の 1 行）。
type QAPair struct {
	QID          string // q_id          : 質問ID
	Question     string // question      : 質問文
	QuestionType string // question_type : "answerable" | "unanswerable"
	RefAnswer    string // ref_answer    : 正解の解答（unanswerable の場合は空）
	RefEvidence  string // ref_evidence  : 正解の根拠テキスト（unanswerable の場合は空）
	RefFile      string // ref_file      : 関連ファイル名（0a.FileName と結合可能）
	RefPage      string // ref_page      : 正解ページ番号（unanswerable の場合は空）
}

// EvalRecord は 1 問の評価結果（0c_evaluation.csv の 1 行）。
type EvalRecord struct {
	// ── 識別情報 ──────────────────────────────────────────
	QID    string // q_id  : 質問ID
	Domain string // domain: ドメイン名

	// ── QA内容（0b_qa_pairs と共通カラム） ─────────────────
	Question     string // question      : 質問文
	QuestionType string // question_type : "answerable" | "unanswerable"
	RefAnswer    string // ref_answer    : 正解の解答（unanswerable の場合は空）
	RefEvidence  string // ref_evidence  : 正解の根拠テキスト（unanswerable の場合は空）
	RefFile      string // ref_file      : 関連ファイル名
	RefPage      string // ref_page      : 正解ページ番号（unanswerable の場合は空）

	// ── RAG システムの回答 ─────────────────────────────────
	RagAnswer  string // rag_answer : RAG の回答テキスト
	RagSources string // rag_sources: RAG が参照したソース（JSON 文字列）

	// ── LLM なし自動メトリクス ─────────────────────────────
	FileHit            int     // file_hit   : rag_sources に ref_file が含まれるか（0/1）
	PageHit            int     // page_hit   : rag_sources に ref_file + ref_page の組が含まれるか（0/1）
	RetrievedFilePages string  // retrieved_file_pages: 取得ソース一覧（JSON文字列）
	RefFileFound       int     // ref_file_found: ref_file が取得ソースに含まれるか（0/1）
	RefPageFound       int     // ref_page_found: ref_page が取得ソースに含まれるか（0/1）
	RefFilePageFound   int     // ref_file_page_found: ref_file + ref_page 厳密一致（0/1）
	RougeL             float64 // rouge_l    : ROUGE-L スコア（0.0〜1.0）
	ExactMatch         int     // exact_match: 完全一致（0/1）
	RagRefused         int     // rag_refused : RAGが「情報なし」と回答したか（0/1）。unanswerable 専用
	LatencyMS          int     // latency_ms : レスポンスタイム（ms）
	LoopCount          int     // loop_count: Librarian の検索ループ回数
	LibrarianMS        int     // librarian_ms: Librarian フェーズ所要時間（ms）
	AnswerGenMS        int     // answer_gen_ms: 回答生成フェーズ所要時間（ms）

	// ── LLM-as-Judge スコア ────────────────────────────────
	JudgeAccuracy      string // judge_accuracy    : 正確性（1〜5）。answerable 専用
	JudgeFaithful      string // judge_faithfulness: 忠実性（1〜5）。answerable 専用
	JudgeComplete      string // judge_completeness: 完全性（1〜5）。answerable 専用
	JudgeOverall       string // judge_overall     : 総合評価（1〜5）
	JudgeHallucination string // judge_hallucination: ハルシネーション度（1〜5）。unanswerable 専用
	JudgeReasoning     string // judge_reasoning   : 採点根拠（テキスト）
}
