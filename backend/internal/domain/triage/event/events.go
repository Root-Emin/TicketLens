package event

import (
	"time"

	"github.com/google/uuid"
)

// TicketCreated is emitted when a new ticket is raised.
//
// Subject and Body are carried on the event so the classification consumer can
// call the classifier without reading back from the database. Classification is
// never done inline in the HTTP handler: a cold model service can take 30+
// seconds to wake and that must not block ticket creation.
type TicketCreated struct {
	TicketID       uuid.UUID `json:"ticket_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	CustomerID     uuid.UUID `json:"customer_id"`
	DepartmentID   uuid.UUID `json:"department_id"`
	Subject        string    `json:"subject"`
	Body           string    `json:"body"`
	Timestamp      time.Time `json:"timestamp"`
}

// TicketUpdated is emitted when a ticket's status, priority or department changes.
type TicketUpdated struct {
	TicketID       uuid.UUID `json:"ticket_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Timestamp      time.Time `json:"timestamp"`
}

// TicketAssigned is emitted when a ticket's assignee changes.
type TicketAssigned struct {
	TicketID       uuid.UUID  `json:"ticket_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	AssigneeID     *uuid.UUID `json:"assignee_id,omitempty"`
	Timestamp      time.Time  `json:"timestamp"`
}

// AnalysisCompleted is emitted after a classifier run is persisted.
//
// Frontends subscribe via WebSocket to refresh the AI panel without polling.
type AnalysisCompleted struct {
	TicketID         uuid.UUID `json:"ticket_id"`
	OrganizationID   uuid.UUID `json:"organization_id"`
	AnalysisID       uuid.UUID `json:"analysis_id"`
	NeedsHumanReview bool      `json:"needs_human_review"`
	ModelName        string    `json:"model_name"`
	ModelVersion     string    `json:"model_version"`
	Timestamp        time.Time `json:"timestamp"`
}
