// Package config は環境変数からアプリケーション設定を読み込む。
package config

import (
	"fmt"
	"os"
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

	// Gemini AI
	GeminiAPIKey string

	// Gemini モデル設定
	// Phase 1: OCR / チャンク分割
	ModelIngestion string
	// Phase 4: 最終回答生成
	ModelAnswer string

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
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://eduanima:eduanima_password@localhost:5432/eduanima_professor?sslmode=disable"),
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_ROOT_PASSWORD", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "eduanima-materials"),
		MinioUseSSL:    false,
		KafkaBrokers:   getEnv("KAFKA_BROKERS", "localhost:9094"),
		KafkaTopic:     getEnv("KAFKA_TOPIC_INGEST", "eduanima.ingest.jobs"),
		GeminiAPIKey:   getEnv("GEMINI_API_KEY", ""),
		ModelIngestion: getEnv("PROFESSOR_MODEL_INGESTION", "gemini-2.0-flash-lite"),
		// 先頭ダッシュは「本番非推奨」マーカーとして使われることがある。
		// 実際の Gemini API 呼び出し時に不正モデル名にならないよう除去する。
		ModelAnswer:       strings.TrimPrefix(getEnv("PROFESSOR_MODEL_ANSWER", "gemini-2.0-flash-lite"), "-"),
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
