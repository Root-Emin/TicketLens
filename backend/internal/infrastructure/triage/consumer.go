// Package triage wires domain events onto the triage use cases.
package triage

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	triageUC "github.com/masterfabric-go/masterfabric/internal/application/triage/usecase"
	triageEvent "github.com/masterfabric-go/masterfabric/internal/domain/triage/event"
	"github.com/masterfabric-go/masterfabric/internal/shared/events"
)

// TicketConsumer classifies newly created tickets off the event bus.
//
// Classification never runs inline in the HTTP handler: a cold model service
// can take 30+ seconds to wake, and ticket creation must not wait for it.
type TicketConsumer struct {
	analyzeUC *triageUC.AnalyzeTicketUseCase
	logger    *slog.Logger
}

// NewTicketConsumer creates a new TicketConsumer.
func NewTicketConsumer(analyzeUC *triageUC.AnalyzeTicketUseCase, logger *slog.Logger) *TicketConsumer {
	return &TicketConsumer{analyzeUC: analyzeUC, logger: logger}
}

// Register subscribes the consumer to the triage topic.
func (c *TicketConsumer) Register(bus events.EventBus) {
	bus.Subscribe(events.TopicTriage, c.handle)
}

func (c *TicketConsumer) handle(ctx context.Context, event events.Event) error {
	payload, ok := decodeTicketCreated(event)
	if !ok {
		return nil // not a ticket.created event
	}

	// At-least-once delivery means this can fire twice for the same ticket.
	// An already classified ticket is skipped; the manual re-run endpoint goes
	// straight to the use case and bypasses this check on purpose.
	if c.analyzeUC.HasAnalysis(ctx, payload.OrganizationID, payload.TicketID) {
		c.logger.Debug("ticket already analyzed, skipping",
			"ticket_id", payload.TicketID)
		return nil
	}

	if _, err := c.analyzeUC.Execute(ctx, payload.OrganizationID, payload.TicketID); err != nil {
		// A classification failure must not damage the ticket. It simply stays
		// unanalyzed and the UI renders latest_analysis as null.
		c.logger.Error("ticket classification failed",
			"ticket_id", payload.TicketID,
			"organization_id", payload.OrganizationID,
			"error", err,
		)
		return nil
	}

	c.logger.Info("ticket classified", "ticket_id", payload.TicketID)
	return nil
}

// decodeTicketCreated copes with both shapes the bus can deliver.
//
// The in-process bus hands over the original typed struct, while the Kafka bus
// hands over an *events.Envelope whose Data holds the JSON. Supporting both
// keeps the consumer working whether or not KAFKA_ENABLED is set.
func decodeTicketCreated(event events.Event) (triageEvent.TicketCreated, bool) {
	switch e := event.(type) {
	case triageEvent.TicketCreated:
		return e, true
	case *triageEvent.TicketCreated:
		return *e, true
	case *events.Envelope:
		if e.Type != events.EventTypeTicketCreated {
			return triageEvent.TicketCreated{}, false
		}
		var payload triageEvent.TicketCreated
		if err := json.Unmarshal(e.Data, &payload); err != nil {
			return triageEvent.TicketCreated{}, false
		}
		if payload.TicketID == uuid.Nil || payload.OrganizationID == uuid.Nil {
			return triageEvent.TicketCreated{}, false
		}
		return payload, true
	default:
		return triageEvent.TicketCreated{}, false
	}
}
