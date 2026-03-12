package llm

import "testing"

func TestNormalizeAnswerability(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		answer   string
		evidence int
		expected string
	}{
		{
			name:     "unknown label with no evidence falls back to unanswerable",
			raw:      "unknown",
			answer:   "",
			evidence: 0,
			expected: "unanswerable",
		},
		{
			name:     "answerable downgraded by refusal style answer in english",
			raw:      "answerable",
			answer:   "The provided materials do not contain this information.",
			evidence: 2,
			expected: "unanswerable",
		},
		{
			name:     "answerable downgraded by refusal style answer in japanese",
			raw:      "answerable",
			answer:   "資料には記載がないため回答できません。",
			evidence: 1,
			expected: "unanswerable",
		},
		{
			name:     "partial with no evidence and refusal answer downgraded",
			raw:      "partial",
			answer:   "insufficient information to answer",
			evidence: 0,
			expected: "unanswerable",
		},
		{
			name:     "valid answerable keeps label when not refusal",
			raw:      "answerable",
			answer:   "The method is described in Section 3.",
			evidence: 2,
			expected: "answerable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAnswerability(tt.raw, tt.answer, tt.evidence)
			if got != tt.expected {
				t.Fatalf("normalizeAnswerability() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsRefusalStyleAnswer(t *testing.T) {
	if !isRefusalStyleAnswer("not mentioned in the provided document") {
		t.Fatalf("expected english refusal to be detected")
	}
	if !isRefusalStyleAnswer("情報が不足しているため判断できません") {
		t.Fatalf("expected japanese refusal to be detected")
	}
	if isRefusalStyleAnswer("The paper reports 92%% accuracy in Table 4.") {
		t.Fatalf("expected normal factual answer to be non-refusal")
	}
}
