package kafka

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMessageTypeRequired = errors.New("kafka message type is required")
	ErrMessagePayloadEmpty = errors.New("kafka message payload is required")
)

type Message struct {
	Type       string            `json:"type"`
	Key        string            `json:"key,omitempty"`
	Payload    json.RawMessage   `json:"payload"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	OccurredAt time.Time         `json:"occurredAt"`
}

func NewMessage(eventType string, key string, payload any, metadata map[string]string) (Message, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("marshal kafka payload: %w", err)
	}

	msg := Message{
		Type:       strings.TrimSpace(eventType),
		Key:        strings.TrimSpace(key),
		Payload:    encodedPayload,
		Metadata:   cloneMetadata(metadata),
		OccurredAt: time.Now().UTC(),
	}

	if err := msg.Validate(); err != nil {
		return Message{}, err
	}

	return msg, nil
}

func DecodeMessage(data []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, fmt.Errorf("decode kafka message: %w", err)
	}

	if err := msg.Validate(); err != nil {
		return Message{}, err
	}

	return msg, nil
}

func (m Message) Bytes() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal kafka message: %w", err)
	}

	return data, nil
}

func (m Message) Validate() error {
	if strings.TrimSpace(m.Type) == "" {
		return ErrMessageTypeRequired
	}
	if len(m.Payload) == 0 {
		return ErrMessagePayloadEmpty
	}
	if m.OccurredAt.IsZero() {
		return errors.New("kafka message occurredAt is required")
	}
	return nil
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}
