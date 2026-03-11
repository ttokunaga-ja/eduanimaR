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

// difficultyOrder は難易度表示順。
var difficultyOrder = []string{"simple", "reasoning", "multi_chunk"}

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

	var semSimSum float64
	var latencySum, loopCountSum, librarianMSSum, answerGenMSSum int
	var fileHitSum, pageHitSum int
	var refFileFoundSum, refPageFoundSum, refFilePageFoundSum int

	for _, r := range answered {
		semSimSum += r.SemanticSimilarity
		latencySum += r.LatencyMS
		loopCountSum += r.LoopCount
		librarianMSSum += r.LibrarianMS
		answerGenMSSum += r.AnswerGenMS
		fileHitSum += r.FileHit
		pageHitSum += r.PageHit
		refFileFoundSum += r.RefFileFound
		refPageFoundSum += r.RefPageFound
		refFilePageFoundSum += r.RefFilePageFound
	}

	// Judge 3 軸（answerable 全体 — 回答なし含む）
	var judgeSum, judgeAccSum, judgeFaithSum, judgeComplSum float64
	judgeCount, judgeAccCount, judgeFaithCount, judgeComplCount := 0, 0, 0, 0
	for _, r := range answerableRecs {
		if r.JudgeOverall != "" {
			var v int
			if _, err := fmt.Sscanf(r.JudgeOverall, "%d", &v); err == nil {
				judgeSum += float64(v)
				judgeCount++
			}
		}
		if r.JudgeAccuracy != "" {
			var v int
			if _, err := fmt.Sscanf(r.JudgeAccuracy, "%d", &v); err == nil {
				judgeAccSum += float64(v)
				judgeAccCount++
			}
		}
		if r.JudgeFaithful != "" {
			var v int
			if _, err := fmt.Sscanf(r.JudgeFaithful, "%d", &v); err == nil {
				judgeFaithSum += float64(v)
				judgeFaithCount++
			}
		}
		if r.JudgeComplete != "" {
			var v int
			if _, err := fmt.Sscanf(r.JudgeComplete, "%d", &v); err == nil {
				judgeComplSum += float64(v)
				judgeComplCount++
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
		AvgJudgeOverall:       -1,
		AvgJudgeAccuracy:      -1,
		AvgJudgeFaithful:      -1,
		AvgJudgeComplete:      -1,
	}

	if len(answered) > 0 {
		s.AvgSemanticSimilarity = roundFloat(semSimSum/float64(len(answered)), 4)
		s.AvgLatencyMS = latencySum / len(answered)
		s.AvgLoopCount = roundFloat(float64(loopCountSum)/float64(len(answered)), 2)
		s.AvgLibrarianMS = librarianMSSum / len(answered)
		s.AvgAnswerGenMS = answerGenMSSum / len(answered)
		s.FileHitRate = roundFloat(float64(fileHitSum)/float64(len(answered)), 4)
		s.PageHitRate = roundFloat(float64(pageHitSum)/float64(len(answered)), 4)
		s.RefFileFoundRate = roundFloat(float64(refFileFoundSum)/float64(len(answered)), 4)
		s.RefPageFoundRate = roundFloat(float64(refPageFoundSum)/float64(len(answered)), 4)
		s.RefFilePageFoundRate = roundFloat(float64(refFilePageFoundSum)/float64(len(answered)), 4)
	}
	if judgeCount > 0 {
		s.AvgJudgeOverall = roundFloat(judgeSum/float64(judgeCount), 2)
	}
	if judgeAccCount > 0 {
		s.AvgJudgeAccuracy = roundFloat(judgeAccSum/float64(judgeAccCount), 2)
	}
	if judgeFaithCount > 0 {
		s.AvgJudgeFaithful = roundFloat(judgeFaithSum/float64(judgeFaithCount), 2)
	}
	if judgeComplCount > 0 {
		s.AvgJudgeComplete = roundFloat(judgeComplSum/float64(judgeComplCount), 2)
	}

	// ─── 難易度別集計（answerable 回答済みのみ） ───────────────
	diffGroups := map[string][]domain.EvalRecord{}
	for _, r := range answered {
		diff := r.Difficulty
		if diff == "" || diff == "N/A" {
			continue
		}
		diffGroups[diff] = append(diffGroups[diff], r)
	}
	if len(diffGroups) > 0 {
		breakdown := map[string]metrics.DifficultyStats{}
		for diff, recs := range diffGroups {
			var semSimGrpSum float64
			var judgeGrpSum float64
			judgeGrpCount := 0
			var fileHitGrpSum, latencyGrpSum, loopGrpSum int
			for _, r := range recs {
				semSimGrpSum += r.SemanticSimilarity
				latencyGrpSum += r.LatencyMS
				loopGrpSum += r.LoopCount
				fileHitGrpSum += r.FileHit
				if r.JudgeOverall != "" {
					var v int
					if _, err := fmt.Sscanf(r.JudgeOverall, "%d", &v); err == nil {
						judgeGrpSum += float64(v)
						judgeGrpCount++
					}
				}
			}
			n := len(recs)
			avgJudge := -1.0
			if judgeGrpCount > 0 {
				avgJudge = roundFloat(judgeGrpSum/float64(judgeGrpCount), 2)
			}
			breakdown[diff] = metrics.DifficultyStats{
				Count:                 n,
				AvgSemanticSimilarity: roundFloat(semSimGrpSum/float64(n), 4),
				AvgJudgeOverall:       avgJudge,
				FileHitRate:           roundFloat(float64(fileHitGrpSum)/float64(n), 4),
				AvgLoopCount:          roundFloat(float64(loopGrpSum)/float64(n), 2),
				AvgLatencyMS:          latencyGrpSum / n,
			}
		}
		s.DifficultyBreakdown = breakdown
	}

	// ─── Hallucination metrics（unanswerable 問題群） ──────────
	if len(unanswerableRecs) > 0 {
		// answerability == "unanswerable" の場合に正しく拒否できたとみなす
		var refusedSum int
		var hallucSum float64
		hallucCount := 0
		var unLatencySum, unLoopSum int
		for _, r := range unanswerableRecs {
			if r.Answerability == "unanswerable" {
				refusedSum++
			}
			unLatencySum += r.LatencyMS
			unLoopSum += r.LoopCount
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
		s.AvgUnanswerableLatencyMS = unLatencySum / len(unanswerableRecs)
		s.AvgUnanswerableLoopCount = roundFloat(float64(unLoopSum)/float64(len(unanswerableRecs)), 2)
	}

	return s
}

// WriteMarkdownReport は Markdown レポートを 0d_evaluation_report.md に出力する。
func WriteMarkdownReport(records []domain.EvalRecord, domainDir, domainName, backendInfo string, s metrics.Summary) error {
	outPath := filepath.Join(domainDir, domain.FileReport)
	now := time.Now().Format("2006-01-02 15:04:05")

	var sb strings.Builder

	// ─── ヘッダー ──────────────────────────────────────────────
	fmt.Fprintf(&sb, "# pageBench Evaluation Report\n\n")
	fmt.Fprintf(&sb, "| Item | Value |\n|------|-------|\n")
	fmt.Fprintf(&sb, "| **Domain** | `%s` |\n", domainName)
	fmt.Fprintf(&sb, "| **Backend** | `%s` |\n", backendInfo)
	fmt.Fprintf(&sb, "| **Date** | %s |\n", now)
	fmt.Fprintf(&sb, "| **Total QA** | %d |\n", s.Total)
	fmt.Fprintf(&sb, "| **Answered (answerable)** | %d |\n", s.Answered)
	fmt.Fprintf(&sb, "| **Unanswerable** | %d |\n\n", s.TotalUnanswerable)

	// ─── Executive Summary ─────────────────────────────────────
	fmt.Fprintf(&sb, "## 🔖 Executive Summary\n\n")
	fmt.Fprintf(&sb, "%s\n\n", buildExecutiveSummary(s))

	// ─── スコアサマリー（answerable のみ） ─────────────────────
	judgeDisplay := formatJudge(s.AvgJudgeOverall)
	fmt.Fprintf(&sb, "## 📊 Score Summary\n\n")
	fmt.Fprintf(&sb, "> 📌 以下の指標は **answerable（回答可能）** 問題のみを対象に集計しています。\n\n")
	fmt.Fprintf(&sb, "| Metric | Score |\n|--------|-------|\n")
	fmt.Fprintf(&sb, "| **File Hit Rate** | %s |\n", formatRate(s.FileHitRate))
	fmt.Fprintf(&sb, "| **Page Hit Rate** | %s |\n", formatRate(s.PageHitRate))
	fmt.Fprintf(&sb, "| **Ref File Found Rate** | %s |\n", formatRate(s.RefFileFoundRate))
	fmt.Fprintf(&sb, "| **Ref Page Found Rate** | %s |\n", formatRate(s.RefPageFoundRate))
	fmt.Fprintf(&sb, "| **Ref File+Page Found Rate** | %s |\n", formatRate(s.RefFilePageFoundRate))
	fmt.Fprintf(&sb, "| **Semantic Similarity** (avg) | %.4f |\n", s.AvgSemanticSimilarity)
	fmt.Fprintf(&sb, "| **Judge Accuracy** (avg) | %s |\n", formatJudge(s.AvgJudgeAccuracy))
	fmt.Fprintf(&sb, "| **Judge Faithfulness** (avg) | %s |\n", formatJudge(s.AvgJudgeFaithful))
	fmt.Fprintf(&sb, "| **Judge Completeness** (avg) | %s |\n", formatJudge(s.AvgJudgeComplete))
	fmt.Fprintf(&sb, "| **Judge Overall** (avg) | %s |\n", judgeDisplay)
	fmt.Fprintf(&sb, "| **Latency** (avg) | %d ms |\n\n", s.AvgLatencyMS)

	// ─── フェーズ別レイテンシ ──────────────────────────────────
	fmt.Fprintf(&sb, "## ⏱️ Phase Latency Breakdown\n\n")
	fmt.Fprintf(&sb, "> 📌 answerable 回答済みレコードのみ対象\n\n")
	fmt.Fprintf(&sb, "| Metric | Value |\n|--------|-------|\n")
	fmt.Fprintf(&sb, "| **Loop Count** (avg) | %.2f |\n", s.AvgLoopCount)
	fmt.Fprintf(&sb, "| **Librarian Phase** (avg) | %d ms |\n", s.AvgLibrarianMS)
	fmt.Fprintf(&sb, "| **Answer Generation** (avg) | %d ms |\n\n", s.AvgAnswerGenMS)

	// ─── 難易度別ブレークダウン ────────────────────────────────
	if len(s.DifficultyBreakdown) > 0 {
		fmt.Fprintf(&sb, "## 📋 Difficulty Breakdown\n\n")
		fmt.Fprintf(&sb, "> answerable 回答済みレコードのみ集計。難易度別の弱点を把握できます。\n\n")
		fmt.Fprintf(&sb, "| Difficulty | N | SemanticSim | Judge | FileHitRate | LoopCount(avg) | Latency(avg) |\n")
		fmt.Fprintf(&sb, "|:-----------|--:|------------:|------:|------------:|---------------:|-------------:|\n")

		// 定義順で出力
		for _, diff := range difficultyOrder {
			if stats, ok := s.DifficultyBreakdown[diff]; ok && stats.Count > 0 {
				judgeStr := "N/A"
				if stats.AvgJudgeOverall >= 0 {
					judgeStr = fmt.Sprintf("%.2f", stats.AvgJudgeOverall)
				}
				fmt.Fprintf(&sb, "| %-11s | %d | %.4f | %s | %s | %.2f | %d ms |\n",
					diff,
					stats.Count,
					stats.AvgSemanticSimilarity,
					judgeStr,
					formatRate(stats.FileHitRate),
					stats.AvgLoopCount,
					stats.AvgLatencyMS,
				)
			}
		}
		// 定義外の難易度があれば末尾に追加
		for diff, stats := range s.DifficultyBreakdown {
			inOrder := false
			for _, o := range difficultyOrder {
				if o == diff {
					inOrder = true
					break
				}
			}
			if !inOrder && stats.Count > 0 {
				judgeStr := "N/A"
				if stats.AvgJudgeOverall >= 0 {
					judgeStr = fmt.Sprintf("%.2f", stats.AvgJudgeOverall)
				}
				fmt.Fprintf(&sb, "| %-11s | %d | %.4f | %s | %s | %.2f | %d ms |\n",
					diff,
					stats.Count,
					stats.AvgSemanticSimilarity,
					judgeStr,
					formatRate(stats.FileHitRate),
					stats.AvgLoopCount,
					stats.AvgLatencyMS,
				)
			}
		}
		fmt.Fprintf(&sb, "\n")
	}

	// ─── LoopCount 分布 ────────────────────────────────────────
	var ansLoopCounts []int
	for _, r := range records {
		if r.QuestionType != "unanswerable" && r.RagAnswer != "" {
			ansLoopCounts = append(ansLoopCounts, r.LoopCount)
		}
	}
	if len(ansLoopCounts) > 0 {
		count1, count2, count3plus := 0, 0, 0
		for _, lc := range ansLoopCounts {
			switch {
			case lc <= 1:
				count1++
			case lc == 2:
				count2++
			default:
				count3plus++
			}
		}
		total := float64(len(ansLoopCounts))
		fmt.Fprintf(&sb, "## 🔄 Loop Count Distribution\n\n")
		fmt.Fprintf(&sb, "> answerable 回答済みレコードのみ対象。多ループは検索の苦戦を示す可能性があります。\n\n")
		fmt.Fprintf(&sb, "| Loops | Count | Bar |\n|------:|------:|-----|\n")
		fmt.Fprintf(&sb, "| 1     | %d | %s |\n", count1, strings.Repeat("█", barLen(count1, total)))
		fmt.Fprintf(&sb, "| 2     | %d | %s |\n", count2, strings.Repeat("█", barLen(count2, total)))
		fmt.Fprintf(&sb, "| 3+    | %d | %s |\n\n", count3plus, strings.Repeat("█", barLen(count3plus, total)))
	}

	// ─── Semantic Similarity 分布 ──────────────────────────────
	var semSimScores []float64
	for _, r := range records {
		if r.RagAnswer != "" && r.QuestionType != "unanswerable" && r.SemanticSimilarity > 0 {
			semSimScores = append(semSimScores, r.SemanticSimilarity)
		}
	}
	if len(semSimScores) > 0 {
		buckets := []struct {
			label  string
			lo, hi float64
		}{
			{"0.9–1.0", 0.9, 1.01},
			{"0.7–0.9", 0.7, 0.9},
			{"0.5–0.7", 0.5, 0.7},
			{"0.3–0.5", 0.3, 0.5},
			{"0.0–0.3", 0.0, 0.3},
		}
		fmt.Fprintf(&sb, "## 📈 Semantic Similarity Distribution\n\n")
		fmt.Fprintf(&sb, "> gemini-embedding-001 (SEMANTIC_SIMILARITY) によるコサイン類似度分布\n\n")
		fmt.Fprintf(&sb, "| Range | Count | Bar |\n|-------|------:|-----|\n")
		total := float64(len(semSimScores))
		for _, bkt := range buckets {
			count := 0
			for _, sc := range semSimScores {
				if sc >= bkt.lo && sc < bkt.hi {
					count++
				}
			}
			fmt.Fprintf(&sb, "| %s | %d | %s |\n", bkt.label, count, strings.Repeat("█", barLen(count, total)))
		}
		fmt.Fprintf(&sb, "\n")
	}

	// ─── Judge 分布（answerable のみ） ────────────────────────
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
		fmt.Fprintf(&sb, "| Score | Count | Bar |\n|------:|------:|-----|\n")
		total := float64(len(judgeScores))
		for sc := 1; sc <= 5; sc++ {
			count := 0
			for _, v := range judgeScores {
				if v == sc {
					count++
				}
			}
			fmt.Fprintf(&sb, "| %d / 5 | %d | %s |\n", sc, count, strings.Repeat("█", barLen(count, total)))
		}
		fmt.Fprintf(&sb, "\n")
	}

	// ─── Hallucination Check（unanswerable 問題群） ────────────
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
		fmt.Fprintf(&sb, "| **Avg Hallucination Score** (1=OK, 5=worst) | %s |\n", formatJudge(s.AvgHallucinationScore))
		fmt.Fprintf(&sb, "| **Avg Latency (unanswerable)** | %d ms |\n", s.AvgUnanswerableLatencyMS)
		fmt.Fprintf(&sb, "| **Avg Loop Count (unanswerable)** | %.2f |\n\n", s.AvgUnanswerableLoopCount)

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
			fmt.Fprintf(&sb, "| Score | Meaning | Count | Bar |\n|------:|---------|------:|-----|\n")
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
				fmt.Fprintf(&sb, "| %d | %s | %d | %s |\n", sc, meanings[sc], count, strings.Repeat("█", barLen(count, total)))
			}
			fmt.Fprintf(&sb, "\n")
		}

		// 幻覚発生ケース（answerability != "unanswerable" → 正しく拒否できなかった）
		var hallucCases []domain.EvalRecord
		for _, r := range unanswerableRecs {
			if r.Answerability == "answerable" || r.Answerability == "" {
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
				fmt.Fprintf(&sb, "**%d. [%s]**\n- Question: %s\n- rag_answer: %s\n- answerability: %s\n- judge_hallucination: %s\n- reasoning: %s\n\n",
					i+1, r.QID,
					truncate(r.Question, 80),
					truncate(r.RagAnswer, 120),
					r.Answerability,
					r.JudgeHallucination,
					truncate(r.JudgeReasoning, 200))
			}
		} else {
			fmt.Fprintf(&sb, "✅ すべての回答不能問題で RAG が正しく回答不能と判定しました。\n\n")
		}
	}

	// ─── Top 3 / Bottom 3 by Semantic Similarity（answerable のみ） ──
	var answered []domain.EvalRecord
	for _, r := range records {
		if r.RagAnswer != "" && r.QuestionType != "unanswerable" {
			answered = append(answered, r)
		}
	}
	if len(answered) > 0 {
		sort.Slice(answered, func(i, j int) bool {
			return answered[i].SemanticSimilarity > answered[j].SemanticSimilarity
		})
		fmt.Fprintf(&sb, "## 🏆 Top 3 by Semantic Similarity\n\n")
		for i, r := range answered {
			if i >= 3 {
				break
			}
			fmt.Fprintf(&sb, "**%d. SemanticSim=%.4f** `[%s]` `difficulty=%s`\n- Question: %s\n- ref_answer: %s\n- rag_answer: %s\n\n",
				i+1, r.SemanticSimilarity, r.QID, r.Difficulty,
				truncate(r.Question, 80), truncate(r.RefAnswer, 120), truncate(r.RagAnswer, 120))
		}

		sort.Slice(answered, func(i, j int) bool {
			return answered[i].SemanticSimilarity < answered[j].SemanticSimilarity
		})
		fmt.Fprintf(&sb, "## 🔍 Bottom 3 by Semantic Similarity (for improvement)\n\n")
		for i, r := range answered {
			if i >= 3 {
				break
			}
			fmt.Fprintf(&sb, "**%d. SemanticSim=%.4f** `[%s]` `difficulty=%s`\n- Question: %s\n- ref_answer: %s\n- rag_answer: %s\n\n",
				i+1, r.SemanticSimilarity, r.QID, r.Difficulty,
				truncate(r.Question, 80), truncate(r.RefAnswer, 120), truncate(r.RagAnswer, 120))
		}
	}

	// ─── High Loop Count Cases（LoopCount ≥ 2） ───────────────
	var highLoopCases []domain.EvalRecord
	for _, r := range records {
		if r.QuestionType == "unanswerable" {
			continue
		}
		if r.RagAnswer != "" && r.LoopCount >= 2 {
			highLoopCases = append(highLoopCases, r)
		}
	}
	if len(highLoopCases) > 0 {
		// LoopCount 降順、同値は SemanticSimilarity 昇順
		sort.Slice(highLoopCases, func(i, j int) bool {
			if highLoopCases[i].LoopCount != highLoopCases[j].LoopCount {
				return highLoopCases[i].LoopCount > highLoopCases[j].LoopCount
			}
			return highLoopCases[i].SemanticSimilarity < highLoopCases[j].SemanticSimilarity
		})
		fmt.Fprintf(&sb, "## 🔁 High Loop Count Cases (LoopCount ≥ 2)\n\n")
		fmt.Fprintf(&sb, "> 多ループが発生した問題は RAG の検索に苦戦している可能性があります。\n\n")
		limit := 5
		if len(highLoopCases) < limit {
			limit = len(highLoopCases)
		}
		for i, r := range highLoopCases[:limit] {
			fmt.Fprintf(&sb, "**%d. [%s] loops=%d, SemanticSim=%.4f, difficulty=%s**\n- Question: %s\n- rag_answer: %s\n\n",
				i+1, r.QID, r.LoopCount, r.SemanticSimilarity, r.Difficulty,
				truncate(r.Question, 100),
				truncate(r.RagAnswer, 120))
		}
	}

	// ─── Ref File Miss Cases ────────────────────────────────────
	var missedRefFile []domain.EvalRecord
	for _, r := range records {
		if r.QuestionType == "unanswerable" {
			continue
		}
		if strings.TrimSpace(r.RefFile) == "" {
			continue
		}
		if r.RefFileFound == 0 {
			missedRefFile = append(missedRefFile, r)
		}
	}
	if len(missedRefFile) > 0 {
		fmt.Fprintf(&sb, "## 📁 Ref File Miss Cases\n\n")
		fmt.Fprintf(&sb, "参照ファイルを取得できなかったケースを表示します。\n\n")
		limit := 10
		if len(missedRefFile) < limit {
			limit = len(missedRefFile)
		}
		for i, r := range missedRefFile[:limit] {
			fmt.Fprintf(&sb, "**%d. [%s]**\n- Question: %s\n- ref_file/ref_page: %s / %s\n- retrieved_file_pages: %s\n\n",
				i+1,
				r.QID,
				truncate(r.Question, 100),
				r.RefFile,
				r.RefPage,
				truncate(r.RetrievedFilePages, 500),
			)
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
	fmt.Printf("  総問題数              : %d\n", s.Total)
	fmt.Printf("  回答あり (answerable) : %d\n", s.Answered)
	fmt.Printf("  回答不能 (unanswerable): %d\n", s.TotalUnanswerable)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("  [answerable のみ集計]\n")
	fmt.Printf("  File Hit Rate         : %s\n", formatRate(s.FileHitRate))
	fmt.Printf("  Page Hit Rate         : %s\n", formatRate(s.PageHitRate))
	fmt.Printf("  Ref File Found        : %s\n", formatRate(s.RefFileFoundRate))
	fmt.Printf("  Ref Page Found        : %s\n", formatRate(s.RefPageFoundRate))
	fmt.Printf("  Ref File+Page         : %s\n", formatRate(s.RefFilePageFoundRate))
	fmt.Printf("  Semantic Similarity   : %.4f\n", s.AvgSemanticSimilarity)
	fmt.Printf("  平均 Judge Accuracy   : %s\n", formatJudge(s.AvgJudgeAccuracy))
	fmt.Printf("  平均 Judge Faithfulness: %s\n", formatJudge(s.AvgJudgeFaithful))
	fmt.Printf("  平均 Judge Completeness: %s\n", formatJudge(s.AvgJudgeComplete))
	fmt.Printf("  平均 Judge Overall    : %s\n", judgeDisplay)
	fmt.Printf("  平均レイテンシ        : %d ms\n", s.AvgLatencyMS)
	fmt.Printf("  平均ループ回数        : %.2f\n", s.AvgLoopCount)
	fmt.Printf("  Librarian平均         : %d ms\n", s.AvgLibrarianMS)
	fmt.Printf("  回答生成平均          : %d ms\n", s.AvgAnswerGenMS)

	// ── 難易度別ブレークダウン ──────────────────────────────────
	if len(s.DifficultyBreakdown) > 0 {
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("  [難易度別ブレークダウン]\n")
		fmt.Printf("  %-12s  %4s  %12s  %8s  %6s  %6s\n",
			"Difficulty", "N", "SemanticSim", "Judge", "FileHit", "Loop")
		fmt.Printf("  %-12s  %4s  %12s  %8s  %6s  %6s\n",
			"----------", "----", "------------", "--------", "-------", "------")
		for _, diff := range difficultyOrder {
			if stats, ok := s.DifficultyBreakdown[diff]; ok && stats.Count > 0 {
				judgeStr := "  N/A   "
				if stats.AvgJudgeOverall >= 0 {
					judgeStr = fmt.Sprintf("  %.2f   ", stats.AvgJudgeOverall)
				}
				fmt.Printf("  %-12s  %4d  %.4f%s%s  %.2f\n",
					diff,
					stats.Count,
					stats.AvgSemanticSimilarity,
					judgeStr,
					formatRate(stats.FileHitRate),
					stats.AvgLoopCount,
				)
			}
		}
	}

	// ── Unanswerable メトリクス ────────────────────────────────
	if s.TotalUnanswerable > 0 {
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("  🧪 幻覚拒否率           : %s\n", formatRate(s.HallucinationRefRate))
		fmt.Printf("  🧪 平均幻覚スコア       : %s  (1=OK, 5=最悪)\n", formatJudge(s.AvgHallucinationScore))
		fmt.Printf("  🧪 Unanswerable平均レイテンシ: %d ms\n", s.AvgUnanswerableLatencyMS)
		fmt.Printf("  🧪 Unanswerable平均ループ: %.2f\n", s.AvgUnanswerableLoopCount)
	}
	fmt.Println(strings.Repeat("=", 60))
}

// ─── ヘルパー関数 ──────────────────────────────────────────

// buildExecutiveSummary は評価結果の1行総評を生成する。
func buildExecutiveSummary(s metrics.Summary) string {
	if s.Answered == 0 {
		return "> 評価対象レコードがありません。"
	}
	var parts []string

	// Semantic Similarity 評価
	semSimLabel := "改善の余地あり ⚠️"
	if s.AvgSemanticSimilarity >= 0.85 {
		semSimLabel = "良好 ✅"
	} else if s.AvgSemanticSimilarity >= 0.70 {
		semSimLabel = "標準的 ⚡"
	}
	parts = append(parts, fmt.Sprintf("Semantic Similarity 平均 `%.4f` (%s)", s.AvgSemanticSimilarity, semSimLabel))

	// Judge Overall 評価
	if s.AvgJudgeOverall >= 0 {
		judgeLabel := "要改善 ⚠️"
		if s.AvgJudgeOverall >= 4.0 {
			judgeLabel = "高品質 ✅"
		} else if s.AvgJudgeOverall >= 3.0 {
			judgeLabel = "標準的 ⚡"
		}
		parts = append(parts, fmt.Sprintf("Judge Overall `%.2f/5` (%s)", s.AvgJudgeOverall, judgeLabel))
	}

	// Hallucination 評価
	if s.TotalUnanswerable > 0 && s.HallucinationRefRate >= 0 {
		hallucLabel := "要対応 🚨"
		if s.HallucinationRefRate >= 0.9 {
			hallucLabel = "優秀 ✅"
		} else if s.HallucinationRefRate >= 0.7 {
			hallucLabel = "注意が必要 ⚠️"
		}
		parts = append(parts, fmt.Sprintf("幻覚拒否率 `%.1f%%` (%s)", s.HallucinationRefRate*100, hallucLabel))
	}

	// 難易度別の最弱点
	if len(s.DifficultyBreakdown) > 0 {
		weakest := findWeakestDifficulty(s.DifficultyBreakdown)
		if weakest != "" {
			parts = append(parts, fmt.Sprintf("最弱難易度: `%s` (SemanticSim: %.4f)", weakest, s.DifficultyBreakdown[weakest].AvgSemanticSimilarity))
		}
	}

	result := "> 📝 " + strings.Join(parts, " ／ ")
	return result
}

// findWeakestDifficulty は難易度別統計から最も SemanticSimilarity が低い難易度を返す。
func findWeakestDifficulty(breakdown map[string]metrics.DifficultyStats) string {
	weakest := ""
	weakestScore := 2.0 // 最大値より大きい初期値
	for _, d := range difficultyOrder {
		if stats, ok := breakdown[d]; ok && stats.Count > 0 {
			if stats.AvgSemanticSimilarity < weakestScore {
				weakestScore = stats.AvgSemanticSimilarity
				weakest = d
			}
		}
	}
	return weakest
}

// barLen はヒストグラム用のバー長を計算する（最大 20 文字）。
func barLen(count int, total float64) int {
	if total == 0 {
		return 0
	}
	return int(float64(count) / total * 20)
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
