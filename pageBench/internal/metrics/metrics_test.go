package metrics

import (
	"testing"
)

// ─── RougeL ──────────────────────────────────────────────

func TestRougeL_ExactMatch(t *testing.T) {
	score := RougeL("hello world", "hello world")
	if score != 1.0 {
		t.Errorf("完全一致は 1.0 期待, got %.4f", score)
	}
}

func TestRougeL_EmptyPrediction(t *testing.T) {
	score := RougeL("", "reference answer")
	if score != 0.0 {
		t.Errorf("空の prediction は 0.0 期待, got %.4f", score)
	}
}

func TestRougeL_EmptyReference(t *testing.T) {
	score := RougeL("prediction", "")
	if score != 0.0 {
		t.Errorf("空の reference は 0.0 期待, got %.4f", score)
	}
}

func TestRougeL_PartialMatch(t *testing.T) {
	score := RougeL("the cat sat on the mat", "the cat sat on the mat and the dog")
	if score <= 0.0 || score >= 1.0 {
		t.Errorf("部分一致は 0 < score < 1 期待, got %.4f", score)
	}
}

func TestRougeL_NoMatch(t *testing.T) {
	score := RougeL("foo bar baz", "qux quux corge")
	if score != 0.0 {
		t.Errorf("不一致は 0.0 期待, got %.4f", score)
	}
}

func TestRougeL_JapaneseCJK(t *testing.T) {
	// 日本語テキスト（文字レベルで処理される）
	pred := "東京都は日本の首都です"
	ref := "東京は日本の首都です"
	score := RougeL(pred, ref)
	if score <= 0.0 || score > 1.0 {
		t.Errorf("日本語部分一致は 0 < score <= 1 期待, got %.4f", score)
	}
}

func TestRougeL_JapaneseExact(t *testing.T) {
	text := "人工知能の研究は急速に発展しています"
	score := RougeL(text, text)
	if score != 1.0 {
		t.Errorf("日本語完全一致は 1.0 期待, got %.4f", score)
	}
}

func TestRougeL_Symmetry(t *testing.T) {
	// ROUGE-L は prediction/reference の順序に依存するため非対称
	// ただし値は必ず [0,1] の範囲内
	score1 := RougeL("a b c d", "a b c d e f")
	score2 := RougeL("a b c d e f", "a b c d")
	if score1 < 0 || score1 > 1 || score2 < 0 || score2 > 1 {
		t.Errorf("スコアは [0,1] 範囲外: score1=%.4f, score2=%.4f", score1, score2)
	}
}

// ─── ExactMatch ──────────────────────────────────────────

func TestExactMatch_Equal(t *testing.T) {
	if ExactMatch("hello", "hello") != 1 {
		t.Error("完全一致は 1 期待")
	}
}

func TestExactMatch_CaseInsensitive(t *testing.T) {
	if ExactMatch("Hello World", "hello world") != 1 {
		t.Error("大文字小文字無視の一致は 1 期待")
	}
}

func TestExactMatch_WhitespaceTrimed(t *testing.T) {
	if ExactMatch("  hello  ", "hello") != 1 {
		t.Error("前後空白を除去した一致は 1 期待")
	}
}

func TestExactMatch_Different(t *testing.T) {
	if ExactMatch("hello", "world") != 0 {
		t.Error("不一致は 0 期待")
	}
}

func TestExactMatch_Empty(t *testing.T) {
	if ExactMatch("", "something") != 0 {
		t.Error("空 prediction は 0 期待")
	}
	if ExactMatch("something", "") != 0 {
		t.Error("空 reference は 0 期待")
	}
}

func TestExactMatch_Japanese(t *testing.T) {
	if ExactMatch("東京都", "東京都") != 1 {
		t.Error("日本語完全一致は 1 期待")
	}
	if ExactMatch("東京都", "東京") != 0 {
		t.Error("日本語不一致は 0 期待")
	}
}

// ─── isCJKDominant ───────────────────────────────────────

func TestIsCJKDominant_Japanese(t *testing.T) {
	if !isCJKDominant("東京都は日本の首都です", 0.3) {
		t.Error("日本語主体のテキストは CJK dominant 期待")
	}
}

func TestIsCJKDominant_English(t *testing.T) {
	if isCJKDominant("This is an English sentence", 0.3) {
		t.Error("英語テキストは CJK dominant でない期待")
	}
}

func TestIsCJKDominant_Empty(t *testing.T) {
	if isCJKDominant("", 0.3) {
		t.Error("空文字は CJK dominant でない期待")
	}
}

// ─── lcsLength ───────────────────────────────────────────

func TestLCSLength_Full(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"a", "b", "c"}
	if lcsLength(a, b) != 3 {
		t.Error("完全一致の LCS は 3 期待")
	}
}

func TestLCSLength_Empty(t *testing.T) {
	if lcsLength([]string{}, []string{"a"}) != 0 {
		t.Error("空スライスの LCS は 0 期待")
	}
}

func TestLCSLength_Classic(t *testing.T) {
	// LCS("ABCBDAB", "BDCAB") = 4
	a := []string{"A", "B", "C", "B", "D", "A", "B"}
	b := []string{"B", "D", "C", "A", "B"}
	if got := lcsLength(a, b); got != 4 {
		t.Errorf("LCS は 4 期待, got %d", got)
	}
}
