// Package judge provides LLM-as-Judge scoring using Gemini with thinking support.
package judge

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"

	geminiutil "github.com/ttokunaga-ja/pagebench/internal/gemini"
)

// Result は 1 問分の Judge スコア（answerable 質問用）。
type Result struct {
	Accuracy     int    `json:"accuracy"`
	Faithfulness int    `json:"faithfulness"`
	Completeness int    `json:"completeness"`
	Overall      int    `json:"overall"`
	Reasoning    string `json:"reasoning"`
}

// ResultUnanswerable は 1 問分の Judge スコア（unanswerable 質問用）。
type ResultUnanswerable struct {
	Hallucination int    `json:"hallucination"` // ハルシネーション度（1=なし, 5=完全な作り話）
	Overall       int    `json:"overall"`       // 総合評価（1=回答拒否で正解, 5=ハルシネーション満載）
	Reasoning     string `json:"reasoning"`     // 採点根拠
}

// judgeResponseSchema は Gemini Structured Output 用のスキーマ定義。
var judgeResponseSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"accuracy":     {Type: genai.TypeInteger, Description: "正確性 (1-5)"},
		"faithfulness": {Type: genai.TypeInteger, Description: "忠実性 (1-5)"},
		"completeness": {Type: genai.TypeInteger, Description: "完全性 (1-5)"},
		"overall":      {Type: genai.TypeInteger, Description: "総合評価 (1-5)"},
		"reasoning":    {Type: genai.TypeString, Description: "採点の根拠（簡潔に）"},
	},
	Required: []string{"accuracy", "faithfulness", "completeness", "overall", "reasoning"},
}

const judgePromptTmpl = `あなたは厳格な RAG システム評価者です。
以下の情報をもとに、システムの回答を多角的に評価してください。

【質問】
%s

【参照解答（正解）】
%s

【根拠テキスト（ソース文書からの抜粋）】
%s

【システムの回答】
%s

以下の基準で各項目を 1〜5 点で採点してください:
- accuracy（正確性）: 参照解答と意味的に一致しているか（1=全く違う, 5=完全に一致）
- faithfulness（忠実性）: 根拠テキストから逸脱していないか（1=大幅に逸脱, 5=完全に根拠に基づく）
- completeness（完全性）: 質問に対して必要な情報が揃っているか（1=不十分, 5=完全）
- overall（総合評価）: 上記を総合した評価
- reasoning: 採点の根拠を日本語で 1〜3 文で説明してください`

// Judge は Gemini を使った LLM-as-Judge 採点器。
type Judge struct {
	client        *genai.Client
	model         string
	thinkingLevel string
}

// New は Judge インスタンスを生成する。
// thinkingLevel: "minimal" | "low" | "medium" | "high"
func New(ctx context.Context, model, thinkingLevel string) (*Judge, error) {
	client, err := geminiutil.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Judge{
		client:        client,
		model:         model,
		thinkingLevel: thinkingLevel,
	}, nil
}

// Score は 1 問分の採点を実行して Result を返す。
func (j *Judge) Score(ctx context.Context, question, referenceAnswer, systemAnswer, evidenceText string) (*Result, error) {
	if evidenceText == "" {
		evidenceText = "（なし）"
	}
	if systemAnswer == "" {
		systemAnswer = "（回答なし）"
	}

	prompt := fmt.Sprintf(judgePromptTmpl,
		question,
		truncate(referenceAnswer, 2000),
		truncate(evidenceText, 1500),
		truncate(systemAnswer, 3000),
	)

	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   judgeResponseSchema,
		Temperature:      ptr(float32(0.0)), // 採点は決定論的に固定
		Seed:             ptr(int32(42)),    // 乱数シード固定（再現性確保）
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: geminiutil.ThinkingLevel(j.thinkingLevel),
		},
	}

	resp, err := j.client.Models.GenerateContent(ctx, j.model,
		genai.Text(prompt), cfg)
	if err != nil {
		return nil, fmt.Errorf("Gemini Judge エラー: %w", err)
	}

	text := resp.Text()
	var result Result
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("Judge レスポンスのデコード失敗: %w (raw: %s)", err, text)
	}
	return &result, nil
}

// ── unanswerable 専用スキーマ・プロンプト ──────────────────────────────

// judgeUnanswerableSchema は unanswerable 質問の採点スキーマ。
var judgeUnanswerableSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"hallucination": {Type: genai.TypeInteger, Description: "ハルシネーション度 (1=なし/正しく回答拒否, 5=完全な作り話)"},
		"overall":       {Type: genai.TypeInteger, Description: "総合評価 (1=正しく拒否, 5=完全なハルシネーション)"},
		"reasoning":     {Type: genai.TypeString, Description: "採点の根拠（簡潔に）"},
	},
	Required: []string{"hallucination", "overall", "reasoning"},
}

const judgeUnanswerablePromptTmpl = `あなたは RAG システムのハルシネーション検査評価者です。
以下の質問は文書に答えが含まれない「回答不能質問」です。
RAG システムがこの質問に対して正しく「情報がない」と答えたか、それとも架空の情報を作り上げたかを評価してください。

【質問（回答不能質問）】
%s

【システムの回答】
%s

以下の基準で採点してください:
- hallucination（ハルシネーション度）:
  1 = 正しく「情報なし」「回答できません」と答えた（ハルシネーションなし）
  2 = 不確かさを示しつつ、一部に誤った情報を含む
  3 = 中程度のハルシネーション（一部事実・一部作り話が混在）
  4 = 大部分が架空の情報（信頼性が低い）
  5 = 完全な作り話（文書に存在しない情報を断言）
- overall: hallucination と同じ基準での総合評価
- reasoning: 採点根拠を日本語で 1〜2 文で説明してください`

// ScoreUnanswerable は unanswerable 質問の採点を実行して ResultUnanswerable を返す。
// RAG が回答拒否をしているかどうか（hallucination 抑制の有無）を評価する。
func (j *Judge) ScoreUnanswerable(ctx context.Context, question, systemAnswer string) (*ResultUnanswerable, error) {
	if systemAnswer == "" {
		systemAnswer = "（回答なし）"
	}

	prompt := fmt.Sprintf(judgeUnanswerablePromptTmpl,
		question,
		truncate(systemAnswer, 3000),
	)

	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   judgeUnanswerableSchema,
		Temperature:      ptr(float32(0.0)), // 採点は決定論的に固定
		Seed:             ptr(int32(42)),    // 乱数シード固定（再現性確保）
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: geminiutil.ThinkingLevel(j.thinkingLevel),
		},
	}

	resp, err := j.client.Models.GenerateContent(ctx, j.model,
		genai.Text(prompt), cfg)
	if err != nil {
		return nil, fmt.Errorf("Gemini Judge (unanswerable) エラー: %w", err)
	}

	text := resp.Text()
	var result ResultUnanswerable
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("Judge (unanswerable) レスポンスのデコード失敗: %w (raw: %s)", err, text)
	}
	return &result, nil
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func ptr[T any](v T) *T { return &v }
