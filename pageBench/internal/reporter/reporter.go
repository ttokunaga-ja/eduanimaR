// Package reporter generates Markdown reports and terminal summaries.
package reporter

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ttokunaga-ja/pagebench/internal/domain"
	"github.com/ttokunaga-ja/pagebench/internal/metrics"
)

// ComputeSummary は EvalRecord スライスから Summary を計算する。
func ComputeSummary(records []domain.EvalRecord) metrics.Summary {
	// answerable / unanswerable に分離
	var answerableRecs, unanswerableRecs []domain.EvalRecord
	for _, r := range records {
		if r.QuestionType == "unanswerable" {
			unanswerableRecs = append(unanswerableRecs, r)
		} else {
			answerableRecs = append(answerableRecs, r)
		}
	}

	// RAG 回答が存在する answerable レコードのみで指標計算
	var answered []domain.EvalRecord
	for _, r := range answerableRecs {
		if r.RagAnswer != "" {
			answered = append(answered, r)
		}
	}

	var rougeSum, emSum float64
	var judgeSum float64
	judgeCount := 0
	var latencySum int
	var fileHitSum, pageHitSum int

	for _, r := range answered {
		rougeSum += r.RougeL
		emSum += float64(r.ExactMatch)
		latencySum += r.LatencyMS
		fileHitSum += r.FileHit
		pageHitSum += r.PageHit
	}
	for _, r := range answerableRecs {
		if r.JudgeOverall != "" {
			var v int
			if _, err := fmt.Sscanf(r.JudgeOverall, "%d", &v); err == nil {
				judgeSum += float64(v)
				judgeCount++
			}
		}
	}

	s := metrics.Summary{
		Total:                 len(records),
		Answered:              len(answered),
		TotalUnanswerable:     len(unanswerableRecs),
		FileHitRate:           -1,
		PageHitRate:           -1,
		HallucinationRefRate:  -1,
		AvgHallucinationScore: -1,
	}
	if len(answered) > 0 {
		s.AvgRougeL = roundFloat(rougeSum/float64(len(answered)), 4)
		s.AvgExactMatch = roundFloat(emSum/float64(len(answered)), 4)
		s.AvgLatencyMS = latencySum / len(answered)
		s.FileHitRate = roundFloat(float64(fileHitSum)/float64(len(answered)), 4)
		s.PageHitRate = roundFloat(float64(pageHitSum)/float64(len(answered)), 4)
	}
	if judgeCount > 0 {
		s.AvgJudgeOverall = roundFloat(judgeSum/float64(judgeCount), 2)
	} else {
		s.AvgJudgeOverall = -1
	}

	// Hallucination metrics (unanswerable 問題群)
	if len(unanswerableRecs) > 0 {
		var refusedSum int
		var hallucSum float64
		hallucCount := 0
		for _, r := range unanswerableRecs {
			refusedSum += r.RagRefused
			if r.JudgeHallucination != "" {
				var v int
				if _, err := fmt.Sscanf(r.JudgeHallucination, "%d", &v); err == nil {
					hallucSum += float64(v)
					hallucCount++
				}
			}
		}
		s.HallucinationRefRate = roundFloat(float64(refusedSum)/float64(len(unanswerableRecs)), 4)
		if hallucCount > 0 {
			s.AvgHallucinationScore = roundFloat(hallucSum/float64(hallucCount), 2)
		}
	}

	return s
}

// WriteMarkdownReport は Markdown レポートを 0d_evaluation_report.md に出力する。
func WriteMarkdownReport(records []domain.EvalRecord, domainDir, domainName, backendInfo string, s metrics.Summary) error {
	outPath := filepath.Join(domainDir, domain.FileReport)
	now := time.Now().Format("2006-01-02 15:04:05")

	var sb strings.Builder

	// ヘッダー
	judgeDisplay := formatJudge(s.AvgJudgeOverall)
	fmt.Fprintf(&sb, "# pageBench Evaluation Report\n\n")
	fmt.Fprintf(&sb, "| Item | Value |\n|------|-------|\n")
	fmt.Fprintf(&sb, "| **Domain** | `%s` |\n", domainName)
	fmt.Fprintf(&sb, "| **Backend** | `%s` |\n", backendInfo)
	fmt.Fprintf(&sb, "| **Date** | %s |\n", now)
	fmt.Fprintf(&sb, "| **Total QA** | %d |\n", s.Total)
	fmt.Fprintf(&sb, "| **Answered** | %d |\n", s.Answered)
	fmt.Fprintf(&sb, "| **Unanswerable** | %d |\n\n", s.TotalUnanswerable)

	// スコアサマリー
	fmt.Fprintf(&sb, "## 📊 Score Summary\n\n")
	fmt.Fprintf(&sb, "| Metric | Score |\n|--------|-------|\n")
	fmt.Fprintf(&sb, "| **File Hit Rate** | %s |\n", formatRate(s.FileHitRate))
	fmt.Fprintf(&sb, "| **Page Hit Rate** | %s |\n", formatRate(s.PageHitRate))
	fmt.Fprintf(&sb, "| **ROUGE-L** (avg) | %.4f |\n", s.AvgRougeL)
	fmt.Fprintf(&sb, "| **Exact Match** (avg) | %.4f |\n", s.AvgExactMatch)
	fmt.Fprintf(&sb, "| **Judge Overall** (avg) | %s |\n", judgeDisplay)
	fmt.Fprintf(&sb, "| **Latency** (avg) | %d ms |\n\n", s.AvgLatencyMS)

	// ROUGE-L 分布
	var rougeScores []float64
	for _, r := range records {
		if r.RagAnswer != "" && r.QuestionType != "unanswerable" {
			rougeScores = append(rougeScores, r.RougeL)
		}
	}
	if len(rougeScores) > 0 {
		buckets := []struct {
			label  string
			lo, hi float64
		}{
			{"0.8–1.0", 0.8, 1.01},
			{"0.6–0.8", 0.6, 0.8},
			{"0.4–0.6", 0.4, 0.6},
			{"0.2–0.4", 0.2, 0.4},
			{"0.0–0.2", 0.0, 0.2},
		}
		fmt.Fprintf(&sb, "## 📈 ROUGE-L Distribution\n\n")
		fmt.Fprintf(&sb, "| Range | Count | Bar |\n|-------|-------|-----|\n")
		total := float64(len(rougeScores))
		for _, bkt := range buckets {
			count := 0
			for _, sc := range rougeScores {
				if sc >= bkt.lo && sc < bkt.hi {
					count++
				}
			}
			bar := strings.Repeat("█", int(float64(count)/total*20))
			fmt.Fprintf(&sb, "| %s | %d | %s |\n", bkt.label, count, bar)
		}
		fmt.Fprintf(&sb, "\n")
	}

	// Judge 分布 (answerable のみ)
	var judgeScores []int
	for _, r := range records {
		if r.QuestionType != "unanswerable" && r.JudgeOverall != "" {
			var v int
			if _, err := fmt.Sscanf(r.JudgeOverall, "%d", &v); err == nil {
				judgeScores = append(judgeScores, v)
			}
		}
	}
	if len(judgeScores) > 0 {
		fmt.Fprintf(&sb, "## ⚖️ Judge Overall Distribution\n\n")
		fmt.Fprintf(&sb, "| Score | Count | Bar |\n|-------|-------|-----|\n")
		total := float64(len(judgeScores))
		for sc := 1; sc <= 5; sc++ {
			count := 0
			for _, v := range judgeScores {
				if v == sc {
					count++
				}
			}
			bar := strings.Repeat("█", int(float64(count)/total*20))
			fmt.Fprintf(&sb, "| %d / 5 | %d | %s |\n", sc, count, bar)
		}
		fmt.Fprintf(&sb, "\n")
	}

	// Hallucination Check セクション (unanswerable 問題群)
	var unanswerableRecs []domain.EvalRecord
	for _, r := range records {
		if r.QuestionType == "unanswerable" {
			unanswerableRecs = append(unanswerableRecs, r)
		}
	}
	if len(unanswerableRecs) > 0 {
		fmt.Fprintf(&sb, "## 🧪 Hallucination Check (Unanswerable Questions)\n\n")
		fmt.Fprintf(&sb, "> 回答不能問題 (%d 件) に対する RAG の幻覚抑制チェック\n\n", len(unanswerableRecs))
		fmt.Fprintf(&sb, "| Metric | Value |\n|--------|-------|\n")
		fmt.Fprintf(&sb, "| **Unanswerable Count** | %d |\n", s.TotalUnanswerable)
		fmt.Fprintf(&sb, "| **Hallucination Refusal Rate** | %s |\n", formatRate(s.HallucinationRefRate))
		fmt.Fprintf(&sb, "| **Avg Hallucination Score** (1=OK, 5=worst) | %s |\n\n", formatJudge(s.AvgHallucinationScore))

		// Judge Hallucination 分布
		var hallucScores []int
		for _, r := range unanswerableRecs {
			if r.JudgeHallucination != "" {
				var v int
				if _, err := fmt.Sscanf(r.JudgeHallucination, "%d", &v); err == nil {
					hallucScores = append(hallucScores, v)
				}
			}
		}
		if len(hallucScores) > 0 {
			fmt.Fprintf(&sb, "### Hallucination Score Distribution\n\n")
			fmt.Fprintf(&sb, "| Score | Meaning | Count | Bar |\n|-------|---------|-------|-----|\n")
			meanings := map[int]string{
				1: "正確に拒否",
				2: "ほぼ拒否",
				3: "部分的幻覚",
				4: "重度の幻覚",
				5: "完全な幻覚",
			}
			total := float64(len(hallucScores))
			for sc := 1; sc <= 5; sc++ {
				count := 0
				for _, v := range hallucScores {
					if v == sc {
						count++
					}
				}
				bar := strings.Repeat("█", int(float64(count)/total*20))
				fmt.Fprintf(&sb, "| %d | %s | %d | %s |\n", sc, meanings[sc], count, bar)
			}
			fmt.Fprintf(&sb, "\n")
		}

		// 幻覚発生ケース (rag_refused=0 → 幻覚あり)
		var hallucCases []domain.EvalRecord
		for _, r := range unanswerableRecs {
			if r.RagRefused == 0 {
				hallucCases = append(hallucCases, r)
			}
		}
		if len(hallucCases) > 0 {
			fmt.Fprintf(&sb, "### ⚠️ Hallucination Cases (RAG failed to refuse)\n\n")
			limit := 3
			if len(hallucCases) < limit {
				limit = len(hallucCases)
			}
			for i, r := range hallucCases[:limit] {
				fmt.Fprintf(&sb, "**%d. [%s]**\n- Question: %s\n- rag_answer: %s\n- judge_hallucination: %s\n- reasoning: %s\n\n",
					i+1, r.QID,
					truncate(r.Question, 80),
					truncate(r.RagAnswer, 120),
					r.JudgeHallucination,
					truncate(r.JudgeReasoning, 200))
			}
		} else {
			fmt.Fprintf(&sb, "✅ すべての回答不能問題で RAG が正しく拒否しました。\n\n")
		}
	}

	// Top 3 / Bottom 3 by ROUGE-L (answerable のみ)
	var answered []domain.EvalRecord
	for _, r := range records {
		if r.RagAnswer != "" && r.QuestionType != "unanswerable" {
			answered = append(answered, r)
		}
	}
	if len(answered) > 0 {
		sort.Slice(answered, func(i, j int) bool { return answered[i].RougeL > answered[j].RougeL })
		fmt.Fprintf(&sb, "## 🏆 Top 3 by ROUGE-L\n\n")
		for i, r := range answered {
			if i >= 3 {
				break
			}
			fmt.Fprintf(&sb, "**%d. ROUGE-L=%.4f**\n- Question: %s\n- ref_answer: %s\n- rag_answer: %s\n\n",
				i+1, r.RougeL, truncate(r.Question, 80), truncate(r.RefAnswer, 120), truncate(r.RagAnswer, 120))
		}

		sort.Slice(answered, func(i, j int) bool { return answered[i].RougeL < answered[j].RougeL })
		fmt.Fprintf(&sb, "## 🔍 Bottom 3 by ROUGE-L (for improvement)\n\n")
		for i, r := range answered {
			if i >= 3 {
				break
			}
			fmt.Fprintf(&sb, "**%d. ROUGE-L=%.4f**\n- Question: %s\n- ref_answer: %s\n- rag_answer: %s\n\n",
				i+1, r.RougeL, truncate(r.Question, 80), truncate(r.RefAnswer, 120), truncate(r.RagAnswer, 120))
		}
	}

	fmt.Fprintf(&sb, "---\n*Generated by [pageBench](https://github.com/ttokunaga-ja/pagebench)*\n")

	return os.WriteFile(outPath, []byte(sb.String()), 0o644)
}

// PrintSummary はサマリーをターミナルに出力する。
func PrintSummary(s metrics.Summary, domainName string) {
	judgeDisplay := formatJudge(s.AvgJudgeOverall)
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("  📊 評価サマリー: %s\n", domainName)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  総問題数          : %d\n", s.Total)
	fmt.Printf("  回答あり (answerable): %d\n", s.Answered)
	fmt.Printf("  回答不能 (unanswerable): %d\n", s.TotalUnanswerable)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("  File Hit Rate     : %s\n", formatRate(s.FileHitRate))
	fmt.Printf("  Page Hit Rate     : %s\n", formatRate(s.PageHitRate))
	fmt.Printf("  平均 ROUGE-L      : %.4f\n", s.AvgRougeL)
	fmt.Printf("  平均 Exact Match  : %.4f\n", s.AvgExactMatch)
	fmt.Printf("  平均 Judge Overall: %s\n", judgeDisplay)
	fmt.Printf("  平均レイテンシ    : %d ms\n", s.AvgLatencyMS)
	if s.TotalUnanswerable > 0 {
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("  🧪 幻覚拒否率     : %s\n", formatRate(s.HallucinationRefRate))
		fmt.Printf("  🧪 平均幻覚スコア : %s  (1=OK, 5=最悪)\n", formatJudge(s.AvgHallucinationScore))
	}
	fmt.Println(strings.Repeat("=", 60))
}

func formatJudge(v float64) string {
	if v < 0 {
		return "N/A (Judge skipped)"
	}
	return fmt.Sprintf("%.2f / 5", v)
}

func formatRate(v float64) string {
	if v < 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.2f%%", v*100)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func roundFloat(f float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(f*pow) / pow
}
