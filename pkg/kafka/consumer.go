package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/retry"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Consumer อ่าน message จาก Kafka topic แล้วส่งให้ handler ประมวลผล
// รองรับ Dead Letter Queue (DLQ) สำหรับ message ที่ fail ซ้ำเกิน MaxRetries
type Consumer struct {
	reader  *kafka.Reader
	dlq     *Producer
	cfg     ConsumerConfig
	healthy atomic.Bool // ถูก set เป็น true เมื่อ StartMessages เริ่ม fetch ได้สำเร็จ
}

// ConsumerConfig ตั้งค่า consumer + DLQ
type ConsumerConfig struct {
	Brokers    []string
	Topic      string
	GroupID    string
	MaxRetries int    // จำนวน retry ก่อนส่ง DLQ (default: 3)
	DLQTopic   string // topic สำหรับ dead letter (ว่าง = ปิด DLQ)
	MaxBytes   int    // max message bytes per fetch (default: 10 MB)
}

const defaultMaxBytes = 10 * 1024 * 1024 // 10 MB

func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka brokers is required")
	}
	if cfg.Topic == "" {
		return nil, errors.New("kafka topic is required")
	}
	if cfg.GroupID == "" {
		return nil, errors.New("kafka group id is required")
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}

	maxBytes := cfg.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       maxBytes,
		CommitInterval: time.Second,
	})

	c := &Consumer{
		reader: reader,
		cfg:    cfg,
	}

	// สร้าง DLQ producer ถ้ากำหนด DLQTopic
	if cfg.DLQTopic != "" {
		dlq, err := NewProducer(ProducerConfig{
			Brokers: cfg.Brokers,
			Topic:   cfg.DLQTopic,
		})
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("init DLQ producer: %w", err)
		}
		c.dlq = dlq
	}

	return c, nil
}

// MessageHandler ฟังก์ชันที่ consumer เรียกเมื่อได้รับ message
type MessageHandler func(ctx context.Context, msg Message) error

// StartMessages อ่าน message วนลูปจนกว่า context จะ cancel
// ถ้า handler return error → retry ตาม MaxRetries → ส่ง DLQ ถ้ายัง fail
func (c *Consumer) StartMessages(ctx context.Context, handler MessageHandler) error {
	for {
		raw, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("fetch kafka message: %w", err)
		}

		// mark healthy after first successful fetch
		c.healthy.Store(true)

		c.processMessage(ctx, raw, handler)
	}
}

// processMessage จัดการ message 1 ตัว พร้อม tracing
// แยกออกจาก loop เพื่อให้ defer span.End() ทำงานถูกต้องทุก message
func (c *Consumer) processMessage(ctx context.Context, raw kafka.Message, handler MessageHandler) {
	// Extract trace context ที่ producer inject มา → เชื่อม trace ข้าม service
	msgCtx := extractTraceContext(ctx, raw.Headers)
	msgCtx, span := appOtel.Tracer(appOtel.TracerKafka).Start(msgCtx, c.cfg.Topic+" process",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", c.cfg.Topic),
			attribute.String("messaging.operation.name", "process"),
			attribute.String("messaging.kafka.consumer.group", c.cfg.GroupID),
			attribute.Int64("messaging.kafka.message.offset", raw.Offset),
			attribute.Int("messaging.kafka.partition", raw.Partition),
		),
	)
	defer span.End()

	msg, decodeErr := DecodeMessage(raw.Value)
	if decodeErr != nil {
		span.RecordError(decodeErr)
		span.SetStatus(codes.Error, "decode_error")
		c.sendToDLQ(msgCtx, raw.Value, "decode_error", decodeErr)
		_ = c.reader.CommitMessages(ctx, raw)
		return
	}

	span.SetAttributes(
		attribute.String("messaging.kafka.message.key", msg.Key),
		attribute.String("kafka.event_type", msg.Type),
	)

	if err := c.processWithRetry(msgCtx, handler, msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.sendToDLQ(msgCtx, raw.Value, msg.Type, err)
	}

	_ = c.reader.CommitMessages(ctx, raw)
}

// processWithRetry retry handler ตาม MaxRetries ก่อนยอมแพ้
func (c *Consumer) processWithRetry(ctx context.Context, handler MessageHandler, msg Message) error {
	return retry.Do(ctx, func(ctx context.Context) error {
		return handler(ctx, msg)
	}, retry.MaxAttempts(c.cfg.MaxRetries), retry.Backoff(1*time.Second))
}

// sendToDLQ ส่ง message ที่ fail ไปยัง DLQ topic
func (c *Consumer) sendToDLQ(ctx context.Context, originalValue []byte, eventType string, handlerErr error) {
	if c.dlq == nil {
		return
	}

	dlqMsg, err := NewMessage(
		"dlq."+eventType,
		"",
		dlqPayload{
			OriginalMessage: string(originalValue),
			Error:           handlerErr.Error(),
			Retries:         c.cfg.MaxRetries,
			FailedAt:        time.Now().UTC(),
		},
		map[string]string{
			"source_topic": c.cfg.Topic,
			"error":        handlerErr.Error(),
		},
	)
	if err != nil {
		log.L().Error().Err(err).Str("topic", c.cfg.Topic).Str("event_type", eventType).
			Msg("kafka: failed to build DLQ message, message dropped")
		return
	}

	if err := c.dlq.PublishMessage(ctx, dlqMsg); err != nil {
		log.L().Error().Err(err).Str("dlq_topic", c.cfg.DLQTopic).Str("event_type", eventType).
			Msg("kafka: failed to publish to DLQ, message dropped")
	}
}

type dlqPayload struct {
	OriginalMessage string    `json:"originalMessage"`
	Error           string    `json:"error"`
	Retries         int       `json:"retries"`
	FailedAt        time.Time `json:"failedAt"`
}

// IsHealthy คืน true เมื่อ consumer กำลัง consume อยู่ (fetch ได้อย่างน้อย 1 ครั้ง)
func (c *Consumer) IsHealthy() bool {
	return c.healthy.Load()
}

// Close ปิด consumer + DLQ producer
func (c *Consumer) Close() error {
	c.healthy.Store(false)
	var errs []error
	if err := c.reader.Close(); err != nil {
		errs = append(errs, err)
	}
	if c.dlq != nil {
		if err := c.dlq.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
