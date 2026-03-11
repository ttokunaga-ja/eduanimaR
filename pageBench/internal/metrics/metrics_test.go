package metrics

import (
	"math"
	"testing"
)

// ─── CosineSimilarity (gemini/client.go の関数は別パッケージなので、
//     metrics パッケージでは RoundFloat のみテスト) ──────────────────

func TestRoundFloat_FourDecimals(t *testing.T) {
	got := RoundFloat(0.12345678, 4)
	if got != 0.1235 {
		t.Errorf("RoundFloat(0.12345678, 4): want 0.1235, got %.8f", got)
	}
}

func TestRoundFloat_Zero(t *testing.T) {
	got := RoundFloat(0.0, 4)
	if got != 0.0 {
		t.Errorf("RoundFloat(0.0, 4): want 0.0, got %f", got)
	}
}

func TestRoundFloat_One(t *testing.T) {
	got := RoundFloat(1.0, 4)
	if got != 1.0 {
		t.Errorf("RoundFloat(1.0, 4): want 1.0, got %f", got)
	}
}

func TestRoundFloat_HalfUp(t *testing.T) {
	got := RoundFloat(0.5555, 3)
	if got != 0.556 {
		t.Errorf("RoundFloat(0.5555, 3): want 0.556, got %.8f", got)
	}
}

// ─── Summary 初期値チェック ──────────────────────────────────────────

func TestSummaryDefaults(t *testing.T) {
	s := Summary{
		AvgJudgeOverall:       -1,
		AvgJudgeAccuracy:      -1,
		AvgJudgeFaithful:      -1,
		AvgJudgeComplete:      -1,
		HallucinationRefRate:  -1,
		AvgHallucinationScore: -1,
		FileHitRate:           -1,
		PageHitRate:           -1,
	}
	if s.AvgSemanticSimilarity != 0.0 {
		t.Errorf("初期 AvgSemanticSimilarity は 0.0 期待, got %f", s.AvgSemanticSimilarity)
	}
	if s.AvgJudgeOverall != -1 {
		t.Errorf("初期 AvgJudgeOverall は -1 期待, got %f", s.AvgJudgeOverall)
	}
}

// ─── DifficultyStats チェック ────────────────────────────────────────

func TestDifficultyStatsFields(t *testing.T) {
	d := DifficultyStats{
		Count:                 10,
		AvgSemanticSimilarity: 0.85,
		AvgJudgeOverall:       4.2,
		FileHitRate:           0.9,
		AvgLoopCount:          1.5,
		AvgLatencyMS:          1200,
	}
	if d.AvgSemanticSimilarity != 0.85 {
		t.Errorf("AvgSemanticSimilarity: want 0.85, got %f", d.AvgSemanticSimilarity)
	}
	if d.Count != 10 {
		t.Errorf("Count: want 10, got %d", d.Count)
	}
}

// ─── roundFloat (内部関数) 境界値テスト ─────────────────────────────

func TestRoundFloat_NaN(t *testing.T) {
	// NaN は丸めても NaN
	got := RoundFloat(math.NaN(), 4)
	if !math.IsNaN(got) {
		t.Errorf("RoundFloat(NaN): NaN 期待")
	}
}
