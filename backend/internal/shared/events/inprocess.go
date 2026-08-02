package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// InProcessBus is an in-memory event bus using Go channels.
// Suitable for Phase 1 / testing / single-instance deployments.
type InProcessBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	logger   *slog.Logger
	ch       chan inProcessEnvelope
	done     chan struct{}
}

type inProcessEnvelope struct {
	ctx   context.Context
	topic string
	event Event
}

// Verify interface compliance at compile time.
var _ EventBus = (*InProcessBus)(nil)

// NewInProcessBus creates a new in-process event bus.
func NewInProcessBus(logger *slog.Logger, bufferSize int) *InProcessBus {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	b := &InProcessBus{
		handlers: make(map[string][]Handler),
		logger:   logger,
		ch:       make(chan inProcessEnvelope, bufferSize),
		done:     make(chan struct{}),
	}
	go b.dispatch()
	return b
}

// publishTimeout bounds how long Publish waits for room in the buffer.
//
// A dropped event is not a cosmetic loss: ticket.created is what triggers
// classification, so a drop leaves a ticket permanently unanalyzed with nothing
// recording why. Waiting briefly absorbs bursts, which is what a full buffer
// almost always is. The bound keeps a stalled dispatcher from blocking the
// request that is publishing.
const publishTimeout = 5 * time.Second

// Publish sends an event to all registered handlers asynchronously.
//
// It returns an error when the event could not be queued. Callers that ignore
// the error accept the loss; they no longer do so unknowingly.
func (b *InProcessBus) Publish(ctx context.Context, topic string, event Event) error {
	envelope := inProcessEnvelope{ctx: ctx, topic: topic, event: event}

	select {
	case b.ch <- envelope:
		return nil
	default:
	}

	timer := time.NewTimer(publishTimeout)
	defer timer.Stop()

	select {
	case b.ch <- envelope:
		return nil
	case <-timer.C:
		b.logger.Error("in-process event bus buffer full, event dropped",
			"topic", topic, "waited", publishTimeout)
		return fmt.Errorf("in-process event bus buffer full: dropped event on topic %q", topic)
	}
}

// Subscribe registers a handler for a topic.
func (b *InProcessBus) Subscribe(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
	b.logger.Debug("in-process handler registered", "topic", topic)
}

// Close shuts down the event bus.
func (b *InProcessBus) Close() error {
	close(b.ch)
	<-b.done
	return nil
}

func (b *InProcessBus) dispatch() {
	defer close(b.done)
	for env := range b.ch {
		b.mu.RLock()
		handlers := b.handlers[env.topic]
		b.mu.RUnlock()

		for _, h := range handlers {
			if err := h(env.ctx, env.event); err != nil {
				b.logger.Error("in-process handler error", "topic", env.topic, "error", err)
			}
		}
	}
}
