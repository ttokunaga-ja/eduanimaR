// Package config は環境変数からアプリケーション設定を読み込む。
package config

import "os"

// Config はアプリケーション全体の設定を保持する。
type Config struct {
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

	// Librarian gRPC サービス
	LibrarianAddr string
}

// Load は環境変数から Config を構築して返す。
func Load() *Config {
	return &Config{
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
		LibrarianAddr:  getEnv("LIBRARIAN_GRPC_ADDR", "localhost:50051"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
