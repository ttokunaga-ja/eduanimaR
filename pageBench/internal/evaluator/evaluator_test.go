package evaluator

import (
	"testing"

	"github.com/ttokunaga-ja/pagebench/internal/backend"
)

func TestParseReferencePages(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "single", in: "3", want: []string{"3"}},
		{name: "multi separators", in: "3, 5、7;9/11", want: []string{"3", "5", "7", "9", "11"}},
		{name: "dedupe and trim", in: " 3 , 3 , 5 ", want: []string{"3", "5"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseReferencePages(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len mismatch: got=%v want=%v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("index %d: got=%q want=%q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsNearPage(t *testing.T) {
	if !isNearPage("10", "11", 1) {
		t.Fatalf("expected near page match")
	}
	if isNearPage("10", "12", 1) {
		t.Fatalf("expected non-near page mismatch")
	}
	if isNearPage("x", "12", 1) {
		t.Fatalf("expected parse failure to return false")
	}
}

func TestIsReferencePageMatch(t *testing.T) {
	refPages := []string{"3", "5"}
	if !isReferencePageMatch("3", refPages) {
		t.Fatalf("exact page should match")
	}
	if !isReferencePageMatch("4", refPages) {
		t.Fatalf("near page (+/-1) should match")
	}
	if isReferencePageMatch("8", refPages) {
		t.Fatalf("far page should not match")
	}
	if isReferencePageMatch("", refPages) {
		t.Fatalf("empty retrieved page should not match")
	}
}

func TestEvaluateRetrievedSources_MultiRefPagesAndNearPage(t *testing.T) {
	sources := []backend.Source{
		{Name: "PaperA.pdf", Page: "4"}, // ref "3" に対して ±1 で一致
		{Name: "PaperB.pdf", Page: "10"},
	}

	_, refFileFound, refPageFound, refFilePageFound := evaluateRetrievedSources(sources, "papera.pdf", "3, 5")

	if refFileFound != 1 {
		t.Fatalf("refFileFound: got=%d want=1", refFileFound)
	}
	if refPageFound != 1 {
		t.Fatalf("refPageFound: got=%d want=1", refPageFound)
	}
	if refFilePageFound != 1 {
		t.Fatalf("refFilePageFound: got=%d want=1", refFilePageFound)
	}
}
