// Package llm は Gemini API を使った LLMClient の実装を提供する。
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

const (
	// embeddingModel は埋め込みベクトル生成モデル（768次元）
	embeddingModel = "text-embedding-004"
	// generationModel は OCR・回答生成に使用する高速推論モデル
	generationModel = "gemini-2.0-flash-lite"
)

// geminiClient は ports.LLMClient の Gemini API 実装。
type geminiClient struct {
	client *genai.Client
}

// NewGeminiClient は Gemini API クライアントを作成して ports.LLMClient を返す。
// ctx はクライアントのライフタイム用コンテキスト（通常は main の ctx）。
func NewGeminiClient(ctx context.Context, apiKey string) (ports.LLMClient, error) {
	c, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("gemini: new client: %w", err)
	}
	return &geminiClient{client: c}, nil
}

// ─── OCRAndChunk ──────────────────────────────────────────────────

// OCRAndChunk は PDF/画像ファイルのバイト列を受け取り、
// Markdown 化・意味単位チャンク分割を行う。
// チャンク区切りは "---CHUNK---" を使用する。
func (g *geminiClient) OCRAndChunk(ctx context.Context, fileContent []byte, mimeType string) (*ports.OCRResult, error) {
	model := g.client.GenerativeModel(generationModel)

	prompt := `You are an academic document processor.
Extract and structure ALL text content from this document.
Organize into logical semantic units (paragraphs, sections, slides, exercises, etc.)
separated by exactly "---CHUNK---" on its own line.

Rules:
- Preserve mathematical formulas, code snippets, and tables as text
- Each chunk should be self-contained and coherent
- Do NOT include page headers/footers as separate chunks
- Output ONLY the extracted content with chunk delimiters, no commentary`

	resp, err := model.GenerateContent(ctx,
		genai.Text(prompt),
		genai.Blob{MIMEType: mimeType, Data: fileContent},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini: ocr generate: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return &ports.OCRResult{}, nil
	}

	var fullText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if t, ok := part.(genai.Text); ok {
			fullText.WriteString(string(t))
		}
	}

	rawChunks := strings.Split(fullText.String(), "---CHUNK---")
	chunks := make([]ports.ChunkData, 0, len(rawChunks))
	idx := 0
	for _, raw := range rawChunks {
		content := strings.TrimSpace(raw)
		if content == "" {
			continue
		}
		chunks = append(chunks, ports.ChunkData{
			Index:   idx,
			Content: content,
		})
		idx++
	}

	return &ports.OCRResult{Chunks: chunks}, nil
}

// ─── GenerateEmbedding ────────────────────────────────────────────

// GenerateEmbedding はテキストの埋め込みベクトル（768次元）を生成する。
func (g *geminiClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	em := g.client.EmbeddingModel(embeddingModel)
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("gemini: embedding: %w", err)
	}
	return res.Embedding.Values, nil
}

// ─── GenerateAnswer ───────────────────────────────────────────────

// GenerateAnswer は選定済みエビデンスと質問から最終回答を生成する（非ストリーミング）。
func (g *geminiClient) GenerateAnswer(ctx context.Context, question string, evidences []string) (string, error) {
	model := g.client.GenerativeModel(generationModel)
	prompt := buildAnswerPrompt(question, evidences)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini: generate answer: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", nil
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if t, ok := part.(genai.Text); ok {
			sb.WriteString(string(t))
		}
	}
	return sb.String(), nil
}

// ─── GenerateAnswerStream ─────────────────────────────────────────

// GenerateAnswerStream は選定済みエビデンスと質問から回答をストリーミング生成する。
// onChunk コールバックに回答テキストを逐次的に渡す。
func (g *geminiClient) GenerateAnswerStream(ctx context.Context, question string, evidences []string, onChunk func(text string) error) error {
	model := g.client.GenerativeModel(generationModel)
	prompt := buildAnswerPrompt(question, evidences)

	iter := model.GenerateContentStream(ctx, genai.Text(prompt))
	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("gemini: stream answer: %w", err)
		}
		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			continue
		}
		for _, part := range resp.Candidates[0].Content.Parts {
			if t, ok := part.(genai.Text); ok {
				if err := onChunk(string(t)); err != nil {
					return err
				}
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
