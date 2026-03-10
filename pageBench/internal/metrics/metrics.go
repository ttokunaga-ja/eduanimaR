// Package metrics provides ROUGE-L and Exact Match evaluation metrics.
// 外部ライブラリ不要の純粋 Go 実装。
package metrics

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ─── ROUGE-L ─────────────────────────────────────────────

// isCJKDominant は CJK 文字（漢字・かな等）の割合が閾値を超えるか判定する。
func isCJKDominant(text string, threshold float64) bool {
	if text == "" {
		return false
	}
	total := utf8.RuneCountInString(text)
	cjk := 0
	for _, r := range text {
		if (r >= 0x3000 && r <= 0x9FFF) || (r >= 0xF900 && r <= 0xFAFF) {
			cjk++
		}
	}
	return float64(cjk)/float64(total) > threshold
}

// lcsLength は 1D DP で LCS（最長共通部分列）の長さを計算する。
func lcsLength(a, b []string) int {
	m, n := len(a), len(b)
	prev := make([]int, n+1)
	for i := 0; i < m; i++ {
		curr := make([]int, n+1)
		for j := 0; j < n; j++ {
			if a[i] == b[j] {
				curr[j+1] = prev[j] + 1
			} else if prev[j+1] > curr[j] {
				curr[j+1] = prev[j+1]
			} else {
				curr[j+1] = curr[j]
			}
		}
		prev = curr
	}
	return prev[n]
}

// tokenizeWithMode は文字/単語レベルを明示して分割する。
func tokenizeWithMode(text string, charLevel bool) []string {
	if charLevel {
		runes := []rune(text)
		tokens := make([]string, len(runes))
		for i, r := range runes {
			tokens[i] = string(r)
		}
		return tokens
	}
	return strings.Fields(strings.ToLower(text))
}

// RougeL は ROUGE-L F1 スコアを計算して返す（範囲: 0.0〜1.0）。
func RougeL(prediction, reference string) float64 {
	if prediction == "" || reference == "" {
		return 0.0
	}
	// 片側だけ CJK 主体でもトークン化方式を統一して語彙一致を正しく拾う。
	charLevel := isCJKDominant(prediction, 0.3) || isCJKDominant(reference, 0.3)
	predTokens := tokenizeWithMode(prediction, charLevel)
	refTokens := tokenizeWithMode(reference, charLevel)
	if len(predTokens) == 0 || len(refTokens) == 0 {
		return 0.0
	}

	lcs := lcsLength(refTokens, predTokens)
	precision := float64(lcs) / float64(len(predTokens))
	recall := float64(lcs) / float64(len(refTokens))

	if precision+recall == 0 {
		return 0.0
	}
	f1 := 2.0 * precision * recall / (precision + recall)
	// 小数点 4 桁に丸める
	return roundFloat(f1, 4)
}

// ─── Exact Match ─────────────────────────────────────────

// normalize は Exact Match 用の正規化（NFKC + 小文字 + trim）を行う。
func normalize(text string) string {
	// Unicode 正規化（Go の unicode パッケージは NFKC 相当のうち基本的な部分をカバー）
	var sb strings.Builder
	for _, r := range text {
		sb.WriteRune(unicode.ToLower(r))
	}
	return strings.TrimSpace(sb.String())
}

// ExactMatch は完全一致なら 1、それ以外は 0 を返す。
func ExactMatch(prediction, reference string) int {
	if prediction == "" || reference == "" {
		return 0
	}
	if normalize(prediction) == normalize(reference) {
		return 1
	}
	return 0
}

// ─── サマリー ─────────────────────────────────────────────

// Summary は評価結果の集計値。
type Summary struct {
	// ── answerable 質問のメトリクス ──────────────────────────
	Total                int
	Answered             int
	AvgRougeL            float64
	AvgExactMatch        float64
	AvgJudgeOverall      float64 // -1 = N/A（Judge スキップ）
	AvgLatencyMS         int
	AvgLoopCount         float64
	AvgLibrarianMS       int
	AvgAnswerGenMS       int
	FileHitRate          float64 // file_hit の平均（0.0〜1.0）、-1 = N/A（rag_sources なし）
	PageHitRate          float64 // page_hit の平均（0.0〜1.0）、-1 = N/A（rag_sources なし）
	RefFileFoundRate     float64 // ref_file_found の平均（0.0〜1.0）、-1 = N/A
	RefPageFoundRate     float64 // ref_page_found の平均（0.0〜1.0）、-1 = N/A
	RefFilePageFoundRate float64 // ref_file_page_found の平均（0.0〜1.0）、-1 = N/A

	// ── unanswerable 質問のメトリクス（ハルシネーション検査） ──
	TotalUnanswerable     int     // unanswerable 質問の総数
	HallucinationRefRate  float64 // rag_refused=1 の割合（高いほど良い）。-1 = N/A
	AvgHallucinationScore float64 // judge_hallucination の平均（1〜5、低いほど良い）。-1 = N/A
}

// ─── ユーティリティ ───────────────────────────────────────

func roundFloat(f float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(f*pow+0.5)) / pow
}
