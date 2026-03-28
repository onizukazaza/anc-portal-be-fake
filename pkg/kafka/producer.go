package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/retry"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Producer struct {
	writer *kafka.Writer
}

type ProducerConfig struct {
	Brokers      []string
	Topic        string
	WriteTimeout time.Duration // default: 10s
}

const defaultWriteTimeout = 10 * time.Second

func NewProducer(cfg ProducerConfig) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka brokers is required")
	}
	if cfg.Topic == "" {
		return nil, errors.New("kafka topic is required")
	}

	writeTimeout := cfg.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = defaultWriteTimeout
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		WriteTimeout: writeTimeout,
	}

	return &Producer{writer: writer}, nil
}

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	ctx, span := appOtel.Tracer(appOtel.TracerKafka).Start(ctx, p.writer.Topic+" publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", p.writer.Topic),
			attribute.String("messaging.operation.name", "publish"),
		),
	)
	defer span.End()

	if key != "" {
		span.SetAttributes(attribute.String("messaging.kafka.message.key", key))
	}

	// Inject W3C trace context เข้า Kafka headers → consumer จะ extract ได้
	headers := injectTraceHeaders(ctx)

	err := retry.Do(ctx, func(ctx context.Context) error {
		return p.writer.WriteMessages(ctx, kafka.Message{
			Key:     []byte(key),
			Value:   value,
			Headers: headers,
			Time:    time.Now(),
		})
	}, retry.MaxAttempts(3), retry.Backoff(1*time.Second), retry.WithBackoffFunc(retry.ConstantBackoff))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (p *Producer) PublishMessage(ctx context.Context, msg Message) error {
	if err := msg.Validate(); err != nil {
		return err
	}

	value, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal kafka envelope: %w", err)
	}

	return p.Publish(ctx, msg.Key, value)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
