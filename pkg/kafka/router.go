package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrKafkaHandlerNotFound = errors.New("kafka handler not found")

type Handler func(ctx context.Context, msg Message) error

type Router struct {
	handlers map[string]Handler
	fallback Handler
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]Handler),
	}
}

func (r *Router) Register(eventType string, handler Handler) error {
	normalizedType := strings.TrimSpace(eventType)
	if normalizedType == "" {
		return ErrMessageTypeRequired
	}
	if handler == nil {
		return errors.New("kafka handler is required")
	}
	if _, exists := r.handlers[normalizedType]; exists {
		return fmt.Errorf("kafka handler already registered: %s", normalizedType)
	}

	r.handlers[normalizedType] = handler
	return nil
}

func (r *Router) SetFallback(handler Handler) {
	r.fallback = handler
}

func (r *Router) Dispatch(ctx context.Context, msg Message) error {
	if err := msg.Validate(); err != nil {
		return err
	}

	if handler, exists := r.handlers[msg.Type]; exists {
		return handler(ctx, msg)
	}
	if r.fallback != nil {
		return r.fallback(ctx, msg)
	}

	return fmt.Errorf("%w: %s", ErrKafkaHandlerNotFound, msg.Type)
}
