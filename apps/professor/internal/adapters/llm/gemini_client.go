// Package llm は Gemini API を使った LLMClient の実装を提供する。
//
// SDK: google.golang.org/genai（新公式SDK v1.49.0 以降）
//
//	旧: github.com/google/generative-ai-go（非推奨）
//	新: google.golang.org/genai
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/genai"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

// ocrChunk は Gemini Structured Output で返される 1 チャンクの JSON 構造体。
// ResponseSchema で型を強制することでパース失敗を排除する。
type ocrChunk struct {
	Content    string `json:"content"`
	PageNumber int    `json:"page_number"`
}

const (
	// embeddingModel は埋め込みベクトル生成モデル
	// gemini-embedding-001 は MRL（Matryoshka Representation Learning）に対応し、
	// OutputDimensionality で 128〜3072 次元を指定可能。
	embeddingModel = "gemini-embedding-001"

	// embeddingDimensions は pgvector HNSW インデックスの上限（2000次元）以内で
	// 精度と性能を両立する 1536 次元を採用（MRL で 3072→1536 に縮小）。
	embeddingDimensions = int32(1536)

	// ocrSeed は OCRAndChunk の Seed 値。
	// Temperature=0.0 + Seed 固定で Gemini 3 ベストエフォート決定論を実現する。
	ocrSeed = int32(42)
)

// geminiClient は ports.LLMClient の Gemini API 実装。
type geminiClient struct {
	client         *genai.Client
	httpClient     *http.Client
	apiKey         string
	modelIngestion string
	modelAnswer    string
}

// NewGeminiClient は Gemini API クライアントを作成して ports.LLMClient を返す。
// ctx はクライアントのライフタイム用コンテキスト（通常は main の ctx）。
// modelIngestion は OCR / チャンク分割に使用するモデル名（PROFESSOR_MODEL_INGESTION）。
// modelAnswer は回答生成に使用するモデル名（PROFESSOR_MODEL_ANSWER）。
func NewGeminiClient(ctx context.Context, apiKey, modelIngestion, modelAnswer string) (ports.LLMClient, error) {
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: new client: %w", err)
	}
	return &geminiClient{
		client:         c,
		httpClient:     &http.Client{},
		apiKey:         apiKey,
		modelIngestion: modelIngestion,
		modelAnswer:    modelAnswer,
	}, nil
}

// ─── OCRAndChunk ──────────────────────────────────────────────────

// OCRAndChunk は PDF/画像ファイルのバイト列を受け取り、
// Structured Output（JSON スキーマ強制）でテキスト抽出・意味単位チャンク分割を行う。
//
// ResponseMIMEType="application/json" + ResponseSchema で型安全な JSON 配列を出力させることで、
// パース失敗リスクを排除し、page_number を確実に取得できるようにしている。
// Temperature=0.0 + Seed=42 で決定論的（greedy）デコードを強制する。
func (g *geminiClient) OCRAndChunk(ctx context.Context, fileContent []byte, mimeType string) (*ports.OCRResult, error) {
	prompt := `You are an academic document processor.
Extract and structure ALL text content from this document.
Organize content into logical semantic units (paragraphs, sections, slides, exercises, etc.).
Each unit should be self-contained and coherent.
Do NOT include page headers/footers as separate chunks.
Preserve mathematical formulas, code snippets, and tables as text.`

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
				{InlineData: &genai.Blob{MIMEType: mimeType, Data: fileContent}},
			},
			Role: "user",
		},
	}

	config := &genai.GenerateContentConfig{
		// Structured Output 設定
		// ResponseSchema で JSON 配列の型を強制することでパース失敗を排除する。
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"content": {
						Type:        genai.TypeString,
						Description: "Self-contained semantic text chunk. Preserve math formulas, code, and tables as text.",
					},
					"page_number": {
						Type:        genai.TypeInteger,
						Description: "Page number in the PDF (1-indexed). Set to 0 if unknown.",
					},
				},
				Required: []string{"content", "page_number"},
			},
		},
		// Temperature=0.0 + Seed=42: 決定論的デコード（Gemini 3 ベストエフォート）。
		// 定型変換タスクではランダム性は不要。Seed で再現性を担保する。
		Temperature: genai.Ptr(float32(0.0)),
		Seed:        genai.Ptr(ocrSeed),
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.modelIngestion, contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini: ocr generate: %w", err)
	}

	rawJSON := resp.Text()
	if rawJSON == "" {
		return &ports.OCRResult{}, nil
	}

	// JSON 配列を []ocrChunk にアンマーシャルする。
	var ocrChunks []ocrChunk
	if err := json.Unmarshal([]byte(rawJSON), &ocrChunks); err != nil {
		return nil, fmt.Errorf("gemini: ocr unmarshal json: %w (raw=%q)", err, rawJSON)
	}

	chunks := make([]ports.ChunkData, 0, len(ocrChunks))
	for idx, c := range ocrChunks {
		content := strings.TrimSpace(c.Content)
		if content == "" {
			continue
		}
		var pageNum *int
		if c.PageNumber > 0 {
			p := c.PageNumber
			pageNum = &p
		}
		chunks = append(chunks, ports.ChunkData{
			Index:      idx,
			Content:    content,
			PageNumber: pageNum,
		})
	}

	return &ports.OCRResult{Chunks: chunks}, nil
}

// ─── GenerateDocumentEmbedding / GenerateQueryEmbedding ──────────

// embedRequest は Gemini REST API の embedContent リクエスト構造体。
// OutputDimensionality 指定のため REST API を直接呼び出す。
type embedRequest struct {
	Model                string       `json:"model"`
	Content              embedContent `json:"content"`
	TaskType             string       `json:"taskType,omitempty"`
	OutputDimensionality int32        `json:"outputDimensionality,omitempty"`
}

type embedContent struct {
	Parts []embedPart `json:"parts"`
}

type embedPart struct {
	Text string `json:"text"`
}

type embedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

// generateEmbeddingREST は REST API 経由で埋め込みベクトルを生成する共通実装。
// OutputDimensionality=1536（MRL 縮小）指定のため net/http で直接 Gemini REST API（v1beta）を呼び出す。
func (g *geminiClient) generateEmbeddingREST(ctx context.Context, text, taskType string) ([]float32, error) {
	reqBody := embedRequest{
		Model: embeddingModel,
		Content: embedContent{
			Parts: []embedPart{{Text: text}},
		},
		TaskType:             taskType,
		OutputDimensionality: embeddingDimensions,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal embed request: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s",
		embeddingModel, g.apiKey,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: embed http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read embed response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: embed API error %d: %s", resp.StatusCode, string(respBytes))
	}

	var embedResp embedResponse
	if err := json.Unmarshal(respBytes, &embedResp); err != nil {
		return nil, fmt.Errorf("gemini: unmarshal embed response: %w", err)
	}
	return embedResp.Embedding.Values, nil
}

// GenerateDocumentEmbedding はインジェスト用の埋め込みベクトル（1536次元）を生成する。
// TaskType=RETRIEVAL_DOCUMENT を指定して検索対象ドキュメント向けに品質を最適化する。
func (g *geminiClient) GenerateDocumentEmbedding(ctx context.Context, text string) ([]float32, error) {
	vals, err := g.generateEmbeddingREST(ctx, text, "RETRIEVAL_DOCUMENT")
	if err != nil {
		return nil, fmt.Errorf("gemini: document embedding: %w", err)
	}
	return vals, nil
}

// GenerateQueryEmbedding は検索クエリ用の埋め込みベクトル（1536次元）を生成する。
// TaskType=RETRIEVAL_QUERY を指定して検索クエリ向けに品質を最適化する。
func (g *geminiClient) GenerateQueryEmbedding(ctx context.Context, text string) ([]float32, error) {
	vals, err := g.generateEmbeddingREST(ctx, text, "RETRIEVAL_QUERY")
	if err != nil {
		return nil, fmt.Errorf("gemini: query embedding: %w", err)
	}
	return vals, nil
}

// ─── GenerateAnswer ───────────────────────────────────────────────

// GenerateAnswer は選定済みエビデンスと質問から最終回答を生成する（非ストリーミング）。
func (g *geminiClient) GenerateAnswer(ctx context.Context, question string, evidences []string) (string, error) {
	contents := []*genai.Content{
		{
			Parts: []*genai.Part{{Text: buildAnswerPrompt(question, evidences)}},
			Role:  "user",
		},
	}
	config := &genai.GenerateContentConfig{
		// Temperature=0.4: 自然な教師語調と適度な多様性を保持する。
		// 0.0 は決定論的すぎて文章が機械的になり、1.0 はランダムすぎて一貫性が低下する。
		Temperature: genai.Ptr(float32(0.4)),
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.modelAnswer, contents, config)
	if err != nil {
		return "", fmt.Errorf("gemini: generate answer: %w", err)
	}
	return resp.Text(), nil
}

// ─── GenerateAnswerStream ─────────────────────────────────────────

// GenerateAnswerStream は選定済みエビデンスと質問から回答をストリーミング生成する。
// onChunk コールバックに回答テキストを逐次的に渡す。
func (g *geminiClient) GenerateAnswerStream(ctx context.Context, question string, evidences []string, onChunk func(text string) error) error {
	contents := []*genai.Content{
		{
			Parts: []*genai.Part{{Text: buildAnswerPrompt(question, evidences)}},
			Role:  "user",
		},
	}
	config := &genai.GenerateContentConfig{
		// Temperature=0.4: 自然な教師語調と適度な多様性を保持する。
		Temperature: genai.Ptr(float32(0.4)),
	}

	for chunk, err := range g.client.Models.GenerateContentStream(ctx, g.modelAnswer, contents, config) {
		if err != nil {
			return fmt.Errorf("gemini: stream answer: %w", err)
		}
		text := chunk.Text()
		if text != "" {
			if err := onChunk(text); err != nil {
				return err
			}
		}
	}
	return nil
}

// ─── GenerateAnswerStreamWithPDF ─────────────────────────────────

// GenerateAnswerStreamWithPDF は PDF 原本バイト列とエビデンスチャンクを組み合わせて
// 回答をストリーミング生成する。Gemini に PDF を直接渡すことで原本を参照した
// 高精度な回答を実現する。pdfContent が空の場合は GenerateAnswerStream にフォールバックする。
func (g *geminiClient) GenerateAnswerStreamWithPDF(ctx context.Context, question string, evidences []string, pdfContent []byte, mimeType string, onChunk func(text string) error) error {
	if len(pdfContent) == 0 {
		return g.GenerateAnswerStream(ctx, question, evidences, onChunk)
	}

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: buildAnswerPromptForPDF(question, evidences)},
				{InlineData: &genai.Blob{MIMEType: mimeType, Data: pdfContent}},
			},
			Role: "user",
		},
	}
	config := &genai.GenerateContentConfig{
		// Temperature=0.4: 自然な教師語調と適度な多様性を保持する。
		Temperature: genai.Ptr(float32(0.4)),
	}

	for chunk, err := range g.client.Models.GenerateContentStream(ctx, g.modelAnswer, contents, config) {
		if err != nil {
			return fmt.Errorf("gemini: stream answer with pdf: %w", err)
		}
		text := chunk.Text()
		if text != "" {
			if err := onChunk(text); err != nil {
				return err
			}
		}
	}
	return nil
}

// ─── ヘルパー ─────────────────────────────────────────────────────

// buildAnswerPrompt は question と evidences から LLM へのプロンプトを構築する。
func buildAnswerPrompt(question string, evidences []string) string {
	var sb strings.Builder

	sb.WriteString("You are an expert academic tutor. Answer the student's question based ONLY on the provided course materials.\n\n")
	sb.WriteString("## Course Materials\n\n")

	for i, ev := range evidences {
		fmt.Fprintf(&sb, "### Reference %d\n%s\n\n", i+1, ev)
	}

	sb.WriteString("## Student Question\n\n")
	sb.WriteString(question)
	sb.WriteString("\n\n")
	sb.WriteString("## Instructions\n")
	sb.WriteString("- Answer in the same language as the question\n")
	sb.WriteString("- Be concise but thorough\n")
	sb.WriteString("- Cite specific references when relevant (e.g., \"According to Reference 1...\")\n")
	sb.WriteString("- If the provided materials are insufficient to answer, say so clearly\n")
	sb.WriteString("- Do NOT fabricate information not present in the materials\n")

	return sb.String()
}

// buildAnswerPromptForPDF は PDF 原本と evidence ヒントを組み合わせたプロンプトを構築する。
// PDF を Blob として渡すので、プロンプトはヒント（重点箇所）の提供に留める。
func buildAnswerPromptForPDF(question string, evidences []string) string {
	var sb strings.Builder

	sb.WriteString("You are an expert academic tutor. The course material PDF is attached.\n")
	sb.WriteString("Answer the student's question based on the PDF content.\n\n")

	if len(evidences) > 0 {
		sb.WriteString("## Relevant Excerpts (for reference focus)\n\n")
		for i, ev := range evidences {
			fmt.Fprintf(&sb, "### Excerpt %d\n%s\n\n", i+1, ev)
		}
	}

	sb.WriteString("## Student Question\n\n")
	sb.WriteString(question)
	sb.WriteString("\n\n")
	sb.WriteString("## Instructions\n")
	sb.WriteString("- Answer based primarily on the attached PDF\n")
	sb.WriteString("- Answer in the same language as the question\n")
	sb.WriteString("- Be concise but thorough; cite page or section when possible\n")
	sb.WriteString("- Do NOT fabricate information not present in the PDF\n")

	return sb.String()
}
