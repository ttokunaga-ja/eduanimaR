package messaging

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

type kafkaConsumer struct {
	reader *kafka.Reader
}

// NewKafkaConsumer は Kafka MessageConsumer 実装を返す。
// brokers: カンマ区切りのブローカーアドレス（例: "localhost:9092"）
// topic: 読み取りトピック
// groupID: コンシューマーグループ ID（複数インスタンス時のオフセット管理）
func NewKafkaConsumer(brokers, topic, groupID string) ports.MessageConsumer {
	return &kafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{brokers},
			Topic:       topic,
			GroupID:     groupID,
			MinBytes:    1,        // 1 B - 低レイテンシ優先
			MaxBytes:    10 << 20, // 10 MB - 最大ファイルメタデータサイズ
			StartOffset: kafka.FirstOffset,
		}),
	}
}

// ConsumeIngestJobs はメッセージを継続的に受信し、handler を呼び出す。
// ctx がキャンセルされるとグレースフルに終了する。
func (c *kafkaConsumer) ConsumeIngestJobs(
	ctx context.Context,
	handler func(ctx context.Context, msg ports.IngestMessage) error,
) error {
	slog.Info("kafka consumer started")

	for {
		// ReadMessage は auto-commit される（at-least-once 保証）
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			// コンテキストキャンセル → グレースフルシャットダウン
			if ctx.Err() != nil {
				slog.Info("kafka consumer shutting down")
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
			continue
		}

		slog.Info("kafka message received",
			"job_id", msg.JobID,
			"file_id", msg.FileID,
			"mime_type", msg.MimeType,
		)

		if err := handler(ctx, msg); err != nil {
			// エラーはログのみ（IngestUseCase 側でステータスを failed に更新済み）
			slog.Error("ingest handler error",
				"job_id", msg.JobID,
				"error", err,
			)
		}
	}
}

// Close は Kafka Reader を閉じる。
func (c *kafkaConsumer) Close() error {
	return c.reader.Close()
}
