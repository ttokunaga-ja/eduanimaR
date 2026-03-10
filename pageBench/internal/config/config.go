// Package config provides configuration loading from .env files.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AgentConfig は Agent システム（RAG バックエンド）の接続設定。
// OpenAI互換 model フィールドで品質レベル（professor-fast/professor/professor-pro）を指定できる。
type AgentConfig struct {
	APIBase string
	APIKey  string
	Model   string
}

// UploadConfig は Evaluation Preparation（RAG インデックス作成）の設定。
// OpenAI Files API 互換の POST /files + GET /files/{id} を使用する。
type UploadConfig struct {
	Purpose          string // AGENT_UPLOAD_PURPOSE: POST /files の purpose フィールド
	FileStatusReady  string // AGENT_FILE_STATUS_READY: インデックス完了とみなす status 値
	TimeoutSecs      int    // AGENT_INDEX_TIMEOUT_SECS: 完了待機タイムアウト（秒）。0=無制限
	PollIntervalSecs int    // AGENT_INDEX_POLL_INTERVAL_SECS: ポーリング間隔（秒）
}

// GeminiConfig は Gemini Judge + Generate の設定。
type GeminiConfig struct {
	APIKey           string
	JudgeModel       string
	GenerateModel    string
	ThinkingLevel    string // "minimal"(Flash系のみ) | "low" | "medium" | "high"
	RateLimitSleepMs int
}

// QAConfig は QA 動的生成の設定。
type QAConfig struct {
	Min     int     // PAGEBENCH_QA_MIN: 最低 QA 件数
	Max     int     // PAGEBENCH_QA_MAX: 最大 QA 件数
	Density float64 // PAGEBENCH_QA_DENSITY: 1ページあたりの QA 件数（0 = 動的計算無効、--qa-per-doc を使用）
}

// Config は pageBench 全体の設定。
type Config struct {
	Agent         AgentConfig
	Upload        UploadConfig
	Gemini        GeminiConfig
	QA            QAConfig
	TargetDomains []string // PAGEBENCH_TARGET_DOMAINS: デフォルト実行対象のドメインパス一覧

	// フェーズ実行制御（false のときそのフェーズをスキップ）
	ExecuteRegistry              bool // PAGEBENCH_EXECUTE_REGISTRY:              generate: 0a_registry.csv を生成
	ExecuteQA                    bool // PAGEBENCH_EXECUTE_QA:                    generate: 0b_qa_pairs.csv を生成
	ExecuteEvaluationPreparation bool // PAGEBENCH_EXECUTE_EVALUATION_PREPARATION: prepare: ファイルアップロード + インデックス作成
	ExecuteEvaluation            bool // PAGEBENCH_EXECUTE_EVALUATION:            eval: 0c_evaluation.csv + 0d_evaluation_report.md を生成
}

// BackendDisplay はログ表示用のバックエンド情報文字列を返す。
func (c *Config) BackendDisplay() string {
	return fmt.Sprintf("agent @ %s", c.Agent.APIBase)
}

// Load は環境変数（.env ロード済み前提）から Config を生成して返す。
// godotenv.Load() は cmd 側の PersistentPreRunE で呼ぶ。
func Load() (*Config, error) {
	rateLimitMs, err := strconv.Atoi(getEnvDefault("GEMINI_RATE_LIMIT_SLEEP_MS", "3000"))
	if err != nil {
		return nil, fmt.Errorf("GEMINI_RATE_LIMIT_SLEEP_MS の値が不正です: %w", err)
	}

	// ThinkingLevel: minimal(Flash系のみ) | low | medium | high
	// ※ minimal は gemini-3-flash-preview / gemini-3.1-flash-lite-preview 専用
	// ※ Pro 系モデルでは low が最低レベル（minimal 不可）
	thinkingLevel := getEnvDefault("GEMINI_THINKING_LEVEL", "minimal")
	switch thinkingLevel {
	case "minimal", "low", "medium", "high":
	default:
		return nil, fmt.Errorf("GEMINI_THINKING_LEVEL=%q は無効です。minimal/low/medium/high を指定してください", thinkingLevel)
	}

	// PAGEBENCH_TARGET_DOMAINS: カンマ区切りのドメインパス一覧（空白トリム）
	var targetDomains []string
	if raw := os.Getenv("PAGEBENCH_TARGET_DOMAINS"); raw != "" {
		for _, d := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(d); trimmed != "" {
				targetDomains = append(targetDomains, trimmed)
			}
		}
	}

	// フェーズ実行制御（未設定または "true" → 実行する）
	execRegistry := parseBoolDefault("PAGEBENCH_EXECUTE_REGISTRY", true)
	execQA := parseBoolDefault("PAGEBENCH_EXECUTE_QA", true)
	execEvalPrep := parseBoolDefault("PAGEBENCH_EXECUTE_EVALUATION_PREPARATION", true)
	execEval := parseBoolDefault("PAGEBENCH_EXECUTE_EVALUATION", true)

	// QA 動的生成設定
	qaMin, err := strconv.Atoi(getEnvDefault("PAGEBENCH_QA_MIN", "3"))
	if err != nil || qaMin < 1 {
		qaMin = 3
	}
	qaMax, err := strconv.Atoi(getEnvDefault("PAGEBENCH_QA_MAX", "20"))
	if err != nil || qaMax < qaMin {
		qaMax = 20
	}
	var qaDensity float64
	if v := os.Getenv("PAGEBENCH_QA_DENSITY"); v != "" {
		if d, err := strconv.ParseFloat(v, 64); err == nil && d > 0 {
			qaDensity = d
		}
	}

	// UploadConfig 設定（Evaluation Preparation 用）
	indexTimeoutSecs := 300
	if v := os.Getenv("AGENT_INDEX_TIMEOUT_SECS"); v != "" {
		if n, err2 := strconv.Atoi(v); err2 == nil && n >= 0 {
			indexTimeoutSecs = n
		}
	}
	indexPollIntervalSecs := 5
	if v := os.Getenv("AGENT_INDEX_POLL_INTERVAL_SECS"); v != "" {
		if n, err2 := strconv.Atoi(v); err2 == nil && n >= 1 {
			indexPollIntervalSecs = n
		}
	}

	return &Config{
		TargetDomains:                targetDomains,
		ExecuteRegistry:              execRegistry,
		ExecuteQA:                    execQA,
		ExecuteEvaluationPreparation: execEvalPrep,
		ExecuteEvaluation:            execEval,
		QA: QAConfig{
			Min:     qaMin,
			Max:     qaMax,
			Density: qaDensity,
		},
		Agent: AgentConfig{
			APIBase: getEnvDefault("AGENT_API_BASE", ""),
			APIKey:  os.Getenv("AGENT_API_KEY"),
			Model:   getEnvDefault("PAGEBENCH_MODEL", "professor"),
		},
		Upload: UploadConfig{
			Purpose:          getEnvDefault("AGENT_UPLOAD_PURPOSE", "assistants"),
			FileStatusReady:  getEnvDefault("AGENT_FILE_STATUS_READY", "processed"),
			TimeoutSecs:      indexTimeoutSecs,
			PollIntervalSecs: indexPollIntervalSecs,
		},
		Gemini: GeminiConfig{
			APIKey:           os.Getenv("GEMINI_API_KEY"),
			JudgeModel:       getEnvDefault("GEMINI_JUDGE_MODEL", "gemini-3-flash-preview"),
			GenerateModel:    getEnvDefault("GEMINI_GENERATE_MODEL", "gemini-3-flash-preview"),
			ThinkingLevel:    thinkingLevel,
			RateLimitSleepMs: rateLimitMs,
		},
	}, nil
}

func getEnvDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// parseBoolDefault は環境変数を bool に変換する。未設定の場合は defaultVal を返す。
// "false", "0", "no" → false。それ以外（"true", "1", "yes", ""等）→ defaultVal。
func parseBoolDefault(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0", "no":
		return false
	default:
		return true
	}
}
