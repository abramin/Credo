package consumer

import (
	"context"
	"log/slog"

	"credo/internal/platform/kafka/consumer"
)

// TopicHandler handles messages from a specific topic.
type TopicHandler interface {
	Handle(ctx context.Context, msg *consumer.Message) error
}

// Router dispatches messages to topic-specific handlers.
// Use this when consuming from multiple audit topics.
type Router struct {
	handlers map[string]TopicHandler
	fallback TopicHandler
	logger   *slog.Logger
}

// NewRouter creates a topic router with an optional fallback handler.
func NewRouter(logger *slog.Logger, fallback TopicHandler) *Router {
	return &Router{
		handlers: make(map[string]TopicHandler),
		fallback: fallback,
		logger:   logger,
	}
}

// Register adds a handler for a specific topic.
func (r *Router) Register(topic string, handler TopicHandler) {
	r.handlers[topic] = handler
}

// Handle routes the message to the appropriate topic handler.
func (r *Router) Handle(ctx context.Context, msg *consumer.Message) error {
	handler, ok := r.handlers[msg.Topic]
	if !ok {
		if r.fallback != nil {
			return r.fallback.Handle(ctx, msg)
		}
		r.logger.Warn("no handler for topic, skipping message",
			"topic", msg.Topic,
			"key", string(msg.Key),
		)
		return nil // Commit to avoid redelivery
	}
	return handler.Handle(ctx, msg)
}
