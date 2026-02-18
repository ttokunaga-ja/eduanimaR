// Package main は eduanimaR Professor サービスのエントリーポイント。
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	grpcadapter "github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/grpc"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/http/handlers"
	httpmw "github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/http/middleware"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/llm"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/messaging"
	pgadapter "github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/postgres"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/storage"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/config"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/usecases"
)

func main() {
	// ─── ロガー設定 ───────────────────────────────────────────
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ─── 設定読み込み ─────────────────────────────────────────
	cfg := config.Load()
	slog.Info("config loaded", "port", cfg.Port)

	// ─── ルートコンテキスト（アダプタのライフタイム用） ──────
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// ─── DB 接続 ──────────────────────────────────────────────
	db, err := connectDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	// ─── MinIO アダプター ─────────────────────────────────────
	objectStorage, err := storage.NewMinioAdapter(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.MinioUseSSL,
	)
	if err != nil {
		slog.Error("failed to connect to minio", "error", err)
		os.Exit(1)
	}
	slog.Info("minio connected", "bucket", cfg.MinioBucket)

	// ─── Kafka プロデューサー ─────────────────────────────────
	publisher := messaging.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer publisher.Close()
	slog.Info("kafka producer initialized", "brokers", cfg.KafkaBrokers)

	// ─── Kafka コンシューマー（Ingest Worker 用） ─────────────
	consumer := messaging.NewKafkaConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, "professor-ingest-worker")
	defer consumer.Close()
	slog.Info("kafka consumer initialized")

	// ─── Gemini LLM クライアント ──────────────────────────────
	llmClient, err := llm.NewGeminiClient(rootCtx, cfg.GeminiAPIKey)
	if err != nil {
		slog.Error("failed to create gemini client", "error", err)
		os.Exit(1)
	}
	slog.Info("gemini client initialized")

	// ─── Librarian gRPC クライアント ──────────────────────────
	librarianClient, err := grpcadapter.NewLibrarianClient(cfg.LibrarianAddr)
	if err != nil {
		slog.Error("failed to connect to librarian", "error", err, "addr", cfg.LibrarianAddr)
		os.Exit(1)
	}
	slog.Info("librarian client connected", "addr", cfg.LibrarianAddr)

	// ─── リポジトリ ───────────────────────────────────────────
	subjectRepo := pgadapter.NewSubjectRepo(db)
	fileRepo := pgadapter.NewFileRepo(db)
	ingestJobRepo := pgadapter.NewIngestJobRepo(db)
	chunkRepo := pgadapter.NewChunkRepo(db)
	qaSessionRepo := pgadapter.NewQASessionRepo(db)

	// ─── ユースケース ─────────────────────────────────────────
	subjectUC := usecases.NewSubjectUseCase(subjectRepo)
	materialUC := usecases.NewMaterialUseCase(fileRepo, ingestJobRepo, objectStorage, publisher, subjectRepo)
	chatUC := usecases.NewChatUseCase(subjectRepo, qaSessionRepo, chunkRepo, llmClient, librarianClient)
	ingestUC := usecases.NewIngestUseCase(fileRepo, ingestJobRepo, chunkRepo, objectStorage, llmClient)

	// ─── Echo サーバー設定 ────────────────────────────────────
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// グローバルミドルウェア
	e.Use(echomw.RequestID())
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.CORS())
	e.Use(httpmw.DevUser()) // Phase 1: 固定 dev user を設定

	// ─── ルーティング ─────────────────────────────────────────
	// ヘルスチェック (認証不要)
	e.GET("/healthz", handlers.Healthz)

	// API v1 グループ
	v1 := e.Group("/api/v1")

	// 科目 API
	subjectH := handlers.NewSubjectHandler(subjectUC)
	subjectH.Register(v1.Group("/subjects"))

	// 教材 API (/api/v1/subjects/:subject_id/materials)
	materialH := handlers.NewMaterialHandler(materialUC)
	materialH.Register(v1.Group("/subjects/:subject_id/materials"))

	// チャット API (/api/v1/subjects/:subject_id/chats)
	chatH := handlers.NewChatHandler(chatUC)
	chatH.Register(v1.Group("/subjects/:subject_id/chats"))

	// ─── Kafka Ingest Worker goroutine ───────────────────────
	go func() {
		if err := consumer.ConsumeIngestJobs(rootCtx, ingestUC.ProcessJob); err != nil {
			slog.Error("kafka consumer stopped unexpectedly", "error", err)
		}
	}()

	// ─── グレースフルシャットダウン ──────────────────────────
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCtx.Done()
	slog.Info("shutdown signal received")
	rootCancel() // Gemini クライアントなど rootCtx 依存のリソースを解放

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}

// connectDB は DSN から pgx/v5 を使用して *sql.DB を返す。
func connectDB(dsn string) (*sql.DB, error) {
	pgxCfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	db := stdlib.OpenDB(*pgxCfg)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
