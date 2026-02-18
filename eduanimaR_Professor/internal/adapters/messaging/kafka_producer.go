// Package messaging は Kafka メッセージングアダプターを提供する。
package messaging

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

type kafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer は Kafka MessagePublisher 実装を返す。
// brokers: カンマ区切りのブローカーアドレス（例: "localhost:9092"）
func NewKafkaProducer(brokers, topic string) ports.MessagePublisher {
	return &kafkaProducer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
	}
}

// PublishIngestJob は IngestMessage を Kafka に送信する。
// メッセージキーは JobID（パーティション分散用）。
func (p *kafkaProducer) PublishIngestJob(ctx context.Context, msg ports.IngestMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.JobID),
		Value: b,
	})
}

// Close は Kafka Writer を閉じる。
func (p *kafkaProducer) Close() error {
	return p.writer.Close()
}
