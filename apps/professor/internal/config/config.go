// Package config は環境変数からアプリケーション設定を読み込む。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config はアプリケーション全体の設定を保持する。
type Config struct {
	// 実行環境: "development" | "production"
	// APP_ENV=production の場合は DevUser ミドルウェアが無効化される。
	AppEnv string

	// HTTP サーバー
	Port string

	// PostgreSQL
	DatabaseURL string

	// MinIO (Phase 1 で GCS の代替として使用)
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	// Kafka
	KafkaBrokers string
	KafkaTopic   string
	// KafkaWorkerCount は Kafka コンシューマーの並列 goroutine 数。
	// 複数ファイルを同時に ingestion 処理する。デフォルト: 3
	// 環境変数: PROFESSOR_KAFKA_WORKER_COUNT
	KafkaWorkerCount int

	// Gemini AI
	GeminiAPIKey string

	// Gemini モデル設定
	// Phase 1: OCR / チャンク分割
	ModelIngestion string
	// Phase 4: 最終回答生成（eduanima 標準モードで使用）
	// Level別の回答モデルは openai_chat_handler が直接 env を読む:
	//   PROFESSOR_MODEL_ANSWER_FAST  → eduanima-flash 用
	//   PROFESSOR_MODEL_ANSWER_PRO   → eduanima-pro 用
	ModelAnswer string

	// EmbeddingConcurrency は1ファイル内でチャンク Embedding を並列生成する数。
	// 大きくすると速くなるが Gemini API レート制限に注意。デフォルト: 5
	// 環境変数: PROFESSOR_EMBEDDING_CONCURRENCY
	EmbeddingConcurrency int

	// Librarian gRPC サービス
	LibrarianAddr string

	// 認証 (JWT / Firebase Auth)
	// APP_ENV=production では必須。Firebase プロジェクト ID を設定する。
	// 例: "my-project-12345"
	FirebaseProjectID string

	// OpenTelemetry (optional)
	// 空文字の場合は noop プロバイダーを使用。
	// 例: "http://otel-collector:4317" (OTLP gRPC)
	OtelEndpoint    string
	OtelServiceName string
}

// Load は環境変数から Config を構築して返す。
func Load() *Config {
	return &Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://eduanima:eduanima_password@localhost:5432/eduanima_professor?sslmode=disable"),
		MinioEndpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:       getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinioSecretKey:       getEnv("MINIO_ROOT_PASSWORD", "minioadmin"),
		MinioBucket:          getEnv("MINIO_BUCKET", "eduanima-materials"),
		MinioUseSSL:          false,
		KafkaBrokers:         getEnv("KAFKA_BROKERS", "localhost:9094"),
		KafkaTopic:           getEnv("KAFKA_TOPIC_INGEST", "eduanima.ingest.jobs"),
		KafkaWorkerCount:     getEnvInt("PROFESSOR_KAFKA_WORKER_COUNT", 3),
		GeminiAPIKey:         getEnv("GEMINI_API_KEY", ""),
		EmbeddingConcurrency: getEnvInt("PROFESSOR_EMBEDDING_CONCURRENCY", 5),
		ModelIngestion:       getEnv("PROFESSOR_MODEL_INGESTION", "gemini-3-flash-preview"),
		// 先頭ダッシュは「本番非推奨」マーカーとして使われることがある。
		// 実際の Gemini API 呼び出し時に不正モデル名にならないよう除去する。
		ModelAnswer:       strings.TrimPrefix(getEnv("PROFESSOR_MODEL_ANSWER", "gemini-3-flash-preview"), "-"),
		LibrarianAddr:     getEnv("LIBRARIAN_GRPC_ADDR", "localhost:50051"),
		FirebaseProjectID: getEnv("FIREBASE_PROJECT_ID", ""),
		OtelEndpoint:      getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OtelServiceName:   getEnv("OTEL_SERVICE_NAME", "eduanima-professor"),
	}
}

// Validate は起動に必須の設定項目を検証して最初に発見したエラーを返す。
func Validate(cfg *Config) error {
	if cfg.GeminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY is required")
	}
	if cfg.AppEnv == "production" && cfg.FirebaseProjectID == "" {
		return fmt.Errorf("FIREBASE_PROJECT_ID is required when APP_ENV=production")
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}
