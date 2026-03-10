package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

type kafkaConsumer struct {
	reader      *kafka.Reader
	workerCount int
}

// NewKafkaConsumer は Kafka MessageConsumer 実装を返す。
//
// brokers: カンマ区切りのブローカーアドレス（例: "localhost:9092"）
// topic: 読み取りトピック
// groupID: コンシューマーグループ ID（複数インスタンス時のオフセット管理）
// workerCount: 同時に処理するファイル数（並列 goroutine 数）
func NewKafkaConsumer(brokers, topic, groupID string, workerCount int) ports.MessageConsumer {
	if workerCount <= 0 {
		workerCount = 3
	}
	return &kafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{brokers},
			Topic:       topic,
			GroupID:     groupID,
			MinBytes:    1,        // 1 B - 低レイテンシ優先
			MaxBytes:    10 << 20, // 10 MB - 最大ファイルメタデータサイズ
			StartOffset: kafka.FirstOffset,
		}),
		workerCount: workerCount,
	}
}

// ConsumeIngestJobs はメッセージを継続的に受信し、workerCount 個の goroutine で並列処理する。
//
// 並列化の仕組み:
//   - メインループは FetchMessage でメッセージを1件ずつ取得（シリアル I/O）
//   - セマフォ（バッファ付きチャネル）で同時実行数を workerCount に制限
//   - 各 goroutine が独立して handler を実行し、完了後に自身のオフセットをコミット
//   - ctx キャンセル時はインフライト goroutine の完了を WaitGroup で待機してから終了
//
// at-least-once 保証:
//   - FetchMessage は auto-commit しない（手動コミット）
//   - goroutine が処理完了後に CommitMessages を呼び出す
//   - クラッシュ時はコミット前のオフセットから再配信されるため重複処理の可能性あり
//   - IngestUseCase 側のジョブステータス管理で重複を冪等に処理可能
func (c *kafkaConsumer) ConsumeIngestJobs(
	ctx context.Context,
	handler func(ctx context.Context, msg ports.IngestMessage) error,
) error {
	slog.Info("kafka consumer started", "worker_count", c.workerCount)

	// セマフォ: 同時実行数を workerCount に制限
	sem := make(chan struct{}, c.workerCount)
	var wg sync.WaitGroup

	for {
		// FetchMessage: auto-commit しない（goroutine 側で手動コミット）
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// コンテキストキャンセル → インフライトジョブの完了を待ってから終了
				slog.Info("kafka consumer shutting down, waiting for in-flight jobs...",
					"worker_count", c.workerCount,
				)
				wg.Wait()
				return nil
			}
			slog.Error("kafka read error", "error", err)
			continue
		}

		var msg ports.IngestMessage
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			slog.Error("kafka message unmarshal error",
				"offset", m.Offset,
				"error", err,
			)
			// 不正メッセージはコミットしてスキップ（無限リトライを防ぐ）
			if commitErr := c.reader.CommitMessages(ctx, m); commitErr != nil && ctx.Err() == nil {
				slog.Error("kafka commit error after unmarshal failure",
					"offset", m.Offset,
					"error", commitErr,
				)
			}
			continue
		}

		slog.Info("kafka message received",
			"job_id", msg.JobID,
			"file_id", msg.FileID,
			"mime_type", msg.MimeType,
		)

		// セマフォのスロットを確保（満杯の場合はここでブロック）
		// ctx キャンセル時はインフライトジョブを待って終了
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			slog.Info("kafka consumer context cancelled while waiting for worker slot, waiting for in-flight jobs...")
			wg.Wait()
			return nil
		}

		wg.Add(1)
		go func(km kafka.Message, im ports.IngestMessage) {
			defer wg.Done()
			defer func() { <-sem }() // スロット解放

			if err := handler(ctx, im); err != nil {
				// エラーはログのみ（IngestUseCase 側でステータスを failed に更新済み）
				slog.Error("ingest handler error",
					"job_id", im.JobID,
					"file_id", im.FileID,
					"error", err,
				)
			}

			// 処理完了後にオフセットをコミット
			// context.Background() を使用: shutdown 時も確実にコミットするため
			if commitErr := c.reader.CommitMessages(context.Background(), km); commitErr != nil {
				slog.Error("kafka commit error",
					"job_id", im.JobID,
					"offset", km.Offset,
					"error", commitErr,
				)
			}
		}(m, msg)
	}
}

// Close は Kafka Reader を閉じる。
func (c *kafkaConsumer) Close() error {
	return c.reader.Close()
}
