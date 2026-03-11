// Package metrics provides evaluation metric types and utilities for pageBench.
package metrics

import "math"

// ─── 難易度別統計 ─────────────────────────────────────────

// DifficultyStats は難易度別の評価集計値（answerable 回答済みレコードのみ）。
type DifficultyStats struct {
	Count                 int     // 対象レコード数
	AvgSemanticSimilarity float64 // Semantic Similarity 平均（gemini-embedding-001）
	AvgJudgeOverall       float64 // Judge Overall 平均（-1 = N/A）
	FileHitRate           float64 // File Hit Rate（-1 = N/A）
	AvgLoopCount          float64 // ループ回数平均
	AvgLatencyMS          int     // レイテンシ平均（ms）
}

// ─── サマリー ─────────────────────────────────────────────

// Summary は評価結果の集計値。
type Summary struct {
	// ── answerable 質問のメトリクス ──────────────────────────
	Total                 int
	Answered              int
	AvgSemanticSimilarity float64 // Semantic Similarity 平均（gemini-embedding-001 SEMANTIC_SIMILARITY）
	AvgJudgeOverall       float64 // -1 = N/A（Judge スキップ）
	AvgJudgeAccuracy      float64 // -1 = N/A（Accuracy 軸）
	AvgJudgeFaithful      float64 // -1 = N/A（Faithfulness 軸）
	AvgJudgeComplete      float64 // -1 = N/A（Completeness 軸）
	AvgLatencyMS          int
	AvgLoopCount          float64
	AvgLibrarianMS        int
	AvgAnswerGenMS        int
	FileHitRate           float64 // file_hit の平均（0.0〜1.0）、-1 = N/A
	PageHitRate           float64 // page_hit の平均（0.0〜1.0）、-1 = N/A
	RefFileFoundRate      float64 // ref_file_found の平均（0.0〜1.0）、-1 = N/A
	RefPageFoundRate      float64 // ref_page_found の平均（0.0〜1.0）、-1 = N/A
	RefFilePageFoundRate  float64 // ref_file_page_found の平均（0.0〜1.0）、-1 = N/A

	// ── 難易度別集計 ──────────────────────────────────────────
	// key: "simple" | "reasoning" | "multi_chunk" | etc.
	DifficultyBreakdown map[string]DifficultyStats

	// ── unanswerable 質問のメトリクス（ハルシネーション検査） ──
	TotalUnanswerable        int     // unanswerable 質問の総数
	HallucinationRefRate     float64 // answerability=="unanswerable" の割合（高いほど良い）。-1 = N/A
	AvgHallucinationScore    float64 // judge_hallucination の平均（1〜5、低いほど良い）。-1 = N/A
	AvgUnanswerableLatencyMS int     // unanswerable 質問の平均レイテンシ (ms)
	AvgUnanswerableLoopCount float64 // unanswerable 質問の平均ループ回数
}

// ─── ユーティリティ ───────────────────────────────────────

func roundFloat(f float64, decimals int) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return f
	}
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(f*pow+0.5)) / pow
}

// RoundFloat は外部から丸め演算を利用するためのエクスポート関数。
func RoundFloat(f float64, decimals int) float64 {
	return roundFloat(f, decimals)
}
